// backend/internal/adapters/in/http/handlers/mint_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
)

// MintHandler は、MintUsecase を HTTP 経由で公開するハンドラです。
// GET /mint/inspections を提供します。
type MintHandler struct {
	mintUC *usecase.MintUsecase
}

// NewMintHandler は MintHandler を生成します。
func NewMintHandler(mintUC *usecase.MintUsecase) http.Handler {
	return &MintHandler{
		mintUC: mintUC,
	}
}

func (h *MintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet &&
		(r.URL.Path == "/mint/inspections" || strings.HasPrefix(r.URL.Path, "/mint/inspections/")):
		h.listInspectionsForCurrentCompany(w, r)
		return
	default:
		http.NotFound(w, r)
	}
}

// ------------------------------------------------------------
// GET /mint/inspections
// ------------------------------------------------------------
func (h *MintHandler) listInspectionsForCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	batches, err := h.mintUC.ListInspectionsForCurrentCompany(ctx)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrCompanyIDMissing) {
			status = http.StatusBadRequest
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(batches)
}
