// backend/internal/adapters/in/http/mall/handler/token_blueprint_composite_handler.go
package mallHandler

import (
	"net/http"
	"strings"
)

// TokenBlueprintCompositeHandler routes requests under /mall/**/token-blueprints*
// to either the TokenBlueprint handler or the TokenBlueprintReview handler.
//
// IMPORTANT:
// - ServeMux pattern is registered only once (e.g. "/mall/me/token-blueprints" and "/mall/me/token-blueprints/").
// - This handler decides routing based on request path.
type TokenBlueprintCompositeHandler struct {
	TokenBlueprint       http.Handler
	TokenBlueprintReview http.Handler
}

func NewTokenBlueprintCompositeHandler(
	tb http.Handler,
	review http.Handler,
) *TokenBlueprintCompositeHandler {
	return &TokenBlueprintCompositeHandler{
		TokenBlueprint:       tb,
		TokenBlueprintReview: review,
	}
}

func (h *TokenBlueprintCompositeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || (h.TokenBlueprint == nil && h.TokenBlueprintReview == nil) {
		http.NotFound(w, r)
		return
	}
	if h.TokenBlueprint == nil {
		h.TokenBlueprintReview.ServeHTTP(w, r)
		return
	}
	if h.TokenBlueprintReview == nil {
		h.TokenBlueprint.ServeHTTP(w, r)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	if strings.Contains(path, "/comments") ||
		strings.Contains(path, "/reviews") ||
		strings.Contains(path, "/replies") ||
		strings.Contains(path, "/reactions") {
		h.TokenBlueprintReview.ServeHTTP(w, r)
		return
	}

	h.TokenBlueprint.ServeHTTP(w, r)
}
