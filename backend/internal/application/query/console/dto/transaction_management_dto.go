// backend/internal/application/query/console/dto/transaction_management_dto.go
package dto

// ============================================================
// Constants
// ============================================================

const (
	TransactionTypeReceive = "receive"
	TransactionTypeSend    = "send"
)

// ============================================================
// DTO
// ============================================================

// TransactionManagementRowDTO is a Console transaction list row.
//
// Transaction is derived from Settlement records that belong to the
// authenticated Company.
//
// Type:
//   - receive: Stripe Connect Transfer to the Company's Account.
//   - send: Stripe Connect Transfer Reversal from the Company's Account.
//
// Amount:
//   - Always represented as a positive integer.
//   - Direction is represented by Type instead of the amount sign.
//
// Timestamp:
//   - receive uses Settlement.TransferredAt.
//   - send uses Settlement.ReversedAt.
type TransactionManagementRowDTO struct {
	ID                       string `json:"id"`
	SettlementID             string `json:"settlementId"`
	OrderID                  string `json:"orderId"`
	PaymentID                string `json:"paymentId"`
	AccountID                string `json:"accountId"`
	Type                     string `json:"type"`
	Amount                   int    `json:"amount"`
	Currency                 string `json:"currency"`
	Description              string `json:"description"`
	Status                   string `json:"status"`
	StripeTransferID         string `json:"stripeTransferId,omitempty"`
	StripeTransferReversalID string `json:"stripeTransferReversalId,omitempty"`
	Timestamp                string `json:"timestamp"`
}
