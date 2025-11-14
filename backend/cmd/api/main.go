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
	"narratives/internal/platform/di"
)

func main() {
	ctx := context.Background()

	// ─────────────────────────────────────────────────────────────
	// 先に軽量ヘルスチェックを公開して、PORTがLISTENできる状態を確保
	// ─────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// ─────────────────────────────────────────────────────────────
	// 重い依存初期化（DIコンテナ）
	// 失敗しても /healthz は提供し続け、起動失敗を避ける
	// ─────────────────────────────────────────────────────────────
	var cont *di.Container
	if c, err := di.NewContainer(ctx); err != nil {
		log.Printf("[boot] WARN: di init failed: %v (serving /healthz only)", err)
	} else {
		cont = c
		defer cont.Close()

		// アプリ本体のルータをマウント
		router := httpin.NewRouter(cont.RouterDeps())
		mux.Handle("/", router)
	}

	// ─────────────────────────────────────────────────────────────
	// Port 解決: config → env:PORT → 8080
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

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux, // まず mux（/healthz を含む）を握らせる
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown: Cloud Run の SIGTERM/SIGINT を捕捉
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
