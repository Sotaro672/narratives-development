// backend/cmd/api/main.go
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

	httpin "narratives/internal/adapters/in/http/console"
	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/platform/di"
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

func main() {
	ctx := context.Background()

	// ─────────────────────────────────────────────────────────────
	// Log output: stdout + (best-effort) file
	// Cloud Run filesystem is effectively read-only except /tmp.
	// ─────────────────────────────────────────────────────────────
	{
		logPath := "debug-idtoken.log"
		// Cloud Run 対策: /tmp に出す（ローカルでも動く）
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

	// handler switcher (start with health only)
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
	// - IMPORTANT: DO NOT defer cont.Close() inside the init goroutine.
	//   That was the direct cause of:
	//   "rpc error: code = Canceled desc = grpc: the client connection is closing"
	//   because Firestore client got closed immediately after handler swap.
	// ─────────────────────────────────────────────────────────────
	var contHolder atomic.Value // stores *di.Container (or nil)
	contHolder.Store((*di.Container)(nil))

	shuttingDown := make(chan struct{})

	// ─────────────────────────────────────────────────────────────
	// Graceful shutdown
	// ─────────────────────────────────────────────────────────────
	idleConnsClosed := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		sig := <-c

		// mark shutdown started (for background init goroutine)
		close(shuttingDown)

		log.Printf("[boot] received signal: %v; shutting down...", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[boot] server shutdown error: %v", err)
		}

		// Close DI resources AFTER server shutdown (best-effort)
		if v := contHolder.Load(); v != nil {
			if cont, ok := v.(*di.Container); ok && cont != nil {
				log.Printf("[boot] closing container resources...")
				if err := cont.Close(); err != nil {
					log.Printf("[boot] container close error: %v", err)
				}
				contHolder.Store((*di.Container)(nil))
			}
		}

		close(idleConnsClosed)
	}()

	// Start server NOW (this satisfies Cloud Run startup requirement)
	go func() {
		log.Printf("[boot] listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// fatal: container will exit → Cloud Run shows it in logs
			log.Fatalf("[boot] server error: %v", err)
		}
	}()

	// ─────────────────────────────────────────────────────────────
	// Heavy DI init in background; then swap handler to full app mux
	// ─────────────────────────────────────────────────────────────
	go func() {
		// DI が詰まるケースを切り分けやすくするためタイムアウトを付ける
		initCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		cont, err := di.NewContainer(initCtx)
		if err != nil {
			log.Printf("[boot] WARN: di init failed: %v (serving /healthz only)", err)
			return
		}

		// If shutdown started while DI was initializing, close and exit.
		select {
		case <-shuttingDown:
			_ = cont.Close()
			return
		default:
		}

		// Keep container for the process lifetime; close it on shutdown goroutine.
		contHolder.Store(cont)

		// Cloud Run PORT を最優先。Config があっても PORT が入ってるなら無視する。
		// （PORT が空の環境＝ローカル等では config を使えるようにする）
		if strings.TrimSpace(os.Getenv("PORT")) == "" {
			if p := strings.TrimSpace(cont.Config.Port); p != "" {
				log.Printf("[boot] local: overriding port from config: %s", p)
				// 注意: ここで port を変えても既に Listen 済みなので変えない（ログのみ）
			}
		}

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

		// Build full mux
		fullMux := http.NewServeMux()

		// keep healthz
		fullMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		// SNS routes (best-effort) — register first so it wins over "/"
		di.RegisterSNSFromContainer(fullMux, cont)

		// Console/Admin routes
		router := httpin.NewRouter(deps)
		fullMux.Handle("/", router)

		// Swap handler
		switcher.Store(middleware.CORS(fullMux))
		log.Printf("[boot] handler switched to full router")
	}()

	<-idleConnsClosed
	log.Printf("[boot] server stopped")
}
