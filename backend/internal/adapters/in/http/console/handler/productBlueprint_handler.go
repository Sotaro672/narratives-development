package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	pbuc "narratives/internal/application/usecase"
	pbdom "narratives/internal/domain/productBlueprint"
	categorydom "narratives/internal/domain/productBlueprintCategory"
)

// ProductBlueprintHandler は ProductBlueprint 用の HTTP ハンドラです。
type ProductBlueprintHandler struct {
	uc *pbuc.ProductBlueprintUsecase
}

func NewProductBlueprintHandler(
	uc *pbuc.ProductBlueprintUsecase,
) http.Handler {
	return &ProductBlueprintHandler{
		uc: uc,
	}
}

func (h *ProductBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimRight(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path == "/product-blueprints":
		h.list(w, r)

	case r.Method == http.MethodPost && path == "/product-blueprints":
		h.post(w, r)

	// POST /product-blueprints/{id}/model-refs
	case r.Method == http.MethodPost &&
		strings.HasPrefix(path, "/product-blueprints/") &&
		strings.HasSuffix(path, "/model-refs"):
		trimmed := strings.TrimPrefix(path, "/product-blueprints/")
		trimmed = strings.TrimSuffix(trimmed, "/model-refs")
		id := strings.Trim(trimmed, "/")
		h.appendModelRefs(w, r, id)

	// 重要：suffix 付きルート（/model-refs）より後に置く
	case (r.Method == http.MethodPut || r.Method == http.MethodPatch) &&
		strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.update(w, r, id)

	case r.Method == http.MethodDelete &&
		strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.delete(w, r, id)

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ---------------------------------------------------
// Common DTOs
// ---------------------------------------------------

type ProductIdTagInput struct {
	Type string `json:"type"`
}

// ---------------------------------------------------
// POST /product-blueprints
// ---------------------------------------------------

type CreateProductBlueprintInput struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandId   string `json:"brandId"`
	CompanyId string `json:"companyId"`

	ProductBlueprintCategory categorydom.Snapshot `json:"productBlueprintCategory"`

	// CategoryFields はカテゴリ別の productBlueprint 入力値を受け取る。
	//
	// 例:
	// - alcohol.sake:
	//   vintage, region, material, alcoholContent, volume
	// - apparel.tops:
	//   weight, fit, material
	// - cosmetics.skincare:
	//   material, volume
	//
	// brandId / productName / productIdTagType / description などの共通 field はここには入れない。
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	// 当面 frontend では qr 固定。
	// DTO としては既存互換のため productIdTag.type を受ける。
	ProductIdTag ProductIdTagInput `json:"productIdTag"`

	AssigneeId string `json:"assigneeId"`
	CreatedBy  string `json:"createdBy,omitempty"`
}

// ---------------------------------------------------
// PATCH/PUT /product-blueprints/{id}
// ---------------------------------------------------

type UpdateProductBlueprintInput struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandId   string `json:"brandId"`
	CompanyId string `json:"companyId"`

	ProductBlueprintCategory categorydom.Snapshot `json:"productBlueprintCategory"`

	// nil / empty の扱いは handler / usecase / repository 側の方針に従う。
	// 今回の endpoint 実装では nil または空 map は nil として domain へ渡す。
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	// 当面 frontend では qr 固定。
	// DTO としては既存互換のため productIdTag.type を受ける。
	ProductIdTag ProductIdTagInput `json:"productIdTag"`

	AssigneeId string `json:"assigneeId"`
	UpdatedBy  string `json:"updatedBy,omitempty"`
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/model-refs
// - productBlueprint 起票後に modelRefs（modelId + displayOrder）を追記する
// - updatedAt / updatedBy は更新しない（repo 側で touch しない更新を行う）
//
// 採用方針
//   - 入力: modelIds（順序は「色登録順→サイズ登録順」に並んだもの）
//   - 空 / 重複除外と displayOrder 採番は usecase 側で行う
//   - 出力: detail（既存の toDetailOutput）
// ---------------------------------------------------

type AppendModelRefsInput struct {
	// model テーブルの docId の配列（順序は displayOrder の採番元）
	ModelIds []string `json:"modelIds"`
}

// ---------------------------------------------------
// GET /product-blueprints (list)
// - usecase 側で name 解決済みを返す
// ---------------------------------------------------

type ProductBlueprintListOutput struct {
	ID            string `json:"id"`
	ProductName   string `json:"productName"`
	BrandName     string `json:"brandName"`
	AssigneeName  string `json:"assigneeName"`
	Printed       bool   `json:"printed"`
	CreatedByName string `json:"createdByName"`
	UpdatedByName string `json:"updatedByName"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// ---------------------------------------------------
// GET /product-blueprints/{id} (detail)
// - usecase 側で name 解決済みを返す
// ---------------------------------------------------

type ModelRefOutput struct {
	ModelId      string `json:"modelId"`
	DisplayOrder int    `json:"displayOrder"`
}

type ProductBlueprintDetailOutput struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	Description string `json:"description"`

	CompanyId string `json:"companyId"`
	BrandId   string `json:"brandId"`
	BrandName string `json:"brandName"`

	ProductBlueprintCategoryId string               `json:"productBlueprintCategoryId"`
	ProductBlueprintCategory   categorydom.Snapshot `json:"productBlueprintCategory"`

	// CategoryFields はカテゴリ別の productBlueprint 入力値。
	//
	// 例:
	// - alcohol.sake:
	//   vintage, region, material, alcoholContent, volume
	// - apparel.tops:
	//   weight, fit, material
	// - cosmetics.skincare:
	//   material, volume
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	ProductIdTag *struct {
		Type string `json:"type"`
	} `json:"productIdTag,omitempty"`

	AssigneeId   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	Printed bool `json:"printed"`

	CreatedBy     string `json:"createdBy"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`

	UpdatedBy     string `json:"updatedBy"`
	UpdatedByName string `json:"updatedByName"`
	UpdatedAt     string `json:"updatedAt"`

	// modelRefs（model docId + displayOrder）
	ModelRefs []ModelRefOutput `json:"modelRefs,omitempty"`
}

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

func toCategorySnapshot(in categorydom.Snapshot) pbdom.ProductBlueprintCategorySnapshot {
	return pbdom.ProductBlueprintCategorySnapshot{
		ID:     string(in.ID),
		Code:   string(in.Code),
		NameJa: in.NameJa,
		NameEn: in.NameEn,
		Kind:   in.Kind,
		Path:   append([]string(nil), in.Path...),
	}
}

func toCategoryOutput(
	in pbdom.ProductBlueprintCategorySnapshot,
) categorydom.Snapshot {
	return categorydom.Snapshot{
		ID:     categorydom.CategoryID(in.ID),
		Code:   categorydom.CategoryCode(in.Code),
		NameJa: in.NameJa,
		NameEn: in.NameEn,
		Kind:   categorydom.CategoryKind(in.Kind),
		Path:   append([]string(nil), in.Path...),
	}
}

func normalizeCategoryFields(in map[string]any) pbdom.CategoryFields {
	if len(in) == 0 {
		return nil
	}

	out := make(pbdom.CategoryFields, len(in))
	for key, value := range in {
		if key == "" {
			continue
		}
		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// ---------------------------------------------------
// POST /product-blueprints
// ---------------------------------------------------

func (h *ProductBlueprintHandler) post(w http.ResponseWriter, r *http.Request) {
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
		Description: in.Description,

		BrandID:   in.BrandId,
		CompanyID: in.CompanyId,

		// ProductBlueprintCategory は productBlueprintCategory entity の Snapshot を正として受け取り、
		// productBlueprint domain の denormalized snapshot へ変換する。
		ProductBlueprintCategory: toCategorySnapshot(in.ProductBlueprintCategory),

		// fit / material / weight / qualityAssurance などカテゴリ依存項目は
		// ProductBlueprint 直下ではなく CategoryFields に集約する。
		CategoryFields: normalizeCategoryFields(in.CategoryFields),

		AssigneeID: in.AssigneeId,

		CreatedBy: createdBy,

		// printed は bool。create 時は常に false（未印刷）
		Printed: false,

		ProductIdTag: pbdom.ProductIDTag{
			Type: normalizeTagType(in.ProductIdTag.Type),
		},
	}

	created, err := h.uc.CreateResolved(ctx, pb)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(created)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// PUT/PATCH /product-blueprints/{id}
// ---------------------------------------------------

func (h *ProductBlueprintHandler) update(w http.ResponseWriter, r *http.Request, id string) {
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
		Description: in.Description,

		BrandID:   in.BrandId,
		CompanyID: in.CompanyId,

		// ProductBlueprintCategory は productBlueprintCategory entity の Snapshot を正として受け取り、
		// productBlueprint domain の denormalized snapshot へ変換する。
		ProductBlueprintCategory: toCategorySnapshot(in.ProductBlueprintCategory),

		// fit / material / weight / qualityAssurance などカテゴリ依存項目は
		// ProductBlueprint 直下ではなく CategoryFields に集約する。
		CategoryFields: normalizeCategoryFields(in.CategoryFields),

		AssigneeID: in.AssigneeId,
		UpdatedBy:  updatedBy,

		ProductIdTag: pbdom.ProductIDTag{
			Type: normalizeTagType(in.ProductIdTag.Type),
		},
	}

	updated, err := h.uc.UpdateResolved(ctx, pb)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(updated)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// DELETE /product-blueprints/{id}
// ---------------------------------------------------

func (h *ProductBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	_ = r

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(r.Context(), id); err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------
// GET /product-blueprints/{id}
// ---------------------------------------------------

func (h *ProductBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	pb, err := h.uc.GetByIDResolved(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(pb)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// GET /product-blueprints
// ---------------------------------------------------

func (h *ProductBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.uc.ListByCompanyIDResolved(ctx)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := make([]ProductBlueprintListOutput, 0, len(rows))
	for _, row := range rows {
		pb := row.ProductBlueprint

		createdAt := ""
		if !pb.CreatedAt.IsZero() {
			createdAt = pb.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if !pb.UpdatedAt.IsZero() {
			updatedAt = pb.UpdatedAt.Format(time.RFC3339)
		}

		out = append(out, ProductBlueprintListOutput{
			ID:            pb.ID,
			ProductName:   pb.ProductName,
			BrandName:     row.Names.BrandName,
			AssigneeName:  row.Names.AssigneeName,
			Printed:       pb.Printed,
			CreatedByName: row.Names.CreatedByName,
			UpdatedByName: row.Names.UpdatedByName,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		})
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/model-refs
// ---------------------------------------------------

func (h *ProductBlueprintHandler) appendModelRefs(w http.ResponseWriter, r *http.Request, id string) {
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

	// body として modelIds が未指定 / 空配列の場合だけ handler で弾く。
	// 空文字除外・重複除外・displayOrder 採番は usecase 側に集約する。
	if len(in.ModelIds) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "modelIds is required"})
		return
	}

	updated, err := h.uc.AppendModelRefsResolved(ctx, id, in.ModelIds)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	out := h.toDetailOutput(updated)
	_ = json.NewEncoder(w).Encode(out)
}

// ---------------------------------------------------
// DTO assembler (detail)
// ---------------------------------------------------

func (h *ProductBlueprintHandler) toDetailOutput(
	row pbuc.ProductBlueprintResolved,
) ProductBlueprintDetailOutput {
	pb := row.ProductBlueprint

	createdBy := ""
	if pb.CreatedBy != nil {
		createdBy = *pb.CreatedBy
	}

	updatedBy := ""
	if pb.UpdatedBy != nil {
		updatedBy = *pb.UpdatedBy
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

	category := toCategoryOutput(pb.ProductBlueprintCategory)

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
		Description: pb.Description,

		CompanyId: pb.CompanyID,
		BrandId:   pb.BrandID,
		BrandName: row.Names.BrandName,

		ProductBlueprintCategoryId: pb.ProductBlueprintCategory.ID,
		ProductBlueprintCategory:   category,

		CategoryFields: map[string]any(pb.CategoryFields),

		ProductIdTag: tag,

		AssigneeId:   pb.AssigneeID,
		AssigneeName: row.Names.AssigneeName,

		Printed: pb.Printed,

		CreatedBy:     createdBy,
		CreatedByName: row.Names.CreatedByName,
		CreatedAt:     createdAt,

		UpdatedBy:     updatedBy,
		UpdatedByName: row.Names.UpdatedByName,
		UpdatedAt:     updatedAt,

		ModelRefs: modelRefs,
	}
}

// ---------------------------------------------------
// error helpers
// ---------------------------------------------------

func writeProductBlueprintErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case pbdom.IsInvalid(err):
		code = http.StatusBadRequest
	case pbdom.IsNotFound(err):
		code = http.StatusNotFound
	case pbdom.IsConflict(err):
		code = http.StatusConflict
	case pbdom.IsUnauthorized(err):
		code = http.StatusUnauthorized
	case pbdom.IsForbidden(err):
		code = http.StatusForbidden
	default:
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
