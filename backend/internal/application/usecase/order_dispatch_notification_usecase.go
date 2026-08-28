// backend/internal/application/usecase/order_dispatch_notification_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	orderdom "narratives/internal/domain/order"
)

const (
	defaultOrderDispatchNotificationDispatchLimit = 50
	maxOrderDispatchNotificationDispatchLimit     = 200

	defaultOrderDispatchNotificationLeaseDuration = 5 * time.Minute
	defaultOrderDispatchNotificationRetryDelay    = time.Minute
	maxOrderDispatchNotificationRetryDelay        = 6 * time.Hour

	maxOrderDispatchNotificationErrorLength = 2000
)

// ==============================
// Inbound Port
// ==============================

type OrderDispatchNotificationUsecasePort interface {
	EnsureDelivery(
		ctx context.Context,
		order orderdom.Order,
		targetItems []orderdom.OrderItemSnapshot,
	) (orderdom.DispatchNotificationDelivery, error)

	DispatchDue(
		ctx context.Context,
		limit int,
	) (int, error)

	Process(
		ctx context.Context,
		deliveryID string,
	) error
}

// ==============================
// Usecase
// ==============================

type OrderDispatchNotificationUsecase struct {
	deliveryRepo orderdom.DispatchNotificationRepository

	authUser applicationport.AuthUserReader

	companyIDFromContext applicationport.CompanyIDResolver

	productBlueprint applicationport.ProductBlueprintGetter

	mailer applicationport.OrderDispatchNotificationMailerPort
	queue  applicationport.OrderDispatchNotificationQueuePort

	now           func() time.Time
	leaseDuration time.Duration
	retryDelay    time.Duration
	maxRetryDelay time.Duration
}

func NewOrderDispatchNotificationUsecase(
	deliveryRepo orderdom.DispatchNotificationRepository,
	authUser applicationport.AuthUserReader,
	companyIDFromContext applicationport.CompanyIDResolver,
	productBlueprint applicationport.ProductBlueprintGetter,
	mailer applicationport.OrderDispatchNotificationMailerPort,
	queue applicationport.OrderDispatchNotificationQueuePort,
) *OrderDispatchNotificationUsecase {
	return &OrderDispatchNotificationUsecase{
		deliveryRepo:         deliveryRepo,
		authUser:             authUser,
		companyIDFromContext: companyIDFromContext,
		productBlueprint:     productBlueprint,
		mailer:               mailer,
		queue:                queue,

		now:           time.Now,
		leaseDuration: defaultOrderDispatchNotificationLeaseDuration,
		retryDelay:    defaultOrderDispatchNotificationRetryDelay,
		maxRetryDelay: maxOrderDispatchNotificationRetryDelay,
	}
}

// ==============================
// Ensure Delivery
// ==============================

