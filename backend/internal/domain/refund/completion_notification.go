// backend/internal/domain/refund/completion_notification.go
package refund

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const DefaultCompletionNotificationMaxAttempts = 5

type CompletionNotificationStatus string

const (
	CompletionNotificationStatusPending         CompletionNotificationStatus = "pending"
	CompletionNotificationStatusProcessing      CompletionNotificationStatus = "processing"
	CompletionNotificationStatusRetryableFailed CompletionNotificationStatus = "retryable_failed"
	CompletionNotificationStatusDelivered       CompletionNotificationStatus = "delivered"
	CompletionNotificationStatusFailed          CompletionNotificationStatus = "failed"
)

var (
	ErrCompletionNotificationDeliveryIDRequired = errors.New(
		"refund: completion notification deliveryID is required",
	)
	ErrCompletionNotificationDeliveryIDInvalid = errors.New(
		"refund: completion notification deliveryID is invalid",
	)
	ErrCompletionNotificationPaymentIDRequired = errors.New(
		"refund: completion notification paymentID is required",
	)
	ErrCompletionNotificationOrderIDRequired = errors.New(
		"refund: completion notification orderID is required",
	)
	ErrCompletionNotificationUserIDRequired = errors.New(
		"refund: completion notification userID is required",
	)
	ErrCompletionNotificationStripeRefundIDRequired = errors.New(
		"refund: completion notification stripeRefundID is required",
	)
	ErrCompletionNotificationStripeRefundIDInvalid = errors.New(
		"refund: completion notification stripeRefundID is invalid",
	)
	ErrCompletionNotificationRefundedAmountInvalid = errors.New(
		"refund: completion notification refundedAmount is invalid",
	)
	ErrCompletionNotificationStatusInvalid = errors.New(
		"refund: completion notification status is invalid",
	)
	ErrCompletionNotificationAttemptCountInvalid = errors.New(
		"refund: completion notification attempt count is invalid",
	)
	ErrCompletionNotificationMaxAttemptsInvalid = errors.New(
		"refund: completion notification max attempts is invalid",
	)
	ErrCompletionNotificationAttemptLimit = errors.New(
		"refund: completion notification attempt limit reached",
	)
	ErrCompletionNotificationNotClaimable = errors.New(
		"refund: completion notification is not claimable",
	)
	ErrCompletionNotificationLeaseInvalid = errors.New(
		"refund: completion notification lease is invalid",
	)
	ErrCompletionNotificationErrorRequired = errors.New(
		"refund: completion notification error is required",
	)
	ErrCompletionNotificationNextAttemptInvalid = errors.New(
		"refund: completion notification next attempt is invalid",
	)
)

type CompletionNotificationDelivery struct {
	ID string

	PaymentID string
	OrderID   string
	UserID    string

	StripeRefundID string
	RefundedAmount int

	Status       CompletionNotificationStatus
	AttemptCount int
	MaxAttempts  int

	ProviderMessageID string
	LastError         string

	CreatedAt time.Time
	UpdatedAt time.Time

	NextAttemptAt       *time.Time
	ProcessingStartedAt *time.Time
	ProcessingUntil     *time.Time
	DeliveredAt         *time.Time
	FailedAt            *time.Time
}

func BuildCompletionNotificationDeliveryID(
	paymentID string,
	stripeRefundID string,
) (string, error) {
	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return "",
			ErrCompletionNotificationPaymentIDRequired
	}

	stripeRefundID = strings.TrimSpace(stripeRefundID)
	if stripeRefundID == "" {
		return "",
			ErrCompletionNotificationStripeRefundIDRequired
	}

	if !isCompletionNotificationStripeRefundID(
		stripeRefundID,
	) {
		return "",
			ErrCompletionNotificationStripeRefundIDInvalid
	}

	sum := sha256.Sum256(
		[]byte(
			paymentID +
				"\x00" +
				stripeRefundID,
		),
	)

	return "refund_completion_" +
			hex.EncodeToString(sum[:]),
		nil
}

