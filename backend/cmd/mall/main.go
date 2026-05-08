package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	mallDI "narratives/internal/platform/di/mall"
	shared "narratives/internal/platform/di/shared"
)

type atomicHandler struct {
	v atomic.Value
}

func newAtomicHandler(initial http.Handler) *atomicHandler {
	ah := &atomicHandler{}
	if initial == nil {
		initial = http.NotFoundHandler()
	}
	ah.v.Store(initial)
	return ah
}

func (h *atomicHandler) Store(next http.Handler) {
	if next == nil {
		return
	}
	h.v.Store(next)
}

func (h *atomicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cur := h.v.Load()
	if cur == nil {
		http.NotFound(w, r)
		return
	}
	cur.(http.Handler).ServeHTTP(w, r)
}

type closer interface {
	Close() error
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

	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	switcher := newAtomicHandler(middleware.CORS(healthMux))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      switcher,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	var infraHolder atomic.Value
	infraHolder.Store((*shared.Infra)(nil))

	var mallHolder atomic.Value
	mallHolder.Store((any)(nil))

	shuttingDown := make(chan struct{})

	idleConnsClosed := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		sig := <-c

		close(shuttingDown)
		log.Printf("[boot] received signal: %v; shutting down...", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[boot] server shutdown error: %v", err)
		}

		if v := mallHolder.Load(); v != nil {
			if c, ok := v.(closer); ok && c != nil {
				log.Printf("[boot] closing mall container resources...")
				if err := c.Close(); err != nil {
					log.Printf("[boot] mall container close error: %v", err)
				}
			}
			mallHolder.Store((any)(nil))
		}

		if v := infraHolder.Load(); v != nil {
			if infra, ok := v.(*shared.Infra); ok && infra != nil {
				log.Printf("[boot] closing infra resources...")
				if err := infra.Close(); err != nil {
					log.Printf("[boot] infra close error: %v", err)
				}
				infraHolder.Store((*shared.Infra)(nil))
			}
		}

		close(idleConnsClosed)
	}()

	go func() {
		log.Printf("[boot] listening on :%s (mall)", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[boot] server error: %v", err)
		}
	}()

	go func() {
		initCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		infra, err := shared.NewInfra(initCtx)
		if err != nil {
			log.Printf("[boot] WARN: shared infra init failed: %v (serving /healthz only)", err)
			return
		}
		infraHolder.Store(infra)

		mallCont, err := mallDI.NewContainer(initCtx, infra)
		if err != nil {
			_ = infra.Close()
			infraHolder.Store((*shared.Infra)(nil))
			log.Printf("[boot] WARN: mall di init failed: %v (serving /healthz only)", err)
			return
		}
		mallHolder.Store(mallCont)

		select {
		case <-shuttingDown:
			if c, ok := any(mallCont).(closer); ok {
				_ = c.Close()
			}
			_ = infra.Close()
			return
		default:
		}

		fullMux := http.NewServeMux()

		fullMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		mallDI.Register(fullMux, mallCont)
		log.Printf("[boot] mall routes registered")

		switcher.Store(middleware.CORS(fullMux))
		log.Printf("[boot] handler switched to mall router")
	}()

	<-idleConnsClosed
	log.Printf("[boot] server stopped")
}
