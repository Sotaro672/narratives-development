// backend/internal/adapters/in/http/handlers/mint_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
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

	// ---------------------------------------
	// ① context に保持されている値をログ出力
	// ---------------------------------------
	companyID := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))

	log.Printf(
		"[MintHandler Request] path=%s method=%s companyId=%s query=%v",
		r.URL.Path,
		r.Method,
		companyID,
		r.URL.Query(),
	)

	// ---------------------------------------
	// ② mintUC が nil でないか
	// ---------------------------------------
	if h.mintUC == nil {
		log.Printf("[MintHandler ERROR] mintUC is nil — cannot process request")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	// ---------------------------------------
	// ③ companyId があるかログ（error 状況追跡のため）
	// ---------------------------------------
	if companyID == "" {
		log.Printf("[MintHandler] companyId is EMPTY — request cannot be scoped to a company")
	} else {
		log.Printf("[MintHandler] companyId resolved: %s — calling usecase", companyID)
	}

	// ---------------------------------------
	// ④ Usecase 呼び出し
	// ---------------------------------------
	batches, err := h.mintUC.ListInspectionsForCurrentCompany(ctx)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrCompanyIDMissing) {
			status = http.StatusBadRequest
		}

		log.Printf("[MintHandler ERROR] usecase returned error: %v (status=%d)", err, status)

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// ---------------------------------------
	// ⑤ Usecase 成功時ログ
	// ---------------------------------------
	count := 0
	if batches != nil {
		count = len(batches)
	}

	log.Printf(
		"[MintHandler Response] companyId=%s returned_batches=%d",
		companyID,
		count,
	)

	_ = json.NewEncoder(w).Encode(batches)
}
