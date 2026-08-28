// backend/internal/application/usecase/refund_completion_notification_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	refunddom "narratives/internal/domain/refund"
)

const (
	defaultRefundCompletionNotificationDispatchLimit = 50
	maxRefundCompletionNotificationDispatchLimit     = 200

	defaultRefundCompletionNotificationLeaseDuration = 5 * time.Minute
	defaultRefundCompletionNotificationRetryDelay    = time.Minute
	maxRefundCompletionNotificationRetryDelay        = 6 * time.Hour

	maxRefundCompletionNotificationErrorLength = 2000
)

// ==============================
// Inbound Port
// ==============================

type RefundCompletionNotificationUsecasePort interface {
	EnsureDelivery(
		ctx context.Context,
		in EnsureRefundCompletionNotificationInput,
	) (refunddom.CompletionNotificationDelivery, error)

	DispatchDue(
		ctx context.Context,
		limit int,
	) (int, error)

	Process(
		ctx context.Context,
		deliveryID string,
	) error
}

type EnsureRefundCompletionNotificationInput struct {
	PaymentID string
	OrderID   string
	UserID    string

	StripeRefundID string
	RefundedAmount int
}

// ==============================
// Usecase
// ==============================

type RefundCompletionNotificationUsecase struct {
	deliveryRepo applicationport.RefundCompletionNotificationRepositoryPort
	authUser     applicationport.AuthUserReader
	mailer       applicationport.RefundCompletionNotificationMailerPort
	queue        applicationport.RefundCompletionNotificationQueuePort

	now           func() time.Time
	leaseDuration time.Duration
	retryDelay    time.Duration
	maxRetryDelay time.Duration
}

func NewRefundCompletionNotificationUsecase(
	deliveryRepo applicationport.RefundCompletionNotificationRepositoryPort,
	authUser applicationport.AuthUserReader,
	mailer applicationport.RefundCompletionNotificationMailerPort,
	queue applicationport.RefundCompletionNotificationQueuePort,
) *RefundCompletionNotificationUsecase {
	return &RefundCompletionNotificationUsecase{
		deliveryRepo: deliveryRepo,
		authUser:     authUser,
		mailer:       mailer,
		queue:        queue,

		now:           time.Now,
		leaseDuration: defaultRefundCompletionNotificationLeaseDuration,
		retryDelay:    defaultRefundCompletionNotificationRetryDelay,
		maxRetryDelay: maxRefundCompletionNotificationRetryDelay,
	}
}

// ==============================
// Ensure Delivery
// ==============================

func (u *RefundCompletionNotificationUsecase) EnsureDelivery(
	ctx context.Context,
	in EnsureRefundCompletionNotificationInput,
) (refunddom.CompletionNotificationDelivery, error) {
	if u == nil {
		return refunddom.CompletionNotificationDelivery{},
			fmt.Errorf("refund completion notification usecase is nil")
	}

	if u.deliveryRepo == nil {
		return refunddom.CompletionNotificationDelivery{},
			fmt.Errorf("refund completion notification repository is not configured")
	}

	in.PaymentID = strings.TrimSpace(in.PaymentID)
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.UserID = strings.TrimSpace(in.UserID)
	in.StripeRefundID = strings.TrimSpace(in.StripeRefundID)

	now := u.currentTime()

	candidate, err := refunddom.NewCompletionNotificationDelivery(
		in.PaymentID,
		in.OrderID,
		in.UserID,
		in.StripeRefundID,
		in.RefundedAmount,
		now,
		refunddom.DefaultCompletionNotificationMaxAttempts,
	)
	if err != nil {
		return refunddom.CompletionNotificationDelivery{},
			fmt.Errorf("create refund completion notification delivery: %w", err)
	}

	delivery, _, err := u.deliveryRepo.CreateIfAbsent(
		ctx,
		candidate,
	)
	if err != nil {
		return refunddom.CompletionNotificationDelivery{},
			fmt.Errorf(
				"create refund completion notification delivery if absent: %w",
				err,
			)
	}

	delivery, err = delivery.Normalize()
	if err != nil {
		return refunddom.CompletionNotificationDelivery{},
			fmt.Errorf(
				"normalize refund completion notification delivery %q: %w",
				delivery.ID,
				err,
			)
	}

	if !sameRefundCompletionNotificationScope(
		delivery,
		candidate,
	) {
		return refunddom.CompletionNotificationDelivery{},
			refunddom.ErrConflict
	}

	switch delivery.Status {
	case refunddom.CompletionNotificationStatusPending,
		refunddom.CompletionNotificationStatusRetryableFailed:
		if !delivery.IsDue(now) {
			return delivery, nil
		}

		if u.queue == nil {
			return delivery,
				fmt.Errorf("refund completion notification queue is not configured")
		}

		if err := u.queue.EnqueueRefundCompletionNotification(
			ctx,
			delivery,
		); err != nil {
			// Delivery自体はFirestoreへ保存済み。
			// DispatchDueから後で回収できる。
			return delivery,
				fmt.Errorf(
					"enqueue refund completion notification %q: %w",
					delivery.ID,
					err,
				)
		}

	case refunddom.CompletionNotificationStatusProcessing,
		refunddom.CompletionNotificationStatusDelivered,
		refunddom.CompletionNotificationStatusFailed:
		return delivery, nil
	}

	return delivery, nil
}

