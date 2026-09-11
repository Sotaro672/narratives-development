// backend/internal/adapters/in/http/admin/router.go
package admin

import (
	"encoding/json"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
)

// RouterDeps contains only the handlers and middleware required by the Admin HTTP router.
type RouterDeps struct {
	AuthMw    *middleware.AdminAuthMiddleware
	Me        http.Handler
	Contacts  http.Handler
	Companies http.Handler
	Avatars   http.Handler
	Resales   http.Handler
	Gas       http.Handler
	Mints     http.Handler
	News      http.Handler
	Reports   http.Handler
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
		contactsHandler := withAuth(deps.Contacts)
		mux.Handle("/admin/contacts", contactsHandler)
		mux.Handle("/admin/contacts/", contactsHandler)
	}

	if deps.Companies != nil {
		companiesHandler := withAuth(deps.Companies)
		mux.Handle("/admin/companies", companiesHandler)
		mux.Handle("/admin/companies/", companiesHandler)
	}

	if deps.Avatars != nil {
		avatarsHandler := withAuth(deps.Avatars)
		mux.Handle("/admin/avatars", avatarsHandler)
		mux.Handle("/admin/avatars/{$}", avatarsHandler)
	}

	if deps.Resales != nil {
		resalesHandler := withAuth(deps.Resales)
		mux.Handle("/admin/avatars/{avatarID}/resales", resalesHandler)
		mux.Handle("/admin/avatars/{avatarID}/resales/{resaleID}", resalesHandler)
	}

	if deps.Gas != nil {
		mux.Handle("/admin/gas", withAuth(deps.Gas))
	}

	if deps.Mints != nil {
		mintsHandler := withAuth(deps.Mints)
		mux.Handle("/admin/mints", mintsHandler)
		mux.Handle("/admin/mints/", mintsHandler)
	}

	if deps.News != nil {
		newsHandler := withAuth(deps.News)
		mux.Handle("/admin/news", newsHandler)
		mux.Handle("/admin/news/", newsHandler)
	}

	if deps.Reports != nil {
		reportsHandler := withAuth(deps.Reports)
		mux.Handle("/admin/reports", reportsHandler)
		mux.Handle("/admin/reports/", reportsHandler)
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
