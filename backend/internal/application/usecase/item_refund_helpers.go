// backend/internal/application/usecase/item_refund_helpers.go
package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	refunddom "narratives/internal/domain/refund"
)

// ============================================================
// Helpers
// ============================================================

type itemRefundRetryableError interface {
	Retryable() bool
}

func isRetryableItemRefundError(
	err error,
) bool {
	if err == nil {
		return false
	}

	var retryableError itemRefundRetryableError

	if errors.As(err, &retryableError) {
		return retryableError.Retryable()
	}

	return false
}

func isRefundCreateConflict(
	err error,
) bool {
	return errors.Is(
		err,
		refunddom.ErrConflict,
	) ||
		errors.Is(
			err,
			refunddom.ErrDuplicateInquiry,
		) ||
		errors.Is(
			err,
			refunddom.ErrDuplicateOrderItem,
		)
}

func itemRefundIdempotencyKey(
	operation string,
	refundID string,
) string {
	value := operation + "|" + refundID
	hash := sha256.Sum256([]byte(value))

	return "amol_item_refund_" +
		operation +
		"_" +
		hex.EncodeToString(hash[:])
}

func (u *ItemRefundUsecase) nowUTC() time.Time {
	return u.now().UTC()
}
