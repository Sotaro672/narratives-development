// backend/internal/adapters/in/http/console/handler/model_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	modeldom "narratives/internal/domain/model"
)

var (
	ErrModelAccessPolicyNotConfigured = errors.New(
		"model access policy is not configured",
	)

	ErrModelUnauthenticated = errors.New(
		"model access unauthenticated",
	)

	ErrModelForbidden = errors.New(
		"model access forbidden",
	)

	ErrProductBlueprintPrinted = errors.New(
		"product blueprint is already printed",
	)
)

// ModelAccessModeはModel APIに対する読取・変更操作を表します。
type ModelAccessMode uint8

const (
	ModelAccessRead ModelAccessMode = iota
	ModelAccessWrite
)

// ProductBlueprintAccessはModel APIの認可判定に必要な
// ProductBlueprint情報だけを表します。
type ProductBlueprintAccess struct {
	CompanyID string
	Printed   bool
}

// ProductBlueprintAccessLoaderはProductBlueprintの所有会社と
// printed状態を取得します。
type ProductBlueprintAccessLoader func(
	ctx context.Context,
	productBlueprintID string,
) (ProductBlueprintAccess, error)

// ModelAccessPolicyは、全Model APIで共通利用する
// company境界とprinted lockを扱います。
//
// Read:
//   - 認証ContextのcompanyIdを取得する
//   - ProductBlueprintの所有会社と一致することを確認する
//
// Write:
//   - Readの確認に加え、ProductBlueprintがprinted=falseであることを確認する
//
// Model IDを受け取るAPIでは、HandlerがModelを取得して
// productBlueprintIdを解決した後、この判定を実行します。
type ModelAccessPolicy struct {
	loadProductBlueprintAccess ProductBlueprintAccessLoader
}

func NewModelAccessPolicy(
	loadProductBlueprintAccess ProductBlueprintAccessLoader,
) *ModelAccessPolicy {
	return &ModelAccessPolicy{
		loadProductBlueprintAccess: loadProductBlueprintAccess,
	}
}

func (p *ModelAccessPolicy) RequireProductBlueprint(
	ctx context.Context,
	productBlueprintID string,
	mode ModelAccessMode,
) error {
	if p == nil ||
		p.loadProductBlueprintAccess == nil {
		return ErrModelAccessPolicyNotConfigured
	}

	if productBlueprintID == "" {
		return modeldom.ErrInvalidBlueprintID
	}

	companyID := usecase.CompanyIDFromContext(ctx)
	if companyID == "" {
		return ErrModelUnauthenticated
	}

	access, err := p.loadProductBlueprintAccess(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return err
	}

	if access.CompanyID == "" ||
		access.CompanyID != companyID {
		return ErrModelForbidden
	}

	if mode == ModelAccessWrite &&
		access.Printed {
		return ErrProductBlueprintPrinted
	}

	return nil
}

// ModelHandlerは/models関連のエンドポイントを担当します。
type ModelHandler struct {
	uc           *usecase.ModelUsecase
	accessPolicy *ModelAccessPolicy
}

// NewModelHandlerはHTTP Handlerを初期化します。
func NewModelHandler(
	uc *usecase.ModelUsecase,
	accessPolicy *ModelAccessPolicy,
) http.Handler {
	return &ModelHandler{
		uc:           uc,
		accessPolicy: accessPolicy,
	}
}

