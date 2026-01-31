package inspection

import (
	"net/http"
	"sort"
	"strings"

	inspectiondom "narratives/internal/domain/inspection"
	pbdom "narratives/internal/domain/productBlueprint"
)

type inspectionRecordResponse struct {
	ProductID        string  `json:"productId"`
	ModelID          string  `json:"modelId,omitempty"`
	ModelNumber      string  `json:"modelNumber,omitempty"`
	DisplayOrder     int     `json:"displayOrder,omitempty"` // ✅ 追加
	InspectionResult any     `json:"inspectionResult,omitempty"`
	InspectedBy      string  `json:"inspectedBy,omitempty"`   // ✅ 表示名
	InspectedByID    *string `json:"inspectedById,omitempty"` // ✅ デバッグ用（UIは無視してOK）
	InspectedAt      any     `json:"inspectedAt,omitempty"`
}

type inspectionBatchResponse struct {
	ProductionID string                     `json:"productionId"`
	Status       any                        `json:"status"`
	Quantity     int                        `json:"quantity"`
	TotalPassed  int                        `json:"totalPassed"`
	Inspections  []inspectionRecordResponse `json:"inspections"`
}

func (h *InspectorHandler) getInspectionsByProductionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		writeError(w, http.StatusInternalServerError, "inspection usecase is not configured")
		return
	}

	productionID := strings.TrimSpace(r.URL.Query().Get("productionId"))
	if productionID == "" {
		writeError(w, http.StatusBadRequest, "productionId is required")
		return
	}

	batch, err := h.inspectionUC.GetBatchByProductionID(ctx, productionID)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID:
			code = http.StatusBadRequest
		case inspectiondom.ErrNotFound:
			code = http.StatusNotFound
		}
		writeError(w, code, err.Error())
		return
	}

	// ✅ modelId -> displayOrder をキャッシュ
	displayOrderByModelID := map[string]int{}

	resolveDisplayOrder := func(modelID string) int {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" || h.pbModelRefGetter == nil {
			return 0
		}
		if v, ok := displayOrderByModelID[modelID]; ok {
			return v
		}

		refs, err := h.pbModelRefGetter.GetModelRefsByModelID(ctx, modelID)
		if err != nil {
			displayOrderByModelID[modelID] = 0
			return 0
		}

		order := 0
		for _, ref := range refs {
			if strings.TrimSpace(ref.ModelID) == modelID {
				order = ref.DisplayOrder
				break
			}
		}
		displayOrderByModelID[modelID] = order
		return order
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
		modelNumber := ""
		modelID := strings.TrimSpace(item.ModelID)
		if h.nameResolver != nil && modelID != "" {
			modelNumber = h.nameResolver.ResolveModelNumber(ctx, modelID)
		}

		// ✅ modelId -> displayOrder
		displayOrder := resolveDisplayOrder(modelID)

		resp.Inspections = append(resp.Inspections, inspectionRecordResponse{
			ProductID:        item.ProductID,
			ModelID:          modelID,
			ModelNumber:      strings.TrimSpace(modelNumber),
			DisplayOrder:     displayOrder,
			InspectionResult: item.InspectionResult,
			InspectedBy:      strings.TrimSpace(inspectedByName),
			InspectedByID:    inspectedByID,
			InspectedAt:      item.InspectedAt,
		})
	}

	// ✅ displayOrder 昇順で並べ替え（0は末尾扱い）
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
		// 同順位の安定化（任意）
		return resp.Inspections[i].ProductID < resp.Inspections[j].ProductID
	})

	writeJSON(w, http.StatusOK, resp)
}

// --- domain/productBlueprint.ModelRef の想定 ---
// package productBlueprint
// type ModelRef struct { ModelID string; DisplayOrder int }
var _ pbdom.ModelRef
