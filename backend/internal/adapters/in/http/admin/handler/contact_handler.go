// backend/internal/adapters/in/http/admin/handler/contact_handler.go
package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	contactuc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	contact "narratives/internal/domain/contact"
)

const adminContactsPath = "/admin/contacts"

type ContactHandler struct {
	uc *contactuc.ContactUsecase
}

func NewContactHandler(uc *contactuc.ContactUsecase) http.Handler {
	return http.HandlerFunc((&ContactHandler{uc: uc}).handle)
}

type contactResponse struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Email              string   `json:"email"`
	Company            string   `json:"company"`
	Message            string   `json:"message"`
	AttachmentImageIDs []string `json:"attachmentImageIds"`
	IsRead             bool     `json:"isRead"`
	Source             string   `json:"source"`
	CreatedAt          string   `json:"createdAt"`
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

	contactID, isDetail, valid := resolveContactPath(r.URL.Path)
	if !valid {
		writeJSONError(w, http.StatusNotFound, "contact_not_found")
		return
	}

	if isDetail {
		h.handleGetByID(w, r, contactID)
		return
	}

	h.handleList(w, r)
}

func (h *ContactHandler) handleGetByID(
	w http.ResponseWriter,
	r *http.Request,
	contactID string,
) {
	result, err := h.uc.GetByID(r.Context(), contactID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			writeJSONError(w, http.StatusNotFound, "contact_not_found")
			return
		}

		writeJSONError(w, http.StatusInternalServerError, "contact_get_failed")
		return
	}

	writeJSON(w, http.StatusOK, toAdminContactResponse(result))
}

func (h *ContactHandler) handleList(
	w http.ResponseWriter,
	r *http.Request,
) {
	query := r.URL.Query()

	var filter contact.Filter
	if value := query.Get("isRead"); value != "" {
		isRead, err := strconv.ParseBool(value)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_is_read")
			return
		}

		filter.IsRead = &isRead
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

func resolveContactPath(
	requestPath string,
) (contactID string, isDetail bool, valid bool) {
	if requestPath == adminContactsPath || requestPath == adminContactsPath+"/" {
		return "", false, true
	}

	if !strings.HasPrefix(requestPath, adminContactsPath+"/") {
		return "", false, false
	}

	contactID = strings.TrimSpace(
		strings.TrimPrefix(requestPath, adminContactsPath+"/"),
	)

	if contactID == "" || strings.Contains(contactID, "/") {
		return "", false, false
	}

	return contactID, true, true
}

func toAdminContactResponse(c contact.Contact) contactResponse {
	createdAt := ""
	if !c.CreatedAt.IsZero() {
		createdAt = c.CreatedAt.UTC().Format(time.RFC3339Nano)
	}

	return contactResponse{
		ID:                 c.ID,
		Name:               c.Name,
		Email:              c.Email,
		Company:            c.Company,
		Message:            c.Message,
		AttachmentImageIDs: append([]string(nil), c.AttachmentImageIDs...),
		IsRead:             c.IsRead,
		Source:             c.Source,
		CreatedAt:          createdAt,
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