func NewCompletionNotificationDelivery(
	paymentID string,
	orderID string,
	userID string,
	stripeRefundID string,
	refundedAmount int,
	createdAt time.Time,
	maxAttempts int,
) (CompletionNotificationDelivery, error) {
	paymentID = strings.TrimSpace(paymentID)
	orderID = strings.TrimSpace(orderID)
	userID = strings.TrimSpace(userID)
	stripeRefundID = strings.TrimSpace(stripeRefundID)

	deliveryID, err :=
		BuildCompletionNotificationDeliveryID(
			paymentID,
			stripeRefundID,
		)
	if err != nil {
		return CompletionNotificationDelivery{},
			err
	}

	if orderID == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationOrderIDRequired
	}

	if userID == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationUserIDRequired
	}

	if refundedAmount <= 0 {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationRefundedAmountInvalid
	}

	if maxAttempts == 0 {
		maxAttempts =
			DefaultCompletionNotificationMaxAttempts
	}

	if maxAttempts < 1 {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationMaxAttemptsInvalid
	}

	createdAt =
		normalizeCompletionNotificationTime(
			createdAt,
		)

	return CompletionNotificationDelivery{
		ID:             deliveryID,
		PaymentID:      paymentID,
		OrderID:        orderID,
		UserID:         userID,
		StripeRefundID: stripeRefundID,
		RefundedAmount: refundedAmount,

		Status:       CompletionNotificationStatusPending,
		AttemptCount: 0,
		MaxAttempts:  maxAttempts,

		CreatedAt: createdAt,
		UpdatedAt: createdAt,

		NextAttemptAt: completionNotificationTimePointer(
			createdAt,
		),
	}, nil
}

func (
	d CompletionNotificationDelivery,
) Normalize() (
	CompletionNotificationDelivery,
	error,
) {
	d.ID = strings.TrimSpace(d.ID)
	if d.ID == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationDeliveryIDRequired
	}

	d.PaymentID =
		strings.TrimSpace(
			d.PaymentID,
		)
	if d.PaymentID == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationPaymentIDRequired
	}

	d.OrderID =
		strings.TrimSpace(
			d.OrderID,
		)
	if d.OrderID == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationOrderIDRequired
	}

	d.UserID =
		strings.TrimSpace(
			d.UserID,
		)
	if d.UserID == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationUserIDRequired
	}

	d.StripeRefundID =
		strings.TrimSpace(
			d.StripeRefundID,
		)
	if d.StripeRefundID == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationStripeRefundIDRequired
	}

	if !isCompletionNotificationStripeRefundID(
		d.StripeRefundID,
	) {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationStripeRefundIDInvalid
	}

	if d.RefundedAmount <= 0 {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationRefundedAmountInvalid
	}

	expectedID, err :=
		BuildCompletionNotificationDeliveryID(
			d.PaymentID,
			d.StripeRefundID,
		)
	if err != nil {
		return CompletionNotificationDelivery{},
			err
	}

	if d.ID != expectedID {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationDeliveryIDInvalid
	}

	d.Status =
		CompletionNotificationStatus(
			strings.TrimSpace(
				string(d.Status),
			),
		)

	if d.Status == "" {
		d.Status =
			CompletionNotificationStatusPending
	}

	if !d.Status.IsValid() {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationStatusInvalid
	}

	if d.AttemptCount < 0 {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationAttemptCountInvalid
	}

	if d.MaxAttempts == 0 {
		d.MaxAttempts =
			DefaultCompletionNotificationMaxAttempts
	}

	if d.MaxAttempts < 1 ||
		d.AttemptCount > d.MaxAttempts {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationMaxAttemptsInvalid
	}

	d.ProviderMessageID =
		strings.TrimSpace(
			d.ProviderMessageID,
		)

	d.LastError =
		strings.TrimSpace(
			d.LastError,
		)

	d.CreatedAt =
		normalizeCompletionNotificationTime(
			d.CreatedAt,
		)

	if d.UpdatedAt.IsZero() {
		d.UpdatedAt =
			d.CreatedAt
	} else {
		d.UpdatedAt =
			d.UpdatedAt.UTC()
	}

	d.NextAttemptAt =
		normalizeCompletionNotificationTimePointer(
			d.NextAttemptAt,
		)

	d.ProcessingStartedAt =
		normalizeCompletionNotificationTimePointer(
			d.ProcessingStartedAt,
		)

	d.ProcessingUntil =
		normalizeCompletionNotificationTimePointer(
			d.ProcessingUntil,
		)

	d.DeliveredAt =
		normalizeCompletionNotificationTimePointer(
			d.DeliveredAt,
		)

	d.FailedAt =
		normalizeCompletionNotificationTimePointer(
			d.FailedAt,
		)

	switch d.Status {
	case CompletionNotificationStatusProcessing:
		if d.ProcessingUntil == nil {
			return CompletionNotificationDelivery{},
				ErrCompletionNotificationLeaseInvalid
		}

	case CompletionNotificationStatusDelivered:
		if d.DeliveredAt == nil {
			return CompletionNotificationDelivery{},
				ErrCompletionNotificationStatusInvalid
		}

	case CompletionNotificationStatusRetryableFailed:
		if d.LastError == "" {
			return CompletionNotificationDelivery{},
				ErrCompletionNotificationErrorRequired
		}

		if d.NextAttemptAt == nil {
			return CompletionNotificationDelivery{},
				ErrCompletionNotificationNextAttemptInvalid
		}

	case CompletionNotificationStatusFailed:
		if d.LastError == "" {
			return CompletionNotificationDelivery{},
				ErrCompletionNotificationErrorRequired
		}

		if d.FailedAt == nil {
			return CompletionNotificationDelivery{},
				ErrCompletionNotificationStatusInvalid
		}
	}

	return d, nil
}

