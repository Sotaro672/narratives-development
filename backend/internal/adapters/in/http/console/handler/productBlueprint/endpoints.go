// backend/internal/adapters/in/http/console/handler/productBlueprint/endpoints.go
package productBlueprint

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ---------------------------------------------------
// POST /product-blueprints
// ---------------------------------------------------

func (h *Handler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in CreateProductBlueprintInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	var createdBy *string
	if v := strings.TrimSpace(in.CreatedBy); v != "" {
		createdBy = &v
	}

	pb := pbdom.ProductBlueprint{
		ProductName:      strings.TrimSpace(in.ProductName),
		BrandID:          strings.TrimSpace(in.BrandId),
		ItemType:         pbdom.ItemType(strings.TrimSpace(in.ItemType)),
		Fit:              strings.TrimSpace(in.Fit),
		Material:         strings.TrimSpace(in.Material),
		Weight:           in.Weight,
		QualityAssurance: in.QualityAssurance,
		AssigneeID:       strings.TrimSpace(in.AssigneeId),
		CompanyID:        strings.TrimSpace(in.CompanyId),
		CreatedBy:        createdBy,
		// printed は bool。create 時は常に false（未印刷）
		Printed: false,
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(strings.TrimSpace(in.ProductIdTag.Type)),
		},
	}

	created, err := h.uc.Create(ctx, pb)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(ctx, created)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// PUT/PATCH /product-blueprints/{id}
// ---------------------------------------------------

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var in UpdateProductBlueprintInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	var updatedBy *string
	if v := strings.TrimSpace(in.UpdatedBy); v != "" {
		updatedBy = &v
	}

	// printed は更新させない（印刷済み化は別ユースケースが適切）
	pb := pbdom.ProductBlueprint{
		ID:               id,
		ProductName:      strings.TrimSpace(in.ProductName),
		BrandID:          strings.TrimSpace(in.BrandId),
		ItemType:         pbdom.ItemType(strings.TrimSpace(in.ItemType)),
		Fit:              strings.TrimSpace(in.Fit),
		Material:         strings.TrimSpace(in.Material),
		Weight:           in.Weight,
		QualityAssurance: in.QualityAssurance,
		AssigneeID:       strings.TrimSpace(in.AssigneeId),
		CompanyID:        strings.TrimSpace(in.CompanyId),
		UpdatedBy:        updatedBy,
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(strings.TrimSpace(in.ProductIdTag.Type)),
		},
	}

	updated, err := h.uc.Update(ctx, pb)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(ctx, updated)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// DELETE /product-blueprints/{id}
// ---------------------------------------------------

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.SoftDeleteWithModels(ctx, id, nil); err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/restore
// ---------------------------------------------------

