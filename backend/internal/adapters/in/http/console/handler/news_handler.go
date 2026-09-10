// backend/internal/adapters/in/http/console/handler/news_handler.go
package consoleHandler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	uc "narratives/internal/application/usecase"
	domcommon "narratives/internal/domain/common"
	newsdom "narratives/internal/domain/news"
)

const (
	defaultNewsPage    = 1
	defaultNewsPerPage = 20
	maxNewsPerPage     = 100
)

type NewsHandler struct {
	NewsUC *uc.NewsUsecase
}

func NewNewsHandler(newsUC *uc.NewsUsecase) *NewsHandler {
	return &NewsHandler{NewsUC: newsUC}
}

// Supported:
//
//	GET  /news
//	GET  /news?page=1&perPage=20
//	GET  /news/unread-count
//	GET  /news/{newsId}
//	POST /news/{newsId}/read
//
// News read state is scoped to the authenticated Console member.
func (h *NewsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.NewsUC == nil {
		writeError(w, http.StatusServiceUnavailable, "NewsHandlerNotInitialized")
		return
	}

	newsID, route, matched := parseNewsPath(r.URL.Path)
	if !matched {
		writeError(w, http.StatusNotFound, "NotFound")
		return
	}

	switch route {
	case newsRouteList:
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
			return
		}
		h.list(w, r)

	case newsRouteUnreadCount:
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
			return
		}
		h.unreadCount(w, r)

	case newsRouteDetail:
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
			return
		}
		h.detail(w, r, newsID)

	case newsRouteRead:
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
			return
		}
		h.markRead(w, r, newsID)

	default:
		writeError(w, http.StatusNotFound, "NotFound")
	}
}

// ============================================================
// DTO
// ============================================================

type newsImageResponse struct {
	FileURL    string `json:"fileUrl"`
	ObjectPath string `json:"objectPath"`
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	FileSize   int64  `json:"fileSize"`
	Alt        string `json:"alt,omitempty"`
}

type newsResponse struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Body        string             `json:"body"`
	Image       *newsImageResponse `json:"image,omitempty"`
	PublishedAt time.Time          `json:"publishedAt"`
	CreatedAt   time.Time          `json:"createdAt"`
	IsRead      bool               `json:"isRead"`
	ReadAt      *time.Time         `json:"readAt"`
}

type newsUnreadCountResponse struct {
	UnreadCount int `json:"unreadCount"`
}

// ============================================================
// List
// ============================================================

func (h *NewsHandler) list(w http.ResponseWriter, r *http.Request) {
	memberID := uc.MemberIDFromContext(r.Context())
	if memberID == "" {
		writeError(w, http.StatusForbidden, "MemberIDNotResolved")
		return
	}

	page, err := parseNewsPage(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "InvalidPagination")
		return
	}

	result, err := h.NewsUC.ListNewsForMember(r.Context(), memberID, page)
	if err != nil {
		writeNewsError(w, err)
		return
	}

	items := make([]newsResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toNewsResponse(item))
	}

	writeJSON(
		w,
		http.StatusOK,
		domcommon.PageResult[newsResponse]{
			Items:      items,
			TotalCount: result.TotalCount,
			TotalPages: result.TotalPages,
			Page:       result.Page,
			PerPage:    result.PerPage,
		},
	)
}

// ============================================================
// Detail
// ============================================================

func (h *NewsHandler) detail(
	w http.ResponseWriter,
	r *http.Request,
	newsID string,
) {
	if newsID == "" {
		writeError(w, http.StatusBadRequest, "NewsIDRequired")
		return
	}

	memberID := uc.MemberIDFromContext(r.Context())
	if memberID == "" {
		writeError(w, http.StatusForbidden, "MemberIDNotResolved")
		return
	}

	item, err := h.NewsUC.GetNewsForMember(
		r.Context(),
		newsdom.NewsID(newsID),
		memberID,
	)
	if err != nil {
		writeNewsError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		toNewsResponse(item),
	)
}

// ============================================================
// Unread count
// ============================================================

func (h *NewsHandler) unreadCount(w http.ResponseWriter, r *http.Request) {
	memberID := uc.MemberIDFromContext(r.Context())
	if memberID == "" {
		writeError(w, http.StatusForbidden, "MemberIDNotResolved")
		return
	}

	count, err := h.NewsUC.CountUnreadNewsForMember(
		r.Context(),
		memberID,
	)
	if err != nil {
		writeNewsError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		newsUnreadCountResponse{UnreadCount: count},
	)
}

// ============================================================
// Mark read
// ============================================================

func (h *NewsHandler) markRead(
	w http.ResponseWriter,
	r *http.Request,
	newsID string,
) {
	if newsID == "" {
		writeError(w, http.StatusBadRequest, "NewsIDRequired")
		return
	}

	memberID := uc.MemberIDFromContext(r.Context())
	if memberID == "" {
		writeError(w, http.StatusForbidden, "MemberIDNotResolved")
		return
	}

	read, err := h.NewsUC.MarkNewsReadForMember(
		r.Context(),
		newsdom.NewsID(newsID),
		memberID,
	)
	if err != nil {
		writeNewsError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		struct {
			ID     string    `json:"id"`
			NewsID string    `json:"newsId"`
			IsRead bool      `json:"isRead"`
			ReadAt time.Time `json:"readAt"`
		}{
			ID:     string(read.ID),
			NewsID: string(read.NewsID),
			IsRead: true,
			ReadAt: read.ReadAt,
		},
	)
}

