// backend\internal\adapters\in\http\console\handler\productBlueprintCategory_handler.go
package consoleHandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	usecase "narratives/internal/application/usecase"
	"narratives/internal/domain/common"
	categorydom "narratives/internal/domain/productBlueprintCategory"
)

// ------------------------------------------------------------
// Usecase contract
// ------------------------------------------------------------

type ProductBlueprintCategoryUsecase interface {
	GetByPath(
		ctx context.Context,
		path []string,
	) (categorydom.ProductBlueprintCategory, error)

	List(
		ctx context.Context,
		q usecase.ListProductBlueprintCategoriesQuery,
	) (common.PageResult[categorydom.ProductBlueprintCategory], error)

	ListTree(ctx context.Context) ([]categorydom.ProductBlueprintCategory, error)
}

// ------------------------------------------------------------
// Handler
// ------------------------------------------------------------

type Handler struct {
	uc ProductBlueprintCategoryUsecase
}

func NewProductBlueprintCategoryHandler(uc ProductBlueprintCategoryUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.uc == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "productBlueprintCategory usecase is nil",
		})
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case path == "/console/product-blueprint-categories":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.list(w, r)

	case path == "/console/product-blueprint-categories/tree":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.listTree(w, r)

	case strings.HasPrefix(path, "/console/product-blueprint-categories/"):
		rawPath := strings.TrimPrefix(
			path,
			"/console/product-blueprint-categories/",
		)

		categoryPath, err := parseCategoryPath(rawPath)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
			return
		}

		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getByPath(w, r, categoryPath)

	default:
		notFound(w)
	}
}

// ------------------------------------------------------------
// Response DTOs
// ------------------------------------------------------------

type ProductBlueprintCategoryResponse struct {
	ProductBlueprintCategoryPath []string `json:"productBlueprintCategoryPath"`
}

type ProductBlueprintCategoryListResponse struct {
	Items      []ProductBlueprintCategoryResponse `json:"items"`
	TotalCount int                                `json:"totalCount"`
	TotalPages int                                `json:"totalPages"`
	Page       int                                `json:"page"`
	PerPage    int                                `json:"perPage"`
}

type ProductBlueprintCategoryTreeResponse struct {
	Items []ProductBlueprintCategoryResponse `json:"items"`
}

// ------------------------------------------------------------
// Endpoints
// ------------------------------------------------------------

func (h *Handler) getByPath(
	w http.ResponseWriter,
	r *http.Request,
	path []string,
) {
	category, err := h.uc.GetByPath(
		r.Context(),
		path,
	)
	if err != nil {
		writeProductBlueprintCategoryErr(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		toProductBlueprintCategoryResponse(category),
	)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query, err := buildListProductBlueprintCategoriesQuery(r.URL.Query())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	result, err := h.uc.List(r.Context(), query)
	if err != nil {
		writeProductBlueprintCategoryErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ProductBlueprintCategoryListResponse{
		Items:      toProductBlueprintCategoryResponses(result.Items),
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	})
}

func (h *Handler) listTree(w http.ResponseWriter, r *http.Request) {
	items, err := h.uc.ListTree(r.Context())
	if err != nil {
		writeProductBlueprintCategoryErr(w, err)
		return
	}

	if len(items) == 0 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "product blueprint category master is empty",
		})
		return
	}

	writeJSON(w, http.StatusOK, ProductBlueprintCategoryTreeResponse{
		Items: toProductBlueprintCategoryResponses(items),
	})
}

// ------------------------------------------------------------
// Query mapping
// ------------------------------------------------------------

func buildListProductBlueprintCategoriesQuery(
	values url.Values,
) (usecase.ListProductBlueprintCategoriesQuery, error) {
	paths, err := parseCategoryPaths(
		values.Get("paths"),
	)
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	sortOrder, err := parseOptionalSortOrder(values.Get("order"))
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	page, err := parseStrictPositiveInt("page", values.Get("page"), 1)
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	perPage, err := parseStrictPositiveInt("perPage", values.Get("perPage"), 20)
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	if perPage > 500 {
		return usecase.ListProductBlueprintCategoriesQuery{},
			errors.New("perPage must be less than or equal to 500")
	}

	return usecase.ListProductBlueprintCategoriesQuery{
		SearchQuery: values.Get("search"),
		Paths:       paths,
		SortColumn:  values.Get("sort"),
		SortOrder:   sortOrder,
		Page:        page,
		PerPage:     perPage,
	}, nil
}

