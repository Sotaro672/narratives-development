// backend/internal/domain/refund/completion.go
package refund

// IsFinanciallyCompleted reports whether every purchaser-refund and seller-side
// financial operation represented by this Refund has completed.
//
// Return-shipping cost is recorded as seller burden but is not itself part of
// the Stripe Refund / Transfer Reversal lifecycle.
func (r Refund) IsFinanciallyCompleted() bool {
	if r.Status != StatusSucceeded {
		return false
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusNotRequired,
		TransferReversalStatusSucceeded:
		return true

	default:
		return false
	}
}
