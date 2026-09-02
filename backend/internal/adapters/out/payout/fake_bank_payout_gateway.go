// backend/internal/adapters/out/payout/fake_bank_payout_gateway.go
package payout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
)

var _ applicationport.BankPayoutGateway = (*FakeBankPayoutGateway)(nil)

// FakeBankPayoutGateway simulates a completed seller bank payout.
//
// It never contacts a bank, payment provider, or any external network.
//
// Idempotency:
//   - the same BankPayoutID always produces the same ProviderPayoutID
//   - IdempotencyKey must equal BankPayoutID
//   - no mutable external payout state is created
//
// Security:
//   - AccountNumber is validated only in memory
//   - AccountNumber is never persisted or logged
//   - gateway errors never include bank account values
//
// This adapter is intended only for development/test environments.
type FakeBankPayoutGateway struct {
	now func() time.Time
}

func NewFakeBankPayoutGateway() *FakeBankPayoutGateway {
	return &FakeBankPayoutGateway{
		now: time.Now,
	}
}

// Execute validates the payout instruction and immediately returns a successful
// fake payout result.
//
// No real money movement occurs.
func (g *FakeBankPayoutGateway) Execute(
	ctx context.Context,
	in applicationport.ExecuteBankPayoutInput,
) (*applicationport.ExecuteBankPayoutResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, newFakeBankPayoutError(
			"context_error",
			"context_done",
			"fake bank payout context is done",
			true,
		)
	}

	if g == nil || g.now == nil {
		return nil, newFakeBankPayoutError(
			"configuration_error",
			"gateway_not_configured",
			"fake bank payout gateway is not configured",
			false,
		)
	}

	if err := validateFakeBankPayoutInput(in); err != nil {
		return nil, err
	}

	paidAt := g.now().UTC()
	if paidAt.IsZero() {
		return nil, newFakeBankPayoutError(
			"configuration_error",
			"invalid_clock",
			"fake bank payout clock returned an invalid time",
			false,
		)
	}

	return &applicationport.ExecuteBankPayoutResult{
		ProviderPayoutID: fakeProviderPayoutID(in.BankPayoutID),
		PaidAt:           paidAt,
	}, nil
}

func validateFakeBankPayoutInput(
	in applicationport.ExecuteBankPayoutInput,
) error {
	if strings.TrimSpace(in.BankPayoutID) == "" ||
		strings.Contains(in.BankPayoutID, "/") {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_bank_payout_id",
			"fake bank payout id is invalid",
			false,
		)
	}

	if strings.TrimSpace(in.IdempotencyKey) == "" ||
		in.IdempotencyKey != in.BankPayoutID {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_idempotency_key",
			"fake bank payout idempotency key is invalid",
			false,
		)
	}

	if in.Amount <= 0 {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_amount",
			"fake bank payout amount is invalid",
			false,
		)
	}

	if in.Currency != "JPY" {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_currency",
			"fake bank payout currency is invalid",
			false,
		)
	}

	if !isFixedDigits(in.BankCode, 4) {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_bank_code",
			"fake bank payout bank code is invalid",
			false,
		)
	}

	if strings.TrimSpace(in.BankName) == "" {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_bank_name",
			"fake bank payout bank name is invalid",
			false,
		)
	}

	if !isFixedDigits(in.BranchCode, 3) {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_branch_code",
			"fake bank payout branch code is invalid",
			false,
		)
	}

	if strings.TrimSpace(in.BranchName) == "" {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_branch_name",
			"fake bank payout branch name is invalid",
			false,
		)
	}

	switch string(in.AccountType) {
	case "ordinary", "current":
	default:
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_account_type",
			"fake bank payout account type is invalid",
			false,
		)
	}

	if !isFixedDigits(in.AccountNumber, 7) {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_account_number",
			"fake bank payout account number is invalid",
			false,
		)
	}

	if !isFixedDigits(in.BankLast4, 4) ||
		in.BankLast4 != in.AccountNumber[len(in.AccountNumber)-4:] {
		return newFakeBankPayoutError(
			"invalid_request",
			"bank_last4_mismatch",
			"fake bank payout bank last4 is invalid",
			false,
		)
	}

	if strings.TrimSpace(in.AccountHolderName) == "" {
		return newFakeBankPayoutError(
			"invalid_request",
			"invalid_account_holder_name",
			"fake bank payout account holder name is invalid",
			false,
		)
	}

	return nil
}

// fakeProviderPayoutID creates a stable provider-style identifier without
// exposing the original BankPayoutID.
//
// The value is deterministic across process restarts, so retries of the same
// BankPayout always return the same ProviderPayoutID.
func fakeProviderPayoutID(
	bankPayoutID string,
) string {
	sum := sha256.Sum256([]byte(bankPayoutID))

	// 16 bytes / 32 hex characters are sufficient for a compact deterministic
	// development identifier while avoiding the original payout ID in logs or
	// downstream records.
	return "fake_bp_" + hex.EncodeToString(sum[:16])
}

// FakeBankPayoutError implements applicationport.BankPayoutGatewayError.
type FakeBankPayoutError struct {
	errorType string
	errorCode string
	message   string
	retryable bool
}

func (e *FakeBankPayoutError) Error() string {
	if e == nil {
		return "fake bank payout failed"
	}
	if e.message == "" {
		return "fake bank payout failed"
	}

	return e.message
}

func (e *FakeBankPayoutError) ErrorType() string {
	if e == nil {
		return ""
	}

	return e.errorType
}

func (e *FakeBankPayoutError) ErrorCode() string {
	if e == nil {
		return ""
	}

	return e.errorCode
}

func (e *FakeBankPayoutError) Retryable() bool {
	if e == nil {
		return false
	}

	return e.retryable
}

func newFakeBankPayoutError(
	errorType string,
	errorCode string,
	message string,
	retryable bool,
) *FakeBankPayoutError {
	if strings.TrimSpace(message) == "" {
		message = "fake bank payout failed"
	}

	return &FakeBankPayoutError{
		errorType: strings.TrimSpace(errorType),
		errorCode: strings.TrimSpace(errorCode),
		message:   message,
		retryable: retryable,
	}
}

func isFixedDigits(
	value string,
	length int,
) bool {
	if len(value) != length {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