// ------------------------------------------------------------
// Response mapping
// ------------------------------------------------------------

func toProductBlueprintCategoryResponses(
	categories []categorydom.ProductBlueprintCategory,
) []ProductBlueprintCategoryResponse {
	responses := make([]ProductBlueprintCategoryResponse, 0, len(categories))

	for _, category := range categories {
		responses = append(
			responses,
			toProductBlueprintCategoryResponse(category),
		)
	}

	return responses
}

func toProductBlueprintCategoryResponse(
	category categorydom.ProductBlueprintCategory,
) ProductBlueprintCategoryResponse {
	return ProductBlueprintCategoryResponse{
		ProductBlueprintCategoryPath: append(
			[]string(nil),
			category.Path...,
		),
	}
}

// ------------------------------------------------------------
// Error handling
// ------------------------------------------------------------

func writeProductBlueprintCategoryErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(err, categorydom.ErrNotFound):
		statusCode = http.StatusNotFound

	case errors.Is(err, categorydom.ErrConflict):
		statusCode = http.StatusConflict

	case errors.Is(err, categorydom.ErrUnauthorized):
		statusCode = http.StatusUnauthorized

	case errors.Is(err, categorydom.ErrForbidden):
		statusCode = http.StatusForbidden

	case errors.Is(err, categorydom.ErrInvalid), isCategoryValidationErr(err):
		statusCode = http.StatusBadRequest
	}

	writeJSON(w, statusCode, map[string]string{
		"error": err.Error(),
	})
}

func isCategoryValidationErr(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, categorydom.ErrInvalidPath) ||
		errors.Is(err, categorydom.ErrRepositoryInvalidInput)
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{
		"error": "not found",
	})
}

// ------------------------------------------------------------
// Query helpers
// ------------------------------------------------------------

func parseCategoryPath(value string) ([]string, error) {
	if value == "" {
		return nil, errors.New(
			"productBlueprintCategoryPath must not be empty",
		)
	}

	parts := strings.Split(
		value,
		"/",
	)

	path := make([]string, 0, len(parts))

	for _, segment := range parts {
		if segment == "" {
			return nil, errors.New(
				"productBlueprintCategoryPath must not contain an empty segment",
			)
		}

		path = append(
			path,
			segment,
		)
	}

	return path, nil
}

func parseCategoryPaths(value string) ([][]string, error) {
	if value == "" {
		return nil, nil
	}

	values := strings.Split(
		value,
		",",
	)

	paths := make([][]string, 0, len(values))

	for _, rawPath := range values {
		path, err := parseCategoryPath(rawPath)
		if err != nil {
			return nil, err
		}

		if containsCategoryPath(
			paths,
			path,
		) {
			return nil, fmt.Errorf(
				"paths contains a duplicate value: %s",
				rawPath,
			)
		}

		paths = append(
			paths,
			path,
		)
	}

	return paths, nil
}

func containsCategoryPath(
	paths [][]string,
	path []string,
) bool {
	for _, candidate := range paths {
		if equalCategoryPath(
			candidate,
			path,
		) {
			return true
		}
	}

	return false
}

func equalCategoryPath(
	a []string,
	b []string,
) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func parseOptionalSortOrder(value string) (common.SortOrder, error) {
	switch value {
	case "":
		return "", nil
	case string(common.SortAsc):
		return common.SortAsc, nil
	case string(common.SortDesc):
		return common.SortDesc, nil
	default:
		return "", errors.New("order must be asc or desc")
	}
}

func parseStrictPositiveInt(name string, value string, defaultValue int) (int, error) {
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return parsed, nil
}
