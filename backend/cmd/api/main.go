// backend/cmd/api/main.go
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	httpin "narratives/internal/adapters/in/http/console"
	"narratives/internal/adapters/in/http/middleware"

	consoleDI "narratives/internal/platform/di/console"
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

// logDepsFieldBestEffort prints whether deps has an exported field and its dynamic type.
// (compile-time safe even if the deps struct changes)
func logDepsFieldBestEffort(deps any, fieldName string) {
	fieldName = strings.TrimSpace(fieldName)
	if deps == nil || fieldName == "" {
		return
	}

	rv := reflect.ValueOf(deps)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		log.Printf("[boot] RouterDeps is not a struct: %T", deps)
		return
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() {
		log.Printf("[boot] RouterDeps.%s is MISSING", fieldName)
		return
	}
	if !f.CanInterface() {
		log.Printf("[boot] RouterDeps.%s exists but cannot interface", fieldName)
		return
	}

	v := f.Interface()
	if v == nil {
		log.Printf("[boot] RouterDeps.%s is NIL", fieldName)
		return
	}
	log.Printf("[boot] RouterDeps.%s: %T", fieldName, v)
}

type closer interface {
	Close() error
}

func main() {
	ctx := context.Background()

	// ─────────────────────────────────────────────────────────────
	// Log output: stdout + (best-effort) file
	// Cloud Run filesystem is effectively read-only except /tmp.
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
	// Container lifetime management
	// ─────────────────────────────────────────────────────────────
	var contHolder atomic.Value // stores *consoleDI.Container (or nil)
	contHolder.Store((*consoleDI.Container)(nil))

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

		// Close DI resources AFTER server shutdown (best-effort)
		// NOTE: console container owns shared.Infra; mall container shares the same infra (no separate close needed).
		if v := contHolder.Load(); v != nil {
			if cont, ok := v.(*consoleDI.Container); ok && cont != nil {
				log.Printf("[boot] closing container resources...")
				if c, ok := any(cont).(closer); ok {
					if err := c.Close(); err != nil {
						log.Printf("[boot] container close error: %v", err)
					}
				}
				contHolder.Store((*consoleDI.Container)(nil))
			}
		}

		close(idleConnsClosed)
	}()

	// Start server NOW (Cloud Run startup requirement)
	go func() {
		log.Printf("[boot] listening on :%s", port)
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

		// 1) shared infra (one instance; shared by console & mall)
		infra, err := shared.NewInfra(initCtx)
		if err != nil {
			log.Printf("[boot] WARN: shared infra init failed: %v (serving /healthz only)", err)
			return
		}

		// 2) console container (required)
		cont, err := consoleDI.NewContainer(initCtx, infra)
		if err != nil {
			_ = infra.Close()
			log.Printf("[boot] WARN: console di init failed: %v (serving /healthz only)", err)
			return
		}

		select {
		case <-shuttingDown:
			if c, ok := any(cont).(closer); ok {
				_ = c.Close()
			}
			return
		default:
		}

		contHolder.Store(cont)
		deps := cont.RouterDeps()

		logDepsFieldBestEffort(deps, "FirebaseAuth")
		logDepsFieldBestEffort(deps, "MemberRepo")

		fullMux := http.NewServeMux()

		// keep healthz
		fullMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		// 3) mall container (best-effort; separated DI)
		if mallCont, err := mallDI.NewContainer(initCtx, infra); err != nil {
			log.Printf("[boot] WARN: mall di init failed: %v (mall routes disabled)", err)
		} else {
			mallDI.Register(fullMux, mallCont)
			log.Printf("[boot] mall routes registered")
		}

		// 4) console/admin routes
		router := httpin.NewRouter(deps)

		// mount console under /console AND keep backward compatibility.
		fullMux.Handle("/console/", http.StripPrefix("/console", router))
		fullMux.Handle("/console", http.RedirectHandler("/console/", http.StatusPermanentRedirect))

		// Optional: keep old root paths working during migration
		fullMux.Handle("/", router)

		switcher.Store(middleware.CORS(fullMux))
		log.Printf("[boot] handler switched to full router (mounted /console)")
	}()

	<-idleConnsClosed
	log.Printf("[boot] server stopped")
}
