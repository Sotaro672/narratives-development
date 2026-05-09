// backend/internal/adapters/in/http/mall/handler/inventory_handler.go
package mallHandler

import (
	"errors"
	"net/http"
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

// products map は廃止し、accumulation / reservedCount を返す
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

	path := strings.TrimSuffix(r.URL.Path, "/")

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
		id := parts[0]
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
	pb := q.Get("productBlueprintId")
	tb := q.Get("tokenBlueprintId")

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

	m, err := h.uc.GetByID(ctx, id)
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

// products は返さない。
// accumulation / reservedCount を返す。
// inventory.Stock は domain の正規型を直接読む。
// Firestore の旧表現や int64 -> int 変換は repository mapper 側で吸収する。
func toMallInventoryResponse(m invdom.Mint) MallInventoryResponse {
	stock := make(map[string]MallInventoryModelStock, len(m.Stock))
	modelIDs := make([]string, 0, len(m.Stock))

	for modelID, ms := range m.Stock {
		if modelID == "" {
			continue
		}

		modelIDs = append(modelIDs, modelID)

		acc := ms.Accumulation
		rc := ms.ReservedCount

		// 最低限の整合: reservedCount が負になるケースは 0 に丸める
		if rc < 0 {
			rc = 0
		}

		// accumulation が負になるケースは 0 に丸める
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
		ID:                 m.ID,
		TokenBlueprintID:   m.TokenBlueprintID,
		ProductBlueprintID: m.ProductBlueprintID,
		ModelIDs:           append([]string{}, finalModelIDs...),
		Stock:              stock,
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
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeJSON(w, code, map[string]string{"error": err.Error()})
}
