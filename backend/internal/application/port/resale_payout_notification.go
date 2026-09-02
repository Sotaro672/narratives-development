// backend/internal/application/port/resale_payout_notification.go
package port

import (
	"context"
	"time"

	bankpayoutdom "narratives/internal/domain/bankPayout"
)

type ResalePayoutNotificationRepositoryPort interface {
	CreateIfAbsent(
		ctx context.Context,
		delivery bankpayoutdom.PayoutNotificationDelivery,
	) (bankpayoutdom.PayoutNotificationDelivery, bool, error)

	GetByID(
		ctx context.Context,
		id string,
	) (bankpayoutdom.PayoutNotificationDelivery, error)

	ListDue(
		ctx context.Context,
		now time.Time,
		limit int,
	) ([]bankpayoutdom.PayoutNotificationDelivery, error)

	Claim(
		ctx context.Context,
		id string,
		now time.Time,
		processingUntil time.Time,
	) (bankpayoutdom.PayoutNotificationDelivery, error)

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

type ResalePayoutNotificationQueuePort interface {
	EnqueueResalePayoutNotification(
		ctx context.Context,
		delivery bankpayoutdom.PayoutNotificationDelivery,
	) error
}