func (
	s CompletionNotificationStatus,
) IsValid() bool {
	switch s {
	case CompletionNotificationStatusPending,
		CompletionNotificationStatusProcessing,
		CompletionNotificationStatusRetryableFailed,
		CompletionNotificationStatusDelivered,
		CompletionNotificationStatusFailed:
		return true

	default:
		return false
	}
}

func (
	d CompletionNotificationDelivery,
) IsTerminal() bool {
	return d.Status ==
		CompletionNotificationStatusDelivered ||
		d.Status ==
			CompletionNotificationStatusFailed
}

func (
	d CompletionNotificationDelivery,
) IsDue(
	now time.Time,
) bool {
	now =
		normalizeCompletionNotificationTime(
			now,
		)

	switch d.Status {
	case CompletionNotificationStatusPending,
		CompletionNotificationStatusRetryableFailed:
		return d.NextAttemptAt == nil ||
			!now.Before(
				d.NextAttemptAt.UTC(),
			)

	case CompletionNotificationStatusProcessing:
		return d.ProcessingUntil != nil &&
			!now.Before(
				d.ProcessingUntil.UTC(),
			)

	default:
		return false
	}
}

func (
	d CompletionNotificationDelivery,
) CanClaim(
	now time.Time,
) bool {
	return !d.IsTerminal() &&
		d.AttemptCount < d.MaxAttempts &&
		d.IsDue(now)
}

func (
	d CompletionNotificationDelivery,
) Claim(
	now time.Time,
	processingUntil time.Time,
) (
	CompletionNotificationDelivery,
	error,
) {
	normalized, err :=
		d.Normalize()
	if err != nil {
		return CompletionNotificationDelivery{},
			err
	}

	now =
		normalizeCompletionNotificationTime(
			now,
		)

	processingUntil =
		processingUntil.UTC()

	if !processingUntil.After(now) {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationLeaseInvalid
	}

	if normalized.AttemptCount >=
		normalized.MaxAttempts {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationAttemptLimit
	}

	if !normalized.CanClaim(now) {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationNotClaimable
	}

	normalized.Status =
		CompletionNotificationStatusProcessing

	normalized.AttemptCount++
	normalized.LastError = ""
	normalized.NextAttemptAt = nil

	normalized.ProcessingStartedAt =
		completionNotificationTimePointer(
			now,
		)

	normalized.ProcessingUntil =
		completionNotificationTimePointer(
			processingUntil,
		)

	normalized.UpdatedAt =
		now

	return normalized, nil
}

