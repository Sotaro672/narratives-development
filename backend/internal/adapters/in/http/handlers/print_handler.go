// backend/internal/adapters/in/http/handlers/print_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	productdom "narratives/internal/domain/product"
)

// PrintHandler は /products 関連のエンドポイントを担当します。
type PrintHandler struct {
	uc           *usecase.PrintUsecase
	productionUC *usecase.ProductionUsecase
	modelUC      *usecase.ModelUsecase
}

// NewPrintHandler はHTTPハンドラを初期化します。
func NewPrintHandler(
	uc *usecase.PrintUsecase,
	productionUC *usecase.ProductionUsecase,
	modelUC *usecase.ModelUsecase,
) http.Handler {
	return &PrintHandler{
		uc:           uc,
		productionUC: productionUC,
		modelUC:      modelUC,
	}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *PrintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// ------------------------------------------------------------
	// ★ 追加: GET /inspector/products/{id}
	//   ※ 現在は InspectorHandler に移譲済みなので PrintHandler 側では扱わない想定だが、
	//     一旦既存の get を流用している
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/inspector/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
		h.get(w, r, id)
		return

	// ------------------------------------------------------------
	// POST /products/print-logs
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && r.URL.Path == "/products/print-logs":
		h.createPrintLog(w, r)

	// ------------------------------------------------------------
	// GET /products/print-logs?productionId=xxx
	// ------------------------------------------------------------
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

	// ------------------------------------------------------------
	// POST /products/inspections
	//   → 検品バッチ作成（これはコンソール側の印刷ロット単位）
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && r.URL.Path == "/products/inspections":
		h.createInspectionBatch(w, r)

	// ------------------------------------------------------------
	// GET /products?productionId=xxx
	// ------------------------------------------------------------
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

	// ------------------------------------------------------------
	// GET /products/{id}
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		h.get(w, r, id)

	// ------------------------------------------------------------
	// PATCH /products/{id}
	//   → 個別 Product の更新（検品結果など）
	// ------------------------------------------------------------
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/products/")
		h.update(w, r, id)

	// ------------------------------------------------------------
	// POST /products
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && r.URL.Path == "/products":
		h.create(w, r)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ------------------------------------------------------------
// GET /products/{id}
// ------------------------------------------------------------
func (h *PrintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
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

// ------------------------------------------------------------
// GET /products?productionId={productionId}
//
//	同一 productionId を持つ Product 一覧を返す
//
// ------------------------------------------------------------
func (h *PrintHandler) listByProductionID(w http.ResponseWriter, r *http.Request, productionID string) {
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

	// modelNumber 付与に必要な Usecase が無い場合は従来のまま
	if h.productionUC == nil || h.modelUC == nil {
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	prod, err := h.productionUC.GetByID(ctx, productionID)
	if err != nil {
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	pbID := strings.TrimSpace(prod.ProductBlueprintID)
	if pbID == "" {
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	vars, err := h.modelUC.ListModelVariationsByProductBlueprintID(ctx, pbID)
	if err != nil {
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	idToModelNumber := make(map[string]string, len(vars))
	for _, v := range vars {
		mn := strings.TrimSpace(v.ModelNumber)
		if mn != "" {
			idToModelNumber[v.ID] = mn
		}
	}

	type productWithModelNumber struct {
		ID           string `json:"id"`
		ModelID      string `json:"modelId"`
		ProductionID string `json:"productionId"`
		ModelNumber  string `json:"modelNumber"`
	}

	out := make([]productWithModelNumber, 0, len(list))

	for _, p := range list {
		modelNumber := strings.TrimSpace(idToModelNumber[p.ModelID])

		out = append(out, productWithModelNumber{
			ID:           p.ID,
			ModelID:      p.ModelID,
			ProductionID: p.ProductionID,
			ModelNumber:  modelNumber,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ------------------------------------------------------------
// GET /products/print-logs?productionId={productionId}
// ------------------------------------------------------------
func (h *PrintHandler) listPrintLogsByProductionID(w http.ResponseWriter, r *http.Request, productionID string) {
	ctx := r.Context()
	productionID = strings.TrimSpace(productionID)

	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productionId"})
		return
	}

	logs, err := h.uc.ListPrintLogsByProductionID(ctx, productionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(logs)
}

// ------------------------------------------------------------
// POST /products/print-logs
// ------------------------------------------------------------
func (h *PrintHandler) createPrintLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ProductionID string `json:"productionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	productionID := strings.TrimSpace(req.ProductionID)
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId is required"})
		return
	}

	pl, err := h.uc.CreatePrintLogForProduction(ctx, productionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(pl)
}

// ------------------------------------------------------------
// POST /products/inspections
// ------------------------------------------------------------
func (h *PrintHandler) createInspectionBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ProductionID string `json:"productionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	productionID := strings.TrimSpace(req.ProductionID)
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId is required"})
		return
	}

	batch, err := h.uc.CreateInspectionBatchForProduction(ctx, productionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(batch)
}

// ------------------------------------------------------------
// POST /products
// ------------------------------------------------------------
func (h *PrintHandler) create(w http.ResponseWriter, r *http.Request) {
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

	p := productdom.Product{
		ModelID:          req.ModelID,
		ProductionID:     req.ProductionID,
		InspectionResult: productdom.InspectionNotYet,
		ConnectedToken:   nil,
		PrintedAt:        &req.PrintedAt,
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

// ------------------------------------------------------------
// PATCH /products/{id}
// ------------------------------------------------------------
func (h *PrintHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

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

// ------------------------------------------------------------
// 共通エラーレスポンス
// ------------------------------------------------------------
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
