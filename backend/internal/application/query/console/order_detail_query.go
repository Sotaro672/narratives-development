// backend/internal/application/query/console/order_detail_query.go
package query

import (
	"context"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"
	orderdom "narratives/internal/domain/order"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// Ports
// ============================================================

type OrderDetailGetter interface {
	GetByID(ctx context.Context, id string) (orderdom.Order, error)
}

type OrderDetailInventoryBlueprintResolver interface {
	ResolveBlueprintIDsByInventoryID(
		ctx context.Context,
		inventoryID string,
	) (productBlueprintID string, tokenBlueprintID string, err error)
}

type OrderDetailProductBlueprintNameResolver interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

type OrderDetailTokenBlueprintNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

type OrderDetailAvatarNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

type OrderDetailUserNameResolver interface {
	GetNameByID(ctx context.Context, id string) (string, error)
}

type OrderDetailModelResolver interface {
	ResolveModelResolved(ctx context.Context, variationID string) resolver.ModelResolved
}

// ============================================================
// DTO
// ============================================================

type OrderDetailDTO struct {
	ID       string `json:"id"`
	UserID   string `json:"userId,omitempty"`
	AvatarID string `json:"avatarId,omitempty"`
	CartID   string `json:"cartId,omitempty"`

	UserName   string `json:"userName,omitempty"`
	AvatarName string `json:"avatarName,omitempty"`

	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt,omitempty"`

	ShippingSnapshot      any                  `json:"shippingSnapshot,omitempty"`
	PaymentMethodSnapshot any                  `json:"paymentMethodSnapshot,omitempty"`
	Items                 []OrderDetailItemDTO `json:"items,omitempty"`
}

type OrderDetailItemDTO struct {
	ModelID string `json:"modelId,omitempty"`

	InventoryID string `json:"inventoryId,omitempty"`

	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	ProductName string `json:"productName,omitempty"`
	TokenName   string `json:"tokenName,omitempty"`

	Size        string               `json:"size,omitempty"`
	ModelNumber string               `json:"modelNumber,omitempty"`
	Color       *OrderDetailColorDTO `json:"color,omitempty"`
	RGB         int                  `json:"rgb"`

	ListID string `json:"listId,omitempty"`
	Qty    int    `json:"qty,omitempty"`
	Price  int    `json:"price,omitempty"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"`
}

type OrderDetailColorDTO struct {
	Name string `json:"name,omitempty"`
	RGB  int    `json:"rgb"`
}

// ============================================================
// Query
// ============================================================

type OrderDetailQuery struct {
	orderGetter OrderDetailGetter

	invBlueprint OrderDetailInventoryBlueprintResolver
	pbName       OrderDetailProductBlueprintNameResolver
	tbName       OrderDetailTokenBlueprintNameResolver

	avatarName OrderDetailAvatarNameResolver
	userName   OrderDetailUserNameResolver

	modelResolver OrderDetailModelResolver
}

type NewOrderDetailQueryParams struct {
	OrderGetter OrderDetailGetter

	InvBlueprint OrderDetailInventoryBlueprintResolver
	PBName       OrderDetailProductBlueprintNameResolver
	TBName       OrderDetailTokenBlueprintNameResolver

	AvatarName OrderDetailAvatarNameResolver
	UserName   OrderDetailUserNameResolver

	ModelResolver OrderDetailModelResolver
}

func NewOrderDetailQuery(p NewOrderDetailQueryParams) *OrderDetailQuery {
	return &OrderDetailQuery{
		orderGetter:   p.OrderGetter,
		invBlueprint:  p.InvBlueprint,
		pbName:        p.PBName,
		tbName:        p.TBName,
		avatarName:    p.AvatarName,
		userName:      p.UserName,
		modelResolver: p.ModelResolver,
	}
}

func (q *OrderDetailQuery) GetByID(ctx context.Context, id string) (OrderDetailDTO, error) {
	if q == nil || q.orderGetter == nil {
		return OrderDetailDTO{}, orderdom.ErrInvalidID
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return OrderDetailDTO{}, orderdom.ErrInvalidID
	}

	o, err := q.orderGetter.GetByID(ctx, id)
	if err != nil {
		return OrderDetailDTO{}, err
	}

	return q.toDTO(ctx, o), nil
}

func (q *OrderDetailQuery) toDTO(ctx context.Context, o orderdom.Order) OrderDetailDTO {
	dto := OrderDetailDTO{
		ID:       o.ID,
		UserID:   o.UserID,
		AvatarID: o.AvatarID,
		CartID:   o.CartID,
		Paid:     o.Paid,

		ShippingSnapshot:      o.ShippingSnapshot,
		PaymentMethodSnapshot: o.PaymentMethodSnapshot,
	}

	if !o.CreatedAt.IsZero() {
		dto.CreatedAt = o.CreatedAt.UTC().Format(time.RFC3339)
	}

	if q.userName != nil && o.UserID != "" {
		if n, err := q.userName.GetNameByID(ctx, o.UserID); err == nil {
			dto.UserName = n
		}
	}

	if q.avatarName != nil && o.AvatarID != "" {
		if n, err := q.avatarName.GetNameByID(ctx, o.AvatarID); err == nil {
			dto.AvatarName = n
		}
	}

	pbNameCache := map[string]string{}
	tbNameCache := map[string]string{}
	modelCache := map[string]resolver.ModelResolved{}

	resolveProductName := func(id string) string {
		id = strings.TrimSpace(id)
		if id == "" || q.pbName == nil {
			return ""
		}
		if v, ok := pbNameCache[id]; ok {
			return v
		}

		pb, err := q.pbName.GetByID(ctx, id)
		if err != nil {
			return ""
		}

		name := pb.ProductName
		pbNameCache[id] = name
		return name
	}

	resolveTokenName := func(id string) string {
		id = strings.TrimSpace(id)
		if id == "" || q.tbName == nil {
			return ""
		}
		if v, ok := tbNameCache[id]; ok {
			return v
		}

		name, err := q.tbName.GetNameByID(ctx, id)
		if err != nil {
			return ""
		}

		tbNameCache[id] = name
		return name
	}

	resolveModel := func(variationID string) resolver.ModelResolved {
		variationID = strings.TrimSpace(variationID)
		if variationID == "" || q.modelResolver == nil {
			return resolver.ModelResolved{}
		}
		if v, ok := modelCache[variationID]; ok {
			return v
		}

		resolved := q.modelResolver.ResolveModelResolved(ctx, variationID)
		modelCache[variationID] = resolved
		return resolved
	}

	if len(o.Items) == 0 {
		dto.Items = []OrderDetailItemDTO{}
		return dto
	}

	dto.Items = make([]OrderDetailItemDTO, 0, len(o.Items))

	for _, it := range o.Items {
		invID := strings.TrimSpace(it.InventoryID)

		pbID := ""
		tbID := ""
		if q.invBlueprint != nil && invID != "" {
			pb, tb, err := q.invBlueprint.ResolveBlueprintIDsByInventoryID(ctx, invID)
			if err == nil {
				pbID = strings.TrimSpace(pb)
				tbID = strings.TrimSpace(tb)
			}
		}

		var (
			size        string
			modelNumber string
			colorDTO    *OrderDetailColorDTO
			rgb         int
		)

		mr := resolveModel(it.ModelID)
		if mr.ModelNumber != "" || mr.Size != "" || mr.Color != "" || mr.RGB != nil {
			size = mr.Size
			modelNumber = mr.ModelNumber

			if mr.Color != "" || mr.RGB != nil {
				colorDTO = &OrderDetailColorDTO{
					Name: mr.Color,
					RGB:  0,
				}
				if mr.RGB != nil {
					colorDTO.RGB = *mr.RGB
					rgb = *mr.RGB
				}
			} else if mr.RGB != nil {
				rgb = *mr.RGB
			}
		}

		item := OrderDetailItemDTO{
			ModelID: it.ModelID,

			InventoryID: invID,

			ProductBlueprintID: pbID,
			TokenBlueprintID:   tbID,

			ProductName: resolveProductName(pbID),
			TokenName:   resolveTokenName(tbID),

			Size:        size,
			ModelNumber: modelNumber,
			Color:       colorDTO,
			RGB:         rgb,

			ListID: it.ListID,
			Qty:    it.Qty,
			Price:  it.Price,

			Transferred: it.Transferred,
		}

		if it.TransferredAt != nil && !it.TransferredAt.IsZero() {
			item.TransferredAt = it.TransferredAt.UTC().Format(time.RFC3339)
		}

		dto.Items = append(dto.Items, item)
	}

	return dto
}
