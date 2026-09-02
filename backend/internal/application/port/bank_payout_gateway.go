// backend/internal/application/port/bank_payout_gateway.go
package port

import (
	"context"
	"time"

	payoutdom "narratives/internal/domain/payoutAccount"
)

// BankPayoutGateway executes one seller bank payout instruction.
//
// The current development implementation may be a FakeBankPayoutGateway that
// immediately returns a successful result without contacting any external bank
// or payment provider.
//
// A future production implementation may call a bank-transfer/payment-provider
// API behind the same interface.
//
// Idempotency:
//   - BankPayoutID identifies one deterministic BankPayout.
//   - IdempotencyKey must be stable for retries of the same BankPayout.
//   - Gateway implementations must never intentionally create multiple payouts
//     for repeated requests with the same idempotency key.
//
// Security:
//   - AccountNumber is plaintext only for the lifetime of the gateway call.
//   - AccountNumber must never be persisted, logged, included in errors, or
//     returned from this interface.
//   - The application layer is responsible for decrypting the snapshotted
//     encrypted account number immediately before invoking the gateway.
//
// Completion semantics:
//   - Execute returns successfully only when the gateway considers the payout
//     completed.
//   - The current BankPayout domain therefore maps a successful result directly
//     from processing -> paid.
//   - If a future real provider is asynchronous, introduce an explicit
//     submitted/pending-provider state before adapting that provider rather than
//     treating provider acceptance as payment completion.
type BankPayoutGateway interface {
	Execute(
		ctx context.Context,
		in ExecuteBankPayoutInput,
	) (*ExecuteBankPayoutResult, error)
}

// ExecuteBankPayoutInput contains one immutable payout instruction.
//
// Bank account values must come from the BankPayout destination snapshot, not
// from a freshly re-resolved PayoutAccount. This ensures that changing the
// seller's registered payout account after BankPayout creation does not alter
// the destination of an already-created payout.
type ExecuteBankPayoutInput struct {
	BankPayoutID   string
	IdempotencyKey string

	Amount   int
	Currency string

	BankCode   string
	BankName   string
	BranchCode string
	BranchName string

	AccountType       payoutdom.BankAccountType
	AccountNumber     string
	BankLast4         string
	AccountHolderName string
}

// ExecuteBankPayoutResult represents a successfully completed payout.
//
// ProviderPayoutID is gateway-specific. FakeBankPayoutGateway may return a
// deterministic fake identifier, while a future production gateway may return
// the provider's transfer/instruction ID.
//
// PaidAt is the completion timestamp reported or determined by the gateway.
// The application layer persists it through BankPayout.MarkPaid and coordinates
// the matching SalesReceivable reserved -> paid transition.
type ExecuteBankPayoutResult struct {
	ProviderPayoutID string
	PaidAt           time.Time
}

// BankPayoutGatewayError exposes structured gateway failure metadata.
//
// Gateway implementations should return errors implementing this interface when
// the application must distinguish retryable transport/provider failures from
// terminal failures.
//
// Error() must not contain plaintext bank account numbers or other secrets.
type BankPayoutGatewayError interface {
	error

	ErrorType() string
	ErrorCode() string
	Retryable() bool
}
