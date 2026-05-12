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

	httpin "narratives/internal/adapters/in/http/console"
	"narratives/internal/adapters/in/http/middleware"

	consoleDI "narratives/internal/platform/di/console"
	introductionDI "narratives/internal/platform/di/introduction"
	mallDI "narratives/internal/platform/di/mall"
	shared "narratives/internal/platform/di/shared"
)

type closer interface {
	Close() error
}

func closeIfPossible(name string, value any) {
	if value == nil {
		return
	}

	c, ok := value.(closer)
	if !ok {
		return
	}

	log.Printf("[boot] closing %s resources...", name)
	if err := c.Close(); err != nil {
		log.Printf("[boot] %s close error: %v", name, err)
	}
}

func main() {
	ctx := context.Background()

	{
		logPath := "debug-idtoken.log"
		if _, ok := os.LookupEnv("K_SERVICE"); ok {
			logPath = "/tmp/debug-idtoken.log"
		}

		if f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err == nil {
			mw := io.MultiWriter(os.Stdout, f)
			log.SetOutput(mw)
			log.Printf("[boot] log output = stdout + %s", logPath)
		} else {
			log.Printf("[boot] WARN: could not open %s: %v (stdout only)", logPath, err)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	initCtx, cancelInit := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelInit()

	// ------------------------------------------------------------
	// Shared infra
	// ------------------------------------------------------------
	infra, err := shared.NewInfra(initCtx)
	if err != nil {
		log.Fatalf("[boot] shared infra init failed: %v", err)
	}
	defer closeIfPossible("shared infra", infra)

	// ------------------------------------------------------------
	// Console DI
	// ------------------------------------------------------------
	consoleCont, err := consoleDI.NewContainer(initCtx, infra)
	if err != nil {
		log.Fatalf("[boot] console di init failed: %v", err)
	}
	defer closeIfPossible("console container", consoleCont)

	deps := consoleCont.RouterDeps()
	log.Printf("[boot] console router deps built")

	// ------------------------------------------------------------
	// Build full mux BEFORE ListenAndServe
	// ------------------------------------------------------------
	fullMux := http.NewServeMux()

	fullMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// ------------------------------------------------------------
	// Mall routes
	// ------------------------------------------------------------
	if mallCont, err := mallDI.NewContainer(initCtx, infra); err != nil {
		log.Printf("[boot] WARN: mall di init failed: %v (mall routes disabled)", err)
	} else {
		mallDI.Register(fullMux, mallCont)
		log.Printf("[boot] mall routes registered")
	}

	// ------------------------------------------------------------
	// Introduction routes
	// ------------------------------------------------------------
	var introCont *introductionDI.Container

	projectID := os.Getenv("FIRESTORE_PROJECT_ID")
	if projectID == "" {
		log.Printf("[boot] WARN: FIRESTORE_PROJECT_ID is empty (introduction routes disabled)")
	} else {
		if c, err := introductionDI.NewContainer(initCtx, projectID); err != nil {
			log.Printf("[boot] WARN: introduction di init failed: %v (introduction routes disabled)", err)
		} else {
			introCont = c
			introCont.Register(fullMux)
			log.Printf("[boot] introduction routes registered")
		}
	}
	defer closeIfPossible("introduction container", introCont)

	// ------------------------------------------------------------
	// Console routes
	// ------------------------------------------------------------
	router := httpin.NewRouter(deps)

	fullMux.Handle("/console/", http.StripPrefix("/console", router))
	fullMux.Handle("/console", http.RedirectHandler("/console/", http.StatusPermanentRedirect))
	fullMux.Handle("/", router)

	log.Printf("[boot] full router built")

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      middleware.CORS(fullMux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

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
