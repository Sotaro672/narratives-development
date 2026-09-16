// backend/internal/domain/refund/transfer_reversal_status.go
package refund

// TransferReversalStatus represents seller-side Stripe Transfer Reversal
// processing for a primary List-sale refund.
//
// Consumer resale Refunds do not execute Stripe Transfer Reversal and use
// TransferReversalStatusNotRequired.
type TransferReversalStatus string

const (
	TransferReversalStatusNotRequired     TransferReversalStatus = "not_required"
	TransferReversalStatusPending         TransferReversalStatus = "pending"
	TransferReversalStatusSucceeded       TransferReversalStatus = "succeeded"
	TransferReversalStatusFailedRetryable TransferReversalStatus = "failed_retryable"
	TransferReversalStatusFailed          TransferReversalStatus = "failed"
)

var AllowedTransferReversalStatuses = map[TransferReversalStatus]struct{}{
	TransferReversalStatusNotRequired:     {},
	TransferReversalStatusPending:         {},
	TransferReversalStatusSucceeded:       {},
	TransferReversalStatusFailedRetryable: {},
	TransferReversalStatusFailed:          {},
}

func IsValidTransferReversalStatus(
	status TransferReversalStatus,
) bool {
	if status == "" {
		return false
	}

	_, ok := AllowedTransferReversalStatuses[status]
	return ok
}
