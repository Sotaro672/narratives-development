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

func NewCompanyHandler(
	uc *usecase.CompanyUsecase,
) http.Handler {
	return &CompanyHandler{
		uc: uc,
	}
}

func (h *CompanyHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	path := strings.TrimSuffix(
		r.URL.Path,
		"/",
	)

	switch {
	case r.Method == http.MethodPost &&
		path == "/companies":
		h.create(w, r)

	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/companies/"):
		id := strings.TrimPrefix(
			path,
			"/companies/",
		)

		if id == "" || strings.Contains(id, "/") {
			writeNotFound(w)
			return
		}

		h.get(w, r, id)

	case r.Method == http.MethodPatch &&
		strings.HasPrefix(path, "/companies/"):
		id := strings.TrimPrefix(
			path,
			"/companies/",
		)

		if id == "" || strings.Contains(id, "/") {
			writeNotFound(w)
			return
		}

		h.update(w, r, id)

	case r.Method == http.MethodDelete &&
		strings.HasPrefix(path, "/companies/"):
		id := strings.TrimPrefix(
			path,
			"/companies/",
		)

		if id == "" || strings.Contains(id, "/") {
			writeNotFound(w)
			return
		}

		h.delete(w, r, id)

	default:
		writeNotFound(w)
	}
}

func (h *CompanyHandler) requireUsecase(
	w http.ResponseWriter,
) bool {
	if h != nil && h.uc != nil {
		return true
	}

	writeError(
		w,
		http.StatusServiceUnavailable,
		"company_usecase_not_initialized",
	)

	return false
}

func (h *CompanyHandler) get(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	if !h.requireUsecase(w) {
		return
	}

	if id == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid id",
		)
		return
	}

	company, err := h.uc.GetByID(
		r.Context(),
		id,
	)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		company,
	)
}

type createCompanyRequest struct {
	Name      string  `json:"name"`
	Admin     string  `json:"admin"`
	IsActive  *bool   `json:"isActive,omitempty"`
	CreatedBy *string `json:"createdBy,omitempty"`
}

func (h *CompanyHandler) create(
	w http.ResponseWriter,
	r *http.Request,
) {
	if !h.requireUsecase(w) {
		return
	}

	var request createCompanyRequest
	if err := decodeStrictJSON(
		r,
		&request,
	); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid json",
		)
		return
	}

	if request.Name == "" ||
		request.Admin == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"name and admin are required",
		)
		return
	}

	isActive := true
	if request.IsActive != nil {
		isActive = *request.IsActive
	}

	now := time.Now().UTC()

	company := companydom.Company{
		Name:      request.Name,
		Admin:     request.Admin,
		IsActive:  isActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if request.CreatedBy != nil {
		company.CreatedBy = *request.CreatedBy
		company.UpdatedBy = *request.CreatedBy
	}

	created, err := h.uc.Create(
		r.Context(),
		company,
	)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		created,
	)
}

type updateCompanyRequest struct {
	Name     *string `json:"name,omitempty"`
	Admin    *string `json:"admin,omitempty"`
	IsActive *bool   `json:"isActive,omitempty"`
}

func (h *CompanyHandler) update(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	if !h.requireUsecase(w) {
		return
	}

	if id == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid id",
		)
		return
	}

	var request updateCompanyRequest
	if err := decodeStrictJSON(
		r,
		&request,
	); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid json",
		)
		return
	}

	updatedBy := usecase.MemberIDFromContext(
		r.Context(),
	)

	patch := companydom.CompanyPatch{
		Name:     request.Name,
		Admin:    request.Admin,
		IsActive: request.IsActive,
	}

	if updatedBy != "" {
		patch.UpdatedBy = &updatedBy
	}

	updated, err := h.uc.Update(
		r.Context(),
		id,
		patch,
	)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		updated,
	)
}

func (h *CompanyHandler) delete(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	if !h.requireUsecase(w) {
		return
	}

	if id == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid id",
		)
		return
	}

	if err := h.uc.Delete(
		r.Context(),
		id,
	); err != nil {
		writeCompanyErr(w, err)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func writeCompanyErr(
	w http.ResponseWriter,
	err error,
) {
	statusCode := http.StatusInternalServerError

	switch err {
	case companydom.ErrInvalidID:
		statusCode = http.StatusBadRequest

	case companydom.ErrNotFound:
		statusCode = http.StatusNotFound

	case companydom.ErrConflict:
		statusCode = http.StatusConflict
	}

	_ = json.NewEncoder

	writeError(
		w,
		statusCode,
		err.Error(),
	)
}
