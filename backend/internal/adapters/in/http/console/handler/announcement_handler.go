// backend/internal/adapters/in/http/console/handler/announcement_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	consolequery "narratives/internal/application/query/console"
	uc "narratives/internal/application/usecase"
	ann "narratives/internal/domain/announcement"
)

type AnnouncementHandler struct {
	uc              *uc.AnnouncementUsecase
	managementQuery *consolequery.AnnouncementManagementQuery
	detailQuery     *consolequery.AnnouncementDetailQuery
}

func NewAnnouncementHandler(
	announcementUC *uc.AnnouncementUsecase,
	announcementManagementQuery *consolequery.AnnouncementManagementQuery,
	announcementDetailQuery *consolequery.AnnouncementDetailQuery,
) http.Handler {
	return &AnnouncementHandler{
		uc:              announcementUC,
		managementQuery: announcementManagementQuery,
		detailQuery:     announcementDetailQuery,
	}
}

func (h *AnnouncementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil {
		writeAnnouncementJSON(w, http.StatusInternalServerError, map[string]string{"error": "not_configured"})
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] != "announcements" {
		writeAnnouncementJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		h.listAnnouncements(w, r)
		return

	case len(parts) == 1 && r.Method == http.MethodPost:
		h.createAnnouncement(w, r)
		return

	case len(parts) == 2 && r.Method == http.MethodGet:
		h.getAnnouncement(w, r, parts[1])
		return

	case len(parts) == 2 && r.Method == http.MethodPut:
		h.updateAnnouncement(w, r, parts[1])
		return

	case len(parts) == 2 && r.Method == http.MethodDelete:
		h.deleteAnnouncementCascade(w, r, parts[1])
		return

	case len(parts) == 3 && parts[2] == "publish" && r.Method == http.MethodPost:
		h.markPublished(w, r, parts[1])
		return

	default:
		writeAnnouncementJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
}

// =======================
// Request DTOs
// =======================

type createAnnouncementRequest struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	TargetToken   *string    `json:"targetToken"`
	TargetAvatars []string   `json:"targetAvatars"`
	Attachments   []string   `json:"attachments"`
	Published     bool       `json:"published"`
	PublishedAt   *time.Time `json:"publishedAt"`
	CreatedBy     string     `json:"createdBy"`
}

type updateAnnouncementRequest struct {
	Title         *string    `json:"title"`
	Content       *string    `json:"content"`
	TargetToken   *string    `json:"targetToken"`
	TargetAvatars *[]string  `json:"targetAvatars"`
	Published     *bool      `json:"published"`
	PublishedAt   *time.Time `json:"publishedAt"`
	Attachments   *[]string  `json:"attachments"`
	UpdatedBy     *string    `json:"updatedBy"`
}

type markPublishedRequest struct {
	UpdatedBy *string `json:"updatedBy"`
}

// =======================
// Handlers
// =======================

func (h *AnnouncementHandler) listAnnouncements(w http.ResponseWriter, r *http.Request) {
	if h.managementQuery == nil {
		writeAnnouncementJSON(w, http.StatusInternalServerError, map[string]string{"error": "announcement_management_query_not_configured"})
		return
	}

	companyID := strings.TrimSpace(r.URL.Query().Get("companyId"))
	if companyID == "" {
		writeAnnouncementJSON(w, http.StatusBadRequest, map[string]string{"error": "companyId is required"})
		return
	}

	result, err := h.managementQuery.ListByCompanyID(r.Context(), companyID)
	if err != nil {
		writeAnnouncementErr(w, err)
		return
	}

	writeAnnouncementJSON(w, http.StatusOK, result)
}

func (h *AnnouncementHandler) getAnnouncement(w http.ResponseWriter, r *http.Request, announcementID string) {
	if h.detailQuery == nil {
		writeAnnouncementJSON(w, http.StatusInternalServerError, map[string]string{"error": "announcement_detail_query_not_configured"})
		return
	}

	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		writeAnnouncementJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	result, err := h.detailQuery.GetByID(r.Context(), announcementID)
	if err != nil {
		writeAnnouncementErr(w, err)
		return
	}

	writeAnnouncementJSON(w, http.StatusOK, result)
}

