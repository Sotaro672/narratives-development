// backend/internal/adapters/in/http/console/handler/companyShippingAddress_handler.go
package consoleHandler

import (
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	shadom "narratives/internal/domain/shippingAddress"
)

const companyShippingAddressBasePath = "/companies/me/shipping-addresses"

type CompanyShippingAddressHandler struct {
	uc *usecase.ShippingAddressUsecase
}

func NewCompanyShippingAddressHandler(uc *usecase.ShippingAddressUsecase) http.Handler {
	return &CompanyShippingAddressHandler{uc: uc}
}

type companyShippingAddressCreateRequest struct {
	Name    string  `json:"name"`
	ZipCode string  `json:"zipCode"`
	State   string  `json:"state"`
	City    string  `json:"city"`
	Street  string  `json:"street"`
	Street2 *string `json:"street2,omitempty"`
	Country *string `json:"country,omitempty"`
}

type companyShippingAddressUpdateRequest struct {
	Name    *string `json:"name,omitempty"`
	ZipCode *string `json:"zipCode,omitempty"`
	State   *string `json:"state,omitempty"`
	City    *string `json:"city,omitempty"`
	Street  *string `json:"street,omitempty"`
	Street2 *string `json:"street2,omitempty"`
	Country *string `json:"country,omitempty"`
}

func (h *CompanyShippingAddressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path == companyShippingAddressBasePath:
		h.list(w, r)

	case r.Method == http.MethodGet && strings.HasPrefix(path, companyShippingAddressBasePath+"/"):
		id := strings.TrimPrefix(path, companyShippingAddressBasePath+"/")
		h.get(w, r, id)

	case r.Method == http.MethodPost && path == companyShippingAddressBasePath:
		h.create(w, r)

	case r.Method == http.MethodPatch && strings.HasPrefix(path, companyShippingAddressBasePath+"/"):
		id := strings.TrimPrefix(path, companyShippingAddressBasePath+"/")
		h.update(w, r, id)

	case r.Method == http.MethodDelete && strings.HasPrefix(path, companyShippingAddressBasePath+"/"):
		id := strings.TrimPrefix(path, companyShippingAddressBasePath+"/")
		h.delete(w, r, id)

	default:
		writeNotFound(w)
	}
}

func (h *CompanyShippingAddressHandler) requireUsecase(w http.ResponseWriter) bool {
	if h != nil && h.uc != nil {
		return true
	}

	writeError(w, http.StatusServiceUnavailable, "shipping_address_usecase_not_initialized")
	return false
}

func (h *CompanyShippingAddressHandler) requireCompanyID(w http.ResponseWriter, r *http.Request) (string, bool) {
	companyID, ok := middleware.CompanyID(r)
	if ok && companyID != "" {
		return companyID, true
	}

	writeError(w, http.StatusForbidden, "company_id_not_resolved")
	return "", false
}

func (h *CompanyShippingAddressHandler) requireUID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid, ok := middleware.CurrentUserUID(r)
	if ok && uid != "" {
		return uid, true
	}

	writeError(w, http.StatusUnauthorized, "unauthorized")
	return "", false
}

func (h *CompanyShippingAddressHandler) list(w http.ResponseWriter, r *http.Request) {
	if !h.requireUsecase(w) {
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	addresses, err := h.uc.ListByCompanyID(r.Context(), companyID)
	if err != nil {
		writeCompanyShippingAddressErr(w, err)
		return
	}

	if addresses == nil {
		addresses = []shadom.ShippingAddress{}
	}

	writeJSON(w, http.StatusOK, addresses)
}

func (h *CompanyShippingAddressHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	if !h.requireUsecase(w) {
		return
	}

	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "invalid_shipping_address_id")
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	address, err := h.uc.GetByCompany(r.Context(), id, companyID)
	if err != nil {
		writeCompanyShippingAddressErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, address)
}

func (h *CompanyShippingAddressHandler) create(w http.ResponseWriter, r *http.Request) {
	if !h.requireUsecase(w) {
		return
	}

	uid, ok := h.requireUID(w, r)
	if !ok {
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var request companyShippingAddressCreateRequest
	if err := decodeStrictJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	street2 := ""
	if request.Street2 != nil {
		street2 = *request.Street2
	}

	country := ""
	if request.Country != nil {
		country = *request.Country
	}

	created, err := h.uc.Create(
		r.Context(),
		uid,
		companyID,
		usecase.CreateShippingAddressInput{
			Name:    request.Name,
			ZipCode: request.ZipCode,
			State:   request.State,
			City:    request.City,
			Street:  request.Street,
			Street2: street2,
			Country: country,
		},
	)
	if err != nil {
		writeCompanyShippingAddressErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *CompanyShippingAddressHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	if !h.requireUsecase(w) {
		return
	}

	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "invalid_shipping_address_id")
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var request companyShippingAddressUpdateRequest
	if err := decodeStrictJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	updated, err := h.uc.UpdateByCompany(
		r.Context(),
		id,
		companyID,
		usecase.UpdateShippingAddressInput{
			Name:    request.Name,
			ZipCode: request.ZipCode,
			State:   request.State,
			City:    request.City,
			Street:  request.Street,
			Street2: request.Street2,
			Country: request.Country,
		},
	)
	if err != nil {
		writeCompanyShippingAddressErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *CompanyShippingAddressHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if !h.requireUsecase(w) {
		return
	}

	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "invalid_shipping_address_id")
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	if err := h.uc.DeleteByCompany(r.Context(), id, companyID); err != nil {
		writeCompanyShippingAddressErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeCompanyShippingAddressErr(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(err, shadom.ErrInvalidID),
		errors.Is(err, shadom.ErrInvalidUserID),
		errors.Is(err, shadom.ErrInvalidCompanyID),
		errors.Is(err, shadom.ErrInvalidName),
		errors.Is(err, shadom.ErrInvalidZipCode),
		errors.Is(err, shadom.ErrInvalidState),
		errors.Is(err, shadom.ErrInvalidCity),
		errors.Is(err, shadom.ErrInvalidStreet),
		errors.Is(err, shadom.ErrInvalidCountry),
		errors.Is(err, shadom.ErrInvalidCreatedAt),
		errors.Is(err, shadom.ErrInvalidUpdatedAt):
		statusCode = http.StatusBadRequest

	case errors.Is(err, shadom.ErrNotFound):
		statusCode = http.StatusNotFound

	case errors.Is(err, shadom.ErrConflict):
		statusCode = http.StatusConflict
	}

	writeError(w, statusCode, err.Error())
}
