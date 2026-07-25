// backend/internal/adapters/in/http/console/handler/productBlueprint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	pbquery "narratives/internal/application/query/console"
	pbuc "narratives/internal/application/usecase"
	pbdom "narratives/internal/domain/productBlueprint"
	categorydom "narratives/internal/domain/productBlueprintCategory"
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

func (
	h *ProductBlueprintHandler,
) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	path := strings.TrimRight(
		r.URL.Path,
		"/",
	)

	switch {
	case r.Method == http.MethodGet &&
		path == "/product-blueprints":
		h.list(w, r)

	case r.Method == http.MethodPost &&
		path == "/product-blueprints":
		h.post(w, r)

	case r.Method == http.MethodPost &&
		strings.HasPrefix(
			path,
			"/product-blueprints/",
		) &&
		strings.HasSuffix(
			path,
			"/restore",
		):
		id := strings.TrimSuffix(
			strings.TrimPrefix(
				path,
				"/product-blueprints/",
			),
			"/restore",
		)

		h.restore(
			w,
			r,
			id,
		)

	case (r.Method == http.MethodPut ||
		r.Method == http.MethodPatch) &&
		strings.HasPrefix(
			path,
			"/product-blueprints/",
		):
		id := strings.TrimPrefix(
			path,
			"/product-blueprints/",
		)

		h.update(
			w,
			r,
			id,
		)

	case r.Method == http.MethodDelete &&
		strings.HasPrefix(
			path,
			"/product-blueprints/",
		):
		id := strings.TrimPrefix(
			path,
			"/product-blueprints/",
		)

		h.softDelete(
			w,
			r,
			id,
		)

	case r.Method == http.MethodGet &&
		strings.HasPrefix(
			path,
			"/product-blueprints/",
		):
		id := strings.TrimPrefix(
			path,
			"/product-blueprints/",
		)

		h.get(
			w,
			r,
			id,
		)

	default:
		w.WriteHeader(
			http.StatusNotFound,
		)

		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "not_found",
			},
		)
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

	BrandId string `json:"brandId"`

	// CompanyIdは既存Frontendとの入力互換のため保持します。
	//
	// 永続化に使用するcompanyIdは認証Contextを正とし、
	// ProductBlueprintUsecase側で設定します。
	CompanyId string `json:"companyId"`

	ProductBlueprintCategory categorydom.Snapshot `json:"productBlueprintCategory"`

	// CategoryFieldsはカテゴリ別のProductBlueprint入力値を受け取ります。
	//
	// 例:
	//   - alcohol.sake:
	//     vintage、region、material、alcoholContent
	//   - apparel.tops:
	//     weight、fit、material
	//   - cosmetics.skincare:
	//     material、volume
	//
	// Alcoholの容量はここへ保存せず、
	// Model variationのVolumeだけを正とします。
	//
	// brandId、productName、productIdTagType、descriptionなどの
	// 共通fieldはここには入れません。
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	// 当面Frontendではqr固定です。
	// DTOとしては既存互換のためproductIdTag.typeを受け取ります。
	ProductIdTag ProductIdTagInput `json:"productIdTag"`

	AssigneeId string `json:"assigneeId"`
}

// ---------------------------------------------------
// PATCH/PUT /product-blueprints/{id}
// ---------------------------------------------------

type UpdateProductBlueprintInput struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandId string `json:"brandId"`

	// CompanyIdは既存Frontendとの入力互換のため保持します。
	//
	// 通常更新ではcompanyIdを変更せず、
	// ProductBlueprintUsecase側で認証Contextとの境界を確認します。
	CompanyId string `json:"companyId"`

	ProductBlueprintCategory categorydom.Snapshot `json:"productBlueprintCategory"`

	// nilは「更新しない」、空mapは「空へ更新する」を表します。
	//
	// mapへのポインタにすることで、JSON上の未指定と空objectを
	// 区別します。
	CategoryFields *map[string]any `json:"categoryFields,omitempty"`

	// 当面Frontendではqr固定です。
	// DTOとしては既存互換のためproductIdTag.typeを受け取ります。
	ProductIdTag ProductIdTagInput `json:"productIdTag"`

	AssigneeId string `json:"assigneeId"`
}

