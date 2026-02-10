// backend/internal/adapters/in/http/console/handler/order_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	orderq "narratives/internal/application/query/console" // order_management_query.go は package query
	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// ============================================================
// Ports (for /orders/{id} enrichment)
// ============================================================

// InventoryBlueprintResolver resolves productBlueprintId/tokenBlueprintId from inventoryId.
type InventoryBlueprintResolver interface {
	ResolveBlueprintIDsByInventoryID(ctx context.Context, inventoryID string) (productBlueprintID string, tokenBlueprintID string, err error)
}

// ============================================================
// Response DTO (camelCase JSON)
// ============================================================

type orderResponseDTO struct {
	ID       string `json:"id"`
	UserID   string `json:"userId,omitempty"`
	AvatarID string `json:"avatarId,omitempty"`
	CartID   string `json:"cartId,omitempty"`

	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt,omitempty"` // RFC3339(UTC)

	ShippingSnapshot any            `json:"shippingSnapshot,omitempty"`
	BillingSnapshot  any            `json:"billingSnapshot,omitempty"`
	Items            []orderItemDTO `json:"items,omitempty"`
}

type orderItemDTO struct {
	ModelID string `json:"modelId,omitempty"`

	// ✅ Keep inventoryId for backward-compat & internal use (but UI can ignore it)
	InventoryID string `json:"inventoryId,omitempty"`

	// ✅ NEW: resolve from inventoryId and return to UI
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ListID string `json:"listId,omitempty"`
	Qty    int    `json:"qty,omitempty"`
	Price  int    `json:"price,omitempty"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"` // RFC3339(UTC)
}

func toOrderResponseDTO(ctx context.Context, o orderdom.Order, invBlueprint InventoryBlueprintResolver) orderResponseDTO {
	dto := orderResponseDTO{
		ID:     strings.TrimSpace(o.ID),
		UserID: strings.TrimSpace(o.UserID),

		AvatarID: strings.TrimSpace(o.AvatarID),
		CartID:   strings.TrimSpace(o.CartID),

		Paid: o.Paid,

		// domainの型をそのまま返す（json tag が無い/揺れていても、DTO側のキーに寄せる）
		ShippingSnapshot: o.ShippingSnapshot,
		BillingSnapshot:  o.BillingSnapshot,
	}

	if !o.CreatedAt.IsZero() {
		dto.CreatedAt = o.CreatedAt.UTC().Format(time.RFC3339)
	}

	if len(o.Items) > 0 {
		dto.Items = make([]orderItemDTO, 0, len(o.Items))
		for _, it := range o.Items {
			invID := strings.TrimSpace(it.InventoryID)

			pbID := ""
			tbID := ""
			if invBlueprint != nil && invID != "" {
				// best-effort: if resolve fails, keep empty fields (UI shows "-")
				pb, tb, err := invBlueprint.ResolveBlueprintIDsByInventoryID(ctx, invID)
				if err == nil {
					pbID = strings.TrimSpace(pb)
					tbID = strings.TrimSpace(tb)
				}
			}

			item := orderItemDTO{
				ModelID: strings.TrimSpace(it.ModelID),

				InventoryID: invID,

				ProductBlueprintID: pbID,
				TokenBlueprintID:   tbID,

				ListID: strings.TrimSpace(it.ListID),
				Qty:    it.Qty,
				Price:  it.Price,

				Transferred: it.Transferred,
			}
			if it.TransferredAt != nil && !it.TransferredAt.IsZero() {
				item.TransferredAt = it.TransferredAt.UTC().Format(time.RFC3339)
			}
			dto.Items = append(dto.Items, item)
		}
	} else {
		// 常に配列で返したい場合は空sliceにする（UI側が楽）
		dto.Items = []orderItemDTO{}
	}

	return dto
}

// OrderHandler は /orders 関連のエンドポイントを担当します。
//
// 追加機能:
// - GET /orders/items         : OrderManagementQuery の items(flat) 結果を返す
// - GET /orders/inventory-ids : items から distinct inventoryId を返す
// - GET /orders/{id}          : 単一取得（既存）
type OrderHandler struct {
	uc *usecase.OrderUsecase
	q  *orderq.OrderManagementQuery

	// ✅ NEW: for enriching /orders/{id} items with productBlueprintId/tokenBlueprintId
	invBlueprint InventoryBlueprintResolver
}

