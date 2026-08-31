// backend/internal/domain/dispatch/repository_port.go
package dispatch

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("dispatch: not found")
)

// DispatchNotificationRepository manages shipment notification delivery state.
//
// CreateIfAbsent must be idempotent for the deterministic delivery ID.
// An existing delivery, especially a delivered delivery, must never be reset
// to pending.
//
// State transition methods use expectedAttemptCount so an older worker cannot
// overwrite the result of a newer delivery attempt.
type DispatchNotificationRepository interface {
	CreateIfAbsent(
		ctx context.Context,
		delivery DispatchNotificationDelivery,
	) (DispatchNotificationDelivery, bool, error)

	GetByID(
		ctx context.Context,
		id string,
	) (DispatchNotificationDelivery, error)

	ListDue(
		ctx context.Context,
		now time.Time,
		limit int,
	) ([]DispatchNotificationDelivery, error)

	Claim(
		ctx context.Context,
		id string,
		now time.Time,
		processingUntil time.Time,
	) (DispatchNotificationDelivery, error)

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
