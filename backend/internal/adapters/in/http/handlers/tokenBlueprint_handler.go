package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	uc "narratives/internal/application/usecase"
)

// TokenBlueprintHandler handles /token-blueprints endpoints (GET by id).
type TokenBlueprintHandler struct {
	uc *uc.TokenBlueprintUsecase
}

// NewTokenBlueprintHandler initializes the HTTP handler.
func NewTokenBlueprintHandler(ucase *uc.TokenBlueprintUsecase) http.Handler {
	return &TokenBlueprintHandler{uc: ucase}
}

// ServeHTTP routes requests.
func (h *TokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/token-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /token-blueprints/{id}
func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(tb)
}

// Error handling
func writeTokenBlueprintErr(w http.ResponseWriter, err error) {
	// Return 500 without depending on domain error types.
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
