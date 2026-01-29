// backend/internal/platform/di/console/adapters_inventory.go
package console

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	fs "narratives/internal/adapters/out/firestore"
)

// ✅ Adapter: fs.InventoryRepositoryFS に ApplyTransferResult を付与
type inventoryRepoTransferResultAdapter struct {
	*fs.InventoryRepositoryFS
}

// ApplyTransferResult updates inventory after transfer by removing productId and decrementing reservation for orderId.
func (a *inventoryRepoTransferResultAdapter) ApplyTransferResult(
	ctx context.Context,
	productID string,
	orderID string,
	now time.Time,
) error {
	if a == nil || a.InventoryRepositoryFS == nil {
		return errors.New("inventory repo adapter is nil")
	}

	removed, err := a.InventoryRepositoryFS.ReleaseReservationAfterTransfer(ctx, productID, orderID, now)
	if err != nil {
		return err
	}

	// best-effort log (removed can be 0 on idempotent re-run)
	log.Printf(
		"[inventory_repo_adapter] ApplyTransferResult ok productId=%q orderId=%q removed=%d at=%s",
		strings.TrimSpace(productID),
		strings.TrimSpace(orderID),
		removed,
		now.UTC().Format(time.RFC3339),
	)

	return nil
}
