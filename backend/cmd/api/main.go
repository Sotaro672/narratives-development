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

	if err := c.Close(); err != nil {
		log.Printf("[boot] %s close error: %v", name, err)
	}
}

func main() {
	ctx := context.Background()

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
		}
	}
	defer closeIfPossible("introduction container", introCont)

	// ------------------------------------------------------------
	// Console routes
	// ------------------------------------------------------------
	router := httpin.NewRouter(deps)

	fullMux.Handle("/console/", router)
	fullMux.Handle("/console", http.RedirectHandler("/console/", http.StatusPermanentRedirect))
	fullMux.Handle("/", router)

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
		<-c

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[boot] server shutdown error: %v", err)
		}

		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[boot] server error: %v", err)
	}

	<-idleConnsClosed
}
