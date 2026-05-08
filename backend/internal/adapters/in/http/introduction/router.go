// backend/internal/adapters/in/http/introduction/router.go
package introduction

import "net/http"

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Router bundles introduction endpoints.
type Router struct {
	mux http.Handler
}

// NewRouter creates a router for introduction module.
// You can pass middlewares (outer -> inner) and they will be applied to the underlying mux.
func NewRouter(middlewares ...Middleware) *Router {
	base := http.NewServeMux()

	var h http.Handler = base
	// Apply in reverse so that middlewares[0] becomes the outermost wrapper.
	for i := len(middlewares) - 1; i >= 0; i-- {
		if middlewares[i] == nil {
			continue
		}
		h = middlewares[i](h)
	}

	return &Router{mux: h}
}

// Mux returns the underlying mux (http.Handler).
func (r *Router) Mux() http.Handler {
	return r.mux
}

// Register registers handlers for introduction module.
// You can add more handlers here in the future.
func (r *Router) Register(contactHandler *ContactHandler) {
	// We need the concrete *http.ServeMux to register routes.
	// Since r.mux is wrapped by middleware, keep a local mux for registration.
	// To keep API simple, we reconstruct the base mux and apply middleware in NewRouter.
	// Therefore, in this implementation, NewRouter internally owns the base mux.
	//
	// If you need to access the base mux later, refactor Router to store both base *http.ServeMux and wrapped handler.
	//
	// NOTE: This file expects the registration to be called on the base mux.
	// We achieve this by type asserting the innermost handler chain to *http.ServeMux.
	base, ok := unwrapToServeMux(r.mux)
	if !ok {
		// If not found, do nothing (shouldn't happen if NewRouter created it)
		return
	}

	if contactHandler != nil {
		contactHandler.Register(base)
	}
}

func unwrapToServeMux(h http.Handler) (*http.ServeMux, bool) {
	if h == nil {
		return nil, false
	}
	if mux, ok := h.(*http.ServeMux); ok {
		return mux, true
	}

	// Best-effort unwrap for common middleware patterns that use a struct with a `next http.Handler`.
	// If your middleware differs, refactor Router to store baseMux directly.
	type nextHolder interface{ Next() http.Handler }

	if nh, ok := h.(nextHolder); ok {
		return unwrapToServeMux(nh.Next())
	}

	return nil, false
}
