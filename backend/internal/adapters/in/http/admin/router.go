// backend/internal/adapters/in/http/admin/router.go
package admin

import (
	"encoding/json"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
)

// RouterDeps contains only the handlers and middleware required by the Admin HTTP router.
type RouterDeps struct {
	AuthMw   *middleware.AdminAuthMiddleware
	Me       http.Handler
	Contacts http.Handler
}

// NewRouter creates the Admin router.
//
// All /admin/* endpoints registered here must be protected by AdminAuthMiddleware.
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	withAuth := func(handler http.Handler) http.Handler {
		if handler == nil {
			return unavailableHandler("admin_handler_not_initialized")
		}
		if deps.AuthMw == nil {
			return unavailableHandler("admin_auth_not_initialized")
		}
		return deps.AuthMw.Handler(handler)
	}

	if deps.Me != nil {
		mux.Handle("/admin/me", withAuth(deps.Me))
	}

	if deps.Contacts != nil {
		mux.Handle("/admin/contacts", withAuth(deps.Contacts))
	}

	return mux
}

func unavailableHandler(message string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
	})
}
