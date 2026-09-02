// backend/internal/application/usecase/resale_payout_notification_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	bankpayoutdom "narratives/internal/domain/bankPayout"
)

const (
	defaultResalePayoutNotificationDispatchLimit = 50
	maxResalePayoutNotificationDispatchLimit     = 200

	defaultResalePayoutNotificationLeaseDuration = 5 * time.Minute
	defaultResalePayoutNotificationRetryDelay    = time.Minute
	maxResalePayoutNotificationRetryDelay        = 6 * time.Hour

	maxResalePayoutNotificationErrorLength = 2000
)

// ==============================
// Inbound Port
// ==============================

type ResalePayoutNotificationUsecasePort interface {
	EnsureDelivery(
		ctx context.Context,
		in EnsureResalePayoutNotificationInput,
	) (bankpayoutdom.PayoutNotificationDelivery, error)

	DispatchDue(
		ctx context.Context,
		limit int,
	) (int, error)

	Process(
		ctx context.Context,
		deliveryID string,
	) error
}

type EnsureResalePayoutNotificationInput struct {
	Payout bankpayoutdom.BankPayout
}

// ==============================
// Usecase
// ==============================

type ResalePayoutNotificationUsecase struct {
	deliveryRepo applicationport.ResalePayoutNotificationRepositoryPort
	authUser     applicationport.AuthUserReader
	mailer       applicationport.ResalePayoutNotificationMailerPort
	queue        applicationport.ResalePayoutNotificationQueuePort

	now           func() time.Time
	leaseDuration time.Duration
	retryDelay    time.Duration
	maxRetryDelay time.Duration
}

var _ ResalePayoutNotificationUsecasePort = (*ResalePayoutNotificationUsecase)(nil)

func NewResalePayoutNotificationUsecase(
	deliveryRepo applicationport.ResalePayoutNotificationRepositoryPort,
	authUser applicationport.AuthUserReader,
	mailer applicationport.ResalePayoutNotificationMailerPort,
	queue applicationport.ResalePayoutNotificationQueuePort,
) *ResalePayoutNotificationUsecase {
	return &ResalePayoutNotificationUsecase{
		deliveryRepo: deliveryRepo,
		authUser:     authUser,
		mailer:       mailer,
		queue:        queue,

		now:           time.Now,
		leaseDuration: defaultResalePayoutNotificationLeaseDuration,
		retryDelay:    defaultResalePayoutNotificationRetryDelay,
		maxRetryDelay: maxResalePayoutNotificationRetryDelay,
	}
}

// ==============================
// Ensure Delivery
// ==============================

// EnsureDelivery creates the durable seller-email delivery after a BankPayout
// has reached paid.
//
// The caller must invoke this only after the matching SalesReceivable has also
// been reconciled to paid. This usecase never changes payout or receivable
// financial state.
func (u *ResalePayoutNotificationUsecase) EnsureDelivery(
	ctx context.Context,
	in EnsureResalePayoutNotificationInput,
) (bankpayoutdom.PayoutNotificationDelivery, error) {
	if u == nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf("resale payout notification usecase is nil")
	}
	if u.deliveryRepo == nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf("resale payout notification repository is not configured")
	}

	payout := in.Payout
	if err := payout.Validate(); err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf("validate bank payout before notification: %w", err)
	}
	if payout.Status != bankpayoutdom.StatusPaid ||
		payout.PaidAt == nil ||
		payout.PaidAt.IsZero() {
		return bankpayoutdom.PayoutNotificationDelivery{},
			bankpayoutdom.ErrInvalidStatus
	}

	paidAt := payout.PaidAt.UTC()
	createdAt := u.currentTime()
	if createdAt.Before(paidAt) {
		createdAt = paidAt
	}

	candidate, err := bankpayoutdom.NewPayoutNotificationDelivery(
		payout.ID,
		payout.SalesReceivableID,
		payout.OrderID,
		payout.ResaleID,
		payout.SellerUserID,
		payout.Amount,
		payout.Currency,
		payout.BankDestination.BankName,
		payout.BankDestination.BranchName,
		payout.BankDestination.BankLast4,
		paidAt,
		createdAt,
		bankpayoutdom.DefaultPayoutNotificationMaxAttempts,
	)
	if err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf("create resale payout notification delivery: %w", err)
	}

	delivery, _, err := u.deliveryRepo.CreateIfAbsent(
		ctx,
		candidate,
	)
	if err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf(
				"create resale payout notification delivery if absent: %w",
				err,
			)
	}

	delivery, err = delivery.Normalize()
	if err != nil {
		return bankpayoutdom.PayoutNotificationDelivery{},
			fmt.Errorf(
				"normalize resale payout notification delivery %q: %w",
				delivery.ID,
				err,
			)
	}

	if !sameResalePayoutNotificationScope(
		delivery,
		candidate,
	) {
		return bankpayoutdom.PayoutNotificationDelivery{},
			bankpayoutdom.ErrConflict
	}

	switch delivery.Status {
	case bankpayoutdom.PayoutNotificationStatusPending,
		bankpayoutdom.PayoutNotificationStatusRetryableFailed:
		if !delivery.IsDue(createdAt) {
			return delivery, nil
		}

		if u.queue == nil {
			return delivery,
				fmt.Errorf("resale payout notification queue is not configured")
		}

		if err := u.queue.EnqueueResalePayoutNotification(
			ctx,
			delivery,
		); err != nil {
			// The durable delivery already exists. DispatchDue can recover it.
			return delivery,
				fmt.Errorf(
					"enqueue resale payout notification %q: %w",
					delivery.ID,
					err,
				)
		}

	case bankpayoutdom.PayoutNotificationStatusProcessing,
		bankpayoutdom.PayoutNotificationStatusDelivered,
		bankpayoutdom.PayoutNotificationStatusFailed:
		return delivery, nil
	}

	return delivery, nil
}