func (
	d CompletionNotificationDelivery,
) MarkDelivered(
	providerMessageID string,
	deliveredAt time.Time,
) (
	CompletionNotificationDelivery,
	error,
) {
	normalized, err :=
		d.Normalize()
	if err != nil {
		return CompletionNotificationDelivery{},
			err
	}

	if normalized.Status ==
		CompletionNotificationStatusDelivered {
		return normalized, nil
	}

	if normalized.Status !=
		CompletionNotificationStatusProcessing {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationNotClaimable
	}

	deliveredAt =
		normalizeCompletionNotificationTime(
			deliveredAt,
		)

	normalized.Status =
		CompletionNotificationStatusDelivered

	normalized.ProviderMessageID =
		strings.TrimSpace(
			providerMessageID,
		)

	normalized.LastError = ""
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil

	normalized.DeliveredAt =
		completionNotificationTimePointer(
			deliveredAt,
		)

	normalized.FailedAt = nil
	normalized.UpdatedAt =
		deliveredAt

	return normalized, nil
}

func (
	d CompletionNotificationDelivery,
) MarkRetryableFailed(
	lastError string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) (
	CompletionNotificationDelivery,
	error,
) {
	normalized, err :=
		d.Normalize()
	if err != nil {
		return CompletionNotificationDelivery{},
			err
	}

	if normalized.Status !=
		CompletionNotificationStatusProcessing {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationNotClaimable
	}

	if normalized.AttemptCount >=
		normalized.MaxAttempts {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationAttemptLimit
	}

	lastError =
		strings.TrimSpace(
			lastError,
		)
	if lastError == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationErrorRequired
	}

	failedAt =
		normalizeCompletionNotificationTime(
			failedAt,
		)

	nextAttemptAt =
		nextAttemptAt.UTC()

	if !nextAttemptAt.After(
		failedAt,
	) {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationNextAttemptInvalid
	}

	normalized.Status =
		CompletionNotificationStatusRetryableFailed

	normalized.LastError =
		lastError

	normalized.NextAttemptAt =
		completionNotificationTimePointer(
			nextAttemptAt,
		)

	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.FailedAt = nil
	normalized.UpdatedAt =
		failedAt

	return normalized, nil
}

func (
	d CompletionNotificationDelivery,
) MarkFailed(
	lastError string,
	failedAt time.Time,
) (
	CompletionNotificationDelivery,
	error,
) {
	normalized, err :=
		d.Normalize()
	if err != nil {
		return CompletionNotificationDelivery{},
			err
	}

	if normalized.Status ==
		CompletionNotificationStatusDelivered {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationStatusInvalid
	}

	if normalized.Status ==
		CompletionNotificationStatusFailed {
		return normalized, nil
	}

	lastError =
		strings.TrimSpace(
			lastError,
		)
	if lastError == "" {
		return CompletionNotificationDelivery{},
			ErrCompletionNotificationErrorRequired
	}

	failedAt =
		normalizeCompletionNotificationTime(
			failedAt,
		)

	normalized.Status =
		CompletionNotificationStatusFailed

	normalized.LastError =
		lastError

	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil

	normalized.FailedAt =
		completionNotificationTimePointer(
			failedAt,
		)

	normalized.UpdatedAt =
		failedAt

	return normalized, nil
}

func isCompletionNotificationStripeRefundID(
	value string,
) bool {
	value =
		strings.TrimSpace(
			value,
		)

	return len(value) > len("re_") &&
		strings.HasPrefix(
			value,
			"re_",
		)
}

func normalizeCompletionNotificationTime(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}

func normalizeCompletionNotificationTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil ||
		value.IsZero() {
		return nil
	}

	normalized :=
		value.UTC()

	return &normalized
}

func completionNotificationTimePointer(
	value time.Time,
) *time.Time {
	normalized :=
		value.UTC()

	return &normalized
}