func (u *OrderDispatchNotificationUsecase) EnsureDelivery(
	ctx context.Context,
	order orderdom.Order,
	targetItems []orderdom.OrderItemSnapshot,
) (orderdom.DispatchNotificationDelivery, error) {
	if u == nil {
		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf("order dispatch notification usecase is nil")
	}

	if u.deliveryRepo == nil {
		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf("order dispatch notification repository is not configured")
	}

	if u.companyIDFromContext == nil {
		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf("company ID resolver is not configured")
	}

	if order.ID == "" {
		return orderdom.DispatchNotificationDelivery{},
			orderdom.ErrInvalidID
	}

	if order.UserID == "" {
		return orderdom.DispatchNotificationDelivery{},
			orderdom.ErrInvalidUserID
	}

	if len(targetItems) == 0 {
		return orderdom.DispatchNotificationDelivery{},
			orderdom.ErrDispatchNotificationItemsRequired
	}

	companyID := strings.TrimSpace(
		u.companyIDFromContext(ctx),
	)
	if companyID == "" {
		return orderdom.DispatchNotificationDelivery{},
			orderdom.ErrDispatchNotificationCompanyIDRequired
	}

	items := make(
		[]orderdom.DispatchNotificationItem,
		0,
		len(targetItems),
	)

	for _, targetItem := range targetItems {
		if targetItem.IsCancelled {
			return orderdom.DispatchNotificationDelivery{},
				fmt.Errorf(
					"%w: canceled target item",
					orderdom.ErrDispatchNotificationItemInvalid,
				)
		}

		if !targetItem.IsDispatched {
			return orderdom.DispatchNotificationDelivery{},
				fmt.Errorf(
					"%w: target item is not dispatched",
					orderdom.ErrDispatchNotificationItemInvalid,
				)
		}

		item, err := orderdom.NewDispatchNotificationItem(
			targetItem.InventoryID,
			targetItem.ListID,
			targetItem.ProductBlueprintID,
			targetItem.TokenBlueprintID,
			targetItem.Qty,
		)
		if err != nil {
			return orderdom.DispatchNotificationDelivery{}, err
		}

		items = append(items, item)
	}

	now := u.currentTime()

	candidate, err := orderdom.NewDispatchNotificationDelivery(
		order.ID,
		companyID,
		order.UserID,
		items,
		now,
		orderdom.DefaultDispatchNotificationMaxAttempts,
	)
	if err != nil {
		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf(
				"create order dispatch notification delivery: %w",
				err,
			)
	}

	delivery, _, err := u.deliveryRepo.CreateIfAbsent(
		ctx,
		candidate,
	)
	if err != nil {
		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf(
				"create order dispatch notification delivery if absent: %w",
				err,
			)
	}

	delivery, err = delivery.Normalize()
	if err != nil {
		return orderdom.DispatchNotificationDelivery{},
			fmt.Errorf(
				"normalize order dispatch notification delivery %q: %w",
				delivery.ID,
				err,
			)
	}

	if !sameOrderDispatchNotificationScope(
		delivery,
		candidate,
	) {
		return orderdom.DispatchNotificationDelivery{},
			orderdom.ErrConflict
	}

	switch delivery.Status {
	case orderdom.DispatchNotificationStatusPending,
		orderdom.DispatchNotificationStatusRetryableFailed:
		if !delivery.IsDue(now) {
			return delivery, nil
		}

		if u.queue == nil {
			return delivery,
				fmt.Errorf(
					"order dispatch notification queue is not configured",
				)
		}

		if err := u.queue.EnqueueOrderDispatchNotification(
			ctx,
			delivery,
		); err != nil {
			// delivery自体はFirestoreへ保存済みです。
			// 後続のDispatchDueでも回収できます。
			return delivery,
				fmt.Errorf(
					"enqueue order dispatch notification %q: %w",
					delivery.ID,
					err,
				)
		}

	case orderdom.DispatchNotificationStatusProcessing,
		orderdom.DispatchNotificationStatusDelivered,
		orderdom.DispatchNotificationStatusFailed:
		return delivery, nil
	}

	return delivery, nil
}

// ==============================
// Dispatch Due
// ==============================

