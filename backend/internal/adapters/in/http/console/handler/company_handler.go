// backend/internal/adapters/in/http/console/handler/company_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	query "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	companydom "narratives/internal/domain/company"
)

type CompanyHandler struct {
	uc *usecase.CompanyUsecase
	q  *query.CompanyQuery
}

func NewCompanyHandler(
	uc *usecase.CompanyUsecase,
	q *query.CompanyQuery,
) http.Handler {
	return &CompanyHandler{
		uc: uc,
		q:  q,
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

func (h *CompanyHandler) requireQuery(
	w http.ResponseWriter,
) bool {
	if h != nil && h.q != nil {
		return true
	}

	writeError(
		w,
		http.StatusServiceUnavailable,
		"company_query_not_initialized",
	)

	return false
}

func (h *CompanyHandler) get(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	if !h.requireQuery(w) {
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

	company, err := h.q.GetByID(
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

	if !h.requireQuery(w) {
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

	createdDetail, err := h.q.GetByID(
		r.Context(),
		created.ID,
	)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		createdDetail,
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

	if !h.requireQuery(w) {
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

	updatedBy, _, ok :=
		middleware.CurrentUIDAndEmail(
			r,
		)

	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"current_user_uid_not_resolved",
		)
		return
	}

	patch := companydom.CompanyPatch{
		Name:      request.Name,
		Admin:     request.Admin,
		IsActive:  request.IsActive,
		UpdatedBy: &updatedBy,
	}

	_, err := h.uc.Update(
		r.Context(),
		id,
		patch,
	)
	if err != nil {
		writeCompanyErr(w, err)
		return
	}

	updated, err := h.q.GetByID(
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