// ---------------------------------------------------
// GET /product-blueprints
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
// GET /product-blueprints/{id}
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

	// CategoryFieldsはカテゴリ別のProductBlueprint入力値です。
	//
	// Alcoholの容量はCategoryFieldsへ保存せず、
	// Model variationのVolumeだけを正とします。
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

	ModelRefs []ModelRefOutput `json:"modelRefs,omitempty"`
}

// ---------------------------------------------------
// Internal normalizers
// ---------------------------------------------------

func normalizeTagType(
	value string,
) pbdom.ProductIDTagType {
	switch value {
	case "qr",
		"QRコード",
		"QR":
		return pbdom.TagQR

	case "nfc",
		"NFC":
		return pbdom.TagNFC

	default:
		return pbdom.ProductIDTagType(
			value,
		)
	}
}

func toCategorySnapshot(
	input categorydom.Snapshot,
) pbdom.ProductBlueprintCategorySnapshot {
	return pbdom.ProductBlueprintCategorySnapshot{
		ID:     string(input.ID),
		Code:   string(input.Code),
		NameJa: input.NameJa,
		NameEn: input.NameEn,
		Kind:   input.Kind,
		Path: append(
			[]string(nil),
			input.Path...,
		),
	}
}

func toCategoryOutput(
	input pbdom.ProductBlueprintCategorySnapshot,
) categorydom.Snapshot {
	return categorydom.Snapshot{
		ID: categorydom.CategoryID(
			input.ID,
		),
		Code: categorydom.CategoryCode(
			input.Code,
		),
		NameJa: input.NameJa,
		NameEn: input.NameEn,
		Kind: categorydom.CategoryKind(
			input.Kind,
		),
		Path: append(
			[]string(nil),
			input.Path...,
		),
	}
}

func normalizeCategoryFields(
	input map[string]any,
) pbdom.CategoryFields {
	if input == nil {
		return nil
	}

	output := make(
		pbdom.CategoryFields,
		len(input),
	)

	for key, value := range input {
		if key == "" {
			continue
		}

		output[key] = value
	}

	return output
}

func memberIDPointerFromContext(
	ctx context.Context,
) *string {
	memberID := pbuc.MemberIDFromContext(
		ctx,
	)

	if memberID == "" {
		return nil
	}

	return &memberID
}

func validProductBlueprintID(
	id string,
) bool {
	return id != "" &&
		!strings.Contains(
			id,
			"/",
		)
}

// ---------------------------------------------------
// POST /product-blueprints
// ---------------------------------------------------

func (
	h *ProductBlueprintHandler,
) post(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	var input CreateProductBlueprintInput

	if err := json.NewDecoder(
		r.Body,
	).Decode(
		&input,
	); err != nil {
		w.WriteHeader(
			http.StatusBadRequest,
		)

		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid json",
			},
		)

		return
	}

	productBlueprint := pbdom.ProductBlueprint{
		ProductName: input.ProductName,
		Description: input.Description,

		BrandID: input.BrandId,

		// CompanyIDはProductBlueprintUsecaseが
		// 認証Contextから設定します。
		CompanyID: "",

		ProductBlueprintCategory: toCategorySnapshot(
			input.ProductBlueprintCategory,
		),

		CategoryFields: normalizeCategoryFields(
			input.CategoryFields,
		),

		AssigneeID: input.AssigneeId,

		CreatedBy: memberIDPointerFromContext(
			ctx,
		),

		Printed: false,

		ProductIdTag: pbdom.ProductIDTag{
			Type: normalizeTagType(
				input.ProductIdTag.Type,
			),
		},
	}

	created, err := h.uc.Create(
		ctx,
		productBlueprint,
	)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	row, err := h.detailQuery.GetByID(
		ctx,
		created.ID,
	)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	output := h.toDetailOutput(
		row,
	)

	w.WriteHeader(
		http.StatusCreated,
	)

	_ = json.NewEncoder(w).Encode(
		output,
	)
}

// ---------------------------------------------------
// PUT/PATCH /product-blueprints/{id}
// ---------------------------------------------------

