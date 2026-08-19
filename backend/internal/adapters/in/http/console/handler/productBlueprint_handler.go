// backend/internal/adapters/in/http/console/handler/productBlueprint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	pbquery "narratives/internal/application/query/console"
	pbuc "narratives/internal/application/usecase"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintHandlerはProductBlueprint用のHTTP Handlerです。
type ProductBlueprintHandler struct {
	uc              *pbuc.ProductBlueprintUsecase
	managementQuery *pbquery.ProductBlueprintManagementQuery
	detailQuery     *pbquery.ProductBlueprintDetailQuery
}

func NewProductBlueprintHandler(
	uc *pbuc.ProductBlueprintUsecase,
	managementQuery *pbquery.ProductBlueprintManagementQuery,
	detailQuery *pbquery.ProductBlueprintDetailQuery,
) http.Handler {
	return &ProductBlueprintHandler{
		uc:              uc,
		managementQuery: managementQuery,
		detailQuery:     detailQuery,
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
	case (r.Method == http.MethodPut || r.Method == http.MethodPatch) && strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.update(w, r, id)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/product-blueprints/"):
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
	BrandId     string `json:"brandId"`

	// CompanyIdは既存Frontendとの入力互換のため保持します。
	// 永続化に使用するcompanyIdは認証Contextを正とし、
	// ProductBlueprintUsecase側で設定します。
	CompanyId string `json:"companyId"`

	ProductBlueprintCategoryPath []string `json:"productBlueprintCategoryPath"`

	// CategoryFieldsはカテゴリ別のProductBlueprint入力値を受け取ります。
	// Alcoholの容量はここへ保存せず、Model variationのVolumeだけを正とします。
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	// 当面Frontendではqr固定です。
	ProductIdTag ProductIdTagInput `json:"productIdTag"`
	AssigneeId   string            `json:"assigneeId"`
}

// ---------------------------------------------------
// PATCH/PUT /product-blueprints/{id}
// ---------------------------------------------------

type UpdateProductBlueprintInput struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`
	BrandId     string `json:"brandId"`

	// CompanyIdは既存Frontendとの入力互換のため保持します。
	// 通常更新ではcompanyIdを変更せず、
	// ProductBlueprintUsecase側で認証Contextとの境界を確認します。
	CompanyId string `json:"companyId"`

	ProductBlueprintCategoryPath []string `json:"productBlueprintCategoryPath"`

	// nilは「更新しない」、空mapは「空へ更新する」を表します。
	CategoryFields *map[string]any `json:"categoryFields,omitempty"`

	ProductIdTag ProductIdTagInput `json:"productIdTag"`
	AssigneeId   string            `json:"assigneeId"`
}

// ---------------------------------------------------
// GET /product-blueprints
// ---------------------------------------------------

type ProductBlueprintListOutput struct {
	ID            string `json:"id"`
	ProductName   string `json:"productName"`
	BrandName     string `json:"brandName"`
	AssigneeName  string `json:"assigneeName"`
	ProductIdTag  string `json:"productIdTag"`
	Printed       bool   `json:"printed"`
	CreatedByName string `json:"createdByName"`
	UpdatedByName string `json:"updatedByName"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// ---------------------------------------------------
// GET /product-blueprints/{id}
// ---------------------------------------------------

type ProductBlueprintDetailSizeOutput struct {
	ID           string `json:"id"`
	SizeLabel    string `json:"sizeLabel"`
	Length       *int   `json:"length,omitempty"`
	Width        *int   `json:"width,omitempty"`
	Chest        *int   `json:"chest,omitempty"`
	Shoulder     *int   `json:"shoulder,omitempty"`
	SleeveLength *int   `json:"sleeveLength,omitempty"`
	Waist        *int   `json:"waist,omitempty"`
	Hip          *int   `json:"hip,omitempty"`
	Rise         *int   `json:"rise,omitempty"`
	Inseam       *int   `json:"inseam,omitempty"`
	Thigh        *int   `json:"thigh,omitempty"`
	HemWidth     *int   `json:"hemWidth,omitempty"`
}

type ProductBlueprintDetailApparelModelNumberOutput struct {
	Size  string `json:"size"`
	Color string `json:"color"`
	Code  string `json:"code"`
}

type ProductBlueprintDetailVolumeOutput struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type ProductBlueprintDetailVolumeRowOutput struct {
	ID          string `json:"id"`
	VolumeValue int    `json:"volumeValue"`
	VolumeUnit  string `json:"volumeUnit"`
}

type ProductBlueprintDetailAlcoholModelNumberOutput struct {
	Kind   string                             `json:"kind"`
	Volume ProductBlueprintDetailVolumeOutput `json:"volume"`
	Code   string                             `json:"code"`
}

type ProductBlueprintDetailModelStateOutput struct {
	Colors              []string                                         `json:"colors"`
	Sizes               []ProductBlueprintDetailSizeOutput               `json:"sizes"`
	ModelNumbers        []ProductBlueprintDetailApparelModelNumberOutput `json:"modelNumbers"`
	ColorRgbMap         map[string]string                                `json:"colorRgbMap"`
	Volumes             []ProductBlueprintDetailVolumeRowOutput          `json:"volumes"`
	AlcoholModelNumbers []ProductBlueprintDetailAlcoholModelNumberOutput `json:"alcoholModelNumbers"`
}

type ProductBlueprintDetailOutput struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	Description string `json:"description"`

	CompanyId string `json:"companyId"`
	BrandId   string `json:"brandId"`
	BrandName string `json:"brandName"`

	ProductBlueprintCategoryPath []string `json:"productBlueprintCategoryPath"`

	// CategoryFieldsはカテゴリ別のProductBlueprint入力値です。
	// Alcoholの容量はCategoryFieldsへ保存せず、
	// Model variationのVolumeだけを正とします。
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	ProductIdTag *struct {
		Type string `json:"type"`
	} `json:"productIdTag,omitempty"`

	AssigneeId   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`
	Printed      bool   `json:"printed"`

	CreatedBy     string `json:"createdBy"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`

	UpdatedBy     string `json:"updatedBy"`
	UpdatedByName string `json:"updatedByName"`
	UpdatedAt     string `json:"updatedAt"`

	ModelState ProductBlueprintDetailModelStateOutput `json:"modelState"`
}

// ---------------------------------------------------
// Internal normalizers
// ---------------------------------------------------

func normalizeTagType(value string) pbdom.ProductIDTagType {
	switch value {
	case "qr", "QRコード", "QR":
		return pbdom.TagQR
	case "nfc", "NFC":
		return pbdom.TagNFC
	default:
		return pbdom.ProductIDTagType(value)
	}
}

func normalizeCategoryFields(input map[string]any) pbdom.CategoryFields {
	if input == nil {
		return nil
	}

	output := make(pbdom.CategoryFields, len(input))
	for key, value := range input {
		if key == "" {
			continue
		}
		output[key] = value
	}

	return output
}

func memberIDPointerFromContext(ctx context.Context) *string {
	memberID := pbuc.MemberIDFromContext(ctx)
	if memberID == "" {
		return nil
	}
	return &memberID
}

func validProductBlueprintID(id string) bool {
	return id != "" && !strings.Contains(id, "/")
}

// ---------------------------------------------------
// POST /product-blueprints
// ---------------------------------------------------

func (h *ProductBlueprintHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var input CreateProductBlueprintInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	productBlueprint := pbdom.ProductBlueprint{
		ProductName: input.ProductName,
		Description: input.Description,
		BrandID:     input.BrandId,

		// CompanyIDはProductBlueprintUsecaseが認証Contextから設定します。
		CompanyID: "",

		ProductBlueprintCategoryPath: append(
			[]string(nil),
			input.ProductBlueprintCategoryPath...,
		),
		CategoryFields: normalizeCategoryFields(input.CategoryFields),
		AssigneeID:     input.AssigneeId,
		CreatedBy:      memberIDPointerFromContext(ctx),
		Printed:        false,
		ProductIdTag: pbdom.ProductIDTag{
			Type: normalizeTagType(input.ProductIdTag.Type),
		},
	}

	created, err := h.uc.Create(ctx, productBlueprint)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	row, err := h.detailQuery.GetByID(ctx, created.ID)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	output, err := h.toDetailOutput(row)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(output)
}

// ---------------------------------------------------
// PUT/PATCH /product-blueprints/{id}
// ---------------------------------------------------

func (h *ProductBlueprintHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if !validProductBlueprintID(id) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var input UpdateProductBlueprintInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	var categoryFields pbdom.CategoryFields
	if input.CategoryFields != nil {
		categoryFields = normalizeCategoryFields(*input.CategoryFields)
	}

	// Printedは通常更新APIでは変更しません。
	// 印刷済み化はMarkPrinted Usecaseだけを正とします。
	productBlueprint := pbdom.ProductBlueprint{
		ID:          id,
		ProductName: input.ProductName,
		Description: input.Description,
		BrandID:     input.BrandId,

		// CompanyIDは通常更新では変更しません。
		CompanyID: "",

		ProductBlueprintCategoryPath: append(
			[]string(nil),
			input.ProductBlueprintCategoryPath...,
		),
		CategoryFields: categoryFields,
		AssigneeID:     input.AssigneeId,
		UpdatedBy:      memberIDPointerFromContext(ctx),
		ProductIdTag: pbdom.ProductIDTag{
			Type: normalizeTagType(input.ProductIdTag.Type),
		},
	}

	updated, err := h.uc.Update(ctx, productBlueprint)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	row, err := h.detailQuery.GetByID(ctx, updated.ID)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	output, err := h.toDetailOutput(row)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(output)
}

// ---------------------------------------------------
// DELETE /product-blueprints/{id}
// ---------------------------------------------------

// deleteはprinted=falseのProductBlueprintと配下Modelを物理削除します。
func (h *ProductBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if !validProductBlueprintID(id) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
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

	if !validProductBlueprintID(id) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	productBlueprint, err := h.detailQuery.GetByID(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	output, err := h.toDetailOutput(productBlueprint)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(output)
}

// ---------------------------------------------------
// GET /product-blueprints
// ---------------------------------------------------

func (h *ProductBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.managementQuery.ListByCompanyID(ctx)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}

	output := make([]ProductBlueprintListOutput, 0, len(rows))
	for _, row := range rows {
		productBlueprint := row.ProductBlueprint

		createdAt := ""
		if !productBlueprint.CreatedAt.IsZero() {
			createdAt = productBlueprint.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if !productBlueprint.UpdatedAt.IsZero() {
			updatedAt = productBlueprint.UpdatedAt.Format(time.RFC3339)
		}

		output = append(output, ProductBlueprintListOutput{
			ID:            productBlueprint.ID,
			ProductName:   productBlueprint.ProductName,
			BrandName:     row.Names.BrandName,
			AssigneeName:  row.Names.AssigneeName,
			ProductIdTag:  string(productBlueprint.ProductIdTag.Type),
			Printed:       productBlueprint.Printed,
			CreatedByName: row.Names.CreatedByName,
			UpdatedByName: row.Names.UpdatedByName,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		})
	}

	_ = json.NewEncoder(w).Encode(output)
}

// ---------------------------------------------------
// DTO assembler
// ---------------------------------------------------

func toProductBlueprintDetailModelStateOutput(
	state pbquery.ProductBlueprintDetailModelState,
) ProductBlueprintDetailModelStateOutput {
	sizes := make([]ProductBlueprintDetailSizeOutput, 0, len(state.Sizes))
	for _, size := range state.Sizes {
		sizes = append(sizes, ProductBlueprintDetailSizeOutput{
			ID:           size.ID,
			SizeLabel:    size.SizeLabel,
			Length:       size.Length,
			Width:        size.Width,
			Chest:        size.Chest,
			Shoulder:     size.Shoulder,
			SleeveLength: size.SleeveLength,
			Waist:        size.Waist,
			Hip:          size.Hip,
			Rise:         size.Rise,
			Inseam:       size.Inseam,
			Thigh:        size.Thigh,
			HemWidth:     size.HemWidth,
		})
	}

	modelNumbers := make([]ProductBlueprintDetailApparelModelNumberOutput, 0, len(state.ModelNumbers))
	for _, modelNumber := range state.ModelNumbers {
		modelNumbers = append(modelNumbers, ProductBlueprintDetailApparelModelNumberOutput{
			Size:  modelNumber.Size,
			Color: modelNumber.Color,
			Code:  modelNumber.Code,
		})
	}

	volumes := make([]ProductBlueprintDetailVolumeRowOutput, 0, len(state.Volumes))
	for _, volume := range state.Volumes {
		volumes = append(volumes, ProductBlueprintDetailVolumeRowOutput{
			ID:          volume.ID,
			VolumeValue: volume.VolumeValue,
			VolumeUnit:  volume.VolumeUnit,
		})
	}

	alcoholModelNumbers := make([]ProductBlueprintDetailAlcoholModelNumberOutput, 0, len(state.AlcoholModelNumbers))
	for _, modelNumber := range state.AlcoholModelNumbers {
		alcoholModelNumbers = append(alcoholModelNumbers, ProductBlueprintDetailAlcoholModelNumberOutput{
			Kind: modelNumber.Kind,
			Volume: ProductBlueprintDetailVolumeOutput{
				Value: modelNumber.Volume.Value,
				Unit:  modelNumber.Volume.Unit,
			},
			Code: modelNumber.Code,
		})
	}

	colors := append([]string(nil), state.Colors...)
	colorRgbMap := make(map[string]string, len(state.ColorRGBMap))
	for name, rgb := range state.ColorRGBMap {
		colorRgbMap[name] = rgb
	}

	return ProductBlueprintDetailModelStateOutput{
		Colors:              colors,
		Sizes:               sizes,
		ModelNumbers:        modelNumbers,
		ColorRgbMap:         colorRgbMap,
		Volumes:             volumes,
		AlcoholModelNumbers: alcoholModelNumbers,
	}
}

func (h *ProductBlueprintHandler) toDetailOutput(
	row pbquery.ProductBlueprintDetailResolved,
) (ProductBlueprintDetailOutput, error) {
	resolved := row.ProductBlueprint
	productBlueprint := resolved.ProductBlueprint

	createdBy := ""
	if productBlueprint.CreatedBy != nil {
		createdBy = *productBlueprint.CreatedBy
	}

	updatedBy := ""
	if productBlueprint.UpdatedBy != nil {
		updatedBy = *productBlueprint.UpdatedBy
	}

	createdAt := ""
	if !productBlueprint.CreatedAt.IsZero() {
		createdAt = productBlueprint.CreatedAt.Format(time.RFC3339)
	}

	updatedAt := ""
	if !productBlueprint.UpdatedAt.IsZero() {
		updatedAt = productBlueprint.UpdatedAt.Format(time.RFC3339)
	}

	var productIDTag *struct {
		Type string `json:"type"`
	}

	if string(productBlueprint.ProductIdTag.Type) != "" {
		productIDTag = &struct {
			Type string `json:"type"`
		}{
			Type: string(productBlueprint.ProductIdTag.Type),
		}
	}

	return ProductBlueprintDetailOutput{
		ID:          productBlueprint.ID,
		ProductName: productBlueprint.ProductName,
		Description: productBlueprint.Description,
		CompanyId:   productBlueprint.CompanyID,
		BrandId:     productBlueprint.BrandID,
		BrandName:   resolved.Names.BrandName,
		ProductBlueprintCategoryPath: append(
			[]string(nil),
			productBlueprint.ProductBlueprintCategoryPath...,
		),
		CategoryFields: map[string]any(productBlueprint.CategoryFields),
		ProductIdTag:   productIDTag,
		AssigneeId:     productBlueprint.AssigneeID,
		AssigneeName:   resolved.Names.AssigneeName,
		Printed:        productBlueprint.Printed,
		CreatedBy:      createdBy,
		CreatedByName:  resolved.Names.CreatedByName,
		CreatedAt:      createdAt,
		UpdatedBy:      updatedBy,
		UpdatedByName:  resolved.Names.UpdatedByName,
		UpdatedAt:      updatedAt,
		ModelState:     toProductBlueprintDetailModelStateOutput(row.ModelState),
	}, nil
}

// ---------------------------------------------------
// Error helpers
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
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
