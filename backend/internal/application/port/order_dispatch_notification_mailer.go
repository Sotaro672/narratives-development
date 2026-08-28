// backend/internal/application/port/order_dispatch_notification_mailer.go
package port

import "context"

type OrderDispatchNotificationMailItem struct {
	ProductName string
	Qty         int
}

type OrderDispatchNotificationMailMessage struct {
	IdempotencyKey string
	ToEmail        string

	OrderID string
	Items   []OrderDispatchNotificationMailItem
}

type OrderDispatchNotificationMailSendResult struct {
	ProviderMessageID string
	Retryable         bool
}

type OrderDispatchNotificationMailerPort interface {
	SendOrderDispatchNotification(
		ctx context.Context,
		message OrderDispatchNotificationMailMessage,
	) (OrderDispatchNotificationMailSendResult, error)
}
