// backend/internal/adapters/in/http/console/handler/productBlueprintCategory_handler.go
package consoleHandler

import (
	"context"
	"errors"
	"net/http"
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
	GetByID(
		ctx context.Context,
		id string,
	) (categorydom.ProductBlueprintCategory, error)

	GetByCode(
		ctx context.Context,
		code string,
	) (categorydom.ProductBlueprintCategory, error)

	List(
		ctx context.Context,
		q usecase.ListProductBlueprintCategoriesQuery,
	) (common.PageResult[categorydom.ProductBlueprintCategory], error)

	ListTree(
		ctx context.Context,
	) ([]categorydom.ProductBlueprintCategory, error)
}

// ------------------------------------------------------------
// Handler
// ------------------------------------------------------------

type Handler struct {
	uc ProductBlueprintCategoryUsecase
}

func NewHandler(uc ProductBlueprintCategoryUsecase) *Handler {
	return &Handler{
		uc: uc,
	}
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
		return

	case path == "/console/product-blueprint-categories/tree":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.listTree(w, r)
		return

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
		return

	default:
		notFound(w)
		return
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

	ParentID *string  `json:"parentId,omitempty"`
	Path     []string `json:"path"`

	Kind string `json:"kind"`

	DisplayOrder int `json:"displayOrder"`

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

	writeJSON(w, http.StatusOK, toResponse(category))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()

	query := usecase.ListProductBlueprintCategoriesQuery{
		SearchQuery: strings.TrimSpace(qp.Get("search")),

		IDs: parseCSV(qp.Get("ids")),

		RootOnly: parseBoolDefaultFalse(qp.Get("rootOnly")),

		SortColumn: strings.TrimSpace(qp.Get("sort")),
		SortOrder:  parseSortOrder(qp.Get("order")),

		Page:    parseIntDefault(qp.Get("page"), 1),
		PerPage: parseIntDefault(qp.Get("perPage"), 20),
	}

	if v := strings.TrimSpace(qp.Get("code")); v != "" {
		query.Code = &v
	}

	if v := strings.TrimSpace(qp.Get("kind")); v != "" {
		query.Kind = &v
	}

	if v := strings.TrimSpace(qp.Get("parentId")); v != "" {
		query.ParentID = &v
	}

	if t := parseTimePtr(qp.Get("createdFrom")); t != nil {
		query.CreatedFrom = t
	}

	if t := parseTimePtr(qp.Get("createdTo")); t != nil {
		query.CreatedTo = t
	}

	if t := parseTimePtr(qp.Get("updatedFrom")); t != nil {
		query.UpdatedFrom = t
	}

	if t := parseTimePtr(qp.Get("updatedTo")); t != nil {
		query.UpdatedTo = t
	}

	result, err := h.uc.List(r.Context(), query)
	if err != nil {
		writeProductBlueprintCategoryErr(w, err)
		return
	}

	items := make([]ProductBlueprintCategoryResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toResponse(item))
	}

	writeJSON(w, http.StatusOK, ProductBlueprintCategoryListResponse{
		Items:      items,
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

	out := make([]ProductBlueprintCategoryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toResponse(item))
	}

	writeJSON(w, http.StatusOK, ProductBlueprintCategoryTreeResponse{
		Items: out,
	})
}

// ------------------------------------------------------------
// Mapping
// ------------------------------------------------------------

func toResponse(
	category categorydom.ProductBlueprintCategory,
) ProductBlueprintCategoryResponse {
	var parentID *string
	if category.ParentID != nil {
		v := string(*category.ParentID)
		parentID = &v
	}

	return ProductBlueprintCategoryResponse{
		ID:       string(category.ID),
		Code:     string(category.Code),
		NameJa:   category.NameJa,
		NameEn:   category.NameEn,
		ParentID: parentID,
		Path:     append([]string(nil), category.Path...),

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
// Helpers
// ------------------------------------------------------------

func writeProductBlueprintCategoryErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, categorydom.ErrNotFound) || categorydom.IsNotFound(err):
		code = http.StatusNotFound

	case errors.Is(err, categorydom.ErrConflict) || categorydom.IsConflict(err):
		code = http.StatusConflict

	case errors.Is(err, categorydom.ErrUnauthorized) || categorydom.IsUnauthorized(err):
		code = http.StatusUnauthorized

	case errors.Is(err, categorydom.ErrForbidden) || categorydom.IsForbidden(err):
		code = http.StatusForbidden

	case errors.Is(err, categorydom.ErrInvalid) ||
		categorydom.IsInvalid(err) ||
		isCategoryValidationErr(err):
		code = http.StatusBadRequest

	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeJSON(w, code, map[string]string{
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

func parseCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		out = append(out, v)
	}

	return out
}

func parseBoolDefaultFalse(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func parseSortOrder(s string) common.SortOrder {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(common.SortDesc), "descending":
		return common.SortDesc
	case string(common.SortAsc), "ascending":
		return common.SortAsc
	default:
		return ""
	}
}

func parseTimePtr(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		utc := t.UTC()
		return &utc
	}

	if t, err := time.Parse("2006-01-02", s); err == nil {
		utc := t.UTC()
		return &utc
	}

	return nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}