// NewOrderHandler はHTTPハンドラを初期化します。
// 既存の uc に加えて、OrderManagementQuery を DI して使えるようにします。
// さらに /orders/{id} 用に inventoryId -> (pbId,tbId) 解決ポートも注入します（nil可）。
func NewOrderHandler(
	uc *usecase.OrderUsecase,
	q *orderq.OrderManagementQuery,
	invBlueprint InventoryBlueprintResolver,
) http.Handler {
	return &OrderHandler{uc: uc, q: q, invBlueprint: invBlueprint}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// ✅ 一覧（itemsをフラット化）
	case r.Method == http.MethodGet && r.URL.Path == "/orders/items":
		h.listItemRows(w, r)
		return

	// ✅ distinct inventoryId 一覧
	case r.Method == http.MethodGet && r.URL.Path == "/orders/inventory-ids":
		h.listDistinctInventoryIDs(w, r)
		return

	// ✅ 単一取得
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/orders/"):
		id := strings.TrimPrefix(r.URL.Path, "/orders/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// GET /orders/{id}
func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	order, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	// ✅ domain をそのまま返さず、camelCase の DTO に詰め替えて返す
	// ✅ /orders/{id} でも productBlueprintId/tokenBlueprintId を返す（invBlueprint があれば）
	dto := toOrderResponseDTO(ctx, order, h.invBlueprint)
	_ = json.NewEncoder(w).Encode(dto)
}

// GET /orders/items?page=1&perPage=20&id=...&userId=...&avatarId=...&cartId=...&createdFrom=...&createdTo=...
func (h *OrderHandler) listItemRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.q == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_management_query_not_wired"})
		return
	}

	filter, page, err := parseOrderListParams(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// sort は repo 実装に依存しやすいので、まずはゼロ値で渡す（必要なら後で拡張）
	var sort common.Sort

	// ✅ OrderManagementQuery 側で productBlueprintId/tokenBlueprintId を解決して返す前提
	pr, err := h.q.ListItemInventoryRows(ctx, filter, sort, page)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// pr.Items[] の各行に productBlueprintId/tokenBlueprintId が含まれる
	_ = json.NewEncoder(w).Encode(pr)
}

// GET /orders/inventory-ids?page=1&perPage=200&userId=... （filter/page は /orders/items と同じ）
func (h *OrderHandler) listDistinctInventoryIDs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.q == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_management_query_not_wired"})
		return
	}

	filter, page, err := parseOrderListParams(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var sort common.Sort

	pr, err := h.q.ListDistinctInventoryIDs(ctx, filter, sort, page)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(pr)
}

// ============================================================
// Query param parsing
// ============================================================

// ✅ OrderManagementQuery は usecase.OrderFilter / common.Page を使う
func parseOrderListParams(r *http.Request) (usecase.OrderFilter, common.Page, error) {
	q := r.URL.Query()

	// page/perPage
	pageNum := parseIntDefault(q.Get("page"), 1)
	perPage := parseIntDefault(q.Get("perPage"), 20)

	// filter（usecase.OrderFilter）
	f := usecase.OrderFilter{
		ID: strings.TrimSpace(q.Get("id")),
	}

	// *string fields: "" -> nil, otherwise &v
	if v := strings.TrimSpace(q.Get("userId")); v != "" {
		f.UserID = strPtr(v)
	}
	// ✅ usecase.OrderFilter に AvatarID *string を追加した前提
	if v := strings.TrimSpace(q.Get("avatarId")); v != "" {
		f.AvatarID = strPtr(v)
	}
	if v := strings.TrimSpace(q.Get("cartId")); v != "" {
		f.CartID = strPtr(v)
	}

	// createdFrom/createdTo: RFC3339
	if v := strings.TrimSpace(q.Get("createdFrom")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return usecase.OrderFilter{}, common.Page{}, errors.New("invalid createdFrom (expected RFC3339)")
		}
		f.CreatedFrom = &t
	}
	if v := strings.TrimSpace(q.Get("createdTo")); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return usecase.OrderFilter{}, common.Page{}, errors.New("invalid createdTo (expected RFC3339)")
		}
		f.CreatedTo = &t
	}

	p := common.Page{
		Number:  pageNum,
		PerPage: perPage,
	}
	return f, p, nil
}

func strPtr(s string) *string { return &s }

// ============================================================
// エラーハンドリング（既存）
// ============================================================

func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err == orderdom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
