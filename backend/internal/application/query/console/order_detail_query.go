// backend/internal/application/query/console/order_detail_query.go
package query

import (
	"context"
	"errors"
	"fmt"
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
	ResolveBlueprintIDsByInventoryID(ctx context.Context, inventoryID string) (productBlueprintID string, tokenBlueprintID string, err error)
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
	ResolveUserName(ctx context.Context, userID string) string
}

type OrderDetailModelResolver interface {
	ResolveModelResolved(ctx context.Context, variationID string) resolver.ModelResolved
}

type OrderDetailListReadableIDResolver interface {
	GetReadableIDByID(ctx context.Context, id string) (string, error)
}

// ============================================================
// DTO
// ============================================================

type OrderDetailDTO struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	UserName   string `json:"userName"`
	AvatarName string `json:"avatarName"`

	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt"`

	ShippingSnapshot      orderdom.ShippingSnapshot      `json:"shippingSnapshot"`
	PaymentMethodSnapshot orderdom.PaymentMethodSnapshot `json:"paymentMethodSnapshot"`
	Items                 []OrderDetailItemDTO           `json:"items"`
}

type OrderDetailItemDTO struct {
	Type orderdom.OrderItemType `json:"type"`

	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`
	ResaleID    string `json:"resaleId,omitempty"`

	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
	BrandID            string `json:"brandId,omitempty"`

	ProductName string `json:"productName"`
	TokenName   string `json:"tokenName"`

	ListReadableID string `json:"listReadableId,omitempty"`

	CategoryID     string         `json:"categoryId"`
	CategoryCode   string         `json:"categoryCode"`
	CategoryNameJa string         `json:"categoryNameJa"`
	CategoryNameEn string         `json:"categoryNameEn"`
	CategoryKind   string         `json:"categoryKind"`
	CategoryPath   []string       `json:"categoryPath"`
	CategoryFields map[string]any `json:"categoryFields"`

	Kind        string `json:"kind"`
	ModelNumber string `json:"modelNumber"`
	Size        string `json:"size"`
	Color       string `json:"color"`
	RGB         *int   `json:"rgb,omitempty"`

	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit"`

	Qty   int `json:"qty"`
	Price int `json:"price"`

	IsCanceled   bool `json:"isCanceled"`
	IsDispatched bool `json:"isDispatched"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"`
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
	listReadable  OrderDetailListReadableIDResolver
}

type NewOrderDetailQueryParams struct {
	OrderGetter OrderDetailGetter

	InvBlueprint OrderDetailInventoryBlueprintResolver
	PBName       OrderDetailProductBlueprintNameResolver
	TBName       OrderDetailTokenBlueprintNameResolver

	AvatarName OrderDetailAvatarNameResolver
	UserName   OrderDetailUserNameResolver

	ModelResolver OrderDetailModelResolver
	ListReadable  OrderDetailListReadableIDResolver
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
		listReadable:  p.ListReadable,
	}
}

func (q *OrderDetailQuery) GetByID(ctx context.Context, id string) (OrderDetailDTO, error) {
	if err := q.validateConfigured(); err != nil {
		return OrderDetailDTO{}, err
	}
	if id == "" {
		return OrderDetailDTO{}, orderdom.ErrInvalidID
	}

	o, err := q.orderGetter.GetByID(ctx, id)
	if err != nil {
		return OrderDetailDTO{}, err
	}

	return q.toDTO(ctx, o)
}

func (q *OrderDetailQuery) validateConfigured() error {
	if q == nil {
		return errors.New("OrderDetailQuery: query is nil")
	}
	if q.orderGetter == nil {
		return errors.New("OrderDetailQuery: orderGetter is required")
	}
	if q.invBlueprint == nil {
		return errors.New("OrderDetailQuery: invBlueprint is required")
	}
	if q.pbName == nil {
		return errors.New("OrderDetailQuery: productBlueprint resolver is required")
	}
	if q.tbName == nil {
		return errors.New("OrderDetailQuery: tokenBlueprint resolver is required")
	}
	if q.avatarName == nil {
		return errors.New("OrderDetailQuery: avatarName resolver is required")
	}
	if q.userName == nil {
		return errors.New("OrderDetailQuery: userName resolver is required")
	}
	if q.modelResolver == nil {
		return errors.New("OrderDetailQuery: modelResolver is required")
	}
	if q.listReadable == nil {
		return errors.New("OrderDetailQuery: listReadable resolver is required")
	}
	return nil
}

