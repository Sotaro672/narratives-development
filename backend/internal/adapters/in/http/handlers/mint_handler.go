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
// 現状は「現在ログイン中の company に紐づく inspections 一覧」を返す
// GET /mint/inspections のみを提供します。
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
	// ------------------------------------------------------------
	// GET /mint/inspections
	//   ↳ context に埋め込まれた companyId をもとに
	//      MintUsecase.ListInspectionsForCurrentCompany を呼び出す
	// ------------------------------------------------------------
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
//
//   - AuthMiddleware により context に埋め込まれた companyId を起点に
//     MintUsecase.ListInspectionsForCurrentCompany を呼び出す。
//   - 戻り値の []InspectionBatch をそのまま JSON で返却する。
//
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

	// ログ用に、context に埋め込まれた companyId を確認
	companyID := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if companyID == "" {
		log.Printf("[MintHandler] path=%s companyId=<empty>", r.URL.Path)
	} else {
		log.Printf("[MintHandler] path=%s companyId=%s", r.URL.Path, companyID)
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

	// nil のままでも front 側で null チェックしているので問題ないが、
	// 必要であればここで空スライスに差し替えることも可能。
	// if batches == nil {
	//     batches = []inspectiondom.InspectionBatch{}
	// }

	_ = json.NewEncoder(w).Encode(batches)
}
