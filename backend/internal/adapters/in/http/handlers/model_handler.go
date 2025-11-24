package handlers

import (
	"encoding/json"
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
	// POST /models/{productID}/variations
	//   → ModelUsecase.CreateModelVariation を呼び出す
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/models/"):
		// /models/{productID}/variations を分解
		path := strings.TrimPrefix(r.URL.Path, "/models/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")

		// 期待する形式は {productID}/variations のみ
		if len(parts) == 2 && parts[1] == "variations" {
			productID := strings.TrimSpace(parts[0])
			h.createVariation(w, r, productID)
			return
		}

		// 形式が違う場合は 404
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return

	// ------------------------------------------------------------
	// GET /models/{id}
	//   → ModelUsecase.GetByID を呼び出す（従来どおり）
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/"):
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
 * POST /models/{productID}/variations 用のリクエスト型
 *   frontend/console/model/src/application/modelCreateService.tsx
 *   の CreateModelVariationRequest / NewModelVariationPayload に対応
 * ==========================================================*/

type createModelVariationRequest struct {
	ModelNumber  string             `json:"modelNumber"`            // "LM-SB-S-WHT" など
	Size         string             `json:"size"`                   // "S" / "M" / ...
	Color        string             `json:"color"`                  // "ホワイト" など
	Measurements map[string]float64 `json:"measurements,omitempty"` // chest / shoulder / waist / length など
}

// POST /models/{productID}/variations
//
// Request Body: createModelVariationRequest JSON
// Response    : 作成された ModelVariation を JSON で返す
func (h *ModelHandler) createVariation(w http.ResponseWriter, r *http.Request, productID string) {
	ctx := r.Context()

	productID = strings.TrimSpace(productID)
	if productID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productID"})
		return
	}

	var req createModelVariationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// frontend から来る measurements(map[string]float64) → domain 側の map[string]int へ変換
	ms := make(map[string]int)
	for k, v := range req.Measurements {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		// 必要であれば 0 未満を弾くなどのバリデーションもここで可能
		ms[key] = int(v)
	}

	newVar := modeldom.NewModelVariation{
		ModelNumber: strings.TrimSpace(req.ModelNumber),
		Size:        strings.TrimSpace(req.Size),
		Color: modeldom.Color{
			Name: strings.TrimSpace(req.Color),
			RGB:  0, // RGB は現状フロントから来ていないため 0 で初期化
		},
		Measurements: ms,
	}

	mv, err := h.uc.CreateModelVariation(ctx, productID, newVar)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mv)
}

// エラーハンドリング
func writeModelErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err == modeldom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
