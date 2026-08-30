// backend/internal/application/usecase/order_item_snapshot.go
package usecase

import (
	"context"
	"strings"

	accountdom "narratives/internal/domain/account"
	listdom "narratives/internal/domain/list"
	orderdom "narratives/internal/domain/order"
	payoutaccountdom "narratives/internal/domain/payoutAccount"
	productblueprintcategorydom "narratives/internal/domain/productBlueprintCategory"
	resaledom "narratives/internal/domain/resale"
)

// =======================
// Order item snapshots
// =======================

func (u *OrderUsecase) resolveOrderItems(
	ctx context.Context,
	input []CreateOrderItemInput,
) ([]orderdom.OrderItemSnapshot, error) {
	items := make([]orderdom.OrderItemSnapshot, 0, len(input))

	for _, item := range input {
		switch item.Type {
		case orderdom.OrderItemTypeList:
			resolved, err := u.resolveListOrderItem(ctx, item)
			if err != nil {
				return nil, err
			}

			items = append(items, resolved)

		case orderdom.OrderItemTypeResale:
			resolved, err := u.resolveResaleOrderItem(ctx, item)
			if err != nil {
				return nil, err
			}

			items = append(items, resolved)

		default:
			return nil, orderdom.ErrInvalidItemSnapshot
		}
	}

	return items, nil
}

func (u *OrderUsecase) resolveProductBlueprintTaxSnapshot(
	ctx context.Context,
	productBlueprintID string,
) ([]string, int, error) {
	if u == nil || u.productBlueprintRepo == nil {
		return nil, 0, orderdom.ErrInvalidItemSnapshot
	}

	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, 0, orderdom.ErrInvalidItemSnapshot
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return nil, 0, err
	}

	if productBlueprint.ID != productBlueprintID {
		return nil, 0, orderdom.ErrInvalidItemSnapshot
	}

	categoryPath := append(
		[]string(nil),
		productBlueprint.ProductBlueprintCategoryPath...,
	)

	taxRate, err := productblueprintcategorydom.GetConsumptionTaxRate(categoryPath)
	if err != nil {
		return nil, 0, err
	}

	return categoryPath, int(taxRate), nil
}

