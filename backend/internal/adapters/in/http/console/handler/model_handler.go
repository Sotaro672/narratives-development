// backend/internal/adapters/in/http/console/handler/model_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	usecase "narratives/internal/application/usecase"
	modeldom "narratives/internal/domain/model"
	"net/http"
	"strings"
	"time"
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

// ProductBlueprintAccessはModel APIの認可判定に必要なPB情報だけを表します。
type ProductBlueprintAccess struct {
	CompanyID string
	Printed   bool
}

// ProductBlueprintAccessLoaderはPBの所有会社とprinted状態を取得します。
type ProductBlueprintAccessLoader func(
	ctx context.Context,
	productBlueprintID string,
) (ProductBlueprintAccess, error)

// ModelAccessPolicyは全Model APIで共通利用する
// company境界・printed lockです。
//
// Read:
//   - 認証ContextのcompanyIdを取得する
//   - PBの所有会社と一致することを確認する
//
// Write:
//   - Readの確認に加え、PBがprinted=falseであることを確認する
//
// Model IDを受け取るAPIでは、HandlerがModelを取得して
// productBlueprintIdを解決した後、このPB判定を実行します。
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
	if mode == ModelAccessWrite && access.Printed {
		return ErrProductBlueprintPrinted
	}
	return nil
}

// ModelHandlerは/models関連のエンドポイントを担当します。
type ModelHandler struct {
	uc           *usecase.ModelUsecase
	accessPolicy *ModelAccessPolicy
}

// NewModelHandlerはHTTPハンドラを初期化します。
func NewModelHandler(
	uc *usecase.ModelUsecase,
	accessPolicy *ModelAccessPolicy,
) http.Handler {
	return &ModelHandler{
		uc:           uc,
		accessPolicy: accessPolicy,
	}
}

