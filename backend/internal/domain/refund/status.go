// backend/internal/domain/refund/status.go
package refund

// RefundStatus represents the lifecycle of one item-level purchaser refund.
//
// This lifecycle is independent from Payment.Status because Payment represents
// the original purchaser payment while Refund represents a subsequent item-level
// partial or full refund.
type RefundStatus string

const (
	StatusCreated        RefundStatus = "created"
	StatusPending        RefundStatus = "pending"
	StatusRequiresAction RefundStatus = "requires_action"
	StatusSucceeded      RefundStatus = "succeeded"
	StatusFailed         RefundStatus = "failed"
	StatusCanceled       RefundStatus = "canceled"
)

var AllowedStatuses = map[RefundStatus]struct{}{
	StatusCreated:        {},
	StatusPending:        {},
	StatusRequiresAction: {},
	StatusSucceeded:      {},
	StatusFailed:         {},
	StatusCanceled:       {},
}

var DefaultStatus = StatusCreated

func IsValidStatus(status RefundStatus) bool {
	if status == "" {
		return false
	}

	_, ok := AllowedStatuses[status]
	return ok
}

func canTransitionRefundStatus(
	current RefundStatus,
	next RefundStatus,
) bool {
	if current == next {
		return true
	}

	switch current {
	case StatusCreated:
		switch next {
		case StatusPending,
			StatusRequiresAction,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled:
			return true
		default:
			return false
		}

	case StatusPending:
		switch next {
		case StatusRequiresAction,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled:
			return true
		default:
			return false
		}

	case StatusRequiresAction:
		switch next {
		case StatusPending,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled:
			return true
		default:
			return false
		}

	case StatusSucceeded,
		StatusFailed,
		StatusCanceled:
		return false

	default:
		return false
	}
}
