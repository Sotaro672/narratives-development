// backend/internal/adapters/in/http/mall/handler/news_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	uc "narratives/internal/application/usecase"
	domcommon "narratives/internal/domain/common"
	newsdom "narratives/internal/domain/news"
)

const (
	defaultMallNewsPage    = 1
	defaultMallNewsPerPage = 20
	maxMallNewsPerPage     = 100

	meNewsPath = "/mall/me/news"
)

type NewsHandler struct {
	NewsUC *uc.NewsUsecase
}

func NewNewsHandler(
	newsUC *uc.NewsUsecase,
) *NewsHandler {
	return &NewsHandler{
		NewsUC: newsUC,
	}
}

// Supported:
//
//	GET  /mall/me/news
//	GET  /mall/me/news?page=1&perPage=20
//	GET  /mall/me/news/unread-count
//	POST /mall/me/news/{newsId}/read
//
// Security:
// - avatarId はクライアント入力から受け取らない。
// - UserAuthMiddleware + AvatarContextMiddleware が解決した current avatarId を利用する。
// - News の既読状態は AVATAR 単位で管理する。
// - Admin の内部識別子 CreatedBy はレスポンスへ公開しない。
func (h *NewsHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.NewsUC == nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "news_handler_not_initialized",
			},
		)
		return
	}

	newsID, route, matched := parseMallNewsPath(r.URL.Path)
	if !matched {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "not_found",
			},
		)
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || strings.TrimSpace(avatarID) == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "avatar_context_required",
			},
		)
		return
	}

	switch route {
	case mallNewsRouteList:
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(
				w,
				http.StatusMethodNotAllowed,
				map[string]string{
					"error": "method_not_allowed",
				},
			)
			return
		}

		h.list(
			w,
			r,
			avatarID,
		)

	case mallNewsRouteUnreadCount:
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(
				w,
				http.StatusMethodNotAllowed,
				map[string]string{
					"error": "method_not_allowed",
				},
			)
			return
		}

		h.unreadCount(
			w,
			r,
			avatarID,
		)

	case mallNewsRouteRead:
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(
				w,
				http.StatusMethodNotAllowed,
				map[string]string{
					"error": "method_not_allowed",
				},
			)
			return
		}

		h.markRead(
			w,
			r,
			avatarID,
			newsID,
		)

	default:
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "not_found",
			},
		)
	}
}

// ============================================================
// DTO
// ============================================================

type newsResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	PublishedAt time.Time  `json:"publishedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	IsRead      bool       `json:"isRead"`
	ReadAt      *time.Time `json:"readAt"`
}

type newsUnreadCountResponse struct {
	UnreadCount int `json:"unreadCount"`
}

type newsReadResponse struct {
	ID     string    `json:"id"`
	NewsID string    `json:"newsId"`
	IsRead bool      `json:"isRead"`
	ReadAt time.Time `json:"readAt"`
}

// ============================================================
// List
// ============================================================

func (h *NewsHandler) list(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
) {
	page, err := parseMallNewsPage(r)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_pagination",
			},
		)
		return
	}

	result, err := h.NewsUC.ListNewsForAvatar(
		r.Context(),
		avatarID,
		page,
	)
	if err != nil {
		writeMallNewsError(w, err)
		return
	}

	items := make(
		[]newsResponse,
		0,
		len(result.Items),
	)

	for _, item := range result.Items {
		items = append(
			items,
			toMallNewsResponse(item),
		)
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
// Unread count
// ============================================================

func (h *NewsHandler) unreadCount(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
) {
	count, err := h.NewsUC.CountUnreadNewsForAvatar(
		r.Context(),
		avatarID,
	)
	if err != nil {
		writeMallNewsError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		newsUnreadCountResponse{
			UnreadCount: count,
		},
	)
}

// ============================================================
// Mark read
// ============================================================

func (h *NewsHandler) markRead(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
	newsID string,
) {
	newsID = strings.TrimSpace(newsID)
	if newsID == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "news_id_required",
			},
		)
		return
	}

	read, err := h.NewsUC.MarkNewsReadForAvatar(
		r.Context(),
		newsdom.NewsID(newsID),
		avatarID,
	)
	if err != nil {
		writeMallNewsError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		newsReadResponse{
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

type mallNewsRouteKind int

const (
	mallNewsRouteList mallNewsRouteKind = iota
	mallNewsRouteUnreadCount
	mallNewsRouteRead
)

func parseMallNewsPath(
	path string,
) (
	newsID string,
	route mallNewsRouteKind,
	matched bool,
) {
	basePath := strings.TrimSuffix(
		meNewsPath,
		"/",
	)
	normalizedPath := strings.TrimSuffix(
		path,
		"/",
	)

	if normalizedPath == basePath {
		return "", mallNewsRouteList, true
	}

	unreadCountPath := basePath + "/unread-count"
	if normalizedPath == unreadCountPath {
		return "", mallNewsRouteUnreadCount, true
	}

	prefix := basePath + "/"
	if !strings.HasPrefix(
		normalizedPath,
		prefix,
	) {
		return "", mallNewsRouteList, false
	}

	relativePath := strings.TrimPrefix(
		normalizedPath,
		prefix,
	)
	parts := strings.Split(
		relativePath,
		"/",
	)

	if len(parts) == 2 &&
		strings.TrimSpace(parts[0]) != "" &&
		parts[1] == "read" {
		return strings.TrimSpace(parts[0]), mallNewsRouteRead, true
	}

	return "", mallNewsRouteList, false
}

// ============================================================
// Pagination
// ============================================================

func parseMallNewsPage(
	r *http.Request,
) (domcommon.Page, error) {
	pageNumber := defaultMallNewsPage
	perPage := defaultMallNewsPerPage

	rawPage := strings.TrimSpace(
		r.URL.Query().Get("page"),
	)
	if rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed <= 0 {
			return domcommon.Page{},
				errors.New("invalid page")
		}

		pageNumber = parsed
	}

	rawPerPage := strings.TrimSpace(
		r.URL.Query().Get("perPage"),
	)
	if rawPerPage != "" {
		parsed, err := strconv.Atoi(rawPerPage)
		if err != nil ||
			parsed <= 0 ||
			parsed > maxMallNewsPerPage {
			return domcommon.Page{},
				errors.New("invalid perPage")
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

func toMallNewsResponse(
	item uc.NewsRecipientItem,
) newsResponse {
	return newsResponse{
		ID:          string(item.News.ID),
		Title:       item.News.Title,
		Body:        item.News.Body,
		PublishedAt: item.News.PublishedAt,
		CreatedAt:   item.News.CreatedAt,
		IsRead:      item.IsRead,
		ReadAt:      item.ReadAt,
	}
}

// ============================================================
// Error mapping
// ============================================================

func writeMallNewsError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "news_internal_error",
			},
		)

	case errors.Is(
		err,
		context.Canceled,
	),
		errors.Is(
			err,
			context.DeadlineExceeded,
		):
		writeJSON(
			w,
			http.StatusRequestTimeout,
			map[string]string{
				"error": "request_timeout",
			},
		)

	case errors.Is(
		err,
		uc.ErrNewsRepositoryNotConfigured,
	),
		errors.Is(
			err,
			uc.ErrNewsReadRepositoryNotConfigured,
		):
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "news_service_unavailable",
			},
		)

	case errors.Is(
		err,
		newsdom.ErrNotFound,
	),
		errors.Is(
			err,
			newsdom.ErrReadNotFound,
		):
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "news_not_found",
			},
		)

	case errors.Is(
		err,
		newsdom.ErrConflict,
	):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": "news_conflict",
			},
		)

	case errors.Is(
		err,
		newsdom.ErrInvalidID,
	),
		errors.Is(
			err,
			newsdom.ErrInvalidTitle,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidBody,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidCreatedBy,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidCreatedAt,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidPublishedAt,
		),
		errors.Is(
			err,
			newsdom.ErrPublishedBeforeCreated,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidReadID,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidNewsID,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidRecipientType,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidRecipientID,
		),
		errors.Is(
			err,
			newsdom.ErrInvalidReadAt,
		):
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_news",
			},
		)

	case isNotFound(err):
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "news_not_found",
			},
		)

	default:
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "news_internal_error",
			},
		)
	}
}
