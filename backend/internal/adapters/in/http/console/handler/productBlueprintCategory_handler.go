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
	"time"

	usecase "narratives/internal/application/usecase"
	"narratives/internal/domain/common"
	categorydom "narratives/internal/domain/productBlueprintCategory"
)

// ------------------------------------------------------------
// Usecase contract
// ------------------------------------------------------------

type ProductBlueprintCategoryUsecase interface {
	GetByID(ctx context.Context, id string) (categorydom.ProductBlueprintCategory, error)

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

// NewHandler is kept for backward compatibility.
// Prefer NewProductBlueprintCategoryHandler in new DI wiring.
func NewHandler(uc ProductBlueprintCategoryUsecase) *Handler {
	return NewProductBlueprintCategoryHandler(uc)
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
		id := strings.TrimPrefix(path, "/console/product-blueprint-categories/")
		if id == "" || strings.Contains(id, "/") {
			notFound(w)
			return
		}

		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getByID(w, r, id)

	default:
		notFound(w)
	}
}

// ------------------------------------------------------------
// Response DTOs
// ------------------------------------------------------------

type ProductBlueprintCategoryResponse struct {
	ID string `json:"id"`

	Code   string `json:"code"`
	NameJa string `json:"nameJa"`
	NameEn string `json:"nameEn"`

	ParentID *string  `json:"parentId"`
	Path     []string `json:"path"`

	Kind         string `json:"kind"`
	DisplayOrder int    `json:"displayOrder"`

	Attributes CategoryAttributesResponse `json:"attributes"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type CategoryAttributesResponse struct {
	RequiresExpirationDate bool `json:"requiresExpirationDate"`
	RequiresLotNumber      bool `json:"requiresLotNumber"`
	RequiresIngredients    bool `json:"requiresIngredients"`
	RequiresAlcoholNotice  bool `json:"requiresAlcoholNotice"`
	RequiresCosmeticNotice bool `json:"requiresCosmeticNotice"`
	RequiresStorageMethod  bool `json:"requiresStorageMethod"`
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

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request, id string) {
	category, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		writeProductBlueprintCategoryErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toProductBlueprintCategoryResponse(category))
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
	ids, err := parseCategoryIDs(values.Get("ids"))
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	rootOnly, err := parseOptionalBool("rootOnly", values.Get("rootOnly"))
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

	createdFrom, err := parseOptionalTime("createdFrom", values.Get("createdFrom"))
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	createdTo, err := parseOptionalTime("createdTo", values.Get("createdTo"))
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	updatedFrom, err := parseOptionalTime("updatedFrom", values.Get("updatedFrom"))
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	updatedTo, err := parseOptionalTime("updatedTo", values.Get("updatedTo"))
	if err != nil {
		return usecase.ListProductBlueprintCategoriesQuery{}, err
	}

	if createdFrom != nil && createdTo != nil && createdFrom.After(*createdTo) {
		return usecase.ListProductBlueprintCategoriesQuery{},
			errors.New("createdFrom must not be after createdTo")
	}

	if updatedFrom != nil && updatedTo != nil && updatedFrom.After(*updatedTo) {
		return usecase.ListProductBlueprintCategoriesQuery{},
			errors.New("updatedFrom must not be after updatedTo")
	}

	query := usecase.ListProductBlueprintCategoriesQuery{
		SearchQuery: values.Get("search"),
		IDs:         ids,
		RootOnly:    rootOnly,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		UpdatedFrom: updatedFrom,
		UpdatedTo:   updatedTo,
		SortColumn:  values.Get("sort"),
		SortOrder:   sortOrder,
		Page:        page,
		PerPage:     perPage,
	}

	if value := values.Get("code"); value != "" {
		query.Code = &value
	}

	if value := values.Get("kind"); value != "" {
		query.Kind = &value
	}

	if value := values.Get("parentId"); value != "" {
		query.ParentID = &value
	}

	return query, nil
}

// ------------------------------------------------------------
// Response mapping
// ------------------------------------------------------------

func toProductBlueprintCategoryResponses(
	categories []categorydom.ProductBlueprintCategory,
) []ProductBlueprintCategoryResponse {
	responses := make([]ProductBlueprintCategoryResponse, 0, len(categories))

	for _, category := range categories {
		responses = append(responses, toProductBlueprintCategoryResponse(category))
	}

	return responses
}

func toProductBlueprintCategoryResponse(
	category categorydom.ProductBlueprintCategory,
) ProductBlueprintCategoryResponse {
	var parentID *string
	if category.ParentID != nil {
		value := string(*category.ParentID)
		parentID = &value
	}

	return ProductBlueprintCategoryResponse{
		ID:           string(category.ID),
		Code:         string(category.Code),
		NameJa:       category.NameJa,
		NameEn:       category.NameEn,
		ParentID:     parentID,
		Path:         append([]string(nil), category.Path...),
		Kind:         string(category.Kind),
		DisplayOrder: category.DisplayOrder,
		Attributes: CategoryAttributesResponse{
			RequiresExpirationDate: category.Attributes.RequiresExpirationDate,
			RequiresLotNumber:      category.Attributes.RequiresLotNumber,
			RequiresIngredients:    category.Attributes.RequiresIngredients,
			RequiresAlcoholNotice:  category.Attributes.RequiresAlcoholNotice,
			RequiresCosmeticNotice: category.Attributes.RequiresCosmeticNotice,
			RequiresStorageMethod:  category.Attributes.RequiresStorageMethod,
		},
		CreatedAt: formatTime(category.CreatedAt),
		UpdatedAt: formatTime(category.UpdatedAt),
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

	return errors.Is(err, categorydom.ErrInvalidID) ||
		errors.Is(err, categorydom.ErrInvalidCode) ||
		errors.Is(err, categorydom.ErrInvalidNameJa) ||
		errors.Is(err, categorydom.ErrInvalidKind) ||
		errors.Is(err, categorydom.ErrInvalidPath) ||
		errors.Is(err, categorydom.ErrInvalidDisplayOrder) ||
		errors.Is(err, categorydom.ErrInvalidCreatedAt) ||
		errors.Is(err, categorydom.ErrInvalidUpdatedAt) ||
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

func parseCategoryIDs(value string) ([]string, error) {
	if value == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, id := range parts {
		if id == "" {
			return nil, errors.New("ids must not contain an empty value")
		}

		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("ids contains a duplicate value: %s", id)
		}

		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids, nil
}

func parseOptionalBool(name string, value string) (bool, error) {
	switch value {
	case "":
		return false, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", name)
	}
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

func parseOptionalTime(name string, value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339: %w", name, err)
	}

	utc := parsed.UTC()
	return &utc, nil
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339)
}
