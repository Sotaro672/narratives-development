// backend/internal/domain/refund/transfer_reversal.go
package refund

import "time"

// MarkTransferReversalPending prepares a seller-side Transfer Reversal retry.
func (r *Refund) MarkTransferReversalPending(
	now time.Time,
) error {
	if r == nil {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.SellerType != SellerTypeAccount {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.Status != StatusSucceeded {
		return ErrTransferReversalRequiresSucceededRefund
	}

	if r.TransferReversalAmount <= 0 {
		return ErrInvalidTransferReversalAmount
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending,
		TransferReversalStatusFailedRetryable:
	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r
	next.TransferReversalStatus =
		TransferReversalStatusPending
	next.UpdatedAt = now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next
	return nil
}

// MarkTransferReversalSucceeded records one successful partial Stripe Transfer
// Reversal.
func (r *Refund) MarkTransferReversalSucceeded(
	stripeTransferReversalID string,
	reversedAt time.Time,
	now time.Time,
) error {
	if r == nil {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.SellerType != SellerTypeAccount {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.Status != StatusSucceeded {
		return ErrTransferReversalRequiresSucceededRefund
	}

	if r.TransferReversalAmount <= 0 {
		return ErrInvalidTransferReversalAmount
	}

	if !isStripeTransferReversalID(
		stripeTransferReversalID,
	) {
		return ErrInvalidStripeTransferReversalID
	}

	if reversedAt.IsZero() {
		return ErrInvalidTransferReversedAt
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending,
		TransferReversalStatusFailedRetryable:

	case TransferReversalStatusSucceeded:
		if r.StripeTransferReversalID ==
			stripeTransferReversalID {
			return nil
		}

		return ErrInvalidStripeTransferReversalID

	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r

	reversedAt = reversedAt.UTC()

	next.StripeTransferReversalID =
		stripeTransferReversalID
	next.TransferReversalStatus =
		TransferReversalStatusSucceeded
	next.TransferReversedAt =
		&reversedAt
	next.UpdatedAt =
		now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next
	return nil
}

func (r *Refund) MarkTransferReversalFailedRetryable(
	now time.Time,
) error {
	return r.markTransferReversalFailed(
		TransferReversalStatusFailedRetryable,
		now,
	)
}

func (r *Refund) MarkTransferReversalFailed(
	now time.Time,
) error {
	return r.markTransferReversalFailed(
		TransferReversalStatusFailed,
		now,
	)
}

func (r *Refund) markTransferReversalFailed(
	status TransferReversalStatus,
	now time.Time,
) error {
	if r == nil {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.SellerType != SellerTypeAccount {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.Status != StatusSucceeded {
		return ErrTransferReversalRequiresSucceededRefund
	}

	if r.TransferReversalAmount <= 0 {
		return ErrInvalidTransferReversalAmount
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if status !=
		TransferReversalStatusFailedRetryable &&
		status !=
			TransferReversalStatusFailed {
		return ErrInvalidTransferReversalStatus
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending,
		TransferReversalStatusFailedRetryable:
	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r

	next.StripeTransferReversalID = ""
	next.TransferReversalStatus = status
	next.TransferReversedAt = nil
	next.UpdatedAt = now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next
	return nil
}

func (r Refund) validateTransferReversalState() error {
	if r.SellerType == SellerTypeResale {
		if r.TransferReversalAmount != 0 ||
			r.TransferReversalStatus !=
				TransferReversalStatusNotRequired ||
			r.StripeTransferReversalID != "" ||
			r.TransferReversedAt != nil {
			return ErrInvalidTransferReversalStatus
		}

		return nil
	}

	if r.SellerType != SellerTypeAccount {
		return ErrInvalidSellerType
	}

	if r.TransferReversalAmount == 0 {
		if r.TransferReversalStatus !=
			TransferReversalStatusNotRequired {
			return ErrInvalidTransferReversalStatus
		}

		if r.StripeTransferReversalID != "" ||
			r.TransferReversedAt != nil {
			return ErrInvalidTransferReversalStatus
		}

		return nil
	}

	if r.TransferReversalStatus ==
		TransferReversalStatusNotRequired {
		return ErrInvalidTransferReversalStatus
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending,
		TransferReversalStatusFailedRetryable,
		TransferReversalStatusFailed:
		if r.StripeTransferReversalID != "" {
			return ErrInvalidStripeTransferReversalID
		}

		if r.TransferReversedAt != nil {
			return ErrInvalidTransferReversedAt
		}

		return nil

	case TransferReversalStatusSucceeded:
		if r.Status != StatusSucceeded {
			return ErrTransferReversalRequiresSucceededRefund
		}

		if !isStripeTransferReversalID(
			r.StripeTransferReversalID,
		) {
			return ErrInvalidStripeTransferReversalID
		}

		if r.TransferReversedAt == nil ||
			r.TransferReversedAt.IsZero() {
			return ErrInvalidTransferReversedAt
		}

		return nil

	default:
		return ErrInvalidTransferReversalStatus
	}
}