// ServeHTTPはHTTPルーティングの入口です。
func (h *ModelHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)
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
			parts[1] == "variations" &&
			!strings.Contains(parts[0], "/") {
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
			h.getVariation(w, r, id)
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
			parts[1] == "variations" &&
			!strings.Contains(parts[0], "/") {
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
	// PUT /models/{id}
	// ------------------------------------------------------------
	case r.Method == http.MethodPut &&
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
			h.updateVariation(w, r, id)
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
			h.deleteVariation(w, r, id)
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
func isSingleModelIDPath(id string) bool {
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
// modelVariationRequestはCREATE / UPDATEのrequestです。
// productBlueprintIdはpathだけを正とするためbodyでは受け取りません。
type modelVariationRequest struct {
	// 必須: "apparel"または"alcohol"
	Kind string `json:"kind"`
	// common
	ModelNumber string `json:"modelNumber"`
	// apparel
	Size         string         `json:"size,omitempty"`
	Color        string         `json:"color,omitempty"`
	RGB          int            `json:"rgb"`
	Measurements map[string]int `json:"measurements,omitempty"`
	// alcohol
	Volume modeldom.Volume `json:"volume,omitempty"`
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
	var req modelVariationRequest
	if err := decodeStrictJSON(
		r,
		&req,
	); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid json",
		)
		return
	}
	newVar, err := toNewModelVariation(
		productBlueprintID,
		req,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	if err := newVar.Validate(); err != nil {
		writeModelErr(w, err)
		return
	}
	mv, err := h.uc.Create(ctx, newVar)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	dto, err := toModelVariationDTO(mv)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(dto)
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
	dtos, err := toModelVariationDTOs(
		variations,
	)
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
	mv, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	if err := h.accessPolicy.RequireProductBlueprint(
		ctx,
		mv.GetProductBlueprintID(),
		ModelAccessRead,
	); err != nil {
		writeModelErr(w, err)
		return
	}
	dto, err := toModelVariationDTO(mv)
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
	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
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
	var req modelVariationRequest
	if err := decodeStrictJSON(
		r,
		&req,
	); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid json",
		)
		return
	}
	updates, err := toModelVariationUpdate(req)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	mv, err := h.uc.Update(
		ctx,
		id,
		updates,
	)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	dto, err := toModelVariationDTO(mv)
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
	current, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
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
// Mapper
// ------------------------------------------------------------
func cloneMeasurements(
	in map[string]int,
) modeldom.Measurements {
	if len(in) == 0 {
		return nil
	}
	measurements := make(
		modeldom.Measurements,
		len(in),
	)
	for key, value := range in {
		if key == "" {
			continue
		}
		measurements[key] = value
	}
	if len(measurements) == 0 {
		return nil
	}
	return measurements
}

// toNewModelVariationはrequestを
// category-specificなdomain inputへ変換します。
func toNewModelVariation(
	productBlueprintID string,
	req modelVariationRequest,
) (modeldom.NewModelVariation, error) {
	switch req.Kind {
	case string(
		modeldom.ModelVariationKindApparel,
	):
		return modeldom.NewModelVariationFromApparel(
			modeldom.NewApparelModelVariation{
				ProductBlueprintID: productBlueprintID,
				ModelNumber:        req.ModelNumber,
				Size:               req.Size,
				Color: modeldom.Color{
					Name: req.Color,
					RGB:  req.RGB,
				},
				Measurements: cloneMeasurements(
					req.Measurements,
				),
			},
		), nil
	case string(
		modeldom.ModelVariationKindAlcohol,
	):
		return modeldom.NewModelVariationFromAlcohol(
			modeldom.NewAlcoholModelVariation{
				ProductBlueprintID: productBlueprintID,
				ModelNumber:        req.ModelNumber,
				Volume:             req.Volume,
			},
		), nil
	default:
		return modeldom.NewModelVariation{},
			modeldom.ErrInvalid
	}
}

// toModelVariationUpdateはrequestを
// category-specificなupdate DTOへ変換します。
func toModelVariationUpdate(
	req modelVariationRequest,
) (modeldom.ModelVariationUpdate, error) {
	modelNumber := req.ModelNumber
	switch req.Kind {
	case string(
		modeldom.ModelVariationKindApparel,
	):
		size := req.Size
		color := modeldom.Color{
			Name: req.Color,
			RGB:  req.RGB,
		}
		return modeldom.ModelVariationUpdate{
			ModelNumber: &modelNumber,
			Size:        &size,
			Color:       &color,
			Measurements: cloneMeasurements(
				req.Measurements,
			),
		}, nil
	case string(
		modeldom.ModelVariationKindAlcohol,
	):
		volume := req.Volume
		return modeldom.ModelVariationUpdate{
			ModelNumber: &modelNumber,
			Volume:      &volume,
		}, nil
	default:
		return modeldom.ModelVariationUpdate{},
			modeldom.ErrInvalid
	}
}
func toModelVariationDTO(
	mv modeldom.ModelVariation,
) (modelVariationDTO, error) {
	if mv == nil {
		return modelVariationDTO{},
			modeldom.ErrInvalid
	}
	switch variation := mv.(type) {
	case modeldom.ApparelModelVariation:
		return toApparelModelVariationDTO(
			variation,
		), nil
	case modeldom.AlcoholModelVariation:
		return toAlcoholModelVariationDTO(
			variation,
		), nil
	default:
		return modelVariationDTO{},
			modeldom.ErrInvalid
	}
}
func toApparelModelVariationDTO(
	mv modeldom.ApparelModelVariation,
) modelVariationDTO {
	return modelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		Kind: string(
			modeldom.ModelVariationKindApparel,
		),
		ModelNumber: mv.ModelNumber,
		Size:        mv.Size,
		Color: &colorDTO{
			Name: mv.Color.Name,
			RGB:  mv.Color.RGB,
		},
		Measurements: cloneMeasurementsForDTO(
			mv.Measurements,
		),
		CreatedAt: timePtrToRFC3339(
			&mv.CreatedAt,
		),
		CreatedBy: mv.CreatedBy,
		UpdatedAt: timePtrToRFC3339(
			&mv.UpdatedAt,
		),
		UpdatedBy: mv.UpdatedBy,
	}
}
func toAlcoholModelVariationDTO(
	mv modeldom.AlcoholModelVariation,
) modelVariationDTO {
	return modelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		Kind: string(
			modeldom.ModelVariationKindAlcohol,
		),
		ModelNumber: mv.ModelNumber,
		Volume: &volumeDTO{
			Value: mv.Volume.Value,
			Unit:  mv.Volume.Unit,
		},
		CreatedAt: timePtrToRFC3339(
			&mv.CreatedAt,
		),
		CreatedBy: mv.CreatedBy,
		UpdatedAt: timePtrToRFC3339(
			&mv.UpdatedAt,
		),
		UpdatedBy: mv.UpdatedBy,
	}
}
func toModelVariationDTOs(
	variations []modeldom.ModelVariation,
) ([]modelVariationDTO, error) {
	out := make(
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
		out = append(out, dto)
	}
	return out, nil
}
func cloneMeasurementsForDTO(
	measurements modeldom.Measurements,
) map[string]int {
	if measurements == nil {
		return nil
	}
	out := make(
		map[string]int,
		len(measurements),
	)
	for key, value := range measurements {
		out[key] = value
	}
	return out
}
func timePtrToRFC3339(
	t *time.Time,
) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	formatted := t.UTC().Format(
		time.RFC3339,
	)
	return &formatted
}

// ------------------------------------------------------------
// JSON helpers
// ------------------------------------------------------------
// decodeStrictJSONは未定義フィールドを拒否します。
// これによりbody側へproductBlueprintIdを送った場合も400になります。
func decodeStrictJSON(
	r *http.Request,
	dst any,
) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(
		&trailing,
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
	code := http.StatusInternalServerError
	switch {
	case errors.Is(
		err,
		ErrModelUnauthenticated,
	):
		code = http.StatusUnauthorized
	case errors.Is(
		err,
		ErrModelForbidden,
	):
		code = http.StatusForbidden
	case errors.Is(
		err,
		ErrProductBlueprintPrinted,
	):
		code = http.StatusConflict
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
			modeldom.ErrInvalid,
		):
		code = http.StatusBadRequest
	case errors.Is(
		err,
		modeldom.ErrNotFound,
	):
		code = http.StatusNotFound
	case errors.Is(
		err,
		modeldom.ErrConflict,
	):
		code = http.StatusConflict
	}
	writeJSONError(
		w,
		code,
		err.Error(),
	)
}
