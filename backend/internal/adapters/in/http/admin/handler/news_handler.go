// backend/internal/adapters/in/http/admin/handler/news_handler.go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	newsdom "narratives/internal/domain/news"
)

const (
	adminNewsPath       = "/admin/news"
	defaultNewsPerPage  = 50
	maxAdminNewsPerPage = 200
)

type NewsHandler struct {
	uc *usecase.NewsUsecase
}

func NewNewsHandler(
	uc *usecase.NewsUsecase,
) http.Handler {
	return http.HandlerFunc((&NewsHandler{
		uc: uc,
	}).handle)
}

// ============================================================
// Request / Response
// ============================================================

type createNewsRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type newsResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	PublishedAt string `json:"publishedAt"`
	CreatedAt   string `json:"createdAt"`
	CreatedBy   string `json:"createdBy"`
}

type newsListResponse struct {
	Items      []newsResponse `json:"items"`
	TotalCount int            `json:"totalCount"`
	TotalPages int            `json:"totalPages"`
	Page       int            `json:"page"`
	PerPage    int            `json:"perPage"`
}

// ============================================================
// Handler
// ============================================================

func (h *NewsHandler) handle(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.uc == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"news_usecase_not_initialized",
		)
		return
	}

	if r.URL.Path != adminNewsPath &&
		r.URL.Path != adminNewsPath+"/" {
		writeJSONError(
			w,
			http.StatusNotFound,
			"news_not_found",
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
	}
}

// ============================================================
// GET /admin/news
// ============================================================

func (h *NewsHandler) handleList(
	w http.ResponseWriter,
	r *http.Request,
) {
	query := r.URL.Query()

	filter := newsdom.Filter{
		FilterCommon: common.FilterCommon{
			SearchQuery: strings.TrimSpace(
				query.Get("search"),
			),
		},
	}

	sortValue, ok := parseNewsSort(
		query.Get("sort"),
		query.Get("order"),
	)
	if !ok {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_sort",
		)
		return
	}

	page := common.Page{
		Number: parsePositiveInt(
			query.Get("page"),
			1,
		),
		PerPage: adminNewsPerPage(
			query.Get("perPage"),
		),
	}

	result, err := h.uc.ListNews(
		r.Context(),
		filter,
		sortValue,
		page,
	)
	if err != nil {
		writeNewsError(
			w,
			err,
			"news_list_failed",
		)
		return
	}

	items := make(
		[]newsResponse,
		0,
		len(result.Items),
	)

	for _, entity := range result.Items {
		items = append(
			items,
			toNewsResponse(entity),
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		newsListResponse{
			Items:      items,
			TotalCount: result.TotalCount,
			TotalPages: result.TotalPages,
			Page:       result.Page,
			PerPage:    result.PerPage,
		},
	)
}

// ============================================================
// POST /admin/news
// ============================================================

func (h *NewsHandler) handleCreate(
	w http.ResponseWriter,
	r *http.Request,
) {
	adminUID, ok := middleware.CurrentAdminUID(r)
	if !ok || strings.TrimSpace(adminUID) == "" {
		writeJSONError(
			w,
			http.StatusUnauthorized,
			"admin_identity_not_found",
		)
		return
	}

	var request createNewsRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_request",
		)
		return
	}

	title := strings.TrimSpace(request.Title)
	if title == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"news_title_required",
		)
		return
	}

	body := strings.TrimSpace(request.Body)
	if body == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"news_body_required",
		)
		return
	}

	created, err := h.uc.CreateNews(
		r.Context(),
		usecase.CreateNewsInput{
			Title:     title,
			Body:      body,
			CreatedBy: adminUID,
		},
	)
	if err != nil {
		writeNewsError(
			w,
			err,
			"news_create_failed",
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		toNewsResponse(created),
	)
}

// ============================================================
// Response mapper
// ============================================================

func toNewsResponse(
	entity newsdom.News,
) newsResponse {
	return newsResponse{
		ID:          string(entity.ID),
		Title:       entity.Title,
		Body:        entity.Body,
		PublishedAt: newsTimeString(entity.PublishedAt),
		CreatedAt:   newsTimeString(entity.CreatedAt),
		CreatedBy:   entity.CreatedBy,
	}
}

func newsTimeString(
	value time.Time,
) string {
	if value.IsZero() {
		return ""
	}

	return value.
		UTC().
		Format(time.RFC3339Nano)
}

// ============================================================
// Sort / Paging
// ============================================================

func parseNewsSort(
	columnValue string,
	orderValue string,
) (common.Sort, bool) {
	column := strings.TrimSpace(columnValue)
	if column == "" {
		column = "publishedAt"
	}

	if _, ok := newsdom.AllowedSortColumns[column]; !ok {
		return common.Sort{}, false
	}

	orderValue = strings.ToLower(
		strings.TrimSpace(orderValue),
	)

	order := common.SortDesc
	if orderValue != "" {
		switch common.SortOrder(orderValue) {
		case common.SortAsc:
			order = common.SortAsc
		case common.SortDesc:
			order = common.SortDesc
		default:
			return common.Sort{}, false
		}
	}

	return common.Sort{
		Column: column,
		Order:  order,
	}, true
}

func adminNewsPerPage(
	value string,
) int {
	perPage := parsePositiveInt(
		value,
		defaultNewsPerPage,
	)

	if perPage > maxAdminNewsPerPage {
		return maxAdminNewsPerPage
	}

	return perPage
}

// ============================================================
// Error mapping
// ============================================================

func writeNewsError(
	w http.ResponseWriter,
	err error,
	fallback string,
) {
	switch {
	case errors.Is(
		err,
		usecase.ErrNewsRepositoryNotConfigured,
	),
		errors.Is(
			err,
			usecase.ErrNewsReadRepositoryNotConfigured,
		):
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"news_service_unavailable",
		)

	case errors.Is(
		err,
		newsdom.ErrNotFound,
	),
		errors.Is(
			err,
			newsdom.ErrReadNotFound,
		):
		writeJSONError(
			w,
			http.StatusNotFound,
			"news_not_found",
		)

	case errors.Is(
		err,
		newsdom.ErrConflict,
	):
		writeJSONError(
			w,
			http.StatusConflict,
			"news_conflict",
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
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_news",
		)

	default:
		writeJSONError(
			w,
			http.StatusInternalServerError,
			fallback,
		)
	}
}
