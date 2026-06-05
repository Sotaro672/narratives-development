// backend/internal/adapters/in/http/console/handler/model_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	modeldom "narratives/internal/domain/model"
)

// ModelHandler は /models 関連のエンドポイントを担当します。
type ModelHandler struct {
	uc *usecase.ModelUsecase
}

// NewModelHandler はHTTPハンドラを初期化します。
func NewModelHandler(uc *usecase.ModelUsecase) http.Handler {
	return &ModelHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *ModelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// ------------------------------------------------------------
	// GET /models/by-blueprint/{productBlueprintID}/variations
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/by-blueprint/"):
		rest := strings.TrimPrefix(r.URL.Path, "/models/by-blueprint/")
		rest = strings.Trim(rest, "/")
		parts := strings.Split(rest, "/")

		if len(parts) == 2 && parts[0] != "" && parts[1] == "variations" && !strings.Contains(parts[0], "/") {
			h.listVariationsByProductBlueprintID(w, r, parts[0])
			return
		}

		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// GET /models/{id}
	//   → ModelUsecase.GetByID
	//
	// MintRequest detail / InspectionResultCard など、
	// modelId から modelNumber / size / color / volume を単体解決する用途。
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/"):
		id := strings.TrimPrefix(r.URL.Path, "/models/")
		id = strings.Trim(id, "/")

		if isSingleModelIDPath(id) {
			h.getVariation(w, r, id)
			return
		}

		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// POST /models/{productBlueprintID}/variations
	//   → ModelUsecase.Create
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/models/"):
		rest := strings.TrimPrefix(r.URL.Path, "/models/")
		rest = strings.Trim(rest, "/")
		parts := strings.Split(rest, "/")

		if len(parts) == 2 && parts[0] != "" && parts[1] == "variations" && !strings.Contains(parts[0], "/") {
			h.createVariation(w, r, parts[0])
			return
		}

		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// PUT /models/{id}
	//   → ModelUsecase.Update
	// ------------------------------------------------------------
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/models/"):
		id := strings.TrimPrefix(r.URL.Path, "/models/")
		id = strings.Trim(id, "/")

		if isSingleModelIDPath(id) {
			h.updateVariation(w, r, id)
			return
		}

		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// DELETE /models/{id}
	//   → ModelUsecase.Delete
	// ------------------------------------------------------------
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/models/"):
		id := strings.TrimPrefix(r.URL.Path, "/models/")
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

// isSingleModelIDPath は /models/{id} 系の単体 ID path だけを許可する。
func isSingleModelIDPath(id string) bool {
	return id != "" &&
		id != "by-blueprint" &&
		id != "variations" &&
		!strings.HasPrefix(id, "by-blueprint/") &&
		!strings.HasPrefix(id, "variations/") &&
		!strings.Contains(id, "/")
}

// ------------------------------------------------------------
// Request DTOs
// ------------------------------------------------------------

// Request struct for CREATE / UPDATE
type createModelVariationRequest struct {
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`

	// category-specific variation kind.
	// 必須: "apparel" または "alcohol"
	Kind string `json:"kind"`

	// common
	ModelNumber string `json:"modelNumber"`

	// apparel
	Size         string             `json:"size,omitempty"`
	Color        string             `json:"color,omitempty"`
	RGB          int                `json:"rgb"` // 0=黒を送れるように omitempty は付けない
	Measurements map[string]float64 `json:"measurements,omitempty"`

	// alcohol
	Volume modeldom.Volume `json:"volume,omitempty"`
}

// ------------------------------------------------------------
// Response DTOs
// ------------------------------------------------------------

type colorDTO struct {
	Name string `json:"name"`
	RGB  int    `json:"rgb"` // 0=黒を正しく返すため omitempty は付けない
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

func (h *ModelHandler) createVariation(w http.ResponseWriter, r *http.Request, productBlueprintID string) {
	ctx := r.Context()

	if productBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productBlueprintID"})
		return
	}

	var req createModelVariationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	newVar, err := toNewModelVariation(productBlueprintID, req)
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

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toModelVariationDTO(mv))
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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productBlueprintID"})
		return
	}

	vars, err := h.uc.ListByProductBlueprintID(ctx, productBlueprintID)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toModelVariationDTOs(vars))
}

// ------------------------------------------------------------
// GET /models/{id}
// ------------------------------------------------------------

func (h *ModelHandler) getVariation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	mv, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toModelVariationDTO(mv))
}

// ------------------------------------------------------------
// PUT /models/{id}
// ------------------------------------------------------------

func (h *ModelHandler) updateVariation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req createModelVariationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	updates, err := toModelVariationUpdate(req)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	mv, err := h.uc.Update(ctx, id, updates)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toModelVariationDTO(mv))
}

// ------------------------------------------------------------
// DELETE /models/{id}
// ------------------------------------------------------------

func (h *ModelHandler) deleteVariation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ------------------------------------------------------------
// Mapper
// ------------------------------------------------------------

func toMeasurements(in map[string]float64) modeldom.Measurements {
	if len(in) == 0 {
		return nil
	}

	ms := make(modeldom.Measurements, len(in))
	for k, v := range in {
		if k == "" {
			continue
		}
		ms[k] = int(v)
	}

	if len(ms) == 0 {
		return nil
	}

	return ms
}

// toNewModelVariation は request を category-specific な domain input に変換する。
func toNewModelVariation(
	productBlueprintID string,
	req createModelVariationRequest,
) (modeldom.NewModelVariation, error) {
	switch req.Kind {
	case string(modeldom.ModelVariationKindApparel):
		return modeldom.NewModelVariationFromApparel(modeldom.NewApparelModelVariation{
			ProductBlueprintID: productBlueprintID,
			ModelNumber:        req.ModelNumber,
			Size:               req.Size,
			Color: modeldom.Color{
				Name: req.Color,
				RGB:  req.RGB,
			},
			Measurements: toMeasurements(req.Measurements),
		}), nil

	case string(modeldom.ModelVariationKindAlcohol):
		return modeldom.NewModelVariationFromAlcohol(modeldom.NewAlcoholModelVariation{
			ProductBlueprintID: productBlueprintID,
			ModelNumber:        req.ModelNumber,
			Volume:             req.Volume,
		}), nil

	default:
		return modeldom.NewModelVariation{}, modeldom.ErrInvalid
	}
}

// toModelVariationUpdate は request を category-specific な update DTO に変換する。
func toModelVariationUpdate(req createModelVariationRequest) (modeldom.ModelVariationUpdate, error) {
	modelNumber := req.ModelNumber

	switch req.Kind {
	case string(modeldom.ModelVariationKindApparel):
		size := req.Size
		color := modeldom.Color{
			Name: req.Color,
			RGB:  req.RGB,
		}

		return modeldom.ModelVariationUpdate{
			ModelNumber:  &modelNumber,
			Size:         &size,
			Color:        &color,
			Measurements: toMeasurements(req.Measurements),
		}, nil

	case string(modeldom.ModelVariationKindAlcohol):
		volume := req.Volume

		return modeldom.ModelVariationUpdate{
			ModelNumber: &modelNumber,
			Volume:      &volume,
		}, nil

	default:
		return modeldom.ModelVariationUpdate{}, modeldom.ErrInvalid
	}
}

func toModelVariationDTO(mv modeldom.ModelVariation) modelVariationDTO {
	if mv == nil {
		return modelVariationDTO{}
	}

	switch v := mv.(type) {
	case modeldom.ApparelModelVariation:
		return toApparelModelVariationDTO(v)

	case modeldom.AlcoholModelVariation:
		return toAlcoholModelVariationDTO(v)

	default:
		return modelVariationDTO{
			ID:                 mv.GetID(),
			ProductBlueprintID: mv.GetProductBlueprintID(),
			Kind:               "",
			ModelNumber:        mv.GetModelNumber(),
		}
	}
}

func toApparelModelVariationDTO(mv modeldom.ApparelModelVariation) modelVariationDTO {
	return modelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		Kind:               string(modeldom.ModelVariationKindApparel),
		ModelNumber:        mv.ModelNumber,
		Size:               mv.Size,
		Color: &colorDTO{
			Name: mv.Color.Name,
			RGB:  mv.Color.RGB,
		},
		Measurements: cloneMeasurementsForDTO(mv.Measurements),
		CreatedAt:    timePtrToRFC3339(&mv.CreatedAt),
		CreatedBy:    mv.CreatedBy,
		UpdatedAt:    timePtrToRFC3339(&mv.UpdatedAt),
		UpdatedBy:    mv.UpdatedBy,
	}
}

func toAlcoholModelVariationDTO(mv modeldom.AlcoholModelVariation) modelVariationDTO {
	return modelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		Kind:               string(modeldom.ModelVariationKindAlcohol),
		ModelNumber:        mv.ModelNumber,
		Volume: &volumeDTO{
			Value: mv.Volume.Value,
			Unit:  mv.Volume.Unit,
		},
		CreatedAt: timePtrToRFC3339(&mv.CreatedAt),
		CreatedBy: mv.CreatedBy,
		UpdatedAt: timePtrToRFC3339(&mv.UpdatedAt),
		UpdatedBy: mv.UpdatedBy,
	}
}

func toModelVariationDTOs(vars []modeldom.ModelVariation) []modelVariationDTO {
	out := make([]modelVariationDTO, 0, len(vars))
	for _, v := range vars {
		out = append(out, toModelVariationDTO(v))
	}
	return out
}

func cloneMeasurementsForDTO(m modeldom.Measurements) map[string]int {
	if m == nil {
		return nil
	}

	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
	}

	return out
}

func timePtrToRFC3339(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// ------------------------------------------------------------
// Error helpers
// ------------------------------------------------------------

func writeNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
}

func writeModelErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	if errors.Is(err, modeldom.ErrInvalidID) ||
		errors.Is(err, modeldom.ErrInvalidProductID) ||
		errors.Is(err, modeldom.ErrInvalidBlueprintID) ||
		errors.Is(err, modeldom.ErrInvalidModelNumber) ||
		errors.Is(err, modeldom.ErrInvalidSize) ||
		errors.Is(err, modeldom.ErrInvalidColor) ||
		errors.Is(err, modeldom.ErrInvalidMeasurements) ||
		errors.Is(err, modeldom.ErrInvalidVolume) ||
		errors.Is(err, modeldom.ErrInvalidVolumeUnit) ||
		errors.Is(err, modeldom.ErrInvalid) {
		code = http.StatusBadRequest
	} else if errors.Is(err, modeldom.ErrNotFound) {
		code = http.StatusNotFound
	} else if errors.Is(err, modeldom.ErrConflict) {
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
