// backend/cmd/api/main.go
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpin "narratives/internal/adapters/in/http"
	"narratives/internal/adapters/in/http/middleware"

	// ✅ SNS query (company boundary なし)
	snsquery "narratives/internal/application/query/sns"

	// ✅ Firestore repository adapter (implements domain/list.Repository)
	fs "narratives/internal/adapters/out/firestore"

	"narratives/internal/platform/di"
)

func main() {
	ctx := context.Background()

	// ─────────────────────────────────────────────────────────────
	// Log output: ファイル + stdout の両方に出す
	// ─────────────────────────────────────────────────────────────
	if f, err := os.OpenFile("debug-idtoken.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err == nil {
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
		log.Printf("[boot] log output = stdout + debug-idtoken.log")
	} else {
		log.Printf("[boot] WARN: could not open debug-idtoken.log: %v", err)
	}

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

		// RouterDeps logging
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

		// ✅ SNS Query（companyId 境界なしで status=listing を取る）
		// NewSNSListQuery は domain/list.Repository を要求するため、
		// Firestore Client ではなく ListRepositoryFS（adapter）を渡す。
		var snsListQuery *snsquery.SNSListQuery
		if cont.Firestore != nil {
			listRepo := fs.NewListRepositoryFS(cont.Firestore) // implements list.Repository
			snsListQuery = snsquery.NewSNSListQuery(listRepo)
		} else {
			log.Printf("[boot] WARN: Firestore client is NIL; sns list query will be nil")
		}

		// ✅ SNS routes（/sns/...）を top-level mux に登録（DIで組み立て）
		snsDeps := di.NewSNSDeps(
			deps.ListUC,
			snsListQuery,
		)
		di.RegisterSNSRoutes(mux, snsDeps)

		// ✅ Console/Admin routes（既存）
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
	// ─────────────────────────────────────────────────────────────
	handler := middleware.CORS(mux)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ─────────────────────────────────────────────────────────────
	// Graceful shutdown for Cloud Run
	// ─────────────────────────────────────────────────────────────
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
