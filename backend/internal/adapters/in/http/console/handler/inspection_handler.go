// backend\internal\adapters\in\http\console\handler\inspection_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	inspectorquery "narratives/internal/application/query/inspector"
	"narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	inspectiondom "narratives/internal/domain/inspection"
	productdom "narratives/internal/domain/product"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprint の modelRefs（displayOrder 含む）を引くための最小ポート
type ProductBlueprintModelRefGetter interface {
	GetModelRefsByModelID(ctx context.Context, modelID string) ([]pbdom.ModelRef, error)
}

type InspectorHandler struct {
	inspectionUC    *usecase.InspectionUsecase
	inspectionQuery *inspectorquery.QueryService
	nameResolver    *resolver.NameResolver

	// modelId -> displayOrder 解決用
	pbModelRefGetter ProductBlueprintModelRefGetter
}

func NewInspectorHandler(
	inspectionUC *usecase.InspectionUsecase,
	inspectionQuery *inspectorquery.QueryService,
	nameResolver *resolver.NameResolver,
	pbModelRefGetter ProductBlueprintModelRefGetter,
) http.Handler {
	return &InspectorHandler{
		inspectionUC:     inspectionUC,
		inspectionQuery:  inspectionQuery,
		nameResolver:     nameResolver,
		pbModelRefGetter: pbModelRefGetter,
	}
}

func (h *InspectorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/inspector/products/"):
		h.getInspectorProductDetail(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products/inspections":
		h.getInspectionsByProductionID(w, r)
		return

	case r.Method == http.MethodPatch && r.URL.Path == "/products/inspections":
		h.updateInspection(w, r)
		return

	case r.Method == http.MethodPatch && r.URL.Path == "/products/inspections/complete":
		h.completeInspection(w, r)
		return

	default:
		writeInspectionError(w, http.StatusNotFound, "not_found")
		return
	}
}

// ------------------------------------------------------------
// Response DTOs
// ------------------------------------------------------------

type inspectionRecordResponse struct {
	ProductID        string  `json:"productId"`
	ModelID          string  `json:"modelId,omitempty"`
	ModelNumber      string  `json:"modelNumber,omitempty"`
	DisplayOrder     int     `json:"displayOrder"` // 0でも返す
	InspectionResult any     `json:"inspectionResult,omitempty"`
	InspectedBy      string  `json:"inspectedBy,omitempty"`   // 表示名
	InspectedByID    *string `json:"inspectedById,omitempty"` // デバッグ用（UIは無視してOK）
	InspectedAt      any     `json:"inspectedAt,omitempty"`
}

type inspectionBatchResponse struct {
	ProductionID string                     `json:"productionId"`
	Status       any                        `json:"status"`
	Quantity     int                        `json:"quantity"`
	TotalPassed  int                        `json:"totalPassed"`
	Inspections  []inspectionRecordResponse `json:"inspections"`
}

// ------------------------------------------------------------
// GET /inspector/products/{productId}
// ------------------------------------------------------------

