// backend/internal/application/port/refund_completion_notification.go
package port

import (
	"context"
	"time"

	refunddom "narratives/internal/domain/refund"
)

type RefundCompletionNotificationRepositoryPort interface {
	CreateIfAbsent(
		ctx context.Context,
		delivery refunddom.CompletionNotificationDelivery,
	) (refunddom.CompletionNotificationDelivery, bool, error)

	GetByID(
		ctx context.Context,
		id string,
	) (refunddom.CompletionNotificationDelivery, error)

	ListDue(
		ctx context.Context,
		now time.Time,
		limit int,
	) ([]refunddom.CompletionNotificationDelivery, error)

	Claim(
		ctx context.Context,
		id string,
		now time.Time,
		processingUntil time.Time,
	) (refunddom.CompletionNotificationDelivery, error)

	MarkDelivered(
		ctx context.Context,
		id string,
		expectedAttemptCount int,
		providerMessageID string,
		deliveredAt time.Time,
	) error

	MarkRetryableFailed(
		ctx context.Context,
		id string,
		expectedAttemptCount int,
		lastError string,
		nextAttemptAt time.Time,
		failedAt time.Time,
	) error

	MarkFailed(
		ctx context.Context,
		id string,
		expectedAttemptCount int,
		lastError string,
		failedAt time.Time,
	) error
}

type RefundCompletionNotificationMailMessage struct {
	IdempotencyKey string
	ToEmail        string

	PaymentID string
	OrderID   string

	StripeRefundID string
	RefundedAmount int
}

type RefundCompletionNotificationMailSendResult struct {
	ProviderMessageID string
	Retryable         bool
}

type RefundCompletionNotificationMailerPort interface {
	SendRefundCompletionNotification(
		ctx context.Context,
		message RefundCompletionNotificationMailMessage,
	) (RefundCompletionNotificationMailSendResult, error)
}

type RefundCompletionNotificationQueuePort interface {
	EnqueueRefundCompletionNotification(
		ctx context.Context,
		delivery refunddom.CompletionNotificationDelivery,
	) error
}