func (h *AnnouncementHandler) createAnnouncement(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		writeAnnouncementJSON(w, http.StatusInternalServerError, map[string]string{"error": "announcement_usecase_not_configured"})
		return
	}

	var req createAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAnnouncementJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	result, err := h.uc.CreateAnnouncement(r.Context(), uc.CreateAnnouncementInput{
		ID:            req.ID,
		Title:         req.Title,
		Content:       req.Content,
		TargetToken:   req.TargetToken,
		TargetAvatars: req.TargetAvatars,
		Attachments:   req.Attachments,
		Published:     req.Published,
		PublishedAt:   req.PublishedAt,
		CreatedBy:     req.CreatedBy,
	})
	if err != nil {
		writeAnnouncementErr(w, err)
		return
	}

	writeAnnouncementJSON(w, http.StatusCreated, result)
}

func (h *AnnouncementHandler) updateAnnouncement(w http.ResponseWriter, r *http.Request, announcementID string) {
	if h.uc == nil {
		writeAnnouncementJSON(w, http.StatusInternalServerError, map[string]string{"error": "announcement_usecase_not_configured"})
		return
	}

	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		writeAnnouncementJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req updateAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAnnouncementJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	result, err := h.uc.UpdateAnnouncement(r.Context(), announcementID, uc.UpdateAnnouncementInput{
		Title:         req.Title,
		Content:       req.Content,
		TargetToken:   req.TargetToken,
		TargetAvatars: req.TargetAvatars,
		Published:     req.Published,
		PublishedAt:   req.PublishedAt,
		Attachments:   req.Attachments,
		UpdatedBy:     req.UpdatedBy,
	})
	if err != nil {
		writeAnnouncementErr(w, err)
		return
	}

	writeAnnouncementJSON(w, http.StatusOK, result)
}

func (h *AnnouncementHandler) markPublished(w http.ResponseWriter, r *http.Request, announcementID string) {
	if h.uc == nil {
		writeAnnouncementJSON(w, http.StatusInternalServerError, map[string]string{"error": "announcement_usecase_not_configured"})
		return
	}

	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		writeAnnouncementJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req markPublishedRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAnnouncementJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
	}

	result, err := h.uc.MarkPublished(r.Context(), announcementID, req.UpdatedBy)
	if err != nil {
		writeAnnouncementErr(w, err)
		return
	}

	writeAnnouncementJSON(w, http.StatusOK, result)
}

func (h *AnnouncementHandler) deleteAnnouncementCascade(w http.ResponseWriter, r *http.Request, announcementID string) {
	if h.uc == nil {
		writeAnnouncementJSON(w, http.StatusInternalServerError, map[string]string{"error": "announcement_usecase_not_configured"})
		return
	}

	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		writeAnnouncementJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.DeleteAnnouncementCascade(r.Context(), announcementID); err != nil {
		writeAnnouncementErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =======================
// Helpers
// =======================

func writeAnnouncementErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, ann.ErrNotFound):
		code = http.StatusNotFound

	case errors.Is(err, ann.ErrInvalidID),
		errors.Is(err, ann.ErrInvalidTitle),
		errors.Is(err, ann.ErrInvalidContent),
		errors.Is(err, ann.ErrInvalidCreatedBy),
		errors.Is(err, ann.ErrInvalidCreatedAt),
		errors.Is(err, ann.ErrInvalidUpdatedAt),
		errors.Is(err, ann.ErrInvalidPublishedAt),
		errors.Is(err, ann.ErrInvalidAvatarID),
		errors.Is(err, ann.ErrInvalidReadAt),
		errors.Is(err, ann.ErrInvalidAnnouncementID),
		errors.Is(err, ann.ErrInvalidFileName),
		errors.Is(err, ann.ErrInvalidFileURL),
		errors.Is(err, ann.ErrInvalidFileSize),
		errors.Is(err, ann.ErrInvalidMimeType),
		errors.Is(err, ann.ErrInvalidObjectPath):
		code = http.StatusBadRequest
	}

	writeAnnouncementJSON(w, code, map[string]string{"error": err.Error()})
}

func writeAnnouncementJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
