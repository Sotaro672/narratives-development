// backend/internal/domain/refund/id.go
package refund

import (
	"strconv"
	"strings"
)

// NewID creates a deterministic Refund document ID.
//
// One Order item may be refunded only once through the return receipt flow.
//
// Deterministic ID:
//
//	{orderId}_{orderItemIndex}
//
// Retries therefore resolve to the same Refund record.
func NewID(
	orderID string,
	orderItemIndex int,
) (string, error) {
	if orderID == "" ||
		strings.Contains(orderID, "/") {
		return "", ErrInvalidOrderID
	}

	if orderItemIndex < 0 {
		return "", ErrInvalidOrderItemIndex
	}

	return orderID +
			"_" +
			strconv.Itoa(orderItemIndex),
		nil
}