func (u *OrderUsecase) resolveSellerSnapshotByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) (orderdom.SellerSnapshot, error) {
	if u == nil || u.productBlueprintRepo == nil || u.brandRepo == nil || u.accountRepo == nil {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return orderdom.SellerSnapshot{}, err
	}

	if productBlueprint.ID != productBlueprintID ||
		productBlueprint.BrandID == "" ||
		productBlueprint.CompanyID == "" {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	brand, err := u.brandRepo.GetByID(ctx, productBlueprint.BrandID)
	if err != nil {
		return orderdom.SellerSnapshot{}, err
	}

	if brand.ID != productBlueprint.BrandID ||
		brand.CompanyID != productBlueprint.CompanyID ||
		brand.AccountID == "" ||
		!brand.IsActive {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	account, err := u.accountRepo.GetByID(ctx, brand.AccountID)
	if err != nil {
		return orderdom.SellerSnapshot{}, err
	}

	if account.ID != brand.AccountID ||
		account.CompanyID != brand.CompanyID ||
		account.Status != accountdom.StatusActive ||
		account.StripeAccountID == "" {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	return orderdom.SellerSnapshot{
		BrandID:         brand.ID,
		CompanyID:       brand.CompanyID,
		AccountID:       account.ID,
		StripeAccountID: account.StripeAccountID,
	}, nil
}

func (u *OrderUsecase) resolveResaleSellerSnapshot(
	ctx context.Context,
	avatarID string,
) (orderdom.SellerSnapshot, error) {
	if u == nil || u.avatarRepo == nil || u.payoutAccountUC == nil {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	avatar, err := u.avatarRepo.GetByID(ctx, avatarID)
	if err != nil {
		return orderdom.SellerSnapshot{}, err
	}

	userID := strings.TrimSpace(avatar.UserID)
	if avatar.ID != avatarID || userID == "" {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	payoutAccount, err := u.payoutAccountUC.GetByUserID(ctx, userID)
	if err != nil {
		return orderdom.SellerSnapshot{}, err
	}
	if payoutAccount == nil {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	provider := strings.TrimSpace(payoutAccount.Provider)
	providerAccountID := strings.TrimSpace(payoutAccount.ProviderAccountID)

	if payoutAccount.UserID != userID ||
		provider != payoutaccountdom.ProviderStripe ||
		payoutAccount.Status != payoutaccountdom.StatusRegistered ||
		!payoutAccount.PayoutReady ||
		providerAccountID == "" ||
		!strings.HasPrefix(providerAccountID, "acct_") {
		return orderdom.SellerSnapshot{}, orderdom.ErrInvalidSellerSnapshot
	}

	return orderdom.SellerSnapshot{
		AvatarID:        avatar.ID,
		UserID:          userID,
		PayoutAccountID: userID,
		StripeAccountID: providerAccountID,
	}, nil
}

func (u *OrderUsecase) resolveListOrderItem(
	ctx context.Context,
	item CreateOrderItemInput,
) (orderdom.OrderItemSnapshot, error) {
	if item.ListID == "" || item.ModelID == "" || item.Qty <= 0 {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	list, err := u.listRepo.GetByID(ctx, item.ListID)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	if list.Status != listdom.StatusListing {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	inventory, err := u.inventoryRepo.GetByID(ctx, list.InventoryID)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	if inventory.ProductBlueprintID == "" || inventory.TokenBlueprintID == "" {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	productBlueprintCategoryPath, consumptionTaxRate, err :=
		u.resolveProductBlueprintTaxSnapshot(ctx, inventory.ProductBlueprintID)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	sellerSnapshot, err :=
		u.resolveSellerSnapshotByProductBlueprintID(ctx, inventory.ProductBlueprintID)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	stock, ok := inventory.Stock[item.ModelID]
	if !ok {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	available := stock.Accumulation - stock.ReservedCount
	if available < item.Qty {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	price, err := resolveListModelPrice(list, item.ModelID)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	return orderdom.OrderItemSnapshot{
		Type:               orderdom.OrderItemTypeList,
		ModelID:            item.ModelID,
		InventoryID:        list.InventoryID,
		ListID:             list.ID,
		ProductBlueprintID: inventory.ProductBlueprintID,
		TokenBlueprintID:   inventory.TokenBlueprintID,

		SellerSnapshot: sellerSnapshot,

		ProductBlueprintCategoryPath: productBlueprintCategoryPath,
		ConsumptionTaxRate:           consumptionTaxRate,

		Qty:                     item.Qty,
		Price:                   price,
		IsCancelled:             false,
		IsDispatched:            false,
		IsReturnRequested:       false,
		ReturnRequestKind:       "",
		ReturnRequestedAt:       nil,
		IsReturnCompleted:       false,
		ReturnCompletedAt:       nil,
		TokenTransferVerifiedAt: nil,
		Transferred:             false,
		TransferredAt:           nil,
	}, nil
}

func resolveListModelPrice(
	list listdom.List,
	modelID string,
) (int, error) {
	for _, price := range list.Prices {
		if price.ModelID == modelID {
			return price.Price, nil
		}
	}

	return 0, orderdom.ErrInvalidItemSnapshot
}

func (u *OrderUsecase) resolveResaleOrderItem(
	ctx context.Context,
	item CreateOrderItemInput,
) (orderdom.OrderItemSnapshot, error) {
	if u == nil || u.resaleRepo == nil {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	resaleID := strings.TrimSpace(item.ResaleID)
	if resaleID == "" || item.Qty != 1 {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	resale, err := u.resaleRepo.GetByID(ctx, resaleID)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	if resale.ID != resaleID ||
		resale.Status != resaledom.StatusListing ||
		resale.AvatarID == "" ||
		resale.ProductID == "" ||
		resale.ProductBlueprintID == "" ||
		resale.TokenBlueprintID == "" ||
		resale.BrandID == "" ||
		resale.Price < 0 {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	productBlueprintCategoryPath, consumptionTaxRate, err :=
		u.resolveProductBlueprintTaxSnapshot(ctx, resale.ProductBlueprintID)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	sellerSnapshot, err := u.resolveResaleSellerSnapshot(ctx, resale.AvatarID)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	return orderdom.OrderItemSnapshot{
		Type:               orderdom.OrderItemTypeResale,
		ResaleID:           resale.ID,
		ProductID:          resale.ProductID,
		ProductBlueprintID: resale.ProductBlueprintID,
		TokenBlueprintID:   resale.TokenBlueprintID,
		BrandID:            resale.BrandID,

		SellerSnapshot: sellerSnapshot,

		ProductBlueprintCategoryPath: productBlueprintCategoryPath,
		ConsumptionTaxRate:           consumptionTaxRate,

		Qty:                     1,
		Price:                   resale.Price,
		IsCancelled:             false,
		IsDispatched:            false,
		IsReturnRequested:       false,
		ReturnRequestKind:       "",
		ReturnRequestedAt:       nil,
		IsReturnCompleted:       false,
		ReturnCompletedAt:       nil,
		TokenTransferVerifiedAt: nil,
		Transferred:             false,
		TransferredAt:           nil,
	}, nil
}
