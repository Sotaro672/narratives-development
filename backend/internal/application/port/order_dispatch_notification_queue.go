// backend/internal/application/port/order_dispatch_notification_queue.go
package port

import (
	"context"

	orderdom "narratives/internal/domain/order"
)

type OrderDispatchNotificationQueuePort interface {
	EnqueueOrderDispatchNotification(
		ctx context.Context,
		delivery orderdom.DispatchNotificationDelivery,
	) error
}
