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
	uc           *usecase.ProductUsecase
	productionUC *usecase.ProductionUsecase
	modelUC      *usecase.ModelUsecase
}

// NewProductHandler はHTTPハンドラを初期化します。
func NewProductHandler(
	uc *usecase.ProductUsecase,
	productionUC *usecase.ProductionUsecase,
	modelUC *usecase.ModelUsecase,
) http.Handler {
	return &ProductHandler{
		uc:           uc,
		productionUC: productionUC,
		modelUC:      modelUC,
	}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ★ リクエストログを追加
	log.Printf("[ProductHandler] method=%s path=%s query=%s", r.Method, r.URL.Path, r.URL.RawQuery)

	switch {

	// ------------------------------------------------------------
	// POST /products/print-logs
	//   body: { "productionId": "xxx" }
	//   → 対象 productionId の products から 1 件の print_log を作成
	//     （内部で QR JSON 文字列も生成して QrPayloads に格納）
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && r.URL.Path == "/products/print-logs":
		h.createPrintLog(w, r)

	// ------------------------------------------------------------
	// GET /products/print-logs?productionId=xxx
	//   → 同一 productionId を持つ print_log 一覧を返す
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
	//   body: { "productionId": "xxx" }
	//   → inspections_by_production/{productionId} ドキュメントを作成
	//      inspections は productId ごとに null 初期化
	//      status は "inspecting" で開始
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && r.URL.Path == "/products/inspections":
		h.createInspectionBatch(w, r)

	// ------------------------------------------------------------
	// PATCH /products/inspections
	//   body: {
	//     "productionId": "xxx",
	//     "productId": "yyy",
	//     "inspectionResult": "passed" | "failed" | "notYet" | "notManufactured",
	//     "inspectedBy": "inspector-id-or-name",
	//     "inspectedAt": "2025-12-02T10:00:00Z",   // RFC3339 (任意)
	//     "status": "inspecting" | "completed"     // 任意
	//   }
	//   → 対象 productId の検査結果を更新し、必要に応じてバッチ status も更新
	// ------------------------------------------------------------
	case r.Method == http.MethodPatch && r.URL.Path == "/products/inspections":
		h.updateInspection(w, r)

	// ------------------------------------------------------------
	// GET /products?productionId=xxx
	//   → 同一 productionId を持つ Product 一覧を返す
	//      ※ ここで modelNumber を付与して返す
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

// ------------------------------------------------------------
// GET /products?productionId={productionId}
//   同一 productionId を持つ Product 一覧を返す
//   ※ modelNumber を backend 側で解決して付与する
// ------------------------------------------------------------

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

	// modelNumber 付与に必要な Usecase が無い場合は、従来どおりそのまま返す
	if h.productionUC == nil || h.modelUC == nil {
		log.Printf("[ProductHandler] listByProductionID: productionUC or modelUC is nil, return raw products")
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	// production から productBlueprintID を取得
	prod, err := h.productionUC.GetByID(ctx, productionID)
	if err != nil {
		log.Printf("[ProductHandler] listByProductionID: GetByID(productionID=%s) failed: %v", productionID, err)
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	pbID := strings.TrimSpace(prod.ProductBlueprintID)
	if pbID == "" {
		log.Printf("[ProductHandler] listByProductionID: empty ProductBlueprintID for productionID=%s", productionID)
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	// 対象 ProductBlueprint の ModelVariations を取得し、
	// modelID -> modelNumber のマップを作る
	vars, err := h.modelUC.ListModelVariationsByProductBlueprintID(ctx, pbID)
	if err != nil {
		log.Printf("[ProductHandler] listByProductionID: ListModelVariationsByProductBlueprintID(%s) failed: %v", pbID, err)
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

	// フロント用のレスポンス型
	type productWithModelNumber struct {
		ID           string `json:"id"`
		ModelID      string `json:"modelId"`
		ProductionID string `json:"productionId"`
		ModelNumber  string `json:"modelNumber"`
	}

	out := make([]productWithModelNumber, 0, len(list))

	for _, p := range list {
		modelNumber := strings.TrimSpace(idToModelNumber[p.ModelID])
		if modelNumber == "" {
			log.Printf(
				"[ProductHandler] listByProductionID: modelNumber not found for productID=%s modelID=%s (pbID=%s)",
				p.ID, p.ModelID, pbID,
			)
		}

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
//   同一 productionId を持つ print_log 一覧を返す
//   （usecase 側で QrPayloads も埋めた状態で返ってくる想定）
// ------------------------------------------------------------

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
		log.Printf("[ProductHandler] listPrintLogsByProductionID error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(logs)
}

// ------------------------------------------------------------
// POST /products/print-logs
//   body: { "productionId": "xxx" }
//
//   1. 該当 productionId の Product 一覧を usecase から取得
//   2. 1 回の印刷バッチとして PrintLog を 1 件作成
//   3. QR JSON 文字列を QrPayloads に詰めた状態で返す
// ------------------------------------------------------------

func (h *ProductHandler) createPrintLog(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("[ProductHandler] createPrintLog productionId=%s", productionID)

	// usecase 側で:
	//   - products を収集
	//   - PrintLog を 1 件作成
	//   - 各 productId から QR JSON を生成して QrPayloads に詰める
	pl, err := h.uc.CreatePrintLogForProduction(ctx, productionID)
	if err != nil {
		log.Printf("[ProductHandler] createPrintLog error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 正常時は作成した print_log 1 件を返す
	_ = json.NewEncoder(w).Encode(pl)
}

// ------------------------------------------------------------
// POST /products/inspections
//   body: { "productionId": "xxx" }
//
//   1. 該当 productionId の Product 一覧を usecase から取得
//   2. inspections_by_production/{productionId} ドキュメントを作成
//      - inspections は productId ごとに null 初期化
//      - status は "inspecting" で開始
// ------------------------------------------------------------

func (h *ProductHandler) createInspectionBatch(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("[ProductHandler] createInspectionBatch productionId=%s", productionID)

	batch, err := h.uc.CreateInspectionBatchForProduction(ctx, productionID)
	if err != nil {
		log.Printf("[ProductHandler] createInspectionBatch error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(batch)
}

// ------------------------------------------------------------
// PATCH /products/inspections
//   body: {
//     "productionId": "xxx",
//     "productId": "yyy",
//     "inspectionResult": "passed" | "failed" | "notYet" | "notManufactured",
//     "inspectedBy": "inspector-id-or-name",
//     "inspectedAt": "2025-12-02T10:00:00Z",   // RFC3339 (任意)
//     "status": "inspecting" | "completed"     // 任意
//   }
//
//   → 対象 productId の InspectionItem を更新し、更新後の InspectionBatch を返す
// ------------------------------------------------------------

func (h *ProductHandler) updateInspection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ProductionID     string                       `json:"productionId"`
		ProductID        string                       `json:"productId"`
		InspectionResult *productdom.InspectionResult `json:"inspectionResult"` // nil の場合は変更しない
		InspectedBy      *string                      `json:"inspectedBy"`      // nil の場合は変更しない
		InspectedAt      *time.Time                   `json:"inspectedAt"`      // nil の場合は変更しない
		Status           *productdom.InspectionStatus `json:"status"`           // nil の場合は変更しない
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ProductionID = strings.TrimSpace(req.ProductionID)
	req.ProductID = strings.TrimSpace(req.ProductID)

	if req.ProductionID == "" || req.ProductID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId and productId are required"})
		return
	}

	log.Printf(
		"[ProductHandler] updateInspection productionId=%s productId=%s result=%v status=%v",
		req.ProductionID, req.ProductID, req.InspectionResult, req.Status,
	)

	batch, err := h.uc.UpdateInspectionForProduct(
		ctx,
		req.ProductionID,
		req.ProductID,
		req.InspectionResult,
		req.InspectedBy,
		req.InspectedAt,
		req.Status,
	)
	if err != nil {
		log.Printf("[ProductHandler] updateInspection error: %v", err)
		writeProductErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(batch)
}

// ------------------------------------------------------------
// POST /products
//   印刷時などに、以下の項目で Product を新規作成する想定:
//   - modelId        : モデルID（必須, 更新不可）
//   - productionId   : 生産計画ID（必須, 更新不可）
//   - printedAt      : 印刷日時（必須, 更新不可）
//   - printedBy      : 印刷者（任意）
//
//   inspectionResult / connectedToken / inspectedAt / inspectedBy は
//   このタイミングでは未設定とし、InspectionResult は notYet を採用します。
// ------------------------------------------------------------

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
//   更新対象:
//     - inspectionResult
//     - connectedToken
//     - inspectedAt
//     - inspectedBy
//   ID / ModelID / ProductionID / PrintedAt / PrintedBy は更新不可
//   （usecase側で維持）
// ------------------------------------------------------------

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

// ------------------------------------------------------------
// エラーハンドリング
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
