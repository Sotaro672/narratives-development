// backend/internal/domain/order/item_lifecycle.go
package order

import "time"

// ========================================
// Item lifecycle
// ========================================

func (o *Order) CancelItem(
	index int,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	item := &o.Items[index]

	if item.IsCancelled {
		return nil
	}

	if item.IsDispatched ||
		item.Transferred ||
		item.TokenTransferVerifiedAt != nil {
		return ErrConflict
	}

	item.IsCancelled = true
	return nil
}

func (o *Order) RequestItemReturn(
	index int,
	kind ReturnRequestKind,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if !isValidReturnRequestKind(kind) {
		return ErrInvalidItemSnapshot
	}

	if at.IsZero() {
		return ErrInvalidItemSnapshot
	}

	item := &o.Items[index]

	if item.IsCancelled ||
		!item.IsDispatched ||
		item.IsReturnCompleted {
		return ErrConflict
	}

	switch kind {
	case ReturnRequestKindUnopened:
		if item.Transferred ||
			item.TokenTransferVerifiedAt != nil {
			return ErrConflict
		}

	case ReturnRequestKindOpened:
	}

	if item.IsReturnRequested {
		switch {
		case item.ReturnRequestKind == kind:
			return nil

		case item.ReturnRequestKind == ReturnRequestKindUnopened &&
			kind == ReturnRequestKindOpened:
			item.ReturnRequestKind = ReturnRequestKindOpened
			return nil

		case item.ReturnRequestKind == ReturnRequestKindOpened &&
			kind == ReturnRequestKindUnopened:
			return ErrConflict

		default:
			return ErrInvalidItemSnapshot
		}
	}

	returnRequestedAt := at.UTC()

	item.IsReturnRequested = true
	item.ReturnRequestKind = kind
	item.ReturnRequestedAt = &returnRequestedAt

	return nil
}

func (o *Order) CompleteItemReturn(
	index int,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if at.IsZero() {
		return ErrInvalidItemSnapshot
	}

	item := &o.Items[index]

	if item.IsReturnCompleted {
		return nil
	}

	if item.IsCancelled ||
		!item.IsDispatched ||
		!item.IsReturnRequested ||
		item.ReturnRequestedAt == nil ||
		item.ReturnRequestedAt.IsZero() {
		return ErrConflict
	}

	returnCompletedAt := at.UTC()

	if returnCompletedAt.Before(
		item.ReturnRequestedAt.UTC(),
	) {
		return ErrInvalidItemSnapshot
	}

	item.IsReturnCompleted = true
	item.ReturnCompletedAt = &returnCompletedAt

	return nil
}

func (o *Order) MarkItemTokenTransferVerified(
	index int,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if at.IsZero() {
		return ErrInvalidItemSnapshot
	}

	item := &o.Items[index]

	if item.IsCancelled ||
		item.IsReturnCompleted {
		return ErrConflict
	}

	if item.TokenTransferVerifiedAt != nil {
		return nil
	}

	verifiedAt := at.UTC()
	item.TokenTransferVerifiedAt = &verifiedAt

	return nil
}

func (o *Order) UpdateItemDispatched(
	index int,
	isDispatched bool,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if isDispatched &&
		o.Items[index].IsCancelled {
		return ErrConflict
	}

	if !isDispatched &&
		o.Items[index].IsReturnRequested {
		return ErrConflict
	}

	o.Items[index].IsDispatched = isDispatched
	return nil
}

// UpdateItemTransferred updates item-level transfer state.
// transferred=true requires a non-zero transferredAt value.
func (o *Order) UpdateItemTransferred(
	index int,
	transferred bool,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}

	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if transferred {
		if o.Items[index].IsCancelled ||
			o.Items[index].IsReturnRequested {
			return ErrConflict
		}

		if at.IsZero() {
			return ErrInvalidItemSnapshot
		}

		transferredAt := at.UTC()

		if o.Items[index].TokenTransferVerifiedAt == nil {
			verifiedAt := transferredAt
			o.Items[index].TokenTransferVerifiedAt = &verifiedAt
		}

		o.Items[index].Transferred = true
		o.Items[index].TransferredAt = &transferredAt
		return nil
	}

	o.Items[index].Transferred = false
	o.Items[index].TransferredAt = nil
	return nil
}
