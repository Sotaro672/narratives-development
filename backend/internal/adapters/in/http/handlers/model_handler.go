// backend/internal/adapters/in/http/handlers/model_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
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

		// /models/by-blueprint/{productBlueprintID}/variations
		path := strings.TrimPrefix(r.URL.Path, "/models/by-blueprint/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")

		// 期待形式: {productBlueprintID}/variations
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

		// /models/{productBlueprintID}/variations を分解
		path := strings.TrimPrefix(r.URL.Path, "/models/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")

		// 期待形式: {productBlueprintID}/variations
		if len(parts) == 2 && parts[1] == "variations" {
			productBlueprintID := strings.TrimSpace(parts[0])
			h.createVariation(w, r, productBlueprintID)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return

	// ------------------------------------------------------------
	// GET /models/{id}
	//   → ModelUsecase.GetByID（既存仕様）
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/models/"):

		id := strings.TrimPrefix(r.URL.Path, "/models/")
		h.get(w, r, id)

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

/* ============================================================
 * POST /models/{productBlueprintID}/variations 用のリクエスト型
 * frontend/console/model/src/infrastructure/repository/modelRepositoryHTTP.ts
 * の CreateModelVariationRequest に対応
 * ==========================================================*/

type createModelVariationRequest struct {
	// URL パスにも含まれているが、body にもあれば一応受ける
	ProductBlueprintID string             `json:"productBlueprintId,omitempty"`
	ModelNumber        string             `json:"modelNumber"`            // "LM-SB-S-WHT" など
	Size               string             `json:"size"`                   // "S" / "M" / ...
	Color              string             `json:"color"`                  // "ホワイト" など（名前）
	RGB                int                `json:"rgb,omitempty"`          // rgb 値（0xRRGGBB 想定）
	Measurements       map[string]float64 `json:"measurements,omitempty"` // 着丈/身幅/…など
}

// POST /models/{productBlueprintID}/variations
//
// Request Body: createModelVariationRequest JSON
// Response    : 作成された ModelVariation を JSON で返す
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

	// ログ: フロントから渡ってきた値を確認
	log.Printf(
		"[ModelHandler] createVariation productBlueprintID(path)=%s, body.productBlueprintId=%s, modelNumber=%s, size=%s, color=%s, rgb=%d, measurements=%v",
		productBlueprintID,
		req.ProductBlueprintID,
		req.ModelNumber,
		req.Size,
		req.Color,
		req.RGB,
		req.Measurements,
	)

	// frontend から来る measurements(map[string]float64) → domain 側の map[string]int へ変換
	ms := make(modeldom.Measurements)
	for k, v := range req.Measurements {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		ms[key] = int(v)
	}

	newVar := modeldom.NewModelVariation{
		// URL から来た productBlueprintID を domain に渡す
		ProductBlueprintID: productBlueprintID,
		ModelNumber:        strings.TrimSpace(req.ModelNumber),
		Size:               strings.TrimSpace(req.Size),
		Color: modeldom.Color{
			Name: strings.TrimSpace(req.Color),
			RGB:  req.RGB,
		},
		Measurements: ms,
	}

	// ログ: NewModelVariation に values が詰め替えられているか確認
	log.Printf("[ModelHandler] createVariation NewModelVariation=%+v", newVar)

	mv, err := h.uc.CreateModelVariation(ctx, newVar)
	if err != nil {
		log.Printf("[ModelHandler] error: %v", err)
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mv)
}

// GET /models/by-blueprint/{productBlueprintID}/variations
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

	// 0件でも 200 & [] を返す
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(vars)
}

// エラーハンドリング
func writeModelErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	// バリデーション系
	if errors.Is(err, modeldom.ErrInvalidID) ||
		errors.Is(err, modeldom.ErrInvalidProductID) ||
		errors.Is(err, modeldom.ErrInvalidBlueprintID) {
		code = http.StatusBadRequest
	} else if errors.Is(err, modeldom.ErrNotFound) {
		// NotFound は 404 にする
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