// ServeHTTPはHTTP routingの入口です。
func (h *ModelHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if h == nil || h.uc == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"model handler is not initialized",
		)
		return
	}

	switch {
	// ------------------------------------------------------------
	// GET /models/by-blueprint/{productBlueprintID}/variations
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(
			r.URL.Path,
			"/models/by-blueprint/",
		):
		rest := strings.TrimPrefix(
			r.URL.Path,
			"/models/by-blueprint/",
		)

		rest = strings.Trim(rest, "/")
		parts := strings.Split(rest, "/")

		if len(parts) == 2 &&
			parts[0] != "" &&
			parts[1] == "variations" {
			h.listVariationsByProductBlueprintID(
				w,
				r,
				parts[0],
			)
			return
		}

		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// GET /models/{id}
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(
			r.URL.Path,
			"/models/",
		):
		id := strings.TrimPrefix(
			r.URL.Path,
			"/models/",
		)

		id = strings.Trim(id, "/")

		if isSingleModelIDPath(id) {
			h.getVariation(
				w,
				r,
				id,
			)
			return
		}

		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// POST /models/{productBlueprintID}/variations
	// ------------------------------------------------------------
	case r.Method == http.MethodPost &&
		strings.HasPrefix(
			r.URL.Path,
			"/models/",
		):
		rest := strings.TrimPrefix(
			r.URL.Path,
			"/models/",
		)

		rest = strings.Trim(rest, "/")
		parts := strings.Split(rest, "/")

		if len(parts) == 2 &&
			parts[0] != "" &&
			parts[1] == "variations" {
			h.createVariation(
				w,
				r,
				parts[0],
			)
			return
		}

		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// PUT /models/{productBlueprintID}/variations
	// PUT /models/{id}
	// ------------------------------------------------------------
	case r.Method == http.MethodPut &&
		strings.HasPrefix(
			r.URL.Path,
			"/models/",
		):
		rest := strings.TrimPrefix(
			r.URL.Path,
			"/models/",
		)

		rest = strings.Trim(rest, "/")
		parts := strings.Split(rest, "/")

		if len(parts) == 2 &&
			parts[0] != "" &&
			parts[1] == "variations" {
			h.replaceVariations(
				w,
				r,
				parts[0],
			)
			return
		}

		if isSingleModelIDPath(rest) {
			h.updateVariation(
				w,
				r,
				rest,
			)
			return
		}

		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// DELETE /models/{id}
	// ------------------------------------------------------------
	case r.Method == http.MethodDelete &&
		strings.HasPrefix(
			r.URL.Path,
			"/models/",
		):
		id := strings.TrimPrefix(
			r.URL.Path,
			"/models/",
		)

		id = strings.Trim(id, "/")

		if isSingleModelIDPath(id) {
			h.deleteVariation(
				w,
				r,
				id,
			)
			return
		}

		writeNotFound(w)
		return

	default:
		writeNotFound(w)
		return
	}
}

// isSingleModelIDPathは/models/{id}系の単体ID pathだけを許可します。
func isSingleModelIDPath(
	id string,
) bool {
	return id != "" &&
		id != "by-blueprint" &&
		id != "variations" &&
		!strings.HasPrefix(
			id,
			"by-blueprint/",
		) &&
		!strings.HasPrefix(
			id,
			"variations/",
		) &&
		!strings.Contains(id, "/")
}

// ------------------------------------------------------------
// Request DTOs
// ------------------------------------------------------------

// modelVariationRequestはCREATEとUPDATEのrequestです。
// productBlueprintIdはpathだけを正とするためbodyでは受け取りません。
type modelVariationRequest struct {
	Kind string `json:"kind"`

	ModelNumber string `json:"modelNumber"`

	Size         string         `json:"size,omitempty"`
	Color        string         `json:"color,omitempty"`
	RGB          int            `json:"rgb"`
	Measurements map[string]int `json:"measurements,omitempty"`

	Volume modeldom.Volume `json:"volume,omitempty"`
}

// replaceModelVariationsRequestはProductBlueprint配下の
// Model variationを一括置換するrequestです。
type replaceModelVariationsRequest struct {
	Variations []modelVariationRequest `json:"variations"`
}

// ------------------------------------------------------------
// Response DTOs
// ------------------------------------------------------------

type colorDTO struct {
	Name string `json:"name"`
	RGB  int    `json:"rgb"`
}

type volumeDTO struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type modelVariationDTO struct {
	ID                 string         `json:"id"`
	ProductBlueprintID string         `json:"productBlueprintId"`
	Kind               string         `json:"kind"`
	ModelNumber        string         `json:"modelNumber"`
	Size               string         `json:"size,omitempty"`
	Color              *colorDTO      `json:"color,omitempty"`
	Measurements       map[string]int `json:"measurements,omitempty"`
	Volume             *volumeDTO     `json:"volume,omitempty"`
	CreatedAt          *string        `json:"createdAt,omitempty"`
	CreatedBy          *string        `json:"createdBy,omitempty"`
	UpdatedAt          *string        `json:"updatedAt,omitempty"`
	UpdatedBy          *string        `json:"updatedBy,omitempty"`
}

// ------------------------------------------------------------
// POST /models/{productBlueprintID}/variations
// ------------------------------------------------------------