func (h *Handler) restore(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.RestoreWithModels(ctx, id, nil); err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	pb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(ctx, pb)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints/{id}
// ---------------------------------------------------

func (h *Handler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	pb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(ctx, pb)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints
// ---------------------------------------------------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.uc.ListByCompanyID(ctx)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := make([]ProductBlueprintListOutput, 0, len(rows))
	for _, pb := range rows {
		assigneeId := strings.TrimSpace(pb.AssigneeID)
		if assigneeId == "" {
			assigneeId = "-"
		}

		brandName := h.getBrandNameByID(ctx, strings.TrimSpace(pb.BrandID))
		if brandName == "" {
			brandName = strings.TrimSpace(pb.BrandID)
		}

		assigneeName := "-"
		if assigneeId != "-" {
			assigneeName = h.getAssigneeNameByID(ctx, assigneeId)
		}

		createdAt := ""
		if !pb.CreatedAt.IsZero() {
			createdAt = pb.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if !pb.UpdatedAt.IsZero() {
			updatedAt = pb.UpdatedAt.Format(time.RFC3339)
		}

		out = append(out, ProductBlueprintListOutput{
			ID:           pb.ID,
			ProductName:  pb.ProductName,
			BrandName:    brandName,
			AssigneeName: assigneeName,
			Printed:      pb.Printed,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints/deleted
// ---------------------------------------------------

func (h *Handler) listDeleted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.uc.ListDeletedByCompanyID(ctx)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := make([]ProductBlueprintDeletedListOutput, 0, len(rows))
	for _, pb := range rows {
		brandId := strings.TrimSpace(pb.BrandID)
		assigneeId := strings.TrimSpace(pb.AssigneeID)
		if assigneeId == "" {
			assigneeId = "-"
		}

		deletedAtStr := ""
		if pb.DeletedAt != nil && !pb.DeletedAt.IsZero() {
			deletedAtStr = pb.DeletedAt.Format(time.RFC3339)
		}

		expireAtStr := ""
		if pb.ExpireAt != nil && !pb.ExpireAt.IsZero() {
			expireAtStr = pb.ExpireAt.Format(time.RFC3339)
		}

		out = append(out, ProductBlueprintDeletedListOutput{
			ID:          pb.ID,
			ProductName: pb.ProductName,
			BrandId:     brandId,
			AssigneeId:  assigneeId,
			DeletedAt:   deletedAtStr,
			ExpireAt:    expireAtStr,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints/{id}/history
// ---------------------------------------------------

func (h *Handler) listHistory(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	rows, err := h.uc.ListHistory(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := make([]ProductBlueprintHistoryOutput, 0, len(rows))
	for _, pb := range rows {
		updatedAtStr := ""
		if !pb.UpdatedAt.IsZero() {
			updatedAtStr = pb.UpdatedAt.Format(time.RFC3339)
		}

		deletedAtStr := ""
		if pb.DeletedAt != nil && !pb.DeletedAt.IsZero() {
			deletedAtStr = pb.DeletedAt.Format(time.RFC3339)
		}

		expireAtStr := ""
		if pb.ExpireAt != nil && !pb.ExpireAt.IsZero() {
			expireAtStr = pb.ExpireAt.Format(time.RFC3339)
		}

		out = append(out, ProductBlueprintHistoryOutput{
			ID:          pb.ID,
			ProductName: pb.ProductName,
			BrandId:     strings.TrimSpace(pb.BrandID),
			AssigneeId:  strings.TrimSpace(pb.AssigneeID),
			UpdatedAt:   updatedAtStr,
			UpdatedBy:   pb.UpdatedBy,
			DeletedAt:   deletedAtStr,
			ExpireAt:    expireAtStr,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/model-refs
//   - 起票後に modelRefs（modelId + displayOrder）を追記
//   - 入力は modelIds（順序が displayOrder の採番元）
//   - updatedAt / updatedBy は更新しない（usecase/repo 側で保証）
//   - 出力は detail（既存の toDetailOutput）
// ---------------------------------------------------

func (h *Handler) appendModelRefs(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var in AppendModelRefsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// 入力は modelIds 必須
	if len(in.ModelIds) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "modelIds is required"})
		return
	}

	// 入力順を保持しつつ、空/重複だけを弾く（順序は保持）
	seen := make(map[string]struct{}, len(in.ModelIds))
	modelIds := make([]string, 0, len(in.ModelIds))
	for _, raw := range in.ModelIds {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		modelIds = append(modelIds, v)
	}

	if len(modelIds) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "modelIds has no valid ids"})
		return
	}

	// usecase: 入力 modelIds（displayOrder 採番は usecase 側）
	updated, err := h.uc.AppendModelRefs(ctx, id, modelIds)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(ctx, updated)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// DTO assembler (detail)
// ---------------------------------------------------

func (h *Handler) toDetailOutput(ctx context.Context, pb pbdom.ProductBlueprint) ProductBlueprintDetailOutput {
	brandId := strings.TrimSpace(pb.BrandID)
	brandName := h.getBrandNameByID(ctx, brandId)
	if brandName == "" {
		brandName = brandId
	}

	assigneeId := strings.TrimSpace(pb.AssigneeID)
	assigneeName := "-"
	if assigneeId != "" {
		assigneeName = h.getAssigneeNameByID(ctx, assigneeId)
		if assigneeName == "" {
			assigneeName = assigneeId
		}
	}

	createdBy := ""
	if pb.CreatedBy != nil {
		createdBy = strings.TrimSpace(*pb.CreatedBy)
	}

	createdByName := ""
	if createdBy != "" {
		createdByName = h.getAssigneeNameByID(ctx, createdBy)
		if createdByName == "" {
			createdByName = createdBy
		}
	}

	createdAt := ""
	if !pb.CreatedAt.IsZero() {
		createdAt = pb.CreatedAt.Format(time.RFC3339)
	}

	updatedAt := ""
	if !pb.UpdatedAt.IsZero() {
		updatedAt = pb.UpdatedAt.Format(time.RFC3339)
	}

	deletedAt := ""
	if pb.DeletedAt != nil && !pb.DeletedAt.IsZero() {
		deletedAt = pb.DeletedAt.Format(time.RFC3339)
	}

	var tag *struct {
		Type string `json:"type"`
	}
	if strings.TrimSpace(string(pb.ProductIdTag.Type)) != "" {
		tag = &struct {
			Type string `json:"type"`
		}{
			Type: string(pb.ProductIdTag.Type),
		}
	}

	// modelRefs
	var modelRefs []ModelRefOutput
	if len(pb.ModelRefs) > 0 {
		modelRefs = make([]ModelRefOutput, 0, len(pb.ModelRefs))
		for _, mr := range pb.ModelRefs {
			modelID := strings.TrimSpace(mr.ModelID)
			if modelID == "" {
				continue
			}
			modelRefs = append(modelRefs, ModelRefOutput{
				ModelId:      modelID,
				DisplayOrder: mr.DisplayOrder,
			})
		}
	}

	return ProductBlueprintDetailOutput{
		ID:               pb.ID,
		ProductName:      pb.ProductName,
		CompanyId:        strings.TrimSpace(pb.CompanyID),
		BrandId:          brandId,
		BrandName:        brandName,
		ItemType:         string(pb.ItemType),
		Fit:              pb.Fit,
		Material:         pb.Material,
		Weight:           pb.Weight,
		QualityAssurance: pb.QualityAssurance,

		ProductIdTag: tag,

		AssigneeId:   assigneeId,
		AssigneeName: assigneeName,

		Printed: pb.Printed,

		CreatedBy:     createdBy,
		CreatedByName: createdByName,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,

		DeletedAt: deletedAt,

		ModelRefs: modelRefs,
	}
}