func (q *OrderDetailQuery) toDTO(ctx context.Context, o orderdom.Order) (OrderDetailDTO, error) {
	dto := OrderDetailDTO{
		ID:                    o.ID,
		UserID:                o.UserID,
		AvatarID:              o.AvatarID,
		CartID:                o.CartID,
		UserName:              q.userName.ResolveUserName(ctx, o.UserID),
		Paid:                  o.Paid,
		ShippingSnapshot:      o.ShippingSnapshot,
		PaymentMethodSnapshot: o.PaymentMethodSnapshot,
		Items:                 make([]OrderDetailItemDTO, 0, len(o.Items)),
	}

	if !o.CreatedAt.IsZero() {
		dto.CreatedAt = o.CreatedAt.UTC().Format(time.RFC3339)
	}

	if o.AvatarID != "" {
		avatarName, err := q.avatarName.GetNameByID(ctx, o.AvatarID)
		if err != nil {
			return OrderDetailDTO{}, fmt.Errorf("resolve avatarName avatarId=%q: %w", o.AvatarID, err)
		}
		dto.AvatarName = avatarName
	}

	productBlueprintCache := make(map[string]pbdom.ProductBlueprint)
	tokenNameCache := make(map[string]string)
	modelCache := make(map[string]resolver.ModelResolved)
	listReadableCache := make(map[string]string)

	resolveProductBlueprint := func(id string) (pbdom.ProductBlueprint, error) {
		if id == "" {
			return pbdom.ProductBlueprint{}, nil
		}
		if cached, ok := productBlueprintCache[id]; ok {
			return cached, nil
		}

		pb, err := q.pbName.GetByID(ctx, id)
		if err != nil {
			return pbdom.ProductBlueprint{}, err
		}

		productBlueprintCache[id] = pb
		return pb, nil
	}

	resolveTokenName := func(id string) (string, error) {
		if id == "" {
			return "", nil
		}
		if cached, ok := tokenNameCache[id]; ok {
			return cached, nil
		}

		name, err := q.tbName.GetNameByID(ctx, id)
		if err != nil {
			return "", err
		}

		tokenNameCache[id] = name
		return name, nil
	}

	resolveModel := func(modelID string) resolver.ModelResolved {
		if modelID == "" {
			return resolver.ModelResolved{}
		}
		if cached, ok := modelCache[modelID]; ok {
			return cached
		}

		resolved := q.modelResolver.ResolveModelResolved(ctx, modelID)
		modelCache[modelID] = resolved
		return resolved
	}

	resolveListReadableID := func(listID string) (string, error) {
		if listID == "" {
			return "", nil
		}
		if cached, ok := listReadableCache[listID]; ok {
			return cached, nil
		}

		readableID, err := q.listReadable.GetReadableIDByID(ctx, listID)
		if err != nil {
			return "", err
		}

		listReadableCache[listID] = readableID
		return readableID, nil
	}

	for _, it := range o.Items {
		pbID := it.ProductBlueprintID
		tbID := it.TokenBlueprintID

		if it.InventoryID != "" && (pbID == "" || tbID == "") {
			resolvedPBID, resolvedTBID, err := q.invBlueprint.ResolveBlueprintIDsByInventoryID(ctx, it.InventoryID)
			if err != nil {
				return OrderDetailDTO{}, fmt.Errorf("resolve blueprint ids inventoryId=%q: %w", it.InventoryID, err)
			}
			if pbID == "" {
				pbID = resolvedPBID
			}
			if tbID == "" {
				tbID = resolvedTBID
			}
		}

		pb, err := resolveProductBlueprint(pbID)
		if err != nil {
			return OrderDetailDTO{}, fmt.Errorf("resolve productBlueprint productBlueprintId=%q: %w", pbID, err)
		}

		tokenName, err := resolveTokenName(tbID)
		if err != nil {
			return OrderDetailDTO{}, fmt.Errorf("resolve tokenName tokenBlueprintId=%q: %w", tbID, err)
		}

		listReadableID, err := resolveListReadableID(it.ListID)
		if err != nil {
			return OrderDetailDTO{}, fmt.Errorf("resolve listReadableId listId=%q: %w", it.ListID, err)
		}

		categoryPath := make([]string, 0, len(pb.ProductBlueprintCategory.Path))
		categoryPath = append(categoryPath, pb.ProductBlueprintCategory.Path...)

		categoryFields := make(map[string]any, len(pb.CategoryFields))
		for key, value := range pb.CategoryFields {
			categoryFields[key] = value
		}

		model := resolveModel(it.ModelID)

		item := OrderDetailItemDTO{
			Type: it.Type,

			ModelID:     it.ModelID,
			InventoryID: it.InventoryID,
			ListID:      it.ListID,
			ResaleID:    it.ResaleID,

			ProductID:          it.ProductID,
			ProductBlueprintID: pbID,
			TokenBlueprintID:   tbID,
			BrandID:            it.BrandID,

			ProductName: pb.ProductName,
			TokenName:   tokenName,

			ListReadableID: listReadableID,

			CategoryID:     pb.ProductBlueprintCategory.ID,
			CategoryCode:   pb.ProductBlueprintCategory.Code,
			CategoryNameJa: pb.ProductBlueprintCategory.NameJa,
			CategoryNameEn: pb.ProductBlueprintCategory.NameEn,
			CategoryKind:   string(pb.ProductBlueprintCategory.Kind),
			CategoryPath:   categoryPath,
			CategoryFields: categoryFields,

			Kind:        model.Kind,
			ModelNumber: model.ModelNumber,
			Size:        model.Size,
			Color:       model.Color,
			RGB:         model.RGB,

			VolumeValue: model.VolumeValue,
			VolumeUnit:  model.VolumeUnit,

			Qty:   it.Qty,
			Price: it.Price,

			IsCanceled:   it.IsCanceled,
			IsDispatched: it.IsDispatched,

			Transferred: it.Transferred,
		}

		if it.TransferredAt != nil && !it.TransferredAt.IsZero() {
			item.TransferredAt = it.TransferredAt.UTC().Format(time.RFC3339)
		}

		dto.Items = append(dto.Items, item)
	}

	return dto, nil
}
