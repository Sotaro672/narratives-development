// backend/internal/application/usecase/settlement_stripe_error.go
package usecase

import "errors"

// ============================================================
// Stripe error helpers
// ============================================================

func isSettlementTransferErrorRetryable(
	err error,
) bool {
	if err == nil {
		return false
	}

	var retryableError RetryableStripeSettlementError
	if errors.As(
		err,
		&retryableError,
	) {
		return retryableError.Retryable()
	}

	// Unknown transport/infrastructure failures are retried.
	//
	// The deterministic Stripe Idempotency-Key prevents a retry from creating
	// another Transfer when Stripe accepted the original request but the
	// response was lost.
	return true
}

func settlementErrorType(
	err error,
) *string {
	if err == nil {
		return nil
	}

	var metadata StripeSettlementErrorMetadata
	if !errors.As(
		err,
		&metadata,
	) {
		return nil
	}

	value := metadata.ErrorType()
	if value == "" {
		return nil
	}

	return &value
}

func settlementErrorCode(
	err error,
) *string {
	if err == nil {
		return nil
	}

	var metadata StripeSettlementErrorMetadata
	if !errors.As(
		err,
		&metadata,
	) {
		return nil
	}

	value := metadata.ErrorCode()
	if value == "" {
		return nil
	}

	return &value
}

func normalizeSettlementErrorString(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	if *value == "" {
		return nil
	}

	return value
}
