package consoleHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	pbuc "narratives/internal/application/usecase"
	brand "narratives/internal/domain/brand"
	"narratives/internal/domain/common"
	memdom "narratives/internal/domain/member"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintHandler は ProductBlueprint 用の HTTP ハンドラです。
type ProductBlueprintHandler struct {
	uc        *pbuc.ProductBlueprintUsecase
	brandSvc  *brand.Service
	memberSvc *memdom.Service
}

func NewProductBlueprintHandler(
	uc *pbuc.ProductBlueprintUsecase,
	brandSvc *brand.Service,
	memberSvc *memdom.Service,
) http.Handler {
	return &ProductBlueprintHandler{
		uc:        uc,
		brandSvc:  brandSvc,
		memberSvc: memberSvc,
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

type ProductBlueprintCategoryInput struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

type ProductBlueprintCategoryOutput struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

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

	ProductBlueprintCategory ProductBlueprintCategoryInput `json:"productBlueprintCategory"`

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

	ProductBlueprintCategory ProductBlueprintCategoryInput `json:"productBlueprintCategory"`

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
//   - 出力: detail（既存の toDetailOutput）
// ---------------------------------------------------

type AppendModelRefsInput struct {
	// model テーブルの docId の配列（順序は displayOrder の採番元）
	ModelIds []string `json:"modelIds"`
}

// ---------------------------------------------------
// GET /product-blueprints (list)
// - backend 側で name 解決済みを返す
// ---------------------------------------------------

type ProductBlueprintListOutput struct {
	ID           string `json:"id"`
	ProductName  string `json:"productName"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	Printed      bool   `json:"printed"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ---------------------------------------------------
// GET /product-blueprints/{id} (detail)
// - backend 側で name 解決済みを返す
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

	ProductBlueprintCategoryId string                         `json:"productBlueprintCategoryId"`
	ProductBlueprintCategory   ProductBlueprintCategoryOutput `json:"productBlueprintCategory"`

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

func toCategorySnapshot(in ProductBlueprintCategoryInput) pbdom.ProductBlueprintCategorySnapshot {
	return pbdom.ProductBlueprintCategorySnapshot{
		ID:     in.ID,
		Code:   in.Code,
		NameJa: in.NameJa,
		NameEn: in.NameEn,
		Kind:   common.ProductCategoryKind(in.Kind),
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

		// ProductBlueprintCategory は productBlueprintCategory ドメインの正データから生成した
		// denormalized snapshot を入れる想定。
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

		// ProductBlueprintCategory は productBlueprintCategory ドメインの正データから生成した
		// denormalized snapshot を入れる想定。
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

func (h *ProductBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
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

func (h *ProductBlueprintHandler) toDetailOutput(
	ctx context.Context,
	pb pbdom.ProductBlueprint,
) ProductBlueprintDetailOutput {
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

	category := ProductBlueprintCategoryOutput{
		ID:     pb.ProductBlueprintCategory.ID,
		Code:   pb.ProductBlueprintCategory.Code,
		NameJa: pb.ProductBlueprintCategory.NameJa,
		NameEn: pb.ProductBlueprintCategory.NameEn,
		Kind:   string(pb.ProductBlueprintCategory.Kind),
		Path:   append([]string(nil), pb.ProductBlueprintCategory.Path...),
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
		Description: pb.Description,

		CompanyId: pb.CompanyID,
		BrandId:   brandId,
		BrandName: brandName,

		ProductBlueprintCategoryId: pb.ProductBlueprintCategory.ID,
		ProductBlueprintCategory:   category,

		CategoryFields: map[string]any(pb.CategoryFields),

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

// ---------------------------------------------------
// name resolvers
// ---------------------------------------------------

// brandId → brandName 解決用ヘルパ
func (h *ProductBlueprintHandler) getBrandNameByID(ctx context.Context, brandID string) string {
	if brandID == "" {
		return ""
	}
	if h.brandSvc == nil {
		return brandID
	}

	name, err := h.brandSvc.GetNameByID(ctx, brandID)
	if err != nil {
		return brandID
	}
	return name
}

// assigneeId → assigneeName 解決用ヘルパ
func (h *ProductBlueprintHandler) getAssigneeNameByID(ctx context.Context, memberID string) string {
	if memberID == "" {
		return ""
	}
	if h.memberSvc == nil {
		return memberID
	}

	name, err := h.memberSvc.GetNameLastFirstByID(ctx, memberID)
	if err != nil {
		return memberID
	}

	if name == "" {
		return memberID
	}
	return name
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
