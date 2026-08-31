// backend/internal/application/query/console/order_detail_query.go
package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	applicationport "narratives/internal/application/port"
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

type OrderDetailUserNameResolver interface {
	ResolveUserName(ctx context.Context, userID string) string
}

// ============================================================
// DTO
// ============================================================

type OrderDetailDTO struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	UserName string `json:"userName"`
	Email    string `json:"email"`

	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt"`

	ShippingAmount int `json:"shippingAmount"`
	ConsumptionTax int `json:"consumptionTax"`

	ShippingSnapshot orderdom.ShippingSnapshot `json:"shippingSnapshot"`
	Items            []OrderDetailItemDTO      `json:"items"`
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

	ProductBlueprintCategoryPath []string       `json:"productBlueprintCategoryPath"`
	CategoryFields               map[string]any `json:"categoryFields"`

	Kind        string `json:"kind"`
	ModelNumber string `json:"modelNumber"`
	Size        string `json:"size"`
	Color       string `json:"color"`
	RGB         *int   `json:"rgb,omitempty"`

	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit"`

	Qty   int `json:"qty"`
	Price int `json:"price"`

	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"`
}

// ============================================================
// Query
// ============================================================

type OrderDetailQuery struct {
	orderGetter OrderDetailGetter
	invRows     InventoryRowsLister

	invBlueprint InventoryBlueprintResolver
	pbName       applicationport.ProductBlueprintGetter
	tbName       applicationport.TokenBlueprintGetter

	userName OrderDetailUserNameResolver
	authUser applicationport.AuthUserReader

	modelResolver ModelResolver
	listReadable  ListReadableIDReader
}

type NewOrderDetailQueryParams struct {
	OrderGetter OrderDetailGetter
	InvRows     InventoryRowsLister

	InvBlueprint InventoryBlueprintResolver
	PBName       applicationport.ProductBlueprintGetter
	TBName       applicationport.TokenBlueprintGetter

	UserName OrderDetailUserNameResolver
	AuthUser applicationport.AuthUserReader

	ModelResolver ModelResolver
	ListReadable  ListReadableIDReader
}