// ==============================
// Dispatch Due
// ==============================

func (u *RefundCompletionNotificationUsecase) DispatchDue(
	ctx context.Context,
	limit int,
) (int, error) {
	if u == nil {
		return 0,
			fmt.Errorf("refund completion notification usecase is nil")
	}

	if u.deliveryRepo == nil {
		return 0,
			fmt.Errorf("refund completion notification repository is not configured")
	}

	if u.queue == nil {
		return 0,
			fmt.Errorf("refund completion notification queue is not configured")
	}

	limit = normalizeRefundCompletionNotificationDispatchLimit(limit)
	now := u.currentTime()

	deliveries, err := u.deliveryRepo.ListDue(
		ctx,
		now,
		limit,
	)
	if err != nil {
		return 0,
			fmt.Errorf("list due refund completion notifications: %w", err)
	}

	enqueuedCount := 0
	var firstErr error

	for _, delivery := range deliveries {
		normalized, err := delivery.Normalize()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"normalize refund completion notification %q: %w",
					delivery.ID,
					err,
				)
			}
			continue
		}

		if normalized.IsTerminal() ||
			!normalized.IsDue(now) {
			continue
		}

		if err := u.queue.EnqueueRefundCompletionNotification(
			ctx,
			normalized,
		); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"enqueue refund completion notification %q: %w",
					normalized.ID,
					err,
				)
			}
			continue
		}

		enqueuedCount++
	}

	return enqueuedCount, firstErr
}

// ==============================
// Process
// ==============================