func (u *OrderDispatchNotificationUsecase) DispatchDue(
	ctx context.Context,
	limit int,
) (int, error) {
	if u == nil {
		return 0, fmt.Errorf(
			"order dispatch notification usecase is nil",
		)
	}

	if u.deliveryRepo == nil {
		return 0, fmt.Errorf(
			"order dispatch notification repository is not configured",
		)
	}

	if u.queue == nil {
		return 0, fmt.Errorf(
			"order dispatch notification queue is not configured",
		)
	}

	limit = normalizeOrderDispatchNotificationDispatchLimit(limit)
	now := u.currentTime()

	deliveries, err := u.deliveryRepo.ListDue(
		ctx,
		now,
		limit,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"list due order dispatch notifications: %w",
			err,
		)
	}

	enqueuedCount := 0
	var firstErr error

	for _, delivery := range deliveries {
		normalized, err := delivery.Normalize()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"normalize order dispatch notification %q: %w",
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

		if err := u.queue.EnqueueOrderDispatchNotification(
			ctx,
			normalized,
		); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"enqueue order dispatch notification %q: %w",
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

func (u *OrderDispatchNotificationUsecase) Process(
	ctx context.Context,
	deliveryID string,
) error {
	if u == nil {
		return fmt.Errorf(
			"order dispatch notification usecase is nil",
		)
	}

	if u.deliveryRepo == nil {
		return fmt.Errorf(
			"order dispatch notification repository is not configured",
		)
	}

	if u.authUser == nil {
		return fmt.Errorf(
			"auth user reader is not configured",
		)
	}

	if u.productBlueprint == nil {
		return fmt.Errorf(
			"product blueprint reader is not configured",
		)
	}

	if u.mailer == nil {
		return fmt.Errorf(
			"order dispatch notification mailer is not configured",
		)
	}

	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" {
		return orderdom.ErrDispatchNotificationDeliveryIDRequired
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
		case errors.Is(err, orderdom.ErrNotFound):
			return nil

		case errors.Is(
			err,
			orderdom.ErrDispatchNotificationNotClaimable,
		):
			return nil

		case errors.Is(
			err,
			orderdom.ErrDispatchNotificationAttemptLimit,
		):
			return nil

		default:
			return fmt.Errorf(
				"claim order dispatch notification %q: %w",
				deliveryID,
				err,
			)
		}
	}

	delivery, err = delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize claimed order dispatch notification %q: %w",
			deliveryID,
			err,
		)
	}

	if delivery.Status !=
		orderdom.DispatchNotificationStatusProcessing {
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

	mailItems, err := u.resolveOrderDispatchNotificationMailItems(
		ctx,
		delivery,
	)
	if err != nil {
		var permanentError *orderDispatchNotificationPermanentError
		if errors.As(err, &permanentError) {
			return u.markPermanentProcessingFailure(
				ctx,
				delivery,
				err,
			)
		}

		return u.markRetryableProcessingFailure(
			ctx,
			delivery,
			err,
		)
	}

	result, sendErr := u.mailer.SendOrderDispatchNotification(
		ctx,
		applicationport.OrderDispatchNotificationMailMessage{
			IdempotencyKey: delivery.ID,
			ToEmail:        toEmail,
			OrderID:        delivery.OrderID,
			Items:          mailItems,
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
				"mark order dispatch notification %q delivered: %w",
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

func (
	u *OrderDispatchNotificationUsecase,
) resolveOrderDispatchNotificationMailItems(
	ctx context.Context,
	delivery orderdom.DispatchNotificationDelivery,
) ([]applicationport.OrderDispatchNotificationMailItem, error) {
	productNames := make(
		map[string]string,
		len(delivery.Items),
	)

	result := make(
		[]applicationport.OrderDispatchNotificationMailItem,
		0,
		len(delivery.Items),
	)

	for _, item := range delivery.Items {
		productName, ok := productNames[item.ProductBlueprintID]

		if !ok {
			productBlueprint, err := u.productBlueprint.GetByID(
				ctx,
				item.ProductBlueprintID,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"get product blueprint %q for dispatch notification: %w",
					item.ProductBlueprintID,
					err,
				)
			}

			if productBlueprint.ID != item.ProductBlueprintID {
				return nil,
					newOrderDispatchNotificationPermanentError(
						fmt.Errorf(
							"product blueprint ID mismatch: expected %q, got %q",
							item.ProductBlueprintID,
							productBlueprint.ID,
						),
					)
			}

			if strings.TrimSpace(
				productBlueprint.CompanyID,
			) != delivery.CompanyID {
				return nil,
					newOrderDispatchNotificationPermanentError(
						fmt.Errorf(
							"product blueprint %q does not belong to company %q",
							item.ProductBlueprintID,
							delivery.CompanyID,
						),
					)
			}

			productName = strings.TrimSpace(
				productBlueprint.ProductName,
			)
			if productName == "" {
				return nil,
					newOrderDispatchNotificationPermanentError(
						fmt.Errorf(
							"product name is empty for product blueprint %q",
							item.ProductBlueprintID,
						),
					)
			}

			productNames[item.ProductBlueprintID] =
				productName
		}

		result = append(
			result,
			applicationport.OrderDispatchNotificationMailItem{
				ProductName: productName,
				Qty:         item.Qty,
			},
		)
	}

	if len(result) == 0 {
		return nil,
			newOrderDispatchNotificationPermanentError(
				orderdom.ErrDispatchNotificationItemsRequired,
			)
	}

	return result, nil
}

func (u *OrderDispatchNotificationUsecase) markRetryableProcessingFailure(
	ctx context.Context,
	delivery orderdom.DispatchNotificationDelivery,
	cause error,
) error {
	return u.markRetryableProcessingFailureAt(
		ctx,
		delivery,
		cause,
		u.currentTime(),
	)
}

func (u *OrderDispatchNotificationUsecase) markRetryableProcessingFailureAt(
	ctx context.Context,
	delivery orderdom.DispatchNotificationDelivery,
	cause error,
	failedAt time.Time,
) error {
	lastError := orderDispatchNotificationErrorText(cause)

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
			"build retryable order dispatch notification %q: %w",
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
			"mark order dispatch notification %q retryable failed: %w",
			delivery.ID,
			err,
		)
	}

	if u.queue == nil {
		// retryable_failedは保存済みなので、
		// DispatchDueから後で回収できます。
		return fmt.Errorf(
			"order dispatch notification queue is not configured",
		)
	}

	if err := u.queue.EnqueueOrderDispatchNotification(
		ctx,
		retryDelivery,
	); err != nil {
		// nextAttemptAtを含むoutbox stateは保存済みです。
		return fmt.Errorf(
			"enqueue retryable order dispatch notification %q: %w",
			delivery.ID,
			err,
		)
	}

	return nil
}