func (h *ModelHandler) createVariation(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
) {
	ctx := r.Context()

	if productBlueprintID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid productBlueprintID",
		)
		return
	}

	if err := h.accessPolicy.RequireProductBlueprint(
		ctx,
		productBlueprintID,
		ModelAccessWrite,
	); err != nil {
		writeModelErr(w, err)
		return
	}

	var request modelVariationRequest

	if err := decodeStrictJSON(
		r,
		&request,
	); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid json",
		)
		return
	}

	newVariation, err := toNewModelVariation(
		productBlueprintID,
		request,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	if err := newVariation.Validate(); err != nil {
		writeModelErr(w, err)
		return
	}

	created, err := h.uc.Create(
		ctx,
		newVariation,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	dto, err := toModelVariationDTO(created)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(dto)
}

// ------------------------------------------------------------
// PUT /models/{productBlueprintID}/variations
// ------------------------------------------------------------

// replaceVariationsはProductBlueprint配下のModel variationを
// 単一requestで一括置換します。
//
// 既存documentの削除と新規documentの作成はRepository側の
// 単一transaction内で実行されます。
func (h *ModelHandler) replaceVariations(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
) {
	ctx := r.Context()

	if productBlueprintID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid productBlueprintID",
		)
		return
	}

	if err := h.accessPolicy.RequireProductBlueprint(
		ctx,
		productBlueprintID,
		ModelAccessWrite,
	); err != nil {
		writeModelErr(w, err)
		return
	}

	var request replaceModelVariationsRequest

	if err := decodeStrictJSON(
		r,
		&request,
	); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid json",
		)
		return
	}

	newVariations := make(
		[]modeldom.NewModelVariation,
		0,
		len(request.Variations),
	)

	for _, variationRequest := range request.Variations {
		newVariation, err := toNewModelVariation(
			productBlueprintID,
			variationRequest,
		)
		if err != nil {
			writeModelErr(w, err)
			return
		}

		if err := newVariation.Validate(); err != nil {
			writeModelErr(w, err)
			return
		}

		newVariations = append(
			newVariations,
			newVariation,
		)
	}

	replaced, err := h.uc.ReplaceByProductBlueprintID(
		ctx,
		productBlueprintID,
		newVariations,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	dtos, err := toModelVariationDTOs(replaced)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dtos)
}

// ------------------------------------------------------------
// GET /models/by-blueprint/{productBlueprintID}/variations
// ------------------------------------------------------------

func (h *ModelHandler) listVariationsByProductBlueprintID(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
) {
	ctx := r.Context()

	if productBlueprintID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid productBlueprintID",
		)
		return
	}

	if err := h.accessPolicy.RequireProductBlueprint(
		ctx,
		productBlueprintID,
		ModelAccessRead,
	); err != nil {
		writeModelErr(w, err)
		return
	}

	variations, err := h.uc.ListByProductBlueprintID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	dtos, err := toModelVariationDTOs(variations)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dtos)
}

// ------------------------------------------------------------
// GET /models/{id}
// ------------------------------------------------------------

func (h *ModelHandler) getVariation(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if id == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid id",
		)
		return
	}

	variation, err := h.uc.GetByID(
		ctx,
		id,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	if variation == nil {
		writeModelErr(
			w,
			modeldom.ErrNotFound,
		)
		return
	}

	if err := h.accessPolicy.RequireProductBlueprint(
		ctx,
		variation.GetProductBlueprintID(),
		ModelAccessRead,
	); err != nil {
		writeModelErr(w, err)
		return
	}

	dto, err := toModelVariationDTO(variation)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dto)
}

// ------------------------------------------------------------
// PUT /models/{id}
// ------------------------------------------------------------

func (h *ModelHandler) updateVariation(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if id == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid id",
		)
		return
	}

	current, err := h.uc.GetByID(
		ctx,
		id,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	if current == nil {
		writeModelErr(
			w,
			modeldom.ErrNotFound,
		)
		return
	}

	if err := h.accessPolicy.RequireProductBlueprint(
		ctx,
		current.GetProductBlueprintID(),
		ModelAccessWrite,
	); err != nil {
		writeModelErr(w, err)
		return
	}

	var request modelVariationRequest

	if err := decodeStrictJSON(
		r,
		&request,
	); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid json",
		)
		return
	}

	updates, err := toModelVariationUpdate(request)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	if err := updates.Validate(
		current.GetKind(),
	); err != nil {
		writeModelErr(w, err)
		return
	}

	updated, err := h.uc.Update(
		ctx,
		id,
		updates,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	dto, err := toModelVariationDTO(updated)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dto)
}

// ------------------------------------------------------------
// DELETE /models/{id}
// ------------------------------------------------------------