// ==============================
// Dispatch Due
// ==============================

func (u *ResalePayoutNotificationUsecase) DispatchDue(
	ctx context.Context,
	limit int,
) (int, error) {
	if u == nil {
		return 0,
			fmt.Errorf("resale payout notification usecase is nil")
	}
	if u.deliveryRepo == nil {
		return 0,
			fmt.Errorf("resale payout notification repository is not configured")
	}
	if u.queue == nil {
		return 0,
			fmt.Errorf("resale payout notification queue is not configured")
	}

	limit = normalizeResalePayoutNotificationDispatchLimit(limit)
	now := u.currentTime()

	deliveries, err := u.deliveryRepo.ListDue(
		ctx,
		now,
		limit,
	)
	if err != nil {
		return 0,
			fmt.Errorf("list due resale payout notifications: %w", err)
	}

	enqueuedCount := 0
	var firstErr error

	for _, delivery := range deliveries {
		normalized, err := delivery.Normalize()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"normalize resale payout notification %q: %w",
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

		if err := u.queue.EnqueueResalePayoutNotification(
			ctx,
			normalized,
		); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"enqueue resale payout notification %q: %w",
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

func (u *ResalePayoutNotificationUsecase) Process(
	ctx context.Context,
	deliveryID string,
) error {
	if u == nil {
		return fmt.Errorf("resale payout notification usecase is nil")
	}
	if u.deliveryRepo == nil {
		return fmt.Errorf("resale payout notification repository is not configured")
	}
	if u.authUser == nil {
		return fmt.Errorf("auth user reader is not configured")
	}
	if u.mailer == nil {
		return fmt.Errorf("resale payout notification mailer is not configured")
	}

	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return bankpayoutdom.ErrPayoutNotificationDeliveryIDRequired
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
		case errors.Is(
			err,
			bankpayoutdom.ErrNotFound,
		):
			return nil

		case errors.Is(
			err,
			bankpayoutdom.ErrPayoutNotificationNotClaimable,
		):
			return nil

		case errors.Is(
			err,
			bankpayoutdom.ErrPayoutNotificationAttemptLimit,
		):
			return nil

		default:
			return fmt.Errorf(
				"claim resale payout notification %q: %w",
				deliveryID,
				err,
			)
		}
	}

	delivery, err = delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize claimed resale payout notification %q: %w",
			deliveryID,
			err,
		)
	}

	if delivery.Status !=
		bankpayoutdom.PayoutNotificationStatusProcessing {
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
				"resolve seller email for user %q: %w",
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
				"seller email is empty for user %q",
				delivery.UserID,
			),
		)
	}

	result, sendErr :=
		u.mailer.SendResalePayoutNotification(
			ctx,
			applicationport.ResalePayoutNotificationMailMessage{
				IdempotencyKey: delivery.ID,
				ToEmail:        toEmail,

				BankPayoutID:      delivery.BankPayoutID,
				SalesReceivableID: delivery.SalesReceivableID,
				OrderID:           delivery.OrderID,
				ResaleID:          delivery.ResaleID,

				Amount:   delivery.Amount,
				Currency: delivery.Currency,

				BankName:   delivery.BankName,
				BranchName: delivery.BranchName,
				BankLast4:  delivery.BankLast4,

				PaidAt: delivery.PaidAt,
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
				"mark resale payout notification %q delivered: %w",
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

func (u *ResalePayoutNotificationUsecase) markRetryableProcessingFailure(
	ctx context.Context,
	delivery bankpayoutdom.PayoutNotificationDelivery,
	cause error,
) error {
	return u.markRetryableProcessingFailureAt(
		ctx,
		delivery,
		cause,
		u.currentTime(),
	)
}

func (u *ResalePayoutNotificationUsecase) markRetryableProcessingFailureAt(
	ctx context.Context,
	delivery bankpayoutdom.PayoutNotificationDelivery,
	cause error,
	failedAt time.Time,
) error {
	lastError := resalePayoutNotificationErrorText(
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
			"build retryable resale payout notification %q: %w",
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
			"mark resale payout notification %q retryable failed: %w",
			delivery.ID,
			err,
		)
	}

	if u.queue == nil {
		// retryable_failed is durable and can later be recovered by DispatchDue.
		return fmt.Errorf(
			"resale payout notification queue is not configured",
		)
	}

	if err := u.queue.EnqueueResalePayoutNotification(
		ctx,
		retryDelivery,
	); err != nil {
		// nextAttemptAt is already persisted, so DispatchDue can recover it.
		return fmt.Errorf(
			"enqueue retryable resale payout notification %q: %w",
			delivery.ID,
			err,
		)
	}

	return nil
}

func (u *ResalePayoutNotificationUsecase) markPermanentProcessingFailure(
	ctx context.Context,
	delivery bankpayoutdom.PayoutNotificationDelivery,
	cause error,
) error {
	return u.markPermanentProcessingFailureAt(
		ctx,
		delivery,
		cause,
		u.currentTime(),
	)
}

func (u *ResalePayoutNotificationUsecase) markPermanentProcessingFailureAt(
	ctx context.Context,
	delivery bankpayoutdom.PayoutNotificationDelivery,
	cause error,
	failedAt time.Time,
) error {
	return u.markPermanentProcessingFailureText(
		ctx,
		delivery,
		resalePayoutNotificationErrorText(cause),
		failedAt,
	)
}

func (u *ResalePayoutNotificationUsecase) markPermanentProcessingFailureText(
	ctx context.Context,
	delivery bankpayoutdom.PayoutNotificationDelivery,
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
			"mark resale payout notification %q failed: %w",
			delivery.ID,
			err,
		)
	}

	// Notification failure is terminal only for email delivery. It never
	// changes the already-completed BankPayout or SalesReceivable.
	return nil
}

