// backend/internal/adapters/in/http/console/handler/company_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	companydom "narratives/internal/domain/company"
)

type CompanyHandler struct {
	uc *usecase.CompanyUsecase
}

func NewCompanyHandler(uc *usecase.CompanyUsecase) http.Handler {
	return &CompanyHandler{uc: uc}
}

func (h *CompanyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// POST /companies
	case r.Method == http.MethodPost && strings.Trim(r.URL.Path, "/") == "companies":
		h.create(w, r)

	// GET /companies/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/companies/"):
		id := strings.TrimPrefix(r.URL.Path, "/companies/")
		h.get(w, r, id)

	// PATCH /companies/{id}
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/companies/"):
		id := strings.TrimPrefix(r.URL.Path, "/companies/")
		h.update(w, r, id)

	// DELETE /companies/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/companies/"):
		id := strings.TrimPrefix(r.URL.Path, "/companies/")
		h.delete(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ---- GET /companies/{id} ----

func (h *CompanyHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

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

// ---- POST /companies ----

type createCompanyRequest struct {
	Name      string  `json:"name"`
	Admin     string  `json:"admin"`
	IsActive  *bool   `json:"isActive,omitempty"`
	CreatedBy *string `json:"createdBy,omitempty"`
}

func (h *CompanyHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	name := req.Name
	admin := req.Admin
	if name == "" || admin == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "name and admin are required"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	c := companydom.Company{
		Name:      name,
		Admin:     admin,
		IsActive:  isActive,
		CreatedAt: time.Now().UTC(),
	}

	if req.CreatedBy != nil {
		c.CreatedBy = *req.CreatedBy
	}

	created, err := h.uc.Create(ctx, c)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// ---- PATCH /companies/{id} ----

type updateCompanyRequest struct {
	Name      *string `json:"name,omitempty"`
	Admin     *string `json:"admin,omitempty"`
	IsActive  *bool   `json:"isActive,omitempty"`
	UpdatedBy *string `json:"updatedBy,omitempty"`
}

func (h *CompanyHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req updateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	patch := companydom.CompanyPatch{
		Name:      req.Name,
		Admin:     req.Admin,
		IsActive:  req.IsActive,
		UpdatedBy: req.UpdatedBy,
	}

	updated, err := h.uc.Update(ctx, id, patch)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

// ---- DELETE /companies/{id} ----

func (h *CompanyHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeCompanyErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---- 共通エラーハンドリング ----

func writeCompanyErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch err {
	case companydom.ErrNotFound:
		code = http.StatusNotFound
	case companydom.ErrConflict:
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