func (h *ModelHandler) deleteVariation(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	if id == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid id",
		)
		return
	}

	current, err := h.uc.GetByID(
		ctx,
		id,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	if current == nil {
		writeModelErr(
			w,
			modeldom.ErrNotFound,
		)
		return
	}

	if err := h.accessPolicy.RequireProductBlueprint(
		ctx,
		current.GetProductBlueprintID(),
		ModelAccessWrite,
	); err != nil {
		writeModelErr(w, err)
		return
	}

	if err := h.uc.Delete(
		ctx,
		id,
	); err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ------------------------------------------------------------
// Request mapper
// ------------------------------------------------------------

func cloneMeasurements(
	input map[string]int,
) (modeldom.Measurements, error) {
	if input == nil {
		return nil, nil
	}

	measurements := make(
		modeldom.Measurements,
		len(input),
	)

	for key, value := range input {
		if key == "" || value < 0 {
			return nil, modeldom.ErrInvalidMeasurements
		}

		measurements[key] = value
	}

	if len(measurements) == 0 {
		return nil, nil
	}

	return measurements, nil
}

// toNewModelVariationはrequestを
// category-specificなDomain入力へ変換します。
func toNewModelVariation(
	productBlueprintID string,
	request modelVariationRequest,
) (modeldom.NewModelVariation, error) {
	if productBlueprintID == "" {
		return modeldom.NewModelVariation{},
			modeldom.ErrInvalidBlueprintID
	}

	switch request.Kind {
	case string(modeldom.ModelVariationKindApparel):
		measurements, err := cloneMeasurements(
			request.Measurements,
		)
		if err != nil {
			return modeldom.NewModelVariation{}, err
		}

		return modeldom.NewModelVariationFromApparel(
			modeldom.NewApparelModelVariation{
				ProductBlueprintID: productBlueprintID,
				ModelNumber:        request.ModelNumber,
				Size:               request.Size,
				Color: modeldom.Color{
					Name: request.Color,
					RGB:  request.RGB,
				},
				Measurements: measurements,
			},
		), nil

	case string(modeldom.ModelVariationKindAlcohol):
		return modeldom.NewModelVariationFromAlcohol(
			modeldom.NewAlcoholModelVariation{
				ProductBlueprintID: productBlueprintID,
				ModelNumber:        request.ModelNumber,
				Volume:             request.Volume,
			},
		), nil

	default:
		return modeldom.NewModelVariation{},
			modeldom.ErrInvalidKind
	}
}

// toModelVariationUpdateはrequestを
// category-specificな更新入力へ変換します。
func toModelVariationUpdate(
	request modelVariationRequest,
) (modeldom.ModelVariationUpdate, error) {
	modelNumber := request.ModelNumber

	switch request.Kind {
	case string(modeldom.ModelVariationKindApparel):
		measurements, err := cloneMeasurements(
			request.Measurements,
		)
		if err != nil {
			return modeldom.ModelVariationUpdate{}, err
		}

		size := request.Size

		color := modeldom.Color{
			Name: request.Color,
			RGB:  request.RGB,
		}

		return modeldom.ModelVariationUpdate{
			ModelNumber:  &modelNumber,
			Size:         &size,
			Color:        &color,
			Measurements: measurements,
		}, nil

	case string(modeldom.ModelVariationKindAlcohol):
		volume := request.Volume

		return modeldom.ModelVariationUpdate{
			ModelNumber: &modelNumber,
			Volume:      &volume,
		}, nil

	default:
		return modeldom.ModelVariationUpdate{},
			modeldom.ErrInvalidKind
	}
}

// ------------------------------------------------------------
// Response mapper
// ------------------------------------------------------------

func toModelVariationDTO(
	variation modeldom.ModelVariation,
) (modelVariationDTO, error) {
	if variation == nil {
		return modelVariationDTO{},
			modeldom.ErrInvalid
	}

	switch modelVariation := variation.(type) {
	case modeldom.ApparelModelVariation:
		return toApparelModelVariationDTO(
			modelVariation,
		), nil

	case modeldom.AlcoholModelVariation:
		return toAlcoholModelVariationDTO(
			modelVariation,
		), nil

	default:
		return modelVariationDTO{},
			modeldom.ErrInvalidKind
	}
}

func toApparelModelVariationDTO(
	variation modeldom.ApparelModelVariation,
) modelVariationDTO {
	return modelVariationDTO{
		ID:                 variation.ID,
		ProductBlueprintID: variation.ProductBlueprintID,
		Kind: string(
			modeldom.ModelVariationKindApparel,
		),
		ModelNumber: variation.ModelNumber,
		Size:        variation.Size,
		Color: &colorDTO{
			Name: variation.Color.Name,
			RGB:  variation.Color.RGB,
		},
		Measurements: cloneMeasurementsForDTO(
			variation.Measurements,
		),
		CreatedAt: timePtrToRFC3339(
			&variation.CreatedAt,
		),
		CreatedBy: variation.CreatedBy,
		UpdatedAt: timePtrToRFC3339(
			&variation.UpdatedAt,
		),
		UpdatedBy: variation.UpdatedBy,
	}
}

func toAlcoholModelVariationDTO(
	variation modeldom.AlcoholModelVariation,
) modelVariationDTO {
	return modelVariationDTO{
		ID:                 variation.ID,
		ProductBlueprintID: variation.ProductBlueprintID,
		Kind: string(
			modeldom.ModelVariationKindAlcohol,
		),
		ModelNumber: variation.ModelNumber,
		Volume: &volumeDTO{
			Value: variation.Volume.Value,
			Unit:  variation.Volume.Unit,
		},
		CreatedAt: timePtrToRFC3339(
			&variation.CreatedAt,
		),
		CreatedBy: variation.CreatedBy,
		UpdatedAt: timePtrToRFC3339(
			&variation.UpdatedAt,
		),
		UpdatedBy: variation.UpdatedBy,
	}
}

func toModelVariationDTOs(
	variations []modeldom.ModelVariation,
) ([]modelVariationDTO, error) {
	output := make(
		[]modelVariationDTO,
		0,
		len(variations),
	)

	for _, variation := range variations {
		dto, err := toModelVariationDTO(
			variation,
		)
		if err != nil {
			return nil, err
		}

		output = append(
			output,
			dto,
		)
	}

	return output, nil
}

func cloneMeasurementsForDTO(
	measurements modeldom.Measurements,
) map[string]int {
	if measurements == nil {
		return nil
	}

	output := make(
		map[string]int,
		len(measurements),
	)

	for key, value := range measurements {
		output[key] = value
	}

	return output
}

func timePtrToRFC3339(
	value *time.Time,
) *string {
	if value == nil || value.IsZero() {
		return nil
	}

	formatted := value.UTC().Format(
		time.RFC3339,
	)

	return &formatted
}

// ------------------------------------------------------------
// JSON helpers
// ------------------------------------------------------------

// decodeStrictJSONは未定義フィールドを拒否します。
// body側へproductBlueprintIdを送った場合も400を返します。
func decodeStrictJSON(
	r *http.Request,
	destination any,
) error {
	if r == nil || r.Body == nil {
		return errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var trailingValue any

	if err := decoder.Decode(
		&trailingValue,
	); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New(
				"multiple json values are not allowed",
			)
		}

		return err
	}

	return nil
}

