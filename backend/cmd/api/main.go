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

	consoleHTTP "narratives/internal/adapters/in/http/console"
	"narratives/internal/adapters/in/http/middleware"

	adminDI "narratives/internal/platform/di/admin"
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

	consoleDeps := consoleCont.RouterDeps()

	// ------------------------------------------------------------
	// Build full mux
	// ------------------------------------------------------------
	fullMux := http.NewServeMux()

	fullMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
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
		introCont, err = introductionDI.NewContainer(initCtx, projectID)
		if err != nil {
			log.Printf("[boot] WARN: introduction di init failed: %v (introduction routes disabled)", err)
			introCont = nil
		} else {
			introCont.Register(fullMux)
		}
	}
	defer closeIfPossible("introduction container", introCont)

	// ------------------------------------------------------------
	// Admin routes
	// ------------------------------------------------------------
	adminCont, err := adminDI.NewContainer(initCtx, infra)
	if err != nil {
		log.Printf("[boot] WARN: admin di init failed: %v (admin routes disabled)", err)
	} else {
		adminDI.Register(fullMux, adminCont)
	}

	// ------------------------------------------------------------
	// Console routes
	// ------------------------------------------------------------
	consoleRouter := consoleHTTP.NewRouter(consoleDeps)

	fullMux.Handle("/console/", consoleRouter)
	fullMux.Handle("/console", http.RedirectHandler("/console/", http.StatusPermanentRedirect))

	// Console frontend currently calls APIs without the /console prefix.
	// Keep this root fallback until all Console API calls have migrated
	// to explicit /console/* paths.
	fullMux.Handle("/", consoleRouter)

	// ------------------------------------------------------------
	// HTTP server
	// ------------------------------------------------------------
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