func (h *InspectorHandler) getInspectorProductDetail(w http.ResponseWriter, r *http.Request) {
	if h.inspectionQuery == nil {
		writeInspectionError(w, http.StatusInternalServerError, "inspection query service is not configured")
		return
	}

	productID := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
	productID = strings.Trim(productID, "/")
	if productID == "" {
		writeInspectionError(w, http.StatusBadRequest, "missing productId")
		return
	}

	p, err := h.inspectionQuery.GetInspectorProductDetail(r.Context(), productID)
	if errors.Is(err, productdom.ErrNotFound) {
		writeInspectionError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeInspectionError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeInspectionJSON(w, http.StatusOK, p)
}

// ------------------------------------------------------------
// GET /products/inspections?productionId=...
// ------------------------------------------------------------

func (h *InspectorHandler) getInspectionsByProductionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionQuery == nil {
		writeInspectionError(w, http.StatusInternalServerError, "inspection query service is not configured")
		return
	}

	productionID := r.URL.Query().Get("productionId")
	if productionID == "" {
		writeInspectionError(w, http.StatusBadRequest, "productionId is required")
		return
	}

	batch, err := h.inspectionQuery.GetByProductionID(ctx, productionID)
	if err != nil {
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, inspectiondom.ErrInvalidInspectionProductionID):
			code = http.StatusBadRequest
		case errors.Is(err, inspectiondom.ErrNotFound):
			code = http.StatusNotFound
		}
		writeInspectionError(w, code, err.Error())
		return
	}

	// modelId -> displayOrder をキャッシュ
	displayOrderByModelID := map[string]int{}

	resolveDisplayOrder := func(modelID string) int {
		if modelID == "" || h.pbModelRefGetter == nil {
			return 0
		}
		if v, ok := displayOrderByModelID[modelID]; ok {
			return v
		}

		refs, err := h.pbModelRefGetter.GetModelRefsByModelID(ctx, modelID)
		if err != nil {
			// 失敗時は固定キャッシュしない（復旧可能にする）
			return 0
		}

		// refs を取れたら全件キャッシュ（同一BP配下のモデルをまとめて埋める）
		for _, ref := range refs {
			if ref.ModelID == "" {
				continue
			}
			displayOrderByModelID[ref.ModelID] = ref.DisplayOrder
		}

		return displayOrderByModelID[modelID]
	}

	resp := inspectionBatchResponse{
		ProductionID: batch.ProductionID,
		Status:       batch.Status,
		Quantity:     batch.Quantity,
		TotalPassed:  batch.TotalPassed,
		Inspections:  make([]inspectionRecordResponse, 0, len(batch.Inspections)),
	}

	for _, item := range batch.Inspections {
		// inspectedBy: *string(memberId) -> 表示名
		inspectedByName := ""
		inspectedByID := item.InspectedBy
		if h.nameResolver != nil {
			inspectedByName = h.nameResolver.ResolveInspectedByName(ctx, inspectedByID)
		}

		// modelId -> modelNumber
		modelID := item.ModelID
		modelNumber := ""
		if h.nameResolver != nil && modelID != "" {
			modelNumber = h.nameResolver.ResolveModelNumber(ctx, modelID)
		}

		// modelId -> displayOrder
		displayOrder := resolveDisplayOrder(modelID)

		resp.Inspections = append(resp.Inspections, inspectionRecordResponse{
			ProductID:        item.ProductID,
			ModelID:          modelID,
			ModelNumber:      modelNumber,
			DisplayOrder:     displayOrder,
			InspectionResult: item.InspectionResult,
			InspectedBy:      inspectedByName,
			InspectedByID:    inspectedByID,
			InspectedAt:      item.InspectedAt,
		})
	}

	// displayOrder 昇順で並べ替え（0は末尾扱い）
	sort.SliceStable(resp.Inspections, func(i, j int) bool {
		ai := resp.Inspections[i].DisplayOrder
		aj := resp.Inspections[j].DisplayOrder

		if ai == 0 {
			ai = 1 << 30
		}
		if aj == 0 {
			aj = 1 << 30
		}
		if ai != aj {
			return ai < aj
		}

		// 同順位の安定化
		return resp.Inspections[i].ProductID < resp.Inspections[j].ProductID
	})

	writeInspectionJSON(w, http.StatusOK, resp)
}

// ------------------------------------------------------------
// PATCH /products/inspections
// ------------------------------------------------------------

