// backend/internal/adapters/in/http/mall/handler/inventory_handler.go
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

// MallInventoryHandler serves buyer-facing inventory endpoints (read-only).
//
// Routes (read-only):
// - GET /mall/inventories?productBlueprintId=...&tokenBlueprintId=...
// - GET /mall/inventories/{id}
//
// NOTE:
// - inventories の docId は "productBlueprintId__tokenBlueprintId" を想定。
// - mall は companyId 境界を持たない（公開・購入客向け）ため、クエリは極力パラメータで完結させる。
type MallInventoryHandler struct {
	uc *usecase.InventoryUsecase
}

func NewMallInventoryHandler(uc *usecase.InventoryUsecase) http.Handler {
	return &MallInventoryHandler{uc: uc}
}

// ------------------------------
// Response DTOs (mall)
// ------------------------------

// ✅ products map は廃止し、accumulation / reservedCount を返す
type MallInventoryModelStock struct {
	Accumulation  int `json:"accumulation"`
	ReservedCount int `json:"reservedCount"`
}

type MallInventoryResponse struct {
	ID                 string                             `json:"id"`
	TokenBlueprintID   string                             `json:"tokenBlueprintId"`
	ProductBlueprintID string                             `json:"productBlueprintId"`
	ModelIDs           []string                           `json:"modelIds"`
	Stock              map[string]MallInventoryModelStock `json:"stock"`
}

// ------------------------------
// http.Handler
// ------------------------------

func (h *MallInventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	// GET /mall/inventories (query by pb/tb)
	if path == "/mall/inventories" {
		h.getByQuery(w, r)
		return
	}

	// GET /mall/inventories/{id}
	if strings.HasPrefix(path, "/mall/inventories/") {
		rest := strings.TrimPrefix(path, "/mall/inventories/")
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

func (h *MallInventoryHandler) getByQuery(w http.ResponseWriter, r *http.Request) {
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

func (h *MallInventoryHandler) getByID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	m, err := h.uc.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, invdom.ErrNotFound) {
			notFound(w)
			return
		}
		writeMallInvErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toMallInventoryResponse(m))
}

// ------------------------------
// Mapping
// ------------------------------

// ✅ products は返さない。
// ✅ accumulation / reservedCount を返す。
// ✅ inventory.Stock の実データ表現差（struct/ptr/any）を吸収して数値を作る。
func toMallInventoryResponse(m invdom.Mint) MallInventoryResponse {
	byModel := extractStockNumbersByModel(m)

	stock := make(map[string]MallInventoryModelStock, len(byModel))
	modelIDs := make([]string, 0, len(byModel))

	for modelID, nums := range byModel {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		modelIDs = append(modelIDs, modelID)

		acc := nums.Accumulation
		rc := nums.ReservedCount

		// ✅ 最低限の整合: reservedCount が負になるケースは 0 に丸める
		if rc < 0 {
			rc = 0
		}
		// ✅ accumulation が負になるケースは 0 に丸める
		if acc < 0 {
			acc = 0
		}

		stock[modelID] = MallInventoryModelStock{
			Accumulation:  acc,
			ReservedCount: rc,
		}
	}

	sort.Strings(modelIDs)

	// 既存の m.ModelIDs があれば尊重（空なら抽出結果を採用）
	finalModelIDs := m.ModelIDs
	if len(finalModelIDs) == 0 {
		finalModelIDs = modelIDs
	}

	return MallInventoryResponse{
		ID:                 strings.TrimSpace(m.ID),
		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ProductBlueprintID: strings.TrimSpace(m.ProductBlueprintID),
		ModelIDs:           append([]string{}, finalModelIDs...),
		Stock:              stock,
	}
}

// ------------------------------
// extractStockNumbersByModel
// ------------------------------

type modelStockNumbers struct {
	Accumulation  int
	ReservedCount int
}

// inventory.Stock の実データが以下どれでも動くように、reflection で吸収します。
// - stock[modelId] = struct{ Accumulation, ReservedCount, Products, ... }
// - stock[modelId] = *struct{ ... }
// - stock[modelId] = map[string]any など（accumulation/reservedCount を含む）
// - stock[modelId] = []string / map[string]bool など旧表現（accumulation=件数, reservedCount=0）
func extractStockNumbersByModel(m invdom.Mint) map[string]modelStockNumbers {
	out := map[string]modelStockNumbers{}

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

		nums := extractNumbersFromStockValue(v)
		out[modelID] = nums
	}

	return out
}

func extractNumbersFromStockValue(v reflect.Value) modelStockNumbers {
	v = derefValue(v)

	// 1) struct なら Accumulation/ReservedCount を読む（無ければ fallback）
	if v.IsValid() && v.Kind() == reflect.Struct {
		acc := readIntField(v, "Accumulation", "accumulation")
		rc := readIntField(v, "ReservedCount", "reservedCount")

		// struct に数値が無いケースは products から件数を推定
		if acc == 0 {
			if pf := v.FieldByName("Products"); pf.IsValid() {
				n := countIDsFromAny(pf)
				if n > 0 {
					acc = n
				}
			}
		}

		return modelStockNumbers{
			Accumulation:  acc,
			ReservedCount: rc,
		}
	}

	// 2) map ならキーから読む（accumulation/reservedCount）or products 件数
	if v.IsValid() && v.Kind() == reflect.Map && v.Type().Key().Kind() == reflect.String {
		acc := 0
		rc := 0

		// accumulation
		if vv := v.MapIndex(reflect.ValueOf("accumulation")); vv.IsValid() {
			acc = toInt(derefValue(vv))
		}
		// reservedCount
		if vv := v.MapIndex(reflect.ValueOf("reservedCount")); vv.IsValid() {
			rc = toInt(derefValue(vv))
		}

		// products fallback
		if acc == 0 {
			if vv := v.MapIndex(reflect.ValueOf("products")); vv.IsValid() {
				n := countIDsFromAny(vv)
				if n > 0 {
					acc = n
				}
			}
		}

		return modelStockNumbers{
			Accumulation:  acc,
			ReservedCount: rc,
		}
	}

	// 3) 旧表現: []string / map[string]bool を件数として扱う
	n := countIDsFromAny(v)
	return modelStockNumbers{
		Accumulation:  n,
		ReservedCount: 0,
	}
}

func derefValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func readIntField(v reflect.Value, names ...string) int {
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return 0
	}
	for _, name := range names {
		f := v.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		f = derefValue(f)
		if !f.IsValid() {
			continue
		}
		return toInt(f)
	}
	return 0
}

func countIDsFromAny(v reflect.Value) int {
	v = derefValue(v)
	if !v.IsValid() {
		return 0
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		n := 0
		for i := 0; i < v.Len(); i++ {
			e := derefValue(v.Index(i))
			if e.IsValid() && e.Kind() == reflect.String {
				if strings.TrimSpace(e.String()) != "" {
					n++
				}
			}
		}
		return n

	case reflect.Map:
		// map[string]bool / map[string]any は key を ID とみなす
		if v.Type().Key().Kind() != reflect.String {
			return 0
		}
		n := 0
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() != reflect.String {
				continue
			}
			if strings.TrimSpace(k.String()) != "" {
				n++
			}
		}
		return n

	default:
		return 0
	}
}

// ------------------------------
// Error mapping
// ------------------------------

func writeMallInvErr(w http.ResponseWriter, err error) {
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
