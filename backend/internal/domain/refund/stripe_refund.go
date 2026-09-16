// backend/internal/domain/refund/stripe_refund.go
package refund

import "time"

// ApplyStripeRefund records the lifecycle returned from one Stripe Refund.
//
// stripeRefundID must continue identifying the same Stripe Refund across
// subsequent updates.
func (r *Refund) ApplyStripeRefund(
	stripeRefundID string,
	status RefundStatus,
	refundedAt *time.Time,
	now time.Time,
) error {
	if r == nil {
		return ErrInvalidStatusTransition
	}

	if !isStripeRefundID(stripeRefundID) {
		return ErrInvalidStripeRefundID
	}

	if !IsValidStatus(status) ||
		status == StatusCreated {
		return ErrInvalidStatus
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if r.StripeRefundID != "" &&
		r.StripeRefundID != stripeRefundID {
		return ErrInvalidStripeRefundID
	}

	if !canTransitionRefundStatus(
		r.Status,
		status,
	) {
		return ErrInvalidStatusTransition
	}

	next := *r
	next.StripeRefundID = stripeRefundID
	next.Status = status

	switch status {
	case StatusSucceeded:
		if refundedAt == nil ||
			refundedAt.IsZero() {
			return ErrInvalidRefundedAt
		}

		value := refundedAt.UTC()
		next.RefundedAt = &value

	case StatusPending,
		StatusRequiresAction,
		StatusFailed,
		StatusCanceled:
		if refundedAt != nil {
			return ErrInvalidRefundedAt
		}

		next.RefundedAt = nil

	default:
		return ErrInvalidStatus
	}

	next.UpdatedAt = now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next
	return nil
}

func (r *Refund) MarkStripeRefundPending(
	stripeRefundID string,
	now time.Time,
) error {
	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusPending,
		nil,
		now,
	)
}

func (r *Refund) MarkStripeRefundRequiresAction(
	stripeRefundID string,
	now time.Time,
) error {
	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusRequiresAction,
		nil,
		now,
	)
}

func (r *Refund) MarkStripeRefundSucceeded(
	stripeRefundID string,
	refundedAt time.Time,
	now time.Time,
) error {
	if refundedAt.IsZero() {
		return ErrInvalidRefundedAt
	}

	value := refundedAt.UTC()

	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusSucceeded,
		&value,
		now,
	)
}

func (r *Refund) MarkStripeRefundFailed(
	stripeRefundID string,
	now time.Time,
) error {
	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusFailed,
		nil,
		now,
	)
}

func (r *Refund) MarkStripeRefundCanceled(
	stripeRefundID string,
	now time.Time,
) error {
	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusCanceled,
		nil,
		now,
	)
}

func (r Refund) validateStripeRefundState() error {
	switch r.Status {
	case StatusCreated:
		if r.StripeRefundID != "" ||
			r.RefundedAt != nil {
			return ErrInvalidStatus
		}

		return nil

	case StatusPending,
		StatusRequiresAction,
		StatusFailed,
		StatusCanceled:
		if !isStripeRefundID(
			r.StripeRefundID,
		) {
			return ErrInvalidStripeRefundID
		}

		if r.RefundedAt != nil {
			return ErrInvalidRefundedAt
		}

		return nil

	case StatusSucceeded:
		if !isStripeRefundID(
			r.StripeRefundID,
		) {
			return ErrInvalidStripeRefundID
		}

		if r.RefundedAt == nil ||
			r.RefundedAt.IsZero() {
			return ErrInvalidRefundedAt
		}

		return nil

	default:
		return ErrInvalidStatus
	}
}
