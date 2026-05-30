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
	// GET /models/variations/{variationId}
	//   → ModelUsecase.GetModelVariationByID
	// ※ mintRequest の「モデル別検査結果」(modelId=variationId) 用
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/variations/"):
		if id, ok := extractSingleID(r.URL.Path, "/models/variations/"); ok {
			h.getVariationByID(w, r, id)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// GET /models/by-blueprint/{productBlueprintID}/variations
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/by-blueprint/"):
		if productBlueprintID, ok := extractBlueprintIDForList(r.URL.Path); ok {
			h.listVariationsByProductBlueprintID(w, r, productBlueprintID)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// POST /models/{productBlueprintID}/variations
	//   → ModelUsecase.CreateModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/models/"):
		if productBlueprintID, ok := extractBlueprintIDForCreate(r.URL.Path); ok {
			h.createVariation(w, r, productBlueprintID)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// PUT /models/{id}
	//   → ModelUsecase.UpdateModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/models/"):
		if id, ok := extractModelID(r.URL.Path); ok {
			h.updateVariation(w, r, id)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// DELETE /models/{id}
	//   → ModelUsecase.DeleteModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/models/"):
		if id, ok := extractModelID(r.URL.Path); ok {
			h.deleteVariation(w, r, id)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// GET /models/{id}
	//   → ModelUsecase.GetByID
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/"):
		if id, ok := extractModelID(r.URL.Path); ok {
			h.get(w, r, id)
			return
		}
		writeNotFound(w)
		return

	default:
		writeNotFound(w)
		return
	}
}

// ------------------------------------------------------------
// Request DTOs
// ------------------------------------------------------------

// Request struct for CREATE / UPDATE
type createModelVariationRequest struct {
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`

	// category-specific variation kind.
	// 未指定の場合は既存互換として apparel 扱いにする。
	Kind string `json:"kind,omitempty"`

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
	Kind               string         `json:"kind,omitempty"`
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

	newVar := toNewModelVariation(productBlueprintID, req)
	if err := newVar.Validate(); err != nil {
		writeModelErr(w, err)
		return
	}

	mv, err := h.uc.CreateModelVariation(ctx, newVar)
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

	vars, err := h.uc.GetModelVariations(ctx, productBlueprintID)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toModelVariationDTOs(vars))
}

// ------------------------------------------------------------
// GET /models/variations/{variationId}
// ------------------------------------------------------------

func (h *ModelHandler) getVariationByID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	mv, err := h.uc.GetModelVariationByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	if mv == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "variation not found"})
		return
	}

	_ = json.NewEncoder(w).Encode(toModelVariationDTO(mv))
}

// ------------------------------------------------------------
// GET /models/{id}
// ------------------------------------------------------------

func (h *ModelHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	m, err := h.uc.GetModelVariationByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	if m == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "variation not found"})
		return
	}

	_ = json.NewEncoder(w).Encode(toModelVariationDTO(m))
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

	updates := toModelVariationUpdate(req)

	mv, err := h.uc.UpdateModelVariation(ctx, id, updates)
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

	mv, err := h.uc.DeleteModelVariation(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toModelVariationDTO(mv))
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
		key := k
		if key == "" {
			continue
		}
		ms[key] = int(v)
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
) modeldom.NewModelVariation {
	if req.Kind == string(modeldom.ModelVariationKindAlcohol) {
		return modeldom.NewModelVariationFromAlcohol(modeldom.NewAlcoholModelVariation{
			ProductBlueprintID: productBlueprintID,
			ModelNumber:        req.ModelNumber,
			Volume:             req.Volume,
		})
	}

	return modeldom.NewModelVariationFromApparel(modeldom.NewApparelModelVariation{
		ProductBlueprintID: productBlueprintID,
		ModelNumber:        req.ModelNumber,
		Size:               req.Size,
		Color: modeldom.Color{
			Name: req.Color,
			RGB:  req.RGB,
		},
		Measurements: toMeasurements(req.Measurements),
	})
}

// toModelVariationUpdate は request を category-specific な update DTO に変換する。
func toModelVariationUpdate(req createModelVariationRequest) modeldom.ModelVariationUpdate {
	modelNumber := req.ModelNumber

	if req.Kind == string(modeldom.ModelVariationKindAlcohol) {
		volume := req.Volume

		return modeldom.ModelVariationUpdate{
			ModelNumber: &modelNumber,
			Volume:      &volume,
		}
	}

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
// Path helpers
// ------------------------------------------------------------

// extractSingleID は prefix 配下の単一IDを抽出します。
// 例: path="/models/variations/123", prefix="/models/variations/" => "123"
func extractSingleID(path string, prefix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.Trim(id, "/")
	if id == "" {
		return "", false
	}
	if strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// extractBlueprintIDForList は以下のパスから productBlueprintID を抽出します。
// GET /models/by-blueprint/{productBlueprintID}/variations
func extractBlueprintIDForList(path string) (string, bool) {
	if !strings.HasPrefix(path, "/models/by-blueprint/") {
		return "", false
	}
	rest := strings.TrimPrefix(path, "/models/by-blueprint/")
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "variations" {
		return "", false
	}
	id := parts[0]
	if id == "" {
		return "", false
	}
	if strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// extractBlueprintIDForCreate は以下のパスから productBlueprintID を抽出します。
// POST /models/{productBlueprintID}/variations
func extractBlueprintIDForCreate(path string) (string, bool) {
	if !strings.HasPrefix(path, "/models/") {
		return "", false
	}
	rest := strings.TrimPrefix(path, "/models/")
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "variations" {
		return "", false
	}
	id := parts[0]
	if id == "" {
		return "", false
	}
	if strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// extractModelID は以下のパスから model variation ID を抽出します。
// GET/PUT/DELETE /models/{id}
func extractModelID(path string) (string, bool) {
	if !strings.HasPrefix(path, "/models/") {
		return "", false
	}
	id := strings.TrimPrefix(path, "/models/")
	id = strings.Trim(id, "/")
	if id == "" {
		return "", false
	}

	if strings.HasPrefix(id, "variations/") || id == "variations" {
		return "", false
	}

	if strings.Contains(id, "/") {
		return "", false
	}

	return id, true
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
