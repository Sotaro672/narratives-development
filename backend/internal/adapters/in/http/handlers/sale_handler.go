package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	saledom "narratives/internal/domain/sale"
)

// SaleHandler は /sales 関連のエンドポイントを担当します（単一取得のみ）。
type SaleHandler struct {
	uc *usecase.SaleUsecase
}

// NewSaleHandler はHTTPハンドラを初期化します。
func NewSaleHandler(uc *usecase.SaleUsecase) http.Handler {
	return &SaleHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *SaleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/sales/"):
		id := strings.TrimPrefix(r.URL.Path, "/sales/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /sales/{id}
func (h *SaleHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	s, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeSaleErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(s)
}

// エラーハンドリング
func writeSaleErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	// NotFound の有無が不明なため InvalidID のみ特別扱い
	if err == saledom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
