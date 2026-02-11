// backend/internal/adapters/in/http/console/handler/order_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	orderq "narratives/internal/application/query/console"
	resolver "narratives/internal/application/resolver"
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

// ProductBlueprintNameResolver resolves productName from productBlueprintId.
type ProductBlueprintNameResolver interface {
	GetProductNameByID(ctx context.Context, id string) (string, error)
}

// TokenBlueprintNameResolver resolves tokenName from tokenBlueprintId.
type TokenBlueprintNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

// AvatarNameResolver resolves avatarName from avatarId.
type AvatarNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

// UserNameResolver resolves userName from userId.
// - userName は "lastName firstName" の順で返す想定
type UserNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

// ✅ ModelResolver: application/resolver の NameResolver を使って modelId(=variationID) を解決する
type ModelResolver interface {
	ResolveModelResolved(ctx context.Context, variationID string) resolver.ModelResolved
}

// ============================================================
// Response DTO (camelCase JSON)
// ============================================================

type orderResponseDTO struct {
	ID       string `json:"id"`
	UserID   string `json:"userId,omitempty"`
	AvatarID string `json:"avatarId,omitempty"`
	CartID   string `json:"cartId,omitempty"`

	// ✅ userId -> userName（UI表示用）
	UserName string `json:"userName,omitempty"`

	// ✅ avatarId -> avatarName（UI表示用）
	AvatarName string `json:"avatarName,omitempty"`

	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt,omitempty"` // RFC3339(UTC)

	ShippingSnapshot any            `json:"shippingSnapshot,omitempty"`
	BillingSnapshot  any            `json:"billingSnapshot,omitempty"`
	Items            []orderItemDTO `json:"items,omitempty"`
}

type orderItemDTO struct {
	ModelID string `json:"modelId,omitempty"`

	// backward-compat & internal use
	InventoryID string `json:"inventoryId,omitempty"`

	// resolved from inventoryId
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	// names for UI
	ProductName string `json:"productName,omitempty"`
	TokenName   string `json:"tokenName,omitempty"`

	// ✅ resolved model fields (from modelId -> variation)
	Size        string    `json:"size,omitempty"`
	ModelNumber string    `json:"modelNumber,omitempty"`
	Color       *ColorDTO `json:"color,omitempty"`
	// ✅ 0(黒) も有効値なので omitempty を外す
	RGB int `json:"rgb"`

	ListID string `json:"listId,omitempty"`
	Qty    int    `json:"qty,omitempty"`
	Price  int    `json:"price,omitempty"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"` // RFC3339(UTC)
}

type ColorDTO struct {
	Name string `json:"name,omitempty"`
	// ✅ 0(黒) も有効値なので omitempty を外す
	RGB int `json:"rgb"`
}

