// backend/internal/adapters/in/http/mall/handler/model_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	malldto "narratives/internal/application/query/mall/dto"
	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
)

type MallCatalogQuery interface {
	GetByListID(ctx context.Context, listID string) (any, error)
}

type MallModelHandler struct {
	Repo    modeldom.RepositoryPort
	Catalog MallCatalogQuery
}

func NewMallModelHandler(repo modeldom.RepositoryPort) http.Handler {
	return &MallModelHandler{Repo: repo, Catalog: nil}
}

func NewMallModelHandlerWithCatalog(repo modeldom.RepositoryPort, catalog MallCatalogQuery) http.Handler {
	return &MallModelHandler{Repo: repo, Catalog: catalog}
}

func (h *MallModelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		internalError(w, "model handler is not ready")
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	if strings.HasPrefix(path, "/mall/catalog/") {
		id := strings.TrimPrefix(path, "/mall/catalog/")
		if id == "" {
			notFound(w)
			return
		}
		h.handleGetCatalogByListID(w, r, id)
		return
	}
	if path == "/mall/catalog" {
		notFound(w)
		return
	}

	if h.Repo == nil {
		internalError(w, "model repo is nil")
		return
	}

	if path == "/mall/models" {
		h.handleListByProductBlueprintID(w, r)
		return
	}

	if strings.HasPrefix(path, "/mall/models/") {
		id := strings.TrimPrefix(path, "/mall/models/")
		if id == "" {
			notFound(w)
			return
		}
		h.handleGetByID(w, r, id)
		return
	}

	notFound(w)
}

type mallModelItem struct {
	ModelID  string `json:"modelId"`
	Metadata any    `json:"metadata"`
}

type mallModelListResponse struct {
	Items      []mallModelItem `json:"items"`
	TotalCount int             `json:"totalCount"`
	TotalPages int             `json:"totalPages"`
	Page       int             `json:"page"`
	PerPage    int             `json:"perPage"`
}

func (h *MallModelHandler) handleListByProductBlueprintID(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	pbID := q.Get("productBlueprintId")
	if pbID == "" {
		pbID = q.Get("pb")
	}
	if pbID == "" {
		badRequest(w, "productBlueprintId is required")
		return
	}

	page := parseIntDefault(q.Get("page"), 1)
	perPage := parseIntDefault(q.Get("perPage"), 50)
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}

	deletedFalse := false

	res, err := h.Repo.ListVariations(
		r.Context(),
		modeldom.VariationFilter{
			ProductBlueprintID: pbID,
			Deleted:            &deletedFalse,
		},
		modeldom.Page{
			Number:  page,
			PerPage: perPage,
		},
	)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	items := make([]mallModelItem, 0, len(res.Items))

	for _, v := range res.Items {
		modelID := extractID(v)
		if modelID == "" {
			continue
		}

		mv, err := h.Repo.GetModelVariationByID(r.Context(), modelID)
		if err != nil {
			internalError(w, err.Error())
			return
		}

		dto, ok := toMallModelVariationDTOAny(mv)
		if !ok {
			dto = malldto.CatalogModelVariationDTO{
				ID:                 modelID,
				ProductBlueprintID: pbID,
				ModelNumber:        "",
				Size:               "",
				ColorName:          "",
				ColorRGB:           0,
				Measurements:       map[string]int{},
				StockKeys:          0,
			}
		}
		if dto.Measurements == nil {
			dto.Measurements = map[string]int{}
		}

		items = append(items, mallModelItem{
			ModelID:  modelID,
			Metadata: dto,
		})
	}

	writeJSON(w, http.StatusOK, mallModelListResponse{
		Items:      items,
		TotalCount: res.TotalCount,
		TotalPages: res.TotalPages,
		Page:       res.Page,
		PerPage:    res.PerPage,
	})
}

func (h *MallModelHandler) handleGetByID(w http.ResponseWriter, r *http.Request, id string) {
	mv, err := h.Repo.GetModelVariationByID(r.Context(), id)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	dto, ok := toMallModelVariationDTOAny(mv)
	if !ok {
		dto = malldto.CatalogModelVariationDTO{
			ID:                 id,
			ProductBlueprintID: "",
			ModelNumber:        "",
			Size:               "",
			ColorName:          "",
			ColorRGB:           0,
			Measurements:       map[string]int{},
			StockKeys:          0,
		}
	}
	if dto.Measurements == nil {
		dto.Measurements = map[string]int{}
	}

	writeJSON(w, http.StatusOK, mallModelItem{
		ModelID:  id,
		Metadata: dto,
	})
}

func (h *MallModelHandler) handleGetCatalogByListID(w http.ResponseWriter, r *http.Request, listID string) {
	if h.Catalog == nil {
		internalError(w, "catalog query is nil")
		return
	}

	dto, err := h.Catalog.GetByListID(r.Context(), listID)
	if err != nil {
		if errors.Is(err, ldom.ErrNotFound) {
			notFound(w)
			return
		}
		internalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

func extractID(v any) string {
	if v == nil {
		return ""
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range []string{"ID", "Id", "ModelID", "ModelId"} {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return f.String()
		}
	}

	return ""
}

func toMallModelVariationDTOAny(v any) (malldto.CatalogModelVariationDTO, bool) {
	if v == nil {
		return malldto.CatalogModelVariationDTO{}, false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return malldto.CatalogModelVariationDTO{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return malldto.CatalogModelVariationDTO{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return malldto.CatalogModelVariationDTO{}, false
	}

	id := pickStringField(rv.Interface(), "ID", "Id", "ModelID", "ModelId", "modelId")
	if id == "" {
		return malldto.CatalogModelVariationDTO{}, false
	}

	pbID := pickStringField(rv.Interface(), "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	modelNumber := pickStringField(rv.Interface(), "ModelNumber", "modelNumber")
	size := pickStringField(rv.Interface(), "Size", "size")

	dto := malldto.CatalogModelVariationDTO{
		ID:                 id,
		ProductBlueprintID: pbID,
		ModelNumber:        modelNumber,
		Size:               size,
		ColorName:          "",
		ColorRGB:           0,
		Measurements:       map[string]int{},
		StockKeys:          0,
	}

	if c := rv.FieldByName("Color"); c.IsValid() {
		if c.Kind() == reflect.Pointer {
			if !c.IsNil() {
				c = c.Elem()
			}
		}
		if c.IsValid() && c.Kind() == reflect.Struct {
			nf := c.FieldByName("Name")
			if nf.IsValid() && nf.Kind() == reflect.String {
				dto.ColorName = nf.String()
			}
			rf := c.FieldByName("RGB")
			if rf.IsValid() {
				dto.ColorRGB = toInt(rf)
			}
		}
	}

	if m := rv.FieldByName("Measurements"); m.IsValid() {
		if m.Kind() == reflect.Map && m.Type().Key().Kind() == reflect.String {
			out := make(map[string]int)
			iter := m.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				if k == "" {
					continue
				}
				out[k] = toInt(iter.Value())
			}
			dto.Measurements = out
		}
	}

	if dto.Measurements == nil {
		dto.Measurements = map[string]int{}
	}

	return dto, true
}

func pickStringField(v any, fieldNames ...string) string {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}