func NewOrderDetailQuery(p NewOrderDetailQueryParams) *OrderDetailQuery {
	return &OrderDetailQuery{
		orderGetter:   p.OrderGetter,
		invRows:       p.InvRows,
		invBlueprint:  p.InvBlueprint,
		pbName:        p.PBName,
		tbName:        p.TBName,
		userName:      p.UserName,
		authUser:      p.AuthUser,
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
	if q.invRows == nil {
		return errors.New("OrderDetailQuery: invRows is required")
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
	if q.userName == nil {
		return errors.New("OrderDetailQuery: userName resolver is required")
	}
	if q.authUser == nil {
		return errors.New("OrderDetailQuery: authUser reader is required")
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
	allowedInventoryIDs, err := AllowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		return OrderDetailDTO{}, err
	}

	visibleItems := make([]orderdom.OrderItemSnapshot, 0, len(o.Items))
	for _, item := range o.Items {
		if item.Type != orderdom.OrderItemTypeList {
			continue
		}
		if !InventoryAllowed(allowedInventoryIDs, item.InventoryID) {
			continue
		}
		visibleItems = append(visibleItems, item)
	}

	if len(visibleItems) == 0 {
		return OrderDetailDTO{}, orderdom.ErrNotFound
	}

	shippingAmount, err := calculateOrderDetailShippingAmount(
		o.ShippingQuoteSnapshot.Items,
		allowedInventoryIDs,
	)
	if err != nil {
		return OrderDetailDTO{}, err
	}

	consumptionTax, err := calculateOrderDetailConsumptionTax(
		visibleItems,
		shippingAmount,
	)
	if err != nil {
		return OrderDetailDTO{}, err
	}

	email, err := q.authUser.GetEmailByUID(ctx, o.UserID)
	if err != nil {
		return OrderDetailDTO{}, fmt.Errorf(
			"resolve user email userId=%q: %w",
			o.UserID,
			err,
		)
	}

	dto := OrderDetailDTO{
		ID:               o.ID,
		UserID:           o.UserID,
		AvatarID:         o.AvatarID,
		CartID:           o.CartID,
		UserName:         q.userName.ResolveUserName(ctx, o.UserID),
		Email:            email,
		Paid:             o.Paid,
		ShippingAmount:   shippingAmount,
		ConsumptionTax:   consumptionTax,
		ShippingSnapshot: o.ShippingSnapshot,
		Items:            make([]OrderDetailItemDTO, 0, len(visibleItems)),
	}

	if !o.CreatedAt.IsZero() {
		dto.CreatedAt = o.CreatedAt.UTC().Format(time.RFC3339)
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

		tb, err := q.tbName.GetByID(ctx, id)
		if err != nil {
			return "", err
		}
		if tb == nil {
			return "", fmt.Errorf("tokenBlueprint is nil: tokenBlueprintId=%q", id)
		}

		tokenNameCache[id] = tb.Name
		return tb.Name, nil
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

	for _, it := range visibleItems {
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

		productBlueprintCategoryPath := make(
			[]string,
			0,
			len(pb.ProductBlueprintCategoryPath),
		)
		productBlueprintCategoryPath = append(
			productBlueprintCategoryPath,
			pb.ProductBlueprintCategoryPath...,
		)

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

			ProductBlueprintCategoryPath: productBlueprintCategoryPath,
			CategoryFields:               categoryFields,

			Kind:        model.Kind,
			ModelNumber: model.ModelNumber,
			Size:        model.Size,
			Color:       model.Color,
			RGB:         model.RGB,

			VolumeValue: model.VolumeValue,
			VolumeUnit:  model.VolumeUnit,

			Qty:   it.Qty,
			Price: it.Price,

			IsCancelled:  it.IsCancelled,
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

func calculateOrderDetailShippingAmount(
	items []orderdom.ShippingQuoteItemSnapshot,
	allowedInventoryIDs map[string]struct{},
) (int, error) {
	maxInt := int(^uint(0) >> 1)
	total := 0

	for _, item := range items {
		if item.Type != orderdom.OrderItemTypeList {
			continue
		}
		if !InventoryAllowed(allowedInventoryIDs, item.InventoryID) {
			continue
		}
		if item.Amount < 0 {
			return 0, errors.New(
				"OrderDetailQuery: shipping amount is invalid",
			)
		}
		if total > maxInt-item.Amount {
			return 0, errors.New(
				"OrderDetailQuery: shipping amount overflows int",
			)
		}
		total += item.Amount
	}

	return total, nil
}

func calculateOrderDetailConsumptionTax(
	items []orderdom.OrderItemSnapshot,
	shippingAmount int,
) (int, error) {
	maxInt := int(^uint(0) >> 1)

	if shippingAmount < 0 {
		return 0, errors.New(
			"OrderDetailQuery: shipping amount is invalid",
		)
	}

	taxableAmount8 := 0
	taxableAmount10 := shippingAmount

	for _, item := range items {
		if item.Type != orderdom.OrderItemTypeList {
			return 0, errors.New(
				"OrderDetailQuery: non-list item reached console detail calculation",
			)
		}
		if item.Price < 0 || item.Qty <= 0 {
			return 0, errors.New(
				"OrderDetailQuery: order item amount is invalid",
			)
		}
		if item.Price > maxInt/item.Qty {
			return 0, errors.New(
				"OrderDetailQuery: order item amount overflows int",
			)
		}

		lineAmount := item.Price * item.Qty

		switch item.ConsumptionTaxRate {
		case orderdom.ConsumptionTaxRateReduced:
			if taxableAmount8 > maxInt-lineAmount {
				return 0, errors.New(
					"OrderDetailQuery: reduced taxable amount overflows int",
				)
			}
			taxableAmount8 += lineAmount

		case orderdom.ConsumptionTaxRateStandard:
			if taxableAmount10 > maxInt-lineAmount {
				return 0, errors.New(
					"OrderDetailQuery: standard taxable amount overflows int",
				)
			}
			taxableAmount10 += lineAmount

		default:
			return 0, errors.New(
				"OrderDetailQuery: consumption tax rate is invalid",
			)
		}
	}

	if taxableAmount8 > maxInt/orderdom.ConsumptionTaxRateReduced {
		return 0, errors.New(
			"OrderDetailQuery: reduced consumption tax overflows int",
		)
	}
	if taxableAmount10 > maxInt/orderdom.ConsumptionTaxRateStandard {
		return 0, errors.New(
			"OrderDetailQuery: standard consumption tax overflows int",
		)
	}

	taxAmount8 := taxableAmount8 * orderdom.ConsumptionTaxRateReduced / 100
	taxAmount10 := taxableAmount10 * orderdom.ConsumptionTaxRateStandard / 100

	if taxAmount8 > maxInt-taxAmount10 {
		return 0, errors.New(
			"OrderDetailQuery: consumption tax overflows int",
		)
	}

	return taxAmount8 + taxAmount10, nil
}
