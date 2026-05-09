// backend/internal/adapters/in/http/mall/handler/model_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
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
		modelID := v.ID
		if modelID == "" {
			continue
		}

		mv, err := h.Repo.GetModelVariationByID(r.Context(), modelID)
		if err != nil {
			internalError(w, err.Error())
			return
		}

		dto, ok := toMallModelVariationDTO(mv)
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

	dto, ok := toMallModelVariationDTO(mv)
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

func toMallModelVariationDTO(mv *modeldom.ModelVariation) (malldto.CatalogModelVariationDTO, bool) {
	if mv == nil || mv.ID == "" {
		return malldto.CatalogModelVariationDTO{}, false
	}

	measurements := map[string]int{}
	for k, v := range mv.Measurements {
		if k == "" {
			continue
		}
		measurements[k] = v
	}

	return malldto.CatalogModelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		ModelNumber:        mv.ModelNumber,
		Size:               mv.Size,
		ColorName:          mv.Color.Name,
		ColorRGB:           mv.Color.RGB,
		Measurements:       measurements,
		StockKeys:          0,
	}, true
}
