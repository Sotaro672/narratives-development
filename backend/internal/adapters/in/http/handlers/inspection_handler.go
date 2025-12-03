// backend/internal/adapters/in/http/handlers/inspector_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
	productdom "narratives/internal/domain/product"
)

type InspectorHandler struct {
	productUC    *usecase.PrintUsecase
	inspectionUC *usecase.InspectionUsecase
}

func NewInspectorHandler(
	productUC *usecase.PrintUsecase,
	inspectionUC *usecase.InspectionUsecase,
) http.Handler {
	return &InspectorHandler{
		productUC:    productUC,
		inspectionUC: inspectionUC,
	}
}

func (h *InspectorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// ------------------------------------------------------------
	// GET /inspector/products/{productId}
	//   → 検品アプリ（Flutter）用の詳細情報取得
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/inspector/products/"):

		productID := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
		productID = strings.TrimSpace(productID)
		if productID == "" {
			http.Error(w, `{"error":"missing productId"}`, http.StatusBadRequest)
			return
		}

		p, err := h.productUC.GetByID(r.Context(), productID)
		if errors.Is(err, productdom.ErrNotFound) {
			http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		// ★ inspector へ渡すデータ全体像ログ
		log.Printf(
			"[InspectorHandler] GET /inspector/products/%s response payload: %+v",
			productID,
			p,
		)

		if err := json.NewEncoder(w).Encode(p); err != nil {
			http.Error(w, `{"error":"encode error"}`, http.StatusInternalServerError)
			return
		}
		return

	// ------------------------------------------------------------
	// GET /products/inspections?productionId=xxxx
	//   → productionId から inspections バッチをそのまま返す
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && r.URL.Path == "/products/inspections":
		h.getInspectionsByProductionID(w, r)
		return

	// ------------------------------------------------------------
	// PATCH /products/inspections
	//   → inspections テーブル（1 productId 分）更新 API
	//   ※ products テーブル側も同時に更新される（InspectionUsecase 内で統合済）
	// ------------------------------------------------------------
	case r.Method == http.MethodPatch && r.URL.Path == "/products/inspections":
		h.updateInspection(w, r)
		return

	// ------------------------------------------------------------
	// PATCH /products/inspections/complete
	//   → 検品完了（未検品: notYet を notManufactured にし status=completed）
	// ------------------------------------------------------------
	case r.Method == http.MethodPatch && r.URL.Path == "/products/inspections/complete":
		h.completeInspection(w, r)
		return

	default:
		http.NotFound(w, r)
	}
}

// ------------------------------------------------------------
// GET /products/inspections?productionId=xxxx
//
//	inspectionUsecase.GetBatchByProductionID に移譲
//
// ------------------------------------------------------------
func (h *InspectorHandler) getInspectionsByProductionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspection usecase is not configured",
		})
		return
	}

	productionID := strings.TrimSpace(r.URL.Query().Get("productionId"))
	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productionId is required",
		})
		return
	}

	batch, err := h.inspectionUC.GetBatchByProductionID(ctx, productionID)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case productdom.ErrInvalidInspectionProductionID:
			code = http.StatusBadRequest
		case productdom.ErrNotFound:
			code = http.StatusNotFound
		}

		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// ★ inspector へ渡す inspections バッチ全体像ログ
	log.Printf(
		"[InspectorHandler] GET /products/inspections?productionId=%s response payload: status=%s, inspectionsCount=%d, batch=%+v",
		productionID,
		batch.Status,
		len(batch.Inspections),
		batch,
	)

	_ = json.NewEncoder(w).Encode(batch)
}