func (u *RefundCompletionNotificationUsecase) Process(
	ctx context.Context,
	deliveryID string,
) error {
	if u == nil {
		return fmt.Errorf("refund completion notification usecase is nil")
	}

	if u.deliveryRepo == nil {
		return fmt.Errorf("refund completion notification repository is not configured")
	}

	if u.authUser == nil {
		return fmt.Errorf("auth user reader is not configured")
	}

	if u.mailer == nil {
		return fmt.Errorf("refund completion notification mailer is not configured")
	}

	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return refunddom.ErrCompletionNotificationDeliveryIDRequired
	}

	claimedAt := u.currentTime()
	processingUntil := claimedAt.Add(
		u.normalizedLeaseDuration(),
	)

	delivery, err := u.deliveryRepo.Claim(
		ctx,
		deliveryID,
		claimedAt,
		processingUntil,
	)
	if err != nil {
		switch {
		case errors.Is(err, refunddom.ErrNotFound):
			return nil

		case errors.Is(
			err,
			refunddom.ErrCompletionNotificationNotClaimable,
		):
			return nil

		case errors.Is(
			err,
			refunddom.ErrCompletionNotificationAttemptLimit,
		):
			return nil

		default:
			return fmt.Errorf(
				"claim refund completion notification %q: %w",
				deliveryID,
				err,
			)
		}
	}

	delivery, err = delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize claimed refund completion notification %q: %w",
			deliveryID,
			err,
		)
	}

	if delivery.Status !=
		refunddom.CompletionNotificationStatusProcessing {
		return nil
	}

	toEmail, err := u.authUser.GetEmailByUID(
		ctx,
		delivery.UserID,
	)
	if err != nil {
		return u.markRetryableProcessingFailure(
			ctx,
			delivery,
			fmt.Errorf(
				"resolve buyer email for user %q: %w",
				delivery.UserID,
				err,
			),
		)
	}

	toEmail = strings.TrimSpace(toEmail)
	if toEmail == "" {
		return u.markPermanentProcessingFailure(
			ctx,
			delivery,
			fmt.Errorf(
				"buyer email is empty for user %q",
				delivery.UserID,
			),
		)
	}

	result, sendErr :=
		u.mailer.SendRefundCompletionNotification(
			ctx,
			applicationport.RefundCompletionNotificationMailMessage{
				IdempotencyKey: delivery.ID,
				ToEmail:        toEmail,

				PaymentID: delivery.PaymentID,
				OrderID:   delivery.OrderID,

				StripeRefundID: delivery.StripeRefundID,
				RefundedAmount: delivery.RefundedAmount,
			},
		)

	completedAt := u.currentTime()

	if sendErr == nil {
		if err := u.deliveryRepo.MarkDelivered(
			ctx,
			delivery.ID,
			delivery.AttemptCount,
			strings.TrimSpace(
				result.ProviderMessageID,
			),
			completedAt,
		); err != nil {
			return fmt.Errorf(
				"mark refund completion notification %q delivered: %w",
				delivery.ID,
				err,
			)
		}

		return nil
	}

	if result.Retryable {
		return u.markRetryableProcessingFailureAt(
			ctx,
			delivery,
			sendErr,
			completedAt,
		)
	}

	return u.markPermanentProcessingFailureAt(
		ctx,
		delivery,
		sendErr,
		completedAt,
	)
}

func (u *RefundCompletionNotificationUsecase) markRetryableProcessingFailure(
	ctx context.Context,
	delivery refunddom.CompletionNotificationDelivery,
	cause error,
) error {
	return u.markRetryableProcessingFailureAt(
		ctx,
		delivery,
		cause,
		u.currentTime(),
	)
}

func (u *RefundCompletionNotificationUsecase) markRetryableProcessingFailureAt(
	ctx context.Context,
	delivery refunddom.CompletionNotificationDelivery,
	cause error,
	failedAt time.Time,
) error {
	lastError := refundCompletionNotificationErrorText(
		cause,
	)

	if delivery.AttemptCount >= delivery.MaxAttempts {
		return u.markPermanentProcessingFailureText(
			ctx,
			delivery,
			lastError,
			failedAt,
		)
	}

	nextAttemptAt := failedAt.Add(
		u.retryDelayForAttempt(
			delivery.AttemptCount,
		),
	)

	retryDelivery, err := delivery.MarkRetryableFailed(
		lastError,
		nextAttemptAt,
		failedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"build retryable refund completion notification %q: %w",
			delivery.ID,
			err,
		)
	}

	if err := u.deliveryRepo.MarkRetryableFailed(
		ctx,
		delivery.ID,
		delivery.AttemptCount,
		lastError,
		nextAttemptAt,
		failedAt,
	); err != nil {
		return fmt.Errorf(
			"mark refund completion notification %q retryable failed: %w",
			delivery.ID,
			err,
		)
	}

	if u.queue == nil {
		// retryable_failedは保存済みなので、
		// DispatchDueから後で回収できる。
		return fmt.Errorf(
			"refund completion notification queue is not configured",
		)
	}

	if err := u.queue.EnqueueRefundCompletionNotification(
		ctx,
		retryDelivery,
	); err != nil {
		// nextAttemptAtを含むoutbox stateは保存済み。
		return fmt.Errorf(
			"enqueue retryable refund completion notification %q: %w",
			delivery.ID,
			err,
		)
	}

	return nil
}