// ==============================
// Helpers
// ==============================

func sameResalePayoutNotificationScope(
	left bankpayoutdom.PayoutNotificationDelivery,
	right bankpayoutdom.PayoutNotificationDelivery,
) bool {
	return left.ID == right.ID &&
		left.BankPayoutID == right.BankPayoutID &&
		left.SalesReceivableID == right.SalesReceivableID &&
		left.OrderID == right.OrderID &&
		left.ResaleID == right.ResaleID &&
		left.UserID == right.UserID &&
		left.Amount == right.Amount &&
		left.Currency == right.Currency &&
		left.BankName == right.BankName &&
		left.BranchName == right.BranchName &&
		left.BankLast4 == right.BankLast4 &&
		left.PaidAt.Equal(right.PaidAt)
}

func (u *ResalePayoutNotificationUsecase) currentTime() time.Time {
	if u != nil && u.now != nil {
		return u.now().UTC()
	}

	return time.Now().UTC()
}

func (u *ResalePayoutNotificationUsecase) normalizedLeaseDuration() time.Duration {
	if u == nil || u.leaseDuration <= 0 {
		return defaultResalePayoutNotificationLeaseDuration
	}

	return u.leaseDuration
}

func (u *ResalePayoutNotificationUsecase) retryDelayForAttempt(
	attemptCount int,
) time.Duration {
	baseDelay := defaultResalePayoutNotificationRetryDelay
	maxDelay := maxResalePayoutNotificationRetryDelay

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

func normalizeResalePayoutNotificationDispatchLimit(
	limit int,
) int {
	if limit <= 0 {
		return defaultResalePayoutNotificationDispatchLimit
	}

	if limit > maxResalePayoutNotificationDispatchLimit {
		return maxResalePayoutNotificationDispatchLimit
	}

	return limit
}

func resalePayoutNotificationErrorText(
	err error,
) string {
	if err == nil {
		return "resale payout notification failed"
	}

	message := strings.TrimSpace(
		err.Error(),
	)
	if message == "" {
		message = "resale payout notification failed"
	}

	runes := []rune(message)
	if len(runes) <= maxResalePayoutNotificationErrorLength {
		return message
	}

	return string(
		runes[:maxResalePayoutNotificationErrorLength],
	)
}
