// backend/internal/adapters/in/http/sns/handler/model_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
)

// SNSCatalogQuery is the minimal contract to serve /sns/catalog/{listId}.
// NOTE: We keep this as an interface here to avoid tight coupling.
// The concrete implementation is:
// - backend/internal/application/query/sns/catalog_query.go  (SNSCatalogQuery)
type SNSCatalogQuery interface {
	GetByListID(ctx context.Context, listID string) (any, error)
}

// SNSModelHandler serves buyer-facing model endpoints.
//
// Routes (intended):
// - GET /sns/models?productBlueprintId=xxxx
// - GET /sns/models/{modelId}
//
// Additionally (to avoid new catalog_handler.go):
// - GET /sns/catalog/{listId}
//
// IMPORTANT:
// This handler can be mounted to both:
// - mux.Handle("/sns/models", handler)
// - mux.Handle("/sns/models/", handler)
// - mux.Handle("/sns/catalog", handler)
// - mux.Handle("/sns/catalog/", handler)
type SNSModelHandler struct {
	Repo modeldom.RepositoryPort

	// ✅ optional: catalog DTO builder
	Catalog SNSCatalogQuery
}

func NewSNSModelHandler(repo modeldom.RepositoryPort) http.Handler {
	return &SNSModelHandler{Repo: repo, Catalog: nil}
}

// ✅ NEW: use this when you also want to serve /sns/catalog/{listId}
func NewSNSModelHandlerWithCatalog(repo modeldom.RepositoryPort, catalog SNSCatalogQuery) http.Handler {
	return &SNSModelHandler{Repo: repo, Catalog: catalog}
}

func (h *SNSModelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		internalError(w, "model handler is not ready")
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	// ============================================================
	// catalog: /sns/catalog/{listId}
	// ============================================================
	if strings.HasPrefix(path, "/sns/catalog/") {
		id := strings.TrimPrefix(path, "/sns/catalog/")
		id = strings.TrimSpace(id)
		if id == "" {
			notFound(w)
			return
		}
		h.handleGetCatalogByListID(w, r, id)
		return
	}
	if path == "/sns/catalog" {
		// collection endpoint is not defined
		notFound(w)
		return
	}

	// ============================================================
	// models: /sns/models
	// ============================================================
	if h.Repo == nil {
		internalError(w, "model repo is nil")
		return
	}

	// collection: /sns/models
	if path == "/sns/models" {
		h.handleListByProductBlueprintID(w, r)
		return
	}

	// item: /sns/models/{id}
	if strings.HasPrefix(path, "/sns/models/") {
		id := strings.TrimPrefix(path, "/sns/models/")
		id = strings.TrimSpace(id)
		if id == "" {
			notFound(w)
			return
		}
		h.handleGetByID(w, r, id)
		return
	}

	notFound(w)
}

type snsModelItem struct {
	ModelID  string `json:"modelId"`
	Metadata any    `json:"metadata"`
}

type snsModelListResponse struct {
	Items      []snsModelItem `json:"items"`
	TotalCount int            `json:"totalCount"`
	TotalPages int            `json:"totalPages"`
	Page       int            `json:"page"`
	PerPage    int            `json:"perPage"`
}

// GET /sns/models?productBlueprintId=xxxx
func (h *SNSModelHandler) handleListByProductBlueprintID(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// accept both productBlueprintId and pb as alias
	pbID := strings.TrimSpace(q.Get("productBlueprintId"))
	if pbID == "" {
		pbID = strings.TrimSpace(q.Get("pb"))
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
	// protect from huge payloads
	if perPage > 200 {
		perPage = 200
	}

	deletedFalse := false

	// 1) productBlueprintId で variation 一覧を取得（= modelId 一覧の取得）
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

	// 2) それぞれの modelId で metadata を取得
	items := make([]snsModelItem, 0, len(res.Items))
	for _, v := range res.Items {
		modelID := extractID(v)
		if modelID == "" {
			// ID が取れない場合はスキップ（異常データ対策）
			continue
		}

		mv, err := h.Repo.GetModelVariationByID(r.Context(), modelID)
		if err != nil {
			// 1件でも失敗したら全体を失敗にする（必要ならここは「欠損許容」に変更可）
			internalError(w, err.Error())
			return
		}

		items = append(items, snsModelItem{
			ModelID:  modelID,
			Metadata: mv,
		})
	}

	writeJSON(w, http.StatusOK, snsModelListResponse{
		Items:      items,
		TotalCount: res.TotalCount,
		TotalPages: res.TotalPages,
		Page:       res.Page,
		PerPage:    res.PerPage,
	})
}

// GET /sns/models/{modelId}
func (h *SNSModelHandler) handleGetByID(w http.ResponseWriter, r *http.Request, id string) {
	mv, err := h.Repo.GetModelVariationByID(r.Context(), id)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, snsModelItem{
		ModelID:  id,
		Metadata: mv,
	})
}

// GET /sns/catalog/{listId}
func (h *SNSModelHandler) handleGetCatalogByListID(w http.ResponseWriter, r *http.Request, listID string) {
	if h.Catalog == nil {
		internalError(w, "catalog query is nil")
		return
	}

	dto, err := h.Catalog.GetByListID(r.Context(), listID)
	if err != nil {
		// list not found / not listing -> 404
		if errors.Is(err, ldom.ErrNotFound) {
			notFound(w)
			return
		}
		internalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

// extractID tries common field names (ID/Id/ModelID/ModelId) by reflection.
// This avoids compile-time dependency on ModelVariation's concrete fields.
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
			return strings.TrimSpace(f.String())
		}
	}

	return ""
}
