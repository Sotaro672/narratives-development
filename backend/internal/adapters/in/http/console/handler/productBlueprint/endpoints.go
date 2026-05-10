// backend/internal/adapters/in/http/console/handler/productBlueprint/endpoints.go
package productBlueprint

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ---------------------------------------------------
// internal normalizers
// ---------------------------------------------------

func normalizeTagType(s string) pbdom.ProductIDTagType {
	switch s {
	case "qr", "QRコード", "QR":
		return pbdom.TagQR
	case "nfc", "NFC":
		return pbdom.TagNFC
	default:
		return pbdom.ProductIDTagType(s)
	}
}

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
	if in.CreatedBy != "" {
		createdBy = &in.CreatedBy
	}

	pb := pbdom.ProductBlueprint{
		ProductName: in.ProductName,
		BrandID:     in.BrandId,

		// NOTE:
		// ProductBlueprintCategory は productBlueprintCategory ドメインの正データから生成した
		// denormalized snapshot を入れる想定。
		// 現時点では CreateProductBlueprintInput 側の DTO が未改修の場合、
		// 次の修正対象は request DTO / handler 入力になります。
		ProductBlueprintCategory: pbdom.ProductBlueprintCategorySnapshot{
			ID:     in.ProductBlueprintCategory.ID,
			Code:   in.ProductBlueprintCategory.Code,
			NameJa: in.ProductBlueprintCategory.NameJa,
			NameEn: in.ProductBlueprintCategory.NameEn,
			Kind:   in.ProductBlueprintCategory.Kind,
			Path:   append([]string(nil), in.ProductBlueprintCategory.Path...),
		},

		Fit:              in.Fit,
		Material:         in.Material,
		Weight:           in.Weight,
		QualityAssurance: in.QualityAssurance,
		AssigneeID:       in.AssigneeId,

		// NOTE: companyId は usecase で auth context を正として上書きされる想定だが、
		// handler でも一応セットしておく（ログ/デバッグ用）。
		CompanyID: in.CompanyId,

		CreatedBy: createdBy,

		// printed は bool。create 時は常に false（未印刷）
		Printed: false,

		ProductIdTag: pbdom.ProductIDTag{
			Type: normalizeTagType(in.ProductIdTag.Type),
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
	if in.UpdatedBy != "" {
		updatedBy = &in.UpdatedBy
	}

	// printed は更新させない（印刷済み化は別ユースケースが適切）
	pb := pbdom.ProductBlueprint{
		ID:          id,
		ProductName: in.ProductName,
		BrandID:     in.BrandId,

		// NOTE:
		// ProductBlueprintCategory は productBlueprintCategory ドメインの正データから生成した
		// denormalized snapshot を入れる想定。
		ProductBlueprintCategory: pbdom.ProductBlueprintCategorySnapshot{
			ID:     in.ProductBlueprintCategory.ID,
			Code:   in.ProductBlueprintCategory.Code,
			NameJa: in.ProductBlueprintCategory.NameJa,
			NameEn: in.ProductBlueprintCategory.NameEn,
			Kind:   in.ProductBlueprintCategory.Kind,
			Path:   append([]string(nil), in.ProductBlueprintCategory.Path...),
		},

		Fit:              in.Fit,
		Material:         in.Material,
		Weight:           in.Weight,
		QualityAssurance: in.QualityAssurance,
		AssigneeID:       in.AssigneeId,
		CompanyID:        in.CompanyId,
		UpdatedBy:        updatedBy,
		ProductIdTag: pbdom.ProductIDTag{
			Type: normalizeTagType(in.ProductIdTag.Type),
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
//
// 論理削除を廃止したため、このエンドポイントは未提供。
// ---------------------------------------------------

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id string) {
	_ = r
	_ = id

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "delete endpoint is not supported"})
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/restore
// ---------------------------------------------------

func (h *Handler) restore(w http.ResponseWriter, r *http.Request, id string) {
	_ = r
	_ = id

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "restore endpoint is not supported"})
}

// ---------------------------------------------------
// GET /product-blueprints/{id}
// ---------------------------------------------------

func (h *Handler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

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
		assigneeId := pb.AssigneeID
		if assigneeId == "" {
			assigneeId = "-"
		}

		brandName := h.getBrandNameByID(ctx, pb.BrandID)
		if brandName == "" {
			brandName = pb.BrandID
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
//
// 論理削除一覧を廃止したため、このエンドポイントは未提供。
// ---------------------------------------------------

func (h *Handler) listDeleted(w http.ResponseWriter, r *http.Request) {
	_ = r

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "deleted list endpoint is not supported"})
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/model-refs
// ---------------------------------------------------

func (h *Handler) appendModelRefs(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

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
		v := raw
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
	brandId := pb.BrandID
	brandName := h.getBrandNameByID(ctx, brandId)
	if brandName == "" {
		brandName = brandId
	}

	assigneeId := pb.AssigneeID
	assigneeName := "-"
	if assigneeId != "" {
		assigneeName = h.getAssigneeNameByID(ctx, assigneeId)
		if assigneeName == "" {
			assigneeName = assigneeId
		}
	}

	createdBy := ""
	if pb.CreatedBy != nil {
		createdBy = *pb.CreatedBy
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

	var tag *struct {
		Type string `json:"type"`
	}
	if string(pb.ProductIdTag.Type) != "" {
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
			modelID := mr.ModelID
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
		ID:          pb.ID,
		ProductName: pb.ProductName,
		CompanyId:   pb.CompanyID,
		BrandId:     brandId,
		BrandName:   brandName,

		// NOTE:
		// ProductBlueprintDetailOutput 側にも productBlueprintCategory 用フィールドを追加する必要があります。
		// その修正が済むまではここでは既存フィールドのみ返します。

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

		ModelRefs: modelRefs,
	}
}