// ------------------------------------------------------------
// Error helpers
// ------------------------------------------------------------

func writeNotFound(
	w http.ResponseWriter,
) {
	writeJSONError(
		w,
		http.StatusNotFound,
		"not_found",
	)
}

func writeJSONError(
	w http.ResponseWriter,
	statusCode int,
	message string,
) {
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"error": message,
		},
	)
}

func writeModelErr(
	w http.ResponseWriter,
	err error,
) {
	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(
		err,
		ErrModelUnauthenticated,
	):
		statusCode = http.StatusUnauthorized

	case errors.Is(
		err,
		ErrModelForbidden,
	):
		statusCode = http.StatusForbidden

	case errors.Is(
		err,
		ErrProductBlueprintPrinted,
	):
		statusCode = http.StatusConflict

	case errors.Is(
		err,
		modeldom.ErrInvalidID,
	),
		errors.Is(
			err,
			modeldom.ErrInvalidProductID,
		),
		errors.Is(
			err,
			modeldom.ErrInvalidBlueprintID,
		),
		errors.Is(
			err,
			modeldom.ErrInvalidModelNumber,
		),
		errors.Is(
			err,
			modeldom.ErrInvalidSize,
		),
		errors.Is(
			err,
			modeldom.ErrInvalidColor,
		),
		errors.Is(
			err,
			modeldom.ErrInvalidMeasurements,
		),
		errors.Is(
			err,
			modeldom.ErrInvalidVolume,
		),
		errors.Is(
			err,
			modeldom.ErrInvalidVolumeUnit,
		),
		errors.Is(
			err,
			modeldom.ErrInvalidKind,
		),
		errors.Is(
			err,
			modeldom.ErrProductMismatch,
		),
		errors.Is(
			err,
			modeldom.ErrInvalid,
		):
		statusCode = http.StatusBadRequest

	case errors.Is(
		err,
		modeldom.ErrNotFound,
	):
		statusCode = http.StatusNotFound

	case errors.Is(
		err,
		modeldom.ErrAtomicReplaceLimitExceeded,
	),
		errors.Is(
			err,
			modeldom.ErrConflict,
		):
		statusCode = http.StatusConflict
	}

	writeJSONError(
		w,
		statusCode,
		err.Error(),
	)
}