// ============================================================
// Path
// ============================================================

type newsRouteKind int

const (
	newsRouteList newsRouteKind = iota
	newsRouteUnreadCount
	newsRouteDetail
	newsRouteRead
)

func parseNewsPath(
	path string,
) (newsID string, route newsRouteKind, matched bool) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", newsRouteList, false
	}

	parts := strings.Split(trimmed, "/")

	if len(parts) == 1 && parts[0] == "news" {
		return "", newsRouteList, true
	}

	if len(parts) == 2 &&
		parts[0] == "news" &&
		parts[1] == "unread-count" {
		return "", newsRouteUnreadCount, true
	}

	if len(parts) == 2 &&
		parts[0] == "news" &&
		parts[1] != "" {
		return parts[1], newsRouteDetail, true
	}

	if len(parts) == 3 &&
		parts[0] == "news" &&
		parts[1] != "" &&
		parts[2] == "read" {
		return parts[1], newsRouteRead, true
	}

	return "", newsRouteList, false
}

// ============================================================
// Pagination
// ============================================================

func parseNewsPage(r *http.Request) (domcommon.Page, error) {
	pageNumber := defaultNewsPage
	perPage := defaultNewsPerPage

	rawPage := r.URL.Query().Get("page")
	if rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed <= 0 {
			return domcommon.Page{}, errors.New("invalid page")
		}
		pageNumber = parsed
	}

	rawPerPage := r.URL.Query().Get("perPage")
	if rawPerPage != "" {
		parsed, err := strconv.Atoi(rawPerPage)
		if err != nil || parsed <= 0 || parsed > maxNewsPerPage {
			return domcommon.Page{}, errors.New("invalid perPage")
		}
		perPage = parsed
	}

	return domcommon.Page{
		Number:  pageNumber,
		PerPage: perPage,
	}, nil
}

// ============================================================
// Mapping
// ============================================================

func toNewsResponse(item uc.NewsRecipientItem) newsResponse {
	response := newsResponse{
		ID:          string(item.News.ID),
		Title:       item.News.Title,
		Body:        item.News.Body,
		PublishedAt: item.News.PublishedAt,
		CreatedAt:   item.News.CreatedAt,
		IsRead:      item.IsRead,
		ReadAt:      item.ReadAt,
	}

	if item.News.Image != nil {
		response.Image = &newsImageResponse{
			FileURL:    item.News.Image.FileURL,
			ObjectPath: item.News.Image.ObjectPath,
			FileName:   item.News.Image.FileName,
			MimeType:   item.News.Image.MimeType,
			FileSize:   item.News.Image.FileSize,
			Alt:        item.News.Image.Alt,
		}
	}

	return response
}

// ============================================================
// Error mapping
// ============================================================

func writeNewsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, uc.ErrNewsRepositoryNotConfigured),
		errors.Is(err, uc.ErrNewsReadRepositoryNotConfigured),
		errors.Is(err, uc.ErrNewsImageStorageNotConfigured):
		writeError(
			w,
			http.StatusServiceUnavailable,
			"NewsServiceUnavailable",
		)

	case errors.Is(err, newsdom.ErrNotFound),
		errors.Is(err, newsdom.ErrReadNotFound):
		writeError(w, http.StatusNotFound, "NewsNotFound")

	case errors.Is(err, newsdom.ErrConflict):
		writeError(w, http.StatusConflict, "NewsConflict")

	case errors.Is(err, newsdom.ErrInvalidID),
		errors.Is(err, newsdom.ErrInvalidTitle),
		errors.Is(err, newsdom.ErrInvalidBody),
		errors.Is(err, newsdom.ErrInvalidCreatedBy),
		errors.Is(err, newsdom.ErrInvalidCreatedAt),
		errors.Is(err, newsdom.ErrInvalidPublishedAt),
		errors.Is(err, newsdom.ErrPublishedBeforeCreated),
		errors.Is(err, newsdom.ErrInvalidImageFileURL),
		errors.Is(err, newsdom.ErrInvalidImageObjectPath),
		errors.Is(err, newsdom.ErrInvalidImageFileName),
		errors.Is(err, newsdom.ErrInvalidImageMimeType),
		errors.Is(err, newsdom.ErrInvalidImageFileSize),
		errors.Is(err, newsdom.ErrInvalidReadID),
		errors.Is(err, newsdom.ErrInvalidNewsID),
		errors.Is(err, newsdom.ErrInvalidRecipientType),
		errors.Is(err, newsdom.ErrInvalidRecipientID),
		errors.Is(err, newsdom.ErrInvalidReadAt):
		writeError(w, http.StatusBadRequest, "InvalidNews")

	default:
		writeError(
			w,
			http.StatusInternalServerError,
			"NewsInternalError",
		)
	}
}
