package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	productdom "narratives/internal/domain/product"
)

// ProductHandler は /products 関連のエンドポイントを担当します。
type ProductHandler struct {
	uc *usecase.ProductUsecase
}

// NewProductHandler はHTTPハンドラを初期化します。
func NewProductHandler(uc *usecase.ProductUsecase) http.Handler {
	return &ProductHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/products/"):
		// GET /products/{id}
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		h.get(w, r, id)

	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/products/"):
		// PATCH /products/{id}
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		h.update(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /products/{id}
func (h *ProductHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(p)
}

// PATCH /products/{id}
// 更新対象:
// - inspectionResult
// - connectedToken
// - inspectedAt
// - inspectedBy
// ID / ModelID / ProductionID / PrintedAt / PrintedBy は更新不可（usecase側で維持）
func (h *ProductHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// リクエストボディ（更新可能フィールドのみ）
	var req struct {
		InspectionResult productdom.InspectionResult `json:"inspectionResult"`
		ConnectedToken   *string                     `json:"connectedToken"`
		InspectedAt      *time.Time                  `json:"inspectedAt"`
		InspectedBy      *string                     `json:"inspectedBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// usecase.Update に渡す Product（更新対象フィールドのみ設定）
	var p productdom.Product
	p.InspectionResult = req.InspectionResult
	p.ConnectedToken = req.ConnectedToken
	p.InspectedAt = req.InspectedAt
	p.InspectedBy = req.InspectedBy

	updated, err := h.uc.Update(ctx, id, p)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

// エラーハンドリング
func writeProductErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case productdom.ErrInvalidID:
		code = http.StatusBadRequest
	case productdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