func (u *RefundCompletionNotificationUsecase) markPermanentProcessingFailure(
	ctx context.Context,
	delivery refunddom.CompletionNotificationDelivery,
	cause error,
) error {
	return u.markPermanentProcessingFailureAt(
		ctx,
		delivery,
		cause,
		u.currentTime(),
	)
}

func (u *RefundCompletionNotificationUsecase) markPermanentProcessingFailureAt(
	ctx context.Context,
	delivery refunddom.CompletionNotificationDelivery,
	cause error,
	failedAt time.Time,
) error {
	return u.markPermanentProcessingFailureText(
		ctx,
		delivery,
		refundCompletionNotificationErrorText(cause),
		failedAt,
	)
}

func (u *RefundCompletionNotificationUsecase) markPermanentProcessingFailureText(
	ctx context.Context,
	delivery refunddom.CompletionNotificationDelivery,
	lastError string,
	failedAt time.Time,
) error {
	if err := u.deliveryRepo.MarkFailed(
		ctx,
		delivery.ID,
		delivery.AttemptCount,
		lastError,
		failedAt,
	); err != nil {
		return fmt.Errorf(
			"mark refund completion notification %q failed: %w",
			delivery.ID,
			err,
		)
	}

	// failedは通知処理上の終端状態。
	// Refund自体の金融状態には影響させない。
	return nil
}

// ==============================
// Helpers
// ==============================

func sameRefundCompletionNotificationScope(
	left refunddom.CompletionNotificationDelivery,
	right refunddom.CompletionNotificationDelivery,
) bool {
	return left.ID == right.ID &&
		left.PaymentID == right.PaymentID &&
		left.OrderID == right.OrderID &&
		left.UserID == right.UserID &&
		left.StripeRefundID == right.StripeRefundID &&
		left.RefundedAmount == right.RefundedAmount
}

func (u *RefundCompletionNotificationUsecase) currentTime() time.Time {
	if u != nil &&
		u.now != nil {
		return u.now().UTC()
	}

	return time.Now().UTC()
}

func (u *RefundCompletionNotificationUsecase) normalizedLeaseDuration() time.Duration {
	if u == nil ||
		u.leaseDuration <= 0 {
		return defaultRefundCompletionNotificationLeaseDuration
	}

	return u.leaseDuration
}

func (u *RefundCompletionNotificationUsecase) retryDelayForAttempt(
	attemptCount int,
) time.Duration {
	baseDelay := defaultRefundCompletionNotificationRetryDelay
	maxDelay := maxRefundCompletionNotificationRetryDelay

	if u != nil {
		if u.retryDelay > 0 {
			baseDelay = u.retryDelay
		}

		if u.maxRetryDelay > 0 {
			maxDelay = u.maxRetryDelay
		}
	}

	if maxDelay < baseDelay {
		maxDelay = baseDelay
	}

	if attemptCount <= 1 {
		return baseDelay
	}

	delay := baseDelay

	for attempt := 1; attempt < attemptCount; attempt++ {
		if delay >= maxDelay {
			return maxDelay
		}

		if delay > maxDelay/2 {
			return maxDelay
		}

		delay *= 2
	}

	if delay > maxDelay {
		return maxDelay
	}

	return delay
}

func normalizeRefundCompletionNotificationDispatchLimit(
	limit int,
) int {
	if limit <= 0 {
		return defaultRefundCompletionNotificationDispatchLimit
	}

	if limit > maxRefundCompletionNotificationDispatchLimit {
		return maxRefundCompletionNotificationDispatchLimit
	}

	return limit
}

func refundCompletionNotificationErrorText(
	err error,
) string {
	if err == nil {
		return "refund completion notification failed"
	}

	message := strings.TrimSpace(
		err.Error(),
	)
	if message == "" {
		message =
			"refund completion notification failed"
	}

	runes := []rune(message)
	if len(runes) <=
		maxRefundCompletionNotificationErrorLength {
		return message
	}

	return string(
		runes[:maxRefundCompletionNotificationErrorLength],
	)
}