func (
	h *ProductBlueprintHandler,
) update(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if !validProductBlueprintID(id) {
		w.WriteHeader(
			http.StatusBadRequest,
		)

		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid id",
			},
		)

		return
	}

	var input UpdateProductBlueprintInput

	if err := json.NewDecoder(
		r.Body,
	).Decode(
		&input,
	); err != nil {
		w.WriteHeader(
			http.StatusBadRequest,
		)

		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid json",
			},
		)

		return
	}

	var categoryFields pbdom.CategoryFields

	if input.CategoryFields != nil {
		categoryFields =
			normalizeCategoryFields(
				*input.CategoryFields,
			)
	}

	// Printedは通常更新APIでは変更しません。
	// 印刷済み化はMarkPrinted Usecaseだけを正とします。
	productBlueprint := pbdom.ProductBlueprint{
		ID:          id,
		ProductName: input.ProductName,
		Description: input.Description,

		BrandID: input.BrandId,

		// CompanyIDは通常更新では変更しません。
		// Usecaseが現在のEntityと認証Contextを比較します。
		CompanyID: "",

		ProductBlueprintCategory: toCategorySnapshot(
			input.ProductBlueprintCategory,
		),

		CategoryFields: categoryFields,

		AssigneeID: input.AssigneeId,

		UpdatedBy: memberIDPointerFromContext(
			ctx,
		),

		ProductIdTag: pbdom.ProductIDTag{
			Type: normalizeTagType(
				input.ProductIdTag.Type,
			),
		},
	}

	updated, err := h.uc.Update(
		ctx,
		productBlueprint,
	)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	row, err := h.detailQuery.GetByID(
		ctx,
		updated.ID,
	)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	output := h.toDetailOutput(
		row,
	)

	_ = json.NewEncoder(w).Encode(
		output,
	)
}

// ---------------------------------------------------
// DELETE /product-blueprints/{id}
// ---------------------------------------------------

// softDeleteはProductBlueprintと配下Modelを論理削除します。
//
// DocumentはこのHandlerから物理削除しません。
// 物理削除は復旧期限経過後にPurge batchだけが実行します。
func (
	h *ProductBlueprintHandler,
) softDelete(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if !validProductBlueprintID(id) {
		w.WriteHeader(
			http.StatusBadRequest,
		)

		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid id",
			},
		)

		return
	}

	_, err := h.uc.SoftDelete(
		ctx,
		id,
		memberIDPointerFromContext(
			ctx,
		),
	)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/restore
// ---------------------------------------------------

// restoreは論理削除から30日以内のProductBlueprintと
// 配下Modelを復旧します。
func (
	h *ProductBlueprintHandler,
) restore(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if !validProductBlueprintID(id) {
		w.WriteHeader(
			http.StatusBadRequest,
		)

		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid id",
			},
		)

		return
	}

	restored, err := h.uc.Restore(
		ctx,
		id,
		memberIDPointerFromContext(
			ctx,
		),
	)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	// Restore後はactive状態へ戻っているため、
	// 通常のDetailQueryから取得します。
	row, err := h.detailQuery.GetByID(
		ctx,
		restored.ID,
	)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	output := h.toDetailOutput(
		row,
	)

	_ = json.NewEncoder(w).Encode(
		output,
	)
}

// ---------------------------------------------------
// GET /product-blueprints/{id}
// ---------------------------------------------------

func (
	h *ProductBlueprintHandler,
) get(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if !validProductBlueprintID(id) {
		w.WriteHeader(
			http.StatusBadRequest,
		)

		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid id",
			},
		)

		return
	}

	productBlueprint, err :=
		h.detailQuery.GetByID(
			ctx,
			id,
		)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	output := h.toDetailOutput(
		productBlueprint,
	)

	_ = json.NewEncoder(w).Encode(
		output,
	)
}

// ---------------------------------------------------
// GET /product-blueprints
// ---------------------------------------------------

