package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	modeldom "narratives/internal/domain/model"
)

// ModelHandler は /models 関連のエンドポイントを担当します（単一取得など）。
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
	// GET /models/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/"):
		id := strings.TrimPrefix(r.URL.Path, "/models/")
		h.get(w, r, id)

	// POST /models/{productID}/variations
	// 指定 Product のバリエーションを 1 件追加する
	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/models/") &&
		strings.HasSuffix(r.URL.Path, "/variations"):

		h.createVariation(w, r)

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

// POST /models/{productID}/variations
//
// 期待するリクエストボディ例（modeldom.NewModelVariation に合わせて調整してください）:
//
//	{
//	  "sizeLabel": "M",
//	  "color": "Black",
//	  "code": "ABC-123",
//	  ...  // NewModelVariation に必要なフィールド
//	}
func (h *ModelHandler) createVariation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// /models/{productID}/variations から productID を抽出
	path := strings.TrimPrefix(r.URL.Path, "/models/")
	productID := strings.TrimSuffix(path, "/variations")
	productID = strings.TrimSpace(productID)

	if productID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productID"})
		return
	}

	// ここでは NewModelVariation とほぼ同じ構造を受け取る想定。
	// 実際の modeldom.NewModelVariation の定義に合わせてフィールドを調整してください。
	var req modeldom.NewModelVariation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json body"})
		return
	}

	created, err := h.uc.CreateModelVariation(ctx, productID, req)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	// 作成した Variation を返却
	_ = json.NewEncoder(w).Encode(created)
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
