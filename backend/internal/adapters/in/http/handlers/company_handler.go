package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	companydom "narratives/internal/domain/company"
)

// CompanyHandler は /companies 関連のエンドポイントを担当します（単一取得のみ）。
type CompanyHandler struct {
	uc *usecase.CompanyUsecase
}

// NewCompanyHandler はHTTPハンドラを初期化します。
func NewCompanyHandler(uc *usecase.CompanyUsecase) http.Handler {
	return &CompanyHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *CompanyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/companies/"):
		id := strings.TrimPrefix(r.URL.Path, "/companies/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /companies/{id}
func (h *CompanyHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	company, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(company)
}

// エラーハンドリング
func writeCompanyErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case companydom.ErrInvalidID:
		code = http.StatusBadRequest
	case companydom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