func (h *InspectorHandler) updateInspection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		writeInspectionError(w, http.StatusInternalServerError, "inspection usecase is not configured")
		return
	}

	var req struct {
		ProductionID     string                          `json:"productionId"`
		ProductID        string                          `json:"productId"`
		InspectionResult *inspectiondom.InspectionResult `json:"inspectionResult"`
		InspectedAt      *time.Time                      `json:"inspectedAt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInspectionError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.ProductionID == "" || req.ProductID == "" {
		writeInspectionError(w, http.StatusBadRequest, "productionId and productId are required")
		return
	}

	if req.InspectionResult == nil {
		writeInspectionError(w, http.StatusBadRequest, "inspectionResult is required")
		return
	}

	switch *req.InspectionResult {
	case inspectiondom.InspectionPassed,
		inspectiondom.InspectionFailed,
		inspectiondom.InspectionNotManufactured:
	default:
		writeInspectionError(w, http.StatusBadRequest, inspectiondom.ErrInvalidInspectionResult.Error())
		return
	}

	// inspectedBy は「表示名」ではなく memberId（認証UID）を保存する方針
	uid := currentInspectionMemberUID(r)
	if uid == "" {
		writeInspectionError(w, http.StatusBadRequest, "inspectedBy (current member uid) could not be resolved")
		return
	}
	inspectedByMemberID := &uid

	inspectedAt := time.Now().UTC()
	if req.InspectedAt != nil {
		inspectedAt = req.InspectedAt.UTC()
	}
	if inspectedAt.IsZero() {
		writeInspectionError(w, http.StatusBadRequest, inspectiondom.ErrInvalidInspectedAt.Error())
		return
	}

	batch, err := h.inspectionUC.UpdateInspectionForProduct(
		ctx,
		req.ProductionID,
		req.ProductID,
		req.InspectionResult,
		inspectedByMemberID,
		&inspectedAt,
	)
	if err != nil {
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, inspectiondom.ErrInvalidInspectionProductionID),
			errors.Is(err, inspectiondom.ErrInvalidInspectionProductIDs),
			errors.Is(err, inspectiondom.ErrInvalidInspectionResult),
			errors.Is(err, inspectiondom.ErrInvalidInspectedBy),
			errors.Is(err, inspectiondom.ErrInvalidInspectedAt),
			errors.Is(err, inspectiondom.ErrInvalidInspectionStatus):
			code = http.StatusBadRequest
		case errors.Is(err, inspectiondom.ErrNotFound):
			code = http.StatusNotFound
		}
		writeInspectionError(w, code, err.Error())
		return
	}

	writeInspectionJSON(w, http.StatusOK, batch)
}

// ------------------------------------------------------------
// PATCH /products/inspections/complete
// ------------------------------------------------------------

func (h *InspectorHandler) completeInspection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		writeInspectionError(w, http.StatusInternalServerError, "inspection usecase is not configured")
		return
	}

	var req struct {
		ProductionID string     `json:"productionId"`
		InspectedAt  *time.Time `json:"inspectedAt"` // 任意。nil の場合はサーバ時刻を使う
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInspectionError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.ProductionID == "" {
		writeInspectionError(w, http.StatusBadRequest, "productionId is required")
		return
	}

	// by は「表示名」ではなく memberId（認証UID）を保存する方針
	by := currentInspectionMemberUID(r)
	if by == "" {
		writeInspectionError(w, http.StatusBadRequest, "inspectedBy (current member uid) could not be resolved")
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
		by,
		at,
	)
	if err != nil {
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, inspectiondom.ErrInvalidInspectionProductionID),
			errors.Is(err, inspectiondom.ErrInvalidInspectionResult),
			errors.Is(err, inspectiondom.ErrInvalidInspectedBy),
			errors.Is(err, inspectiondom.ErrInvalidInspectedAt):
			code = http.StatusBadRequest
		case errors.Is(err, inspectiondom.ErrNotFound):
			code = http.StatusNotFound
		}
		writeInspectionError(w, code, err.Error())
		return
	}

	writeInspectionJSON(w, http.StatusOK, batch)
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func writeInspectionJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeInspectionError(w http.ResponseWriter, status int, msg string) {
	writeInspectionJSON(w, status, map[string]string{"error": msg})
}

// 認証UIDを inspectedBy として使う（取れない場合は "" を返す）
func currentInspectionMemberUID(r *http.Request) string {
	uid, _, ok := middleware.CurrentUIDAndEmail(r)
	if !ok {
		return ""
	}
	return uid
}
