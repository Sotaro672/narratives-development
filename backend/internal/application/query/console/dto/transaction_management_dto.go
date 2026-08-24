// backend/internal/application/query/console/dto/transaction_management_dto.go
package dto

// ============================================================
// Constants
// ============================================================

const (
	TransactionPaymentStatusUnpaid = "unpaid"
)

// ============================================================
// DTO
// ============================================================

// TransactionManagementRowDTO is a Console transaction list row.
//
// OrderAmount:
//   - Amount attributable to the current Company.
//   - Product items and shipping quote items are restricted by the
//     current Company inventory boundary.
//
// PaymentAmount:
//   - Available only when the Order belongs entirely to the current Company.
//   - For a multi-company Order, the Payment represents the full customer
//     charge and must not be exposed as the current Company's revenue.
//   - Multi-company payment allocation will be supplied by Stripe Connect
//     Transfer processing in a later implementation.
type TransactionManagementRowDTO struct {
	OrderID               string `json:"orderId"`
	PaymentID             string `json:"paymentId,omitempty"`
	CreatedAt             string `json:"createdAt,omitempty"`
	OrderCreatedAt        string `json:"orderCreatedAt,omitempty"`
	PaymentCreatedAt      string `json:"paymentCreatedAt,omitempty"`
	Paid                  bool   `json:"paid"`
	OrderAmount           int    `json:"orderAmount"`
	PaymentAmount         *int   `json:"paymentAmount,omitempty"`
	PaymentStatus         string `json:"paymentStatus"`
	StripePaymentIntentID string `json:"stripePaymentIntentId,omitempty"`
	IsMultiCompanyOrder   bool   `json:"isMultiCompanyOrder"`
	AmountMatched         *bool  `json:"amountMatched,omitempty"`
}
