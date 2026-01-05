// backend\internal\adapters\in\http\sns\handler\inventory_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strings"

	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
)

// SNSInventoryHandler serves buyer-facing inventory endpoints (read-only).
//
// Routes (read-only):
// - GET /sns/inventories?productBlueprintId=...&tokenBlueprintId=...
// - GET /sns/inventories/{id}
//
// NOTE:
// - inventories の docId は "productBlueprintId__tokenBlueprintId" を想定。
// - SNS は companyId 境界を持たない（公開・購入客向け）ため、クエリは極力パラメータで完結させる。
type SNSInventoryHandler struct {
	uc *usecase.InventoryUsecase
}

func NewSNSInventoryHandler(uc *usecase.InventoryUsecase) http.Handler {
	return &SNSInventoryHandler{uc: uc}
}

// ------------------------------
// Response DTOs (SNS)
// ------------------------------

type SnsInventoryModelStock struct {
	Products map[string]bool `json:"products"`
}

type SnsInventoryResponse struct {
	ID                 string                            `json:"id"`
	TokenBlueprintID   string                            `json:"tokenBlueprintId"`
	ProductBlueprintID string                            `json:"productBlueprintId"`
	ModelIDs           []string                          `json:"modelIds"`
	Stock              map[string]SnsInventoryModelStock `json:"stock"`
}

// ------------------------------
// http.Handler
// ------------------------------

func (h *SNSInventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.uc == nil {
		internalError(w, "usecase is nil")
		return
	}

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")

	// read-only
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// GET /sns/inventories (query by pb/tb)
	if path == "/sns/inventories" {
		h.getByQuery(w, r)
		return
	}

	// GET /sns/inventories/{id}
	if strings.HasPrefix(path, "/sns/inventories/") {
		rest := strings.TrimPrefix(path, "/sns/inventories/")
		parts := strings.Split(rest, "/")
		id := strings.TrimSpace(parts[0])
		if id == "" {
			badRequest(w, "invalid id")
			return
		}
		// no subroutes
		if len(parts) > 1 {
			notFound(w)
			return
		}

		h.getByID(w, r, id)
		return
	}

	notFound(w)
}

// ------------------------------
// GET handlers
// ------------------------------

func (h *SNSInventoryHandler) getByQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pb := strings.TrimSpace(q.Get("productBlueprintId"))
	tb := strings.TrimSpace(q.Get("tokenBlueprintId"))

	if pb == "" || tb == "" {
		badRequest(w, "productBlueprintId and tokenBlueprintId are required")
		return
	}

	// inventories docId rule: productBlueprintId__tokenBlueprintId
	id := pb + "__" + tb
	h.getByID(w, r, id)
}

func (h *SNSInventoryHandler) getByID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	m, err := h.uc.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, invdom.ErrNotFound) {
			notFound(w)
			return
		}
		writeInvErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toSnsInventoryResponse(m))
}

// ------------------------------
// Mapping
// ------------------------------

// ✅ accumulation は返さない。stock.products (productId set) のみを返す。
// ✅ inventory.Stock の実データ表現差（slice/map/struct）を吸収して products を作る。
func toSnsInventoryResponse(m invdom.Mint) SnsInventoryResponse {
	// modelId -> []productId（重複除去済み） を抽出
	byModel := countStockByModel(m)

	stock := make(map[string]SnsInventoryModelStock, len(byModel))
	modelIDs := make([]string, 0, len(byModel))

	for modelID, ids := range byModel {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		modelIDs = append(modelIDs, modelID)

		products := make(map[string]bool, len(ids))
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			products[id] = true
		}

		stock[modelID] = SnsInventoryModelStock{
			Products: products,
		}
	}

	sort.Strings(modelIDs)

	// 既存の m.ModelIDs があれば尊重（空なら count 結果を採用）
	finalModelIDs := m.ModelIDs
	if len(finalModelIDs) == 0 {
		finalModelIDs = modelIDs
	}

	return SnsInventoryResponse{
		ID:                 strings.TrimSpace(m.ID),
		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ProductBlueprintID: strings.TrimSpace(m.ProductBlueprintID),
		ModelIDs:           finalModelIDs,
		Stock:              stock,
	}
}

// ------------------------------
// countStockByModel
// ------------------------------
//
// inventory.Stock の実データが以下どれでも動くように、reflection で吸収します。
// - stock[modelId] = []string (Firestore: array of productIds)
// - stock[modelId] = map[string]bool (productId set)
// - stock[modelId] = struct{ Products ... }（Products が slice/map のどちらでも）
//
// 返り値: modelId -> productIds（重複除去済み）
func countStockByModel(m invdom.Mint) map[string][]string {
	out := map[string][]string{}

	rv := reflect.ValueOf(m)
	if !rv.IsValid() {
		return out
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return out
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return out
	}

	stock := rv.FieldByName("Stock")
	if !stock.IsValid() {
		return out
	}
	if stock.Kind() == reflect.Ptr {
		if stock.IsNil() {
			return out
		}
		stock = stock.Elem()
	}
	if stock.Kind() != reflect.Map {
		return out
	}

	iter := stock.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		if k.Kind() != reflect.String {
			continue
		}
		modelID := strings.TrimSpace(k.String())
		if modelID == "" {
			continue
		}

		ids := extractProductIDsFromStockValue(v)
		if len(ids) == 0 {
			out[modelID] = []string{}
			continue
		}

		// dedupe
		seen := map[string]struct{}{}
		buf := make([]string, 0, len(ids))
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			buf = append(buf, id)
		}
		sort.Strings(buf)
		out[modelID] = buf
	}

	return out
}

func extractProductIDsFromStockValue(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	// struct { Products: ... } を吸収
	if v.Kind() == reflect.Struct {
		pf := v.FieldByName("Products")
		if pf.IsValid() {
			return extractStringIDs(pf)
		}
	}

	// map/slice が直接入ってるケースを吸収
	return extractStringIDs(v)
}

func extractStringIDs(v reflect.Value) []string {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i)
			if e.Kind() == reflect.Interface && !e.IsNil() {
				e = e.Elem()
			}
			if e.Kind() == reflect.String {
				s := strings.TrimSpace(e.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out

	case reflect.Map:
		// map[string]bool / map[string]struct{} / map[string]any などは key を productId とみなす
		if v.Type().Key().Kind() != reflect.String {
			return nil
		}
		out := make([]string, 0, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() != reflect.String {
				continue
			}
			s := strings.TrimSpace(k.String())
			if s != "" {
				out = append(out, s)
			}
		}
		return out

	default:
		return nil
	}
}

// ------------------------------
// Error mapping
// ------------------------------

func writeInvErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, invdom.ErrNotFound):
		code = http.StatusNotFound
	default:
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