func (
	h *ProductBlueprintHandler,
) list(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	rows, err :=
		h.managementQuery.
			ListByCompanyID(ctx)
	if err != nil {
		writeProductBlueprintErr(
			w,
			err,
		)

		return
	}

	output := make(
		[]ProductBlueprintListOutput,
		0,
		len(rows),
	)

	for _, row := range rows {
		productBlueprint :=
			row.ProductBlueprint

		createdAt := ""

		if !productBlueprint.
			CreatedAt.
			IsZero() {
			createdAt =
				productBlueprint.
					CreatedAt.
					Format(
						time.RFC3339,
					)
		}

		updatedAt := ""

		if !productBlueprint.
			UpdatedAt.
			IsZero() {
			updatedAt =
				productBlueprint.
					UpdatedAt.
					Format(
						time.RFC3339,
					)
		}

		output = append(
			output,
			ProductBlueprintListOutput{
				ID: productBlueprint.ID,

				ProductName: productBlueprint.ProductName,

				BrandName: row.Names.BrandName,

				AssigneeName: row.Names.AssigneeName,

				Printed: productBlueprint.Printed,

				CreatedByName: row.Names.CreatedByName,

				UpdatedByName: row.Names.UpdatedByName,

				CreatedAt: createdAt,

				UpdatedAt: updatedAt,
			},
		)
	}

	_ = json.NewEncoder(w).Encode(
		output,
	)
}

// ---------------------------------------------------
// DTO assembler
// ---------------------------------------------------

func (
	h *ProductBlueprintHandler,
) toDetailOutput(
	row pbquery.ProductBlueprintResolved,
) ProductBlueprintDetailOutput {
	productBlueprint :=
		row.ProductBlueprint

	createdBy := ""

	if productBlueprint.CreatedBy != nil {
		createdBy =
			*productBlueprint.CreatedBy
	}

	updatedBy := ""

	if productBlueprint.UpdatedBy != nil {
		updatedBy =
			*productBlueprint.UpdatedBy
	}

	createdAt := ""

	if !productBlueprint.
		CreatedAt.
		IsZero() {
		createdAt =
			productBlueprint.
				CreatedAt.
				Format(
					time.RFC3339,
				)
	}

	updatedAt := ""

	if !productBlueprint.
		UpdatedAt.
		IsZero() {
		updatedAt =
			productBlueprint.
				UpdatedAt.
				Format(
					time.RFC3339,
				)
	}

	var productIDTag *struct {
		Type string `json:"type"`
	}

	if string(
		productBlueprint.
			ProductIdTag.
			Type,
	) != "" {
		productIDTag = &struct {
			Type string `json:"type"`
		}{
			Type: string(
				productBlueprint.
					ProductIdTag.
					Type,
			),
		}
	}

	category :=
		toCategoryOutput(
			productBlueprint.
				ProductBlueprintCategory,
		)

	var modelRefs []ModelRefOutput

	if len(productBlueprint.ModelRefs) > 0 {
		modelRefs = make(
			[]ModelRefOutput,
			0,
			len(productBlueprint.ModelRefs),
		)

		for _, modelRef := range productBlueprint.ModelRefs {
			if modelRef.ModelID == "" {
				continue
			}

			modelRefs = append(
				modelRefs,
				ModelRefOutput{
					ModelId: modelRef.ModelID,

					DisplayOrder: modelRef.DisplayOrder,
				},
			)
		}
	}

	return ProductBlueprintDetailOutput{
		ID: productBlueprint.ID,

		ProductName: productBlueprint.ProductName,

		Description: productBlueprint.Description,

		CompanyId: productBlueprint.CompanyID,

		BrandId: productBlueprint.BrandID,

		BrandName: row.Names.BrandName,

		ProductBlueprintCategoryId: productBlueprint.
			ProductBlueprintCategory.
			ID,

		ProductBlueprintCategory: category,

		CategoryFields: map[string]any(
			productBlueprint.
				CategoryFields,
		),

		ProductIdTag: productIDTag,

		AssigneeId: productBlueprint.AssigneeID,

		AssigneeName: row.Names.AssigneeName,

		Printed: productBlueprint.Printed,

		CreatedBy: createdBy,

		CreatedByName: row.Names.CreatedByName,

		CreatedAt: createdAt,

		UpdatedBy: updatedBy,

		UpdatedByName: row.Names.UpdatedByName,

		UpdatedAt: updatedAt,

		ModelRefs: modelRefs,
	}
}

// ---------------------------------------------------
// Error helpers
// ---------------------------------------------------

func writeProductBlueprintErr(
	w http.ResponseWriter,
	err error,
) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(
		err,
		pbdom.ErrRestorePeriodExpired,
	):
		code = http.StatusGone

	case errors.Is(
		err,
		pbdom.ErrInvalidDeletionState,
	):
		code = http.StatusConflict

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

	w.WriteHeader(
		code,
	)

	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"error": err.Error(),
		},
	)
}