func toOrderResponseDTO(
	ctx context.Context,
	o orderdom.Order,
	invBlueprint InventoryBlueprintResolver,
	pbName ProductBlueprintNameResolver,
	tbName TokenBlueprintNameResolver,
	avatarName AvatarNameResolver,
	userName UserNameResolver,
	modelResolver ModelResolver, // ✅ NameResolver 互換の interface に変更
) orderResponseDTO {
	dto := orderResponseDTO{
		ID:     strings.TrimSpace(o.ID),
		UserID: strings.TrimSpace(o.UserID),

		AvatarID: strings.TrimSpace(o.AvatarID),
		CartID:   strings.TrimSpace(o.CartID),

		Paid: o.Paid,

		ShippingSnapshot: o.ShippingSnapshot,
		BillingSnapshot:  o.BillingSnapshot,
	}

	if !o.CreatedAt.IsZero() {
		dto.CreatedAt = o.CreatedAt.UTC().Format(time.RFC3339)
	}

	// userName (best-effort)
	if userName != nil && strings.TrimSpace(o.UserID) != "" {
		if n, err := userName.GetNameByID(ctx, strings.TrimSpace(o.UserID)); err == nil {
			dto.UserName = strings.TrimSpace(n)
		}
	}

	// avatarName (best-effort)
	if avatarName != nil && strings.TrimSpace(o.AvatarID) != "" {
		if n, err := avatarName.GetNameByID(ctx, strings.TrimSpace(o.AvatarID)); err == nil {
			dto.AvatarName = strings.TrimSpace(n)
		}
	}

	// caches (best-effort)
	pbNameCache := map[string]string{}
	tbNameCache := map[string]string{}
	modelCache := map[string]resolver.ModelResolved{} // ✅ ModelResolved をキャッシュ

	resolveProductName := func(id string) string {
		id = strings.TrimSpace(id)
		if id == "" || pbName == nil {
			return ""
		}
		if v, ok := pbNameCache[id]; ok {
			return v
		}
		name, err := pbName.GetProductNameByID(ctx, id)
		if err != nil {
			return ""
		}
		name = strings.TrimSpace(name)
		pbNameCache[id] = name
		return name
	}

	resolveTokenName := func(id string) string {
		id = strings.TrimSpace(id)
		if id == "" || tbName == nil {
			return ""
		}
		if v, ok := tbNameCache[id]; ok {
			return v
		}
		name, err := tbName.GetNameByID(ctx, id)
		if err != nil {
			return ""
		}
		name = strings.TrimSpace(name)
		tbNameCache[id] = name
		return name
	}

	// modelId -> ModelResolved (application/resolver)
	resolveModel := func(variationID string) resolver.ModelResolved {
		vid := strings.TrimSpace(variationID)
		if vid == "" || modelResolver == nil {
			return resolver.ModelResolved{}
		}
		if v, ok := modelCache[vid]; ok {
			return v
		}
		resolved := modelResolver.ResolveModelResolved(ctx, vid) // 取れない場合はゼロ値
		modelCache[vid] = resolved
		return resolved
	}

	if len(o.Items) > 0 {
		dto.Items = make([]orderItemDTO, 0, len(o.Items))
		for _, it := range o.Items {
			invID := strings.TrimSpace(it.InventoryID)

			pbID := ""
			tbID := ""
			if invBlueprint != nil && invID != "" {
				pb, tb, err := invBlueprint.ResolveBlueprintIDsByInventoryID(ctx, invID)
				if err == nil {
					pbID = strings.TrimSpace(pb)
					tbID = strings.TrimSpace(tb)
				}
			}

			// ✅ model resolved fields
			var (
				size        string
				modelNumber string
				colorDTO    *ColorDTO
				rgb         int
			)

			mr := resolveModel(strings.TrimSpace(it.ModelID))
			if mr.ModelNumber != "" || mr.Size != "" || mr.Color != "" || mr.RGB != nil {
				size = strings.TrimSpace(mr.Size)
				modelNumber = strings.TrimSpace(mr.ModelNumber)

				// color name は string で返ってくるので DTO を組み立てる
				if strings.TrimSpace(mr.Color) != "" || mr.RGB != nil {
					colorDTO = &ColorDTO{
						Name: strings.TrimSpace(mr.Color),
						RGB:  0,
					}
					if mr.RGB != nil {
						colorDTO.RGB = *mr.RGB
						rgb = *mr.RGB
					}
				} else {
					// rgb だけ取れたケース（Name が空でも rgb は返す）
					if mr.RGB != nil {
						rgb = *mr.RGB
					}
				}
			}

			item := orderItemDTO{
				ModelID: strings.TrimSpace(it.ModelID),

				InventoryID: invID,

				ProductBlueprintID: pbID,
				TokenBlueprintID:   tbID,

				ProductName: resolveProductName(pbID),
				TokenName:   resolveTokenName(tbID),

				Size:        size,
				ModelNumber: modelNumber,
				Color:       colorDTO,
				RGB:         rgb,

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
		dto.Items = []orderItemDTO{}
	}

	return dto
}

// ============================================================
// Handler
// ============================================================

type OrderHandler struct {
	uc *usecase.OrderUsecase
	q  *orderq.OrderManagementQuery

	invBlueprint InventoryBlueprintResolver
	pbName       ProductBlueprintNameResolver
	tbName       TokenBlueprintNameResolver

	avatarName AvatarNameResolver
	userName   UserNameResolver

	// ✅ application/resolver.NameResolver を使う
	modelResolver ModelResolver
}

func NewOrderHandler(
	uc *usecase.OrderUsecase,
	q *orderq.OrderManagementQuery,
	invBlueprint InventoryBlueprintResolver,
	pbName ProductBlueprintNameResolver,
	tbName TokenBlueprintNameResolver,
	avatarName AvatarNameResolver,
	userName UserNameResolver,
	modelResolver ModelResolver, // ✅ ModelVariationResolver から差し替え
) http.Handler {
	return &OrderHandler{
		uc:            uc,
		q:             q,
		invBlueprint:  invBlueprint,
		pbName:        pbName,
		tbName:        tbName,
		avatarName:    avatarName,
		userName:      userName,
		modelResolver: modelResolver,
	}
}

func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/orders/items":
		h.listItemRows(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/orders/inventory-ids":
		h.listDistinctInventoryIDs(w, r)
		return

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

	dto := toOrderResponseDTO(
		ctx,
		order,
		h.invBlueprint,
		h.pbName,
		h.tbName,
		h.avatarName,
		h.userName,
		h.modelResolver,
	)
	_ = json.NewEncoder(w).Encode(dto)
}

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

	var sort common.Sort

	pr, err := h.q.ListItemInventoryRows(ctx, filter, sort, page)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(pr)
}

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

func parseOrderListParams(r *http.Request) (usecase.OrderFilter, common.Page, error) {
	q := r.URL.Query()

	pageNum := parseIntDefault(q.Get("page"), 1)
	perPage := parseIntDefault(q.Get("perPage"), 20)

	f := usecase.OrderFilter{
		ID: strings.TrimSpace(q.Get("id")),
	}

	if v := strings.TrimSpace(q.Get("userId")); v != "" {
		f.UserID = strPtr(v)
	}
	if v := strings.TrimSpace(q.Get("avatarId")); v != "" {
		f.AvatarID = strPtr(v)
	}
	if v := strings.TrimSpace(q.Get("cartId")); v != "" {
		f.CartID = strPtr(v)
	}

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
// Error handling
// ============================================================

func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err == orderdom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
