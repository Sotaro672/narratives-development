// backend/internal/application/port/token_owner_updater.go
package port

import (
	"context"
	"time"
)

// TokenOwnerUpdater updates the recorded owner/address of a transferred token.
type TokenOwnerUpdater interface {
	UpdateToAddressByProductID(
		ctx context.Context,
		productID string,
		newToAddress string,
		now time.Time,
		txSignature string,
	) error
}
