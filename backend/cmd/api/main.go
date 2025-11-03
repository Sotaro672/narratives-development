package main

import (
	"context"
	"log"
	"net/http"
	"time"

	httpin "narratives/internal/adapters/in/http"
	"narratives/internal/platform/di"
)

func main() {
	ctx := context.Background()

	container, err := di.NewContainer(ctx)
	if err != nil {
		log.Fatalf("[boot] FATAL init container: %v", err)
	}
	defer container.Close()

	router := httpin.NewRouter(container.RouterDeps())

	port := container.Config.Port
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("[boot] listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[boot] server error: %v", err)
	}
}
