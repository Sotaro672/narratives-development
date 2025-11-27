// backend/internal/adapters/in/http/handlers/model_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
	//   → ModelUsecase.ListModelVariationsByProductBlueprintID
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/models/by-blueprint/"):

		path := strings.TrimPrefix(r.URL.Path, "/models/by-blueprint/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")

		if len(parts) == 2 && parts[1] == "variations" {
			productBlueprintID := strings.TrimSpace(parts[0])
			h.listVariationsByProductBlueprintID(w, r, productBlueprintID)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return

	// ------------------------------------------------------------
	// POST /models/{productBlueprintID}/variations
	//   → ModelUsecase.CreateModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/models/"):

		path := strings.TrimPrefix(r.URL.Path, "/models/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")

		if len(parts) == 2 && parts[1] == "variations" {
			productBlueprintID := strings.TrimSpace(parts[0])
			h.createVariation(w, r, productBlueprintID)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return

	// ------------------------------------------------------------
	// PUT /models/{id}
	//   → ModelUsecase.UpdateModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodPut &&
		strings.HasPrefix(r.URL.Path, "/models/"):

		id := strings.TrimPrefix(r.URL.Path, "/models/")
		id = strings.TrimSpace(id)
		h.updateVariation(w, r, id)
		return

	// ------------------------------------------------------------
	// DELETE /models/{id}
	//   → ModelUsecase.DeleteModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodDelete &&
		strings.HasPrefix(r.URL.Path, "/models/"):

		id := strings.TrimPrefix(r.URL.Path, "/models/")
		id = strings.TrimSpace(id)
		h.deleteVariation(w, r, id)
		return

	// ------------------------------------------------------------
	// GET /models/{id}
	//   → ModelUsecase.GetByID
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/models/"):

		id := strings.TrimPrefix(r.URL.Path, "/models/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /models/{id}
func (h *ModelHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	m, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(m)
}

// ------------------------------------------------------------
// Request struct for CREATE / UPDATE
// ------------------------------------------------------------

type createModelVariationRequest struct {
	ProductBlueprintID string             `json:"productBlueprintId,omitempty"`
	ModelNumber        string             `json:"modelNumber"`
	Size               string             `json:"size"`
	Color              string             `json:"color"`
	RGB                int                `json:"rgb,omitempty"`
	Measurements       map[string]float64 `json:"measurements,omitempty"`
}

// ------------------------------------------------------------
// POST /models/{productBlueprintID}/variations
// ------------------------------------------------------------
func (h *ModelHandler) createVariation(w http.ResponseWriter, r *http.Request, productBlueprintID string) {
	ctx := r.Context()

	productBlueprintID = strings.TrimSpace(productBlueprintID)
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

	ms := make(modeldom.Measurements)
	for k, v := range req.Measurements {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		ms[key] = int(v)
	}

	newVar := modeldom.NewModelVariation{
		ProductBlueprintID: productBlueprintID,
		ModelNumber:        strings.TrimSpace(req.ModelNumber),
		Size:               strings.TrimSpace(req.Size),
		Color: modeldom.Color{
			Name: strings.TrimSpace(req.Color),
			RGB:  req.RGB,
		},
		Measurements: ms,
	}

	mv, err := h.uc.CreateModelVariation(ctx, newVar)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mv)
}

// ------------------------------------------------------------
// PUT /models/{id}
// ------------------------------------------------------------
func (h *ModelHandler) updateVariation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
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

	ms := make(modeldom.Measurements)
	for k, v := range req.Measurements {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		ms[key] = int(v)
	}

	modelNumber := strings.TrimSpace(req.ModelNumber)
	size := strings.TrimSpace(req.Size)
	color := modeldom.Color{
		Name: strings.TrimSpace(req.Color),
		RGB:  req.RGB,
	}

	updates := modeldom.ModelVariationUpdate{
		ModelNumber:  &modelNumber,
		Size:         &size,
		Color:        &color,
		Measurements: ms,
	}

	mv, err := h.uc.UpdateModelVariation(ctx, id, updates)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mv)
}

// ------------------------------------------------------------
// DELETE /models/{id}
// ------------------------------------------------------------
func (h *ModelHandler) deleteVariation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
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
	_ = json.NewEncoder(w).Encode(mv)
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

	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productBlueprintID"})
		return
	}

	vars, err := h.uc.ListModelVariationsByProductBlueprintID(ctx, productBlueprintID)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(vars)
}

// ------------------------------------------------------------
// 共通エラー処理
// ------------------------------------------------------------
func writeModelErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	if errors.Is(err, modeldom.ErrInvalidID) ||
		errors.Is(err, modeldom.ErrInvalidProductID) ||
		errors.Is(err, modeldom.ErrInvalidBlueprintID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, modeldom.ErrNotFound) {
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
