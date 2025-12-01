// backend/internal/adapters/in/http/handlers/product_handler.go
package handlers

import (
	"encoding/json"
	"log"
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

	// ★ リクエストログを追加
	log.Printf("[ProductHandler] method=%s path=%s query=%s", r.Method, r.URL.Path, r.URL.RawQuery)

	switch {
	// GET /products/print-logs?productionId=xxx
	case r.Method == http.MethodGet && r.URL.Path == "/products/print-logs":
		productionID := strings.TrimSpace(r.URL.Query().Get("productionId"))
		if productionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "productionId query parameter is required",
			})
			return
		}
		h.listPrintLogsByProductionID(w, r, productionID)

	// GET /products?productionId=xxx
	case r.Method == http.MethodGet && r.URL.Path == "/products":
		productionID := strings.TrimSpace(r.URL.Query().Get("productionId"))
		if productionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "productionId query parameter is required",
			})
			return
		}
		h.listByProductionID(w, r, productionID)

	// GET /products/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		h.get(w, r, id)

	// PATCH /products/{id}
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		h.update(w, r, id)

	// POST /products
	case r.Method == http.MethodPost && r.URL.Path == "/products":
		h.create(w, r)

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

// GET /products?productionId={productionId}
// 同一 productionId を持つ Product 一覧を返す
func (h *ProductHandler) listByProductionID(w http.ResponseWriter, r *http.Request, productionID string) {
	ctx := r.Context()

	productionID = strings.TrimSpace(productionID)
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productionId"})
		return
	}

	list, err := h.uc.ListByProductionID(ctx, productionID)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(list)
}

// GET /products/print-logs?productionId={productionId}
// 同一 productionId を持つ print_log 一覧を返す
func (h *ProductHandler) listPrintLogsByProductionID(w http.ResponseWriter, r *http.Request, productionID string) {
	ctx := r.Context()

	productionID = strings.TrimSpace(productionID)
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productionId"})
		return
	}

	logs, err := h.uc.ListPrintLogsByProductionID(ctx, productionID)
	if err != nil {
		// print_log 用の専用エラー型はまだないので、ひとまず 500 を返す
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(logs)
}

// POST /products
// 印刷時などに、以下の項目で Product を新規作成する想定:
// - modelId        : モデルID（必須, 更新不可）
// - productionId   : 生産計画ID（必須, 更新不可）
// - printedAt      : 印刷日時（必須, 更新不可）
// - printedBy      : 印刷者（任意）
//
// inspectionResult / connectedToken / inspectedAt / inspectedBy は
// このタイミングでは未設定とし、InspectionResult は notYet を採用します。
func (h *ProductHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ModelID      string    `json:"modelId"`
		ProductionID string    `json:"productionId"`
		PrintedAt    time.Time `json:"printedAt"`
		PrintedBy    *string   `json:"printedBy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ModelID = strings.TrimSpace(req.ModelID)
	req.ProductionID = strings.TrimSpace(req.ProductionID)

	if req.ModelID == "" || req.ProductionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "modelId and productionId are required"})
		return
	}
	if req.PrintedAt.IsZero() {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "printedAt is required"})
		return
	}

	// domain.Product を構築（POST時に確定させるフィールドのみ設定）
	p := productdom.Product{
		// ID は空のまま渡し、RepositoryFS 側で auto-ID を採番させる
		ModelID:          req.ModelID,
		ProductionID:     req.ProductionID,
		InspectionResult: productdom.InspectionNotYet,
		ConnectedToken:   nil,
		PrintedAt:        &req.PrintedAt,
		PrintedBy:        req.PrintedBy,
		InspectedAt:      nil,
		InspectedBy:      nil,
	}

	created, err := h.uc.Create(ctx, p)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(created)
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
