// backend/internal/adapters/in/http/admin/handler/news_handler.go
package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	newsdom "narratives/internal/domain/news"
)

const (
	adminNewsPath                 = "/admin/news"
	defaultNewsPerPage            = 50
	maxAdminNewsPerPage           = 200
	maxAdminNewsImageSize   int64 = 5 * 1024 * 1024
	maxAdminNewsRequestSize       = 7 * 1024 * 1024
)

// ============================================================
// Handler
// ============================================================

type NewsHandler struct {
	uc *usecase.NewsUsecase
}

func NewNewsHandler(uc *usecase.NewsUsecase) http.Handler {
	return http.HandlerFunc((&NewsHandler{
		uc: uc,
	}).handle)
}

// ============================================================
// Response
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
	PublishedAt string             `json:"publishedAt"`
	CreatedAt   string             `json:"createdAt"`
	CreatedBy   string             `json:"createdBy"`
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

func (h *NewsHandler) handle(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "news_usecase_not_initialized")
		return
	}

	if r.URL.Path != adminNewsPath && r.URL.Path != adminNewsPath+"/" {
		writeJSONError(w, http.StatusNotFound, "news_not_found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

// ============================================================
// GET /admin/news
// ============================================================

func (h *NewsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := newsdom.Filter{
		FilterCommon: common.FilterCommon{
			SearchQuery: query.Get("search"),
		},
	}

	sortValue, ok := parseNewsSort(
		query.Get("sort"),
		query.Get("order"),
	)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid_sort")
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
		writeNewsError(w, err, "news_list_failed")
		return
	}

	items := make([]newsResponse, 0, len(result.Items))
	for _, entity := range result.Items {
		items = append(items, toNewsResponse(entity))
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

func (h *NewsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	adminUID, ok := middleware.CurrentAdminUID(r)
	if !ok || adminUID == "" {
		writeJSONError(w, http.StatusUnauthorized, "admin_identity_not_found")
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxAdminNewsRequestSize,
	)

	if err := r.ParseMultipartForm(maxAdminNewsImageSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "news_request_too_large")
			return
		}

		writeJSONError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	title := r.FormValue("title")
	if title == "" {
		writeJSONError(w, http.StatusBadRequest, "news_title_required")
		return
	}

	body := r.FormValue("body")
	if body == "" {
		writeJSONError(w, http.StatusBadRequest, "news_body_required")
		return
	}

	input := usecase.CreateNewsInput{
		Title:     title,
		Body:      body,
		CreatedBy: adminUID,
	}

	imageFile, imageHeader, err := r.FormFile("image")
	switch {
	case err == nil:
		defer imageFile.Close()

		imageInput, err := createNewsImageInput(
			imageFile,
			imageHeader,
			r.FormValue("imageAlt"),
		)
		if err != nil {
			writeNewsImageError(w, err)
			return
		}

		input.Image = imageInput

	case errors.Is(err, http.ErrMissingFile):
		// image is optional.

	default:
		writeJSONError(w, http.StatusBadRequest, "invalid_news_image")
		return
	}

	created, err := h.uc.CreateNews(
		r.Context(),
		input,
	)
	if err != nil {
		writeNewsError(w, err, "news_create_failed")
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		toNewsResponse(created),
	)
}

// ============================================================
// Image request
// ============================================================

var (
	errNewsImageTooLarge        = errors.New("news image is too large")
	errNewsImageEmpty           = errors.New("news image is empty")
	errNewsImageInvalidMimeType = errors.New("news image has invalid mime type")
)

func createNewsImageInput(
	file multipart.File,
	header *multipart.FileHeader,
	alt string,
) (*usecase.CreateNewsImageInput, error) {
	if file == nil || header == nil {
		return nil, errNewsImageEmpty
	}

	if header.Size <= 0 {
		return nil, errNewsImageEmpty
	}
	if header.Size > maxAdminNewsImageSize {
		return nil, errNewsImageTooLarge
	}

	contentType, err := detectNewsImageContentType(file)
	if err != nil {
		return nil, err
	}

	return &usecase.CreateNewsImageInput{
		FileName:    header.Filename,
		ContentType: contentType,
		FileSize:    header.Size,
		Reader:      file,
		Alt:         alt,
	}, nil
}

func detectNewsImageContentType(file multipart.File) (string, error) {
	if file == nil {
		return "", errNewsImageEmpty
	}

	var header [512]byte
	readBytes, err := file.Read(header[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if readBytes <= 0 {
		return "", errNewsImageEmpty
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	contentType := strings.ToLower(
		http.DetectContentType(header[:readBytes]),
	)

	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return contentType, nil
	default:
		return "", errNewsImageInvalidMimeType
	}
}

func writeNewsImageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNewsImageTooLarge):
		writeJSONError(w, http.StatusRequestEntityTooLarge, "news_image_too_large")

	case errors.Is(err, errNewsImageEmpty):
		writeJSONError(w, http.StatusBadRequest, "news_image_empty")

	case errors.Is(err, errNewsImageInvalidMimeType):
		writeJSONError(w, http.StatusBadRequest, "invalid_news_image_type")

	default:
		writeJSONError(w, http.StatusBadRequest, "invalid_news_image")
	}
}

// ============================================================
// Response mapper
// ============================================================

func toNewsResponse(entity newsdom.News) newsResponse {
	response := newsResponse{
		ID:          string(entity.ID),
		Title:       entity.Title,
		Body:        entity.Body,
		PublishedAt: newsTimeString(entity.PublishedAt),
		CreatedAt:   newsTimeString(entity.CreatedAt),
		CreatedBy:   entity.CreatedBy,
	}

	if entity.Image != nil {
		response.Image = &newsImageResponse{
			FileURL:    entity.Image.FileURL,
			ObjectPath: entity.Image.ObjectPath,
			FileName:   entity.Image.FileName,
			MimeType:   entity.Image.MimeType,
			FileSize:   entity.Image.FileSize,
			Alt:        entity.Image.Alt,
		}
	}

	return response
}

func newsTimeString(value time.Time) string {
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
	column := columnValue
	if column == "" {
		column = "publishedAt"
	}

	if _, ok := newsdom.AllowedSortColumns[column]; !ok {
		return common.Sort{}, false
	}

	orderValue = strings.ToLower(orderValue)

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

func adminNewsPerPage(value string) int {
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
	case errors.Is(err, usecase.ErrNewsRepositoryNotConfigured),
		errors.Is(err, usecase.ErrNewsReadRepositoryNotConfigured),
		errors.Is(err, usecase.ErrNewsImageStorageNotConfigured):
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"news_service_unavailable",
		)

	case errors.Is(err, newsdom.ErrNotFound),
		errors.Is(err, newsdom.ErrReadNotFound):
		writeJSONError(
			w,
			http.StatusNotFound,
			"news_not_found",
		)

	case errors.Is(err, newsdom.ErrConflict):
		writeJSONError(
			w,
			http.StatusConflict,
			"news_conflict",
		)

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