func (u *OrderDispatchNotificationUsecase) markPermanentProcessingFailure(
	ctx context.Context,
	delivery orderdom.DispatchNotificationDelivery,
	cause error,
) error {
	return u.markPermanentProcessingFailureAt(
		ctx,
		delivery,
		cause,
		u.currentTime(),
	)
}

func (u *OrderDispatchNotificationUsecase) markPermanentProcessingFailureAt(
	ctx context.Context,
	delivery orderdom.DispatchNotificationDelivery,
	cause error,
	failedAt time.Time,
) error {
	return u.markPermanentProcessingFailureText(
		ctx,
		delivery,
		orderDispatchNotificationErrorText(cause),
		failedAt,
	)
}

func (u *OrderDispatchNotificationUsecase) markPermanentProcessingFailureText(
	ctx context.Context,
	delivery orderdom.DispatchNotificationDelivery,
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
			"mark order dispatch notification %q failed: %w",
			delivery.ID,
			err,
		)
	}

	// failedはbusiness上の終端状態です。
	// Cloud Tasks側で同じ処理を再試行させません。
	return nil
}

// ==============================
// Helpers
// ==============================

type orderDispatchNotificationPermanentError struct {
	cause error
}

func newOrderDispatchNotificationPermanentError(
	cause error,
) error {
	return &orderDispatchNotificationPermanentError{
		cause: cause,
	}
}

func (
	e *orderDispatchNotificationPermanentError,
) Error() string {
	if e == nil || e.cause == nil {
		return "order dispatch notification permanent error"
	}

	return e.cause.Error()
}

func (
	e *orderDispatchNotificationPermanentError,
) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func sameOrderDispatchNotificationScope(
	left orderdom.DispatchNotificationDelivery,
	right orderdom.DispatchNotificationDelivery,
) bool {
	if left.ID != right.ID ||
		left.OrderID != right.OrderID ||
		left.CompanyID != right.CompanyID ||
		left.UserID != right.UserID ||
		len(left.Items) != len(right.Items) {
		return false
	}

	for index := range left.Items {
		leftItem := left.Items[index]
		rightItem := right.Items[index]

		if leftItem.InventoryID != rightItem.InventoryID ||
			leftItem.ListID != rightItem.ListID ||
			leftItem.ProductBlueprintID != rightItem.ProductBlueprintID ||
			leftItem.TokenBlueprintID != rightItem.TokenBlueprintID ||
			leftItem.Qty != rightItem.Qty {
			return false
		}
	}

	return true
}

func (u *OrderDispatchNotificationUsecase) currentTime() time.Time {
	if u != nil && u.now != nil {
		return u.now().UTC()
	}

	return time.Now().UTC()
}

func (u *OrderDispatchNotificationUsecase) normalizedLeaseDuration() time.Duration {
	if u == nil || u.leaseDuration <= 0 {
		return defaultOrderDispatchNotificationLeaseDuration
	}

	return u.leaseDuration
}

func (u *OrderDispatchNotificationUsecase) retryDelayForAttempt(
	attemptCount int,
) time.Duration {
	baseDelay := defaultOrderDispatchNotificationRetryDelay
	maxDelay := maxOrderDispatchNotificationRetryDelay

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

func normalizeOrderDispatchNotificationDispatchLimit(
	limit int,
) int {
	if limit <= 0 {
		return defaultOrderDispatchNotificationDispatchLimit
	}

	if limit > maxOrderDispatchNotificationDispatchLimit {
		return maxOrderDispatchNotificationDispatchLimit
	}

	return limit
}

func orderDispatchNotificationErrorText(
	err error,
) string {
	if err == nil {
		return "order dispatch notification failed"
	}

	message := strings.TrimSpace(
		err.Error(),
	)
	if message == "" {
		message = "order dispatch notification failed"
	}

	runes := []rune(message)
	if len(runes) <= maxOrderDispatchNotificationErrorLength {
		return message
	}

	return string(
		runes[:maxOrderDispatchNotificationErrorLength],
	)
}
