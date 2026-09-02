// backend/internal/application/port/resale_payout_notification_mailer.go
package port

import (
	"context"
	"time"
)

// ResalePayoutNotificationMailMessage contains the information required to
// notify a resale seller after a BankPayout has completed successfully.
//
// Security:
//   - never include a plaintext bank account number
//   - never include AccountNumberCiphertext
//   - BankLast4 is display-only information
//
// Delivery:
//   - IdempotencyKey must be stable for the same BankPayout
//   - the notification must only be sent after BankPayout status is paid
type ResalePayoutNotificationMailMessage struct {
	IdempotencyKey string
	ToEmail        string

	BankPayoutID      string
	SalesReceivableID string
	OrderID           string
	ResaleID          string

	Amount   int
	Currency string

	BankName   string
	BranchName string
	BankLast4  string

	PaidAt time.Time
}

type ResalePayoutNotificationMailSendResult struct {
	ProviderMessageID string
	Retryable         bool
}

type ResalePayoutNotificationMailerPort interface {
	SendResalePayoutNotification(
		ctx context.Context,
		message ResalePayoutNotificationMailMessage,
	) (ResalePayoutNotificationMailSendResult, error)
}
