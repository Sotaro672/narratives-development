// backend/internal/adapters/in/http/sns/handler/model_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	snsdto "narratives/internal/application/query/mall/dto"
	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
)

// MallCatalogQuery is the minimal contract to serve /mall/catalog/{listId}.
// NOTE: We keep this as an interface here to avoid tight coupling.
// The concrete implementation is:
// - backend/internal/application/query/mall/catalog_query.go  (MallCatalogQuery / SNSCatalogQuery)
type MallCatalogQuery interface {
	GetByListID(ctx context.Context, listID string) (any, error)
}

// MallModelHandler serves buyer-facing model endpoints.
//
// Routes (intended):
// - GET /mall/models?productBlueprintId=xxxx
// - GET /mall/models/{modelId}
//
// Additionally (to avoid new catalog_handler.go):
// - GET /mall/catalog/{listId}
//
// IMPORTANT:
// This handler can be mounted to both:
// - mux.Handle("/mall/models", handler)
// - mux.Handle("/mall/models/", handler)
// - mux.Handle("/mall/catalog", handler)
// - mux.Handle("/mall/catalog/", handler)
type MallModelHandler struct {
	Repo modeldom.RepositoryPort

	// ✅ optional: catalog DTO builder
	Catalog MallCatalogQuery
}

func NewMallModelHandler(repo modeldom.RepositoryPort) http.Handler {
	return &MallModelHandler{Repo: repo, Catalog: nil}
}

// ✅ use this when you also want to serve /mall/catalog/{listId}
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

	// ============================================================
	// catalog: /mall/catalog/{listId}
	// ============================================================
	if strings.HasPrefix(path, "/mall/catalog/") {
		id := strings.TrimPrefix(path, "/mall/catalog/")
		id = strings.TrimSpace(id)
		if id == "" {
			notFound(w)
			return
		}
		h.handleGetCatalogByListID(w, r, id)
		return
	}
	if path == "/mall/catalog" {
		// collection endpoint is not defined
		notFound(w)
		return
	}

	// ============================================================
	// models: /mall/models
	// ============================================================
	if h.Repo == nil {
		internalError(w, "model repo is nil")
		return
	}

	// collection: /mall/models
	if path == "/mall/models" {
		h.handleListByProductBlueprintID(w, r)
		return
	}

	// item: /mall/models/{id}
	if strings.HasPrefix(path, "/mall/models/") {
		id := strings.TrimPrefix(path, "/mall/models/")
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

// GET /mall/models?productBlueprintId=xxxx
func (h *MallModelHandler) handleListByProductBlueprintID(w http.ResponseWriter, r *http.Request) {
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
	items := make([]mallModelItem, 0, len(res.Items))

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

		// ✅ buyer-facing DTO へ変換（lowerCamel）
		dto, ok := toMallModelVariationDTOAny(mv)
		if !ok {
			dto = snsdto.CatalogModelVariationDTO{
				ID:                 strings.TrimSpace(modelID),
				ProductBlueprintID: strings.TrimSpace(pbID),
				ModelNumber:        "",
				Size:               "",
				ColorName:          "",
				ColorRGB:           0,
				Measurements:       map[string]int{}, // ✅ 空でも出す（null回避）
				Products:           nil,
				StockKeys:          0,
			}
		}
		if dto.Measurements == nil {
			dto.Measurements = map[string]int{}
		}

		items = append(items, mallModelItem{
			ModelID:  modelID,
			Metadata: dto, // ✅ DTO を metadata に載せる
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

// GET /mall/models/{modelId}
func (h *MallModelHandler) handleGetByID(w http.ResponseWriter, r *http.Request, id string) {
	mv, err := h.Repo.GetModelVariationByID(r.Context(), id)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	dto, ok := toMallModelVariationDTOAny(mv)
	if !ok {
		dto = snsdto.CatalogModelVariationDTO{
			ID:                 strings.TrimSpace(id),
			ProductBlueprintID: "",
			ModelNumber:        "",
			Size:               "",
			ColorName:          "",
			ColorRGB:           0,
			Measurements:       map[string]int{},
			Products:           nil,
			StockKeys:          0,
		}
	}
	if dto.Measurements == nil {
		dto.Measurements = map[string]int{}
	}

	writeJSON(w, http.StatusOK, mallModelItem{
		ModelID:  id,
		Metadata: dto, // ✅ DTO
	})
}

// GET /mall/catalog/{listId}
func (h *MallModelHandler) handleGetCatalogByListID(w http.ResponseWriter, r *http.Request, listID string) {
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

// ------------------------------------------------------------
// DTO mapper (reflection) - aligned with mall/dto/catalog_dto.go
// (Color integrated: ColorName/ColorRGB)
// ------------------------------------------------------------

func toMallModelVariationDTOAny(v any) (snsdto.CatalogModelVariationDTO, bool) {
	if v == nil {
		return snsdto.CatalogModelVariationDTO{}, false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return snsdto.CatalogModelVariationDTO{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return snsdto.CatalogModelVariationDTO{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return snsdto.CatalogModelVariationDTO{}, false
	}

	// strings
	id := pickStringField(rv.Interface(), "ID", "Id", "ModelID", "ModelId", "modelId")
	if strings.TrimSpace(id) == "" {
		return snsdto.CatalogModelVariationDTO{}, false
	}

	pbID := pickStringField(rv.Interface(), "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId")
	modelNumber := pickStringField(rv.Interface(), "ModelNumber", "modelNumber")
	size := pickStringField(rv.Interface(), "Size", "size")

	dto := snsdto.CatalogModelVariationDTO{
		ID:                 strings.TrimSpace(id),
		ProductBlueprintID: strings.TrimSpace(pbID),
		ModelNumber:        strings.TrimSpace(modelNumber),
		Size:               strings.TrimSpace(size),

		// ✅ Color integrated
		ColorName: "",
		ColorRGB:  0,

		// ✅ always non-nil map for JSON (avoid null)
		Measurements: map[string]int{},

		// stock-related fields are not served by /mall/models, but keep zero-values
		Products:  nil,
		StockKeys: 0,
	}

	// color: Color.{Name,RGB} -> ColorName/ColorRGB
	if c := rv.FieldByName("Color"); c.IsValid() {
		if c.Kind() == reflect.Pointer {
			if !c.IsNil() {
				c = c.Elem()
			}
		}
		if c.IsValid() && c.Kind() == reflect.Struct {
			nf := c.FieldByName("Name")
			if nf.IsValid() && nf.Kind() == reflect.String {
				dto.ColorName = strings.TrimSpace(nf.String())
			}
			rf := c.FieldByName("RGB")
			if rf.IsValid() {
				dto.ColorRGB = toInt(rf)
			}
		}
	}

	// measurements: map[string]int (or map[string]any/number)
	if m := rv.FieldByName("Measurements"); m.IsValid() {
		if m.Kind() == reflect.Map && m.Type().Key().Kind() == reflect.String {
			out := make(map[string]int)
			iter := m.MapRange()
			for iter.Next() {
				k := strings.TrimSpace(iter.Key().String())
				if k == "" {
					continue
				}
				out[k] = toInt(iter.Value())
			}
			// ✅ keep non-nil (len==0 ok)
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
			return strings.TrimSpace(f.String())
		}
	}
	return ""
}

func toInt(v reflect.Value) int {
	if !v.IsValid() {
		return 0
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int(v.Uint())
	case reflect.Float32, reflect.Float64:
		return int(v.Float())
	default:
		return 0
	}
}
