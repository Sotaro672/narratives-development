// backend/internal/application/usecase/transfer_resale_receivable.go
package usecase

import (
	"context"
	"fmt"

	orderdom "narratives/internal/domain/order"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
)

func (u *TransferUsecase) requirePendingResaleReceivable(
	ctx context.Context,
	paymentID string,
	itemIndex int,
	expectedResaleID string,
	item orderdom.OrderItemSnapshot,
	expectedAvatarID string,
) (salesreceivabledom.SalesReceivable, error) {
	if u == nil || u.salesReceivableUC == nil {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableNotConfigured
	}

	if paymentID == "" ||
		itemIndex < 0 ||
		expectedResaleID == "" ||
		item.Type != orderdom.OrderItemTypeResale ||
		item.ResaleID != expectedResaleID ||
		item.Qty != 1 ||
		item.Price <= 0 ||
		item.IsCancelled ||
		item.IsReturnRequested ||
		item.IsReturnCompleted ||
		item.Transferred ||
		expectedAvatarID == "" {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}

	snapshot := item.SellerSnapshot
	if snapshot.AvatarID == "" ||
		snapshot.UserID == "" ||
		snapshot.PayoutAccountID == "" ||
		snapshot.PayoutAccountID != snapshot.UserID ||
		snapshot.AvatarID != expectedAvatarID ||
		snapshot.BrandID != "" ||
		snapshot.CompanyID != "" ||
		snapshot.AccountID != "" ||
		snapshot.StripeAccountID != "" {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}

	receivableID, err := salesreceivabledom.NewID(
		paymentID,
		itemIndex,
	)
	if err != nil {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}

	receivable, err := u.salesReceivableUC.GetByID(
		ctx,
		receivableID,
	)
	if err != nil {
		return salesreceivabledom.SalesReceivable{},
			fmt.Errorf(
				"%w: load receivable %s: %v",
				ErrTransferResaleReceivableUnavailable,
				receivableID,
				err,
			)
	}
	if receivable == nil {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableUnavailable
	}

	if receivable.ID != receivableID ||
		receivable.PaymentID != paymentID ||
		receivable.OrderID != paymentID ||
		receivable.OrderItemIndex != itemIndex ||
		receivable.ResaleID != expectedResaleID ||
		receivable.AvatarID != snapshot.AvatarID ||
		receivable.UserID != snapshot.UserID ||
		receivable.PayoutAccountID != snapshot.PayoutAccountID ||
		receivable.MerchandiseAmount != item.Price {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}

	if err := receivable.Validate(); err != nil {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}
	if receivable.Status != salesreceivabledom.StatusPending {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableUnavailable
	}

	return *receivable, nil
}

func validateCompletedResaleReceivable(
	actual salesreceivabledom.SalesReceivable,
	expected salesreceivabledom.SalesReceivable,
) error {
	if err := validateExistingSalesReceivableAllocation(
		actual,
		expected,
	); err != nil {
		return ErrTransferResaleReceivableMismatch
	}

	if actual.Status != salesreceivabledom.StatusAvailable ||
		actual.AvailableAt == nil ||
		actual.AvailableAt.IsZero() ||
		actual.BankPayoutID != "" {
		return ErrTransferResaleReceivableUnavailable
	}

	return nil
}