// ------------------------------------------------------------
// PATCH /products/inspections
//
//	検品結果を更新する（1 productId 単位）
//	→ InspectionUsecase.UpdateInspectionForProduct に移譲
//
// ------------------------------------------------------------
func (h *InspectorHandler) updateInspection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspection usecase is not configured",
		})
		return
	}

	var req struct {
		ProductionID     string                       `json:"productionId"`
		ProductID        string                       `json:"productId"`
		InspectionResult *productdom.InspectionResult `json:"inspectionResult"`
		InspectedBy      *string                      `json:"inspectedBy"` // ← 互換のため定義だけ残す（実際には無視）
		InspectedAt      *time.Time                   `json:"inspectedAt"`
		Status           *productdom.InspectionStatus `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	req.ProductionID = strings.TrimSpace(req.ProductionID)
	req.ProductID = strings.TrimSpace(req.ProductID)

	if req.ProductionID == "" || req.ProductID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productionId and productId are required",
		})
		return
	}

	// ★ 現在のメンバーの fullName をコンテキストから取得
	fullName, hasFullName := middleware.CurrentFullName(r)

	// ★ フォールバックとして email も拾っておく（fullName が空の場合）
	_, email, hasUIDEmail := middleware.CurrentUIDAndEmail(r)

	var inspectedByName *string
	if hasFullName && strings.TrimSpace(fullName) != "" {
		name := strings.TrimSpace(fullName)
		inspectedByName = &name
	} else if hasUIDEmail && strings.TrimSpace(email) != "" {
		// fullName が取れなかった場合は email を代用
		e := strings.TrimSpace(email)
		inspectedByName = &e
	} else {
		// それでも何もなければ nil のまま → Usecase 側では inspectedBy 更新なし
	}

	// ★ inspector から受け取った更新リクエスト + 解決した inspectedBy のログ
	log.Printf(
		"[InspectorHandler] PATCH /products/inspections request payload: productionId=%s, productId=%s, inspectionResult=%v, inspectedBy(fullName)=%v, inspectedAt=%v, status=%v",
		req.ProductionID,
		req.ProductID,
		req.InspectionResult,
		inspectedByName,
		req.InspectedAt,
		req.Status,
	)

	batch, err := h.inspectionUC.UpdateInspectionForProduct(
		ctx,
		req.ProductionID,
		req.ProductID,
		req.InspectionResult,
		inspectedByName, // ★ ここで fullName（なければ email）を渡す
		req.InspectedAt,
		req.Status,
	)
	if err != nil {

		code := http.StatusInternalServerError
		switch err {
		case productdom.ErrInvalidInspectionProductionID,
			productdom.ErrInvalidInspectionProductIDs,
			productdom.ErrInvalidInspectionResult,
			productdom.ErrInvalidInspectedBy,
			productdom.ErrInvalidInspectedAt,
			productdom.ErrInvalidInspectionStatus:
			code = http.StatusBadRequest

		case productdom.ErrNotFound:
			code = http.StatusNotFound
		}

		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// ★ 更新後に inspector へ返すバッチ全体像ログ
	log.Printf(
		"[InspectorHandler] PATCH /products/inspections response payload: productionId=%s, status=%s, inspectionsCount=%d, batch=%+v",
		batch.ProductionID,
		batch.Status,
		len(batch.Inspections),
		batch,
	)

	_ = json.NewEncoder(w).Encode(batch)
}

// ------------------------------------------------------------
// PATCH /products/inspections/complete
//
//	検品完了処理（未検品 notYet → notManufactured, status → completed）
//	→ InspectionUsecase.CompleteInspectionForProduction に移譲
//
// ------------------------------------------------------------
func (h *InspectorHandler) completeInspection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspection usecase is not configured",
		})
		return
	}

	var req struct {
		ProductionID string     `json:"productionId"`
		InspectedAt  *time.Time `json:"inspectedAt"` // 任意。nil の場合はサーバ時刻を使う
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	req.ProductionID = strings.TrimSpace(req.ProductionID)
	if req.ProductionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productionId is required",
		})
		return
	}

	// ★ by: CurrentFullName → email の順で解決
	fullName, hasFullName := middleware.CurrentFullName(r)
	_, email, hasUIDEmail := middleware.CurrentUIDAndEmail(r)

	by := ""
	if hasFullName && strings.TrimSpace(fullName) != "" {
		by = strings.TrimSpace(fullName)
	} else if hasUIDEmail && strings.TrimSpace(email) != "" {
		by = strings.TrimSpace(email)
	}

	if by == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspectedBy (current member) could not be resolved",
		})
		return
	}

	// inspectedAt: 指定があればそれを UTC に、無ければ now.UTC()
	var at time.Time
	if req.InspectedAt != nil && !req.InspectedAt.IsZero() {
		at = req.InspectedAt.UTC()
	} else {
		at = time.Now().UTC()
	}

	log.Printf(
		"[InspectorHandler] PATCH /products/inspections/complete request payload: productionId=%s, by=%s, at=%s",
		req.ProductionID,
		by,
		at.Format(time.RFC3339Nano),
	)

	batch, err := h.inspectionUC.CompleteInspectionForProduction(
		ctx,
		req.ProductionID,
		by,
		at,
	)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case productdom.ErrInvalidInspectionProductionID,
			productdom.ErrInvalidInspectionResult:
			code = http.StatusBadRequest
		case productdom.ErrNotFound:
			code = http.StatusNotFound
		}

		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Printf(
		"[InspectorHandler] PATCH /products/inspections/complete response payload: productionId=%s, status=%s, inspectionsCount=%d, totalPassed=%d",
		batch.ProductionID,
		batch.Status,
		len(batch.Inspections),
		batch.TotalPassed,
	)

	_ = json.NewEncoder(w).Encode(batch)
}
