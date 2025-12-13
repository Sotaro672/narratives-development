// backend/internal/adapters/in/http/handlers/inspection_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	productdom "narratives/internal/domain/product"
)

type InspectorHandler struct {
	// ★ 検品用 ProductUsecase（Inspector 詳細 DTO を組み立てる）
	productUC    *usecase.ProductUsecase
	inspectionUC *usecase.InspectionUsecase
}

func NewInspectorHandler(
	productUC *usecase.ProductUsecase,
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
	//      ProductUsecase.GetInspectorProductDetail を利用
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/inspector/products/"):

		productID := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
		productID = strings.TrimSpace(productID)
		if productID == "" {
			http.Error(w, `{"error":"missing productId"}`, http.StatusBadRequest)
			return
		}

		p, err := h.productUC.GetInspectorProductDetail(r.Context(), productID)
		if errors.Is(err, productdom.ErrNotFound) {
			http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(p); err != nil {
			http.Error(w, `{"error":"encode error"}`, http.StatusInternalServerError)
			return
		}
		return

	// ------------------------------------------------------------
	// GET /products/inspections/mints?inspectionId=xxxx
	//   → inspectionId (= productionId 扱い) に紐づく mints を返す
	//
	// NOTE:
	// - query は inspectionId を優先し、互換のため productionId も受け付ける
	// - 戻り値は []Mint（複数行対応）
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && r.URL.Path == "/products/inspections/mints":
		h.getMintsByInspectionID(w, r)
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
// GET /products/inspections/mints?inspectionId=xxxx
//
//	inspectionUsecase.ListMintsByInspectionID に移譲
//
// ------------------------------------------------------------
func (h *InspectorHandler) getMintsByInspectionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspection usecase is not configured",
		})
		return
	}

	// inspectionId 優先、互換で productionId も許容
	inspectionID := strings.TrimSpace(r.URL.Query().Get("inspectionId"))
	if inspectionID == "" {
		inspectionID = strings.TrimSpace(r.URL.Query().Get("productionId"))
	}
	if inspectionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspectionId is required",
		})
		return
	}

	mints, err := h.inspectionUC.ListMintsByInspectionID(ctx, inspectionID)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID:
			code = http.StatusBadRequest
		}

		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// nil でも [] を返す
	if mints == nil {
		mints = []mintdom.Mint{}
	}

	_ = json.NewEncoder(w).Encode(mints)
}

// ------------------------------------------------------------
// GET /products/inspections?productionId=xxxx
//
//	inspectionUsecase.ListByProductionID に移譲
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

	// 現状の ListByProductionID は単一バッチを返す実装なので、そのまま返す
	batch, err := h.inspectionUC.ListByProductionID(ctx, productionID)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID:
			code = http.StatusBadRequest
		case inspectiondom.ErrNotFound:
			code = http.StatusNotFound
		}

		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

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
		ProductionID     string                          `json:"productionId"`
		ProductID        string                          `json:"productId"`
		InspectionResult *inspectiondom.InspectionResult `json:"inspectionResult"`
		InspectedBy      *string                         `json:"inspectedBy"` // ← 互換のため定義だけ残す（実際には無視）
		InspectedAt      *time.Time                      `json:"inspectedAt"`
		Status           *inspectiondom.InspectionStatus `json:"status"`
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

	// ★ inspectedBy は「表示名」ではなく memberId（認証UID）を保存する方針
	uid, _, hasUIDEmail := middleware.CurrentUIDAndEmail(r)
	var inspectedByMemberID *string
	if hasUIDEmail && strings.TrimSpace(uid) != "" {
		v := strings.TrimSpace(uid)
		inspectedByMemberID = &v
	} else {
		// ここで uid が取れないのは認証が不正 or middleware 構成不備なので 400
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspectedBy (current member uid) could not be resolved",
		})
		return
	}

	batch, err := h.inspectionUC.UpdateInspectionForProduct(
		ctx,
		req.ProductionID,
		req.ProductID,
		req.InspectionResult,
		inspectedByMemberID, // ★ memberId (uid) を渡す
		req.InspectedAt,
		req.Status,
	)
	if err != nil {

		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID,
			inspectiondom.ErrInvalidInspectionProductIDs,
			inspectiondom.ErrInvalidInspectionResult,
			inspectiondom.ErrInvalidInspectedBy,
			inspectiondom.ErrInvalidInspectedAt,
			inspectiondom.ErrInvalidInspectionStatus:
			code = http.StatusBadRequest

		case inspectiondom.ErrNotFound:
			code = http.StatusNotFound
		}

		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

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

	// ★ by は「表示名」ではなく memberId（認証UID）を保存する方針
	uid, _, hasUIDEmail := middleware.CurrentUIDAndEmail(r)
	by := ""
	if hasUIDEmail && strings.TrimSpace(uid) != "" {
		by = strings.TrimSpace(uid)
	}

	if by == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspectedBy (current member uid) could not be resolved",
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

	batch, err := h.inspectionUC.CompleteInspectionForProduction(
		ctx,
		req.ProductionID,
		by, // ★ memberId (uid)
		at,
	)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID,
			inspectiondom.ErrInvalidInspectionResult:
			code = http.StatusBadRequest
		case inspectiondom.ErrNotFound:
			code = http.StatusNotFound
		}

		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(batch)
}
