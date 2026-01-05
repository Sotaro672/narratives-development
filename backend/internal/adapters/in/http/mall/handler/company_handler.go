// backend/internal/adapters/in/http/sns/handler/company_handler.go
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	companydom "narratives/internal/domain/company"
)

// SNSCompanyHandler serves buyer-facing company endpoint.
//
// Route:
// - GET /sns/companies/{id}
type SNSCompanyHandler struct {
	uc *usecase.CompanyUsecase
}

func NewSNSCompanyHandler(uc *usecase.CompanyUsecase) http.Handler {
	return &SNSCompanyHandler{uc: uc}
}

func (h *SNSCompanyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// NOTE: sns/handler では共通 helper（writeJSON/notFound/methodNotAllowed/internalError 等）が
	// 既にある前提の設計が多いので、ここでは重複定義しない。
	// ただし、この handler 単体でも動くように最低限の JSON はここで返す。

	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "company handler is not ready"})
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	// GET /sns/companies/{id}
	if strings.HasPrefix(path, "/sns/companies/") {
		id := strings.TrimSpace(strings.TrimPrefix(path, "/sns/companies/"))
		h.get(w, r, id)
		return
	}

	// collection endpoint is not defined
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
}

// ---- GET /sns/companies/{id} ----
func (h *SNSCompanyHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	company, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeSNSCompanyErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(company)
}

func writeSNSCompanyErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case companydom.ErrInvalidID:
		code = http.StatusBadRequest
	case companydom.ErrNotFound:
		code = http.StatusNotFound
	case companydom.ErrConflict:
		code = http.StatusConflict
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
