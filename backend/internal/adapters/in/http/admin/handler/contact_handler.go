// backend/internal/adapters/in/http/admin/handler/contact_handler.go
package handler

import (
	"net/http"
	"strconv"

	contactuc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	contact "narratives/internal/domain/contact"
)

type ContactHandler struct {
	uc *contactuc.ContactUsecase
}

func NewContactHandler(uc *contactuc.ContactUsecase) http.Handler {
	h := &ContactHandler{uc: uc}
	return http.HandlerFunc(h.handle)
}

type contactResponse struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Company   string         `json:"company"`
	Message   string         `json:"message"`
	Status    contact.Status `json:"status"`
	Source    string         `json:"source"`
	CreatedAt string         `json:"createdAt"`
}

type contactListResponse struct {
	Items      []contactResponse `json:"items"`
	TotalCount int               `json:"totalCount"`
	TotalPages int               `json:"totalPages"`
	Page       int               `json:"page"`
	PerPage    int               `json:"perPage"`
}

func (h *ContactHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.uc == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "contact_usecase_not_initialized")
		return
	}

	h.handleList(w, r)
}

func (h *ContactHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var filter contact.Filter
	if value := query.Get("status"); value != "" {
		status := contact.Status(value)
		filter.Status = &status
	}

	page := common.Page{
		Number:  parsePositiveInt(query.Get("page"), 1),
		PerPage: parsePositiveInt(query.Get("perPage"), 50),
	}

	sort := common.Sort{
		Column: query.Get("sort"),
		Order:  common.SortOrder(query.Get("order")),
	}

	result, err := h.uc.List(r.Context(), filter, sort, page)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "contact_list_failed")
		return
	}

	items := make([]contactResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAdminContactResponse(item))
	}

	writeJSON(w, http.StatusOK, contactListResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	})
}

func toAdminContactResponse(c contact.Contact) contactResponse {
	createdAt := ""
	if !c.CreatedAt.IsZero() {
		createdAt = c.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}

	return contactResponse{
		ID:        c.ID,
		Name:      c.Name,
		Email:     c.Email,
		Company:   c.Company,
		Message:   c.Message,
		Status:    c.Status,
		Source:    c.Source,
		CreatedAt: createdAt,
	}
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}

	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}
