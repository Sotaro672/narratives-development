// backend/cmd/mall/main.go
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	mallDI "narratives/internal/platform/di/mall"
	shared "narratives/internal/platform/di/shared"
)

// atomicHandler allows swapping the underlying handler at runtime safely.
type atomicHandler struct {
	v atomic.Value // stores http.Handler
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

	// ─────────────────────────────────────────────────────────────
	// Log output: stdout + (best-effort) file
	// ─────────────────────────────────────────────────────────────
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

	// ─────────────────────────────────────────────────────────────
	// Port resolution: env PORT (Cloud Run) → 8080
	// ─────────────────────────────────────────────────────────────
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	// ─────────────────────────────────────────────────────────────
	// Start listening ASAP with lightweight mux (healthz only)
	// ─────────────────────────────────────────────────────────────
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

	// ─────────────────────────────────────────────────────────────
	// Lifetime management (infra/container)
	// ─────────────────────────────────────────────────────────────
	var infraHolder atomic.Value // stores *shared.Infra (or nil)
	infraHolder.Store((*shared.Infra)(nil))

	var mallHolder atomic.Value // stores any (mall container) (or nil)
	mallHolder.Store((any)(nil))

	shuttingDown := make(chan struct{})

	// ─────────────────────────────────────────────────────────────
	// Graceful shutdown
	// ─────────────────────────────────────────────────────────────
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

		// Close mall container best-effort (if it implements Close)
		if v := mallHolder.Load(); v != nil {
			if c, ok := v.(closer); ok && c != nil {
				log.Printf("[boot] closing mall container resources...")
				if err := c.Close(); err != nil {
					log.Printf("[boot] mall container close error: %v", err)
				}
			}
			mallHolder.Store((any)(nil))
		}

		// Close infra
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

	// Start server NOW (Cloud Run startup requirement)
	go func() {
		log.Printf("[boot] listening on :%s (mall)", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[boot] server error: %v", err)
		}
	}()

	// ─────────────────────────────────────────────────────────────
	// Heavy DI init in background; then swap handler to full app mux
	// ─────────────────────────────────────────────────────────────
	go func() {
		initCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		// 1) shared infra (mall service owns it)
		infra, err := shared.NewInfra(initCtx)
		if err != nil {
			log.Printf("[boot] WARN: shared infra init failed: %v (serving /healthz only)", err)
			return
		}
		infraHolder.Store(infra)

		// 2) mall container (required)
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

		// keep healthz
		fullMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		// 3) mall routes
		mallDI.Register(fullMux, mallCont)
		log.Printf("[boot] mall routes registered")

		switcher.Store(middleware.CORS(fullMux))
		log.Printf("[boot] handler switched to mall router")
	}()

	<-idleConnsClosed
	log.Printf("[boot] server stopped")
}
