// backend/internal/domain/refund/debug.go
package refund

import "fmt"

// String provides a concise identifier for logs without exposing Stripe IDs.
func (r Refund) String() string {
	return fmt.Sprintf(
		"refund{id=%s orderId=%s itemIndex=%d status=%s reversalStatus=%s}",
		r.ID,
		r.OrderID,
		r.OrderItemIndex,
		r.Status,
		r.TransferReversalStatus,
	)
}
