// backend/internal/application/port/order_dispatch_notification_queue.go
package port

import (
	"context"

	dispatchdom "narratives/internal/domain/dispatch"
)

type OrderDispatchNotificationQueuePort interface {
	EnqueueOrderDispatchNotification(
		ctx context.Context,
		delivery dispatchdom.DispatchNotificationDelivery,
	) error
}
