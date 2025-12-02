// backend/cmd/api/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpin "narratives/internal/adapters/in/http"
	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/platform/di"
)

func main() {
	ctx := context.Background()

	// ─────────────────────────────────────────────────────────────
	// Lightweight healthz first so PORT is LISTENed quickly
	// ─────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// ─────────────────────────────────────────────────────────────
	// DI container & heavy deps; keep /healthz even on failure
	// ─────────────────────────────────────────────────────────────
	var cont *di.Container
	if c, err := di.NewContainer(ctx); err != nil {
		log.Printf("[boot] WARN: di init failed: %v (serving /healthz only)", err)
	} else {
		cont = c
		defer cont.Close()

		// RouterDeps を取得して FirebaseAuth / MemberRepo の状態をログ出力
		deps := cont.RouterDeps()

		if deps.FirebaseAuth == nil {
			log.Printf("[boot] RouterDeps.FirebaseAuth is NIL")
		} else {
			log.Printf("[boot] RouterDeps.FirebaseAuth: %T", deps.FirebaseAuth)
		}

		if deps.MemberRepo == nil {
			log.Printf("[boot] RouterDeps.MemberRepo is NIL")
		} else {
			log.Printf("[boot] RouterDeps.MemberRepo: %T", deps.MemberRepo)
		}

		// Attach app router under "/"
		router := httpin.NewRouter(deps)
		mux.Handle("/", router)
	}

	// ─────────────────────────────────────────────────────────────
	// Port resolution: config → env:PORT → 8080
	// ─────────────────────────────────────────────────────────────
	port := ""
	if cont != nil && cont.Config.Port != "" {
		port = cont.Config.Port
	}
	if port == "" {
		if p := os.Getenv("PORT"); p != "" {
			port = p
		} else {
			port = "8080"
		}
	}

	// ─────────────────────────────────────────────────────────────
	// Global CORS wrapper (covers /healthz and app routes)
	//
	// 許可する Origin 自体は middleware.CORS 側で
	//   - https://narratives.jp
	//   - https://narratives-console-dev.web.app
	//   - https://narratives-development-26c2d.web.app
	// などを動的に判定している。
	// ─────────────────────────────────────────────────────────────
	handler := middleware.CORS(mux)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler, // CORS applied here (dynamic by Origin)
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown for Cloud Run
	idleConnsClosed := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		sig := <-c
		log.Printf("[boot] received signal: %v; shutting down...", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[boot] server shutdown error: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("[boot] listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[boot] server error: %v", err)
	}

	<-idleConnsClosed
	log.Printf("[boot] server stopped")
}
