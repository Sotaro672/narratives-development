// backend/internal/domain/bankPayout/payout_notification.go
package bankPayout

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const DefaultPayoutNotificationMaxAttempts = 5

type PayoutNotificationStatus string

const (
	PayoutNotificationStatusPending         PayoutNotificationStatus = "pending"
	PayoutNotificationStatusProcessing      PayoutNotificationStatus = "processing"
	PayoutNotificationStatusRetryableFailed PayoutNotificationStatus = "retryable_failed"
	PayoutNotificationStatusDelivered       PayoutNotificationStatus = "delivered"
	PayoutNotificationStatusFailed          PayoutNotificationStatus = "failed"
)

var (
	ErrPayoutNotificationDeliveryIDRequired = errors.New(
		"bankPayout: payout notification deliveryID is required",
	)
	ErrPayoutNotificationDeliveryIDInvalid = errors.New(
		"bankPayout: payout notification deliveryID is invalid",
	)
	ErrPayoutNotificationBankPayoutIDRequired = errors.New(
		"bankPayout: payout notification bankPayoutID is required",
	)
	ErrPayoutNotificationSalesReceivableIDRequired = errors.New(
		"bankPayout: payout notification salesReceivableID is required",
	)
	ErrPayoutNotificationOrderIDRequired = errors.New(
		"bankPayout: payout notification orderID is required",
	)
	ErrPayoutNotificationResaleIDRequired = errors.New(
		"bankPayout: payout notification resaleID is required",
	)
	ErrPayoutNotificationUserIDRequired = errors.New(
		"bankPayout: payout notification userID is required",
	)
	ErrPayoutNotificationAmountInvalid = errors.New(
		"bankPayout: payout notification amount is invalid",
	)
	ErrPayoutNotificationCurrencyInvalid = errors.New(
		"bankPayout: payout notification currency is invalid",
	)
	ErrPayoutNotificationBankNameRequired = errors.New(
		"bankPayout: payout notification bankName is required",
	)
	ErrPayoutNotificationBranchNameRequired = errors.New(
		"bankPayout: payout notification branchName is required",
	)
	ErrPayoutNotificationBankLast4Invalid = errors.New(
		"bankPayout: payout notification bankLast4 is invalid",
	)
	ErrPayoutNotificationPaidAtInvalid = errors.New(
		"bankPayout: payout notification paidAt is invalid",
	)
	ErrPayoutNotificationCreatedAtInvalid = errors.New(
		"bankPayout: payout notification createdAt is invalid",
	)
	ErrPayoutNotificationStatusInvalid = errors.New(
		"bankPayout: payout notification status is invalid",
	)
	ErrPayoutNotificationAttemptCountInvalid = errors.New(
		"bankPayout: payout notification attempt count is invalid",
	)
	ErrPayoutNotificationMaxAttemptsInvalid = errors.New(
		"bankPayout: payout notification max attempts is invalid",
	)
	ErrPayoutNotificationAttemptLimit = errors.New(
		"bankPayout: payout notification attempt limit reached",
	)
	ErrPayoutNotificationNotClaimable = errors.New(
		"bankPayout: payout notification is not claimable",
	)
	ErrPayoutNotificationLeaseInvalid = errors.New(
		"bankPayout: payout notification lease is invalid",
	)
	ErrPayoutNotificationErrorRequired = errors.New(
		"bankPayout: payout notification error is required",
	)
	ErrPayoutNotificationNextAttemptInvalid = errors.New(
		"bankPayout: payout notification next attempt is invalid",
	)
)

// PayoutNotificationDelivery is the durable email-delivery state created after
// one resale BankPayout has completed.
//
// The delivery snapshots only non-secret values needed by the seller
// notification. Plaintext account numbers and AccountNumberCiphertext must never
// be stored here.
type PayoutNotificationDelivery struct {
	ID string

	BankPayoutID      string
	SalesReceivableID string
	OrderID           string
	ResaleID          string
	UserID            string

	Amount   int
	Currency string

	BankName   string
	BranchName string
	BankLast4  string

	PaidAt time.Time

	Status       PayoutNotificationStatus
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

// BuildPayoutNotificationDeliveryID creates the deterministic email-delivery ID
// for one BankPayout.
//
// One BankPayout may produce at most one payout-completion notification.
func BuildPayoutNotificationDeliveryID(
	bankPayoutID string,
) (string, error) {
	bankPayoutID = strings.TrimSpace(bankPayoutID)
	if bankPayoutID == "" {
		return "", ErrPayoutNotificationBankPayoutIDRequired
	}

	sum := sha256.Sum256([]byte(bankPayoutID))
	return "bank_payout_notification_" + hex.EncodeToString(sum[:]), nil
}

func NewPayoutNotificationDelivery(
	bankPayoutID string,
	salesReceivableID string,
	orderID string,
	resaleID string,
	userID string,
	amount int,
	currency string,
	bankName string,
	branchName string,
	bankLast4 string,
	paidAt time.Time,
	createdAt time.Time,
	maxAttempts int,
) (PayoutNotificationDelivery, error) {
	bankPayoutID = strings.TrimSpace(bankPayoutID)
	salesReceivableID = strings.TrimSpace(salesReceivableID)
	orderID = strings.TrimSpace(orderID)
	resaleID = strings.TrimSpace(resaleID)
	userID = strings.TrimSpace(userID)
	currency = strings.TrimSpace(currency)
	bankName = strings.TrimSpace(bankName)
	branchName = strings.TrimSpace(branchName)
	bankLast4 = strings.TrimSpace(bankLast4)

	deliveryID, err := BuildPayoutNotificationDeliveryID(bankPayoutID)
	if err != nil {
		return PayoutNotificationDelivery{}, err
	}

	if salesReceivableID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationSalesReceivableIDRequired
	}
	if orderID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationOrderIDRequired
	}
	if resaleID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationResaleIDRequired
	}
	if userID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationUserIDRequired
	}
	if amount <= 0 {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationAmountInvalid
	}
	if currency != CurrencyJPY {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationCurrencyInvalid
	}
	if bankName == "" ||
		(MaxBankNameLength > 0 && len([]rune(bankName)) > MaxBankNameLength) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationBankNameRequired
	}
	if branchName == "" ||
		(MaxBranchNameLength > 0 && len([]rune(branchName)) > MaxBranchNameLength) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationBranchNameRequired
	}
	if !isFixedDigits(bankLast4, 4) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationBankLast4Invalid
	}

	paidAt = normalizePayoutNotificationTime(paidAt)
	if paidAt.IsZero() {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationPaidAtInvalid
	}

	createdAt = normalizePayoutNotificationTime(createdAt)
	if createdAt.IsZero() || createdAt.Before(paidAt) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationCreatedAtInvalid
	}

	if maxAttempts == 0 {
		maxAttempts = DefaultPayoutNotificationMaxAttempts
	}
	if maxAttempts < 1 {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationMaxAttemptsInvalid
	}

	return PayoutNotificationDelivery{
		ID:                deliveryID,
		BankPayoutID:      bankPayoutID,
		SalesReceivableID: salesReceivableID,
		OrderID:           orderID,
		ResaleID:          resaleID,
		UserID:            userID,
		Amount:            amount,
		Currency:          currency,
		BankName:          bankName,
		BranchName:        branchName,
		BankLast4:         bankLast4,
		PaidAt:            paidAt,
		Status:            PayoutNotificationStatusPending,
		AttemptCount:      0,
		MaxAttempts:       maxAttempts,
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
		NextAttemptAt:     payoutNotificationTimePointer(createdAt),
	}, nil
}

func (d PayoutNotificationDelivery) Normalize() (
	PayoutNotificationDelivery,
	error,
) {
	d.ID = strings.TrimSpace(d.ID)
	if d.ID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationDeliveryIDRequired
	}

	d.BankPayoutID = strings.TrimSpace(d.BankPayoutID)
	if d.BankPayoutID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationBankPayoutIDRequired
	}

	expectedID, err := BuildPayoutNotificationDeliveryID(d.BankPayoutID)
	if err != nil {
		return PayoutNotificationDelivery{}, err
	}
	if d.ID != expectedID {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationDeliveryIDInvalid
	}

	d.SalesReceivableID = strings.TrimSpace(d.SalesReceivableID)
	if d.SalesReceivableID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationSalesReceivableIDRequired
	}

	d.OrderID = strings.TrimSpace(d.OrderID)
	if d.OrderID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationOrderIDRequired
	}

	d.ResaleID = strings.TrimSpace(d.ResaleID)
	if d.ResaleID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationResaleIDRequired
	}

	d.UserID = strings.TrimSpace(d.UserID)
	if d.UserID == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationUserIDRequired
	}

	if d.Amount <= 0 {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationAmountInvalid
	}

	d.Currency = strings.TrimSpace(d.Currency)
	if d.Currency != CurrencyJPY {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationCurrencyInvalid
	}

	d.BankName = strings.TrimSpace(d.BankName)
	if d.BankName == "" ||
		(MaxBankNameLength > 0 && len([]rune(d.BankName)) > MaxBankNameLength) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationBankNameRequired
	}

	d.BranchName = strings.TrimSpace(d.BranchName)
	if d.BranchName == "" ||
		(MaxBranchNameLength > 0 && len([]rune(d.BranchName)) > MaxBranchNameLength) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationBranchNameRequired
	}

	d.BankLast4 = strings.TrimSpace(d.BankLast4)
	if !isFixedDigits(d.BankLast4, 4) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationBankLast4Invalid
	}

	d.PaidAt = normalizePayoutNotificationTime(d.PaidAt)
	if d.PaidAt.IsZero() {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationPaidAtInvalid
	}

	d.Status = PayoutNotificationStatus(
		strings.TrimSpace(string(d.Status)),
	)
	if d.Status == "" {
		d.Status = PayoutNotificationStatusPending
	}
	if !d.Status.IsValid() {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationStatusInvalid
	}

	if d.AttemptCount < 0 {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationAttemptCountInvalid
	}

	if d.MaxAttempts == 0 {
		d.MaxAttempts = DefaultPayoutNotificationMaxAttempts
	}
	if d.MaxAttempts < 1 || d.AttemptCount > d.MaxAttempts {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationMaxAttemptsInvalid
	}

	d.ProviderMessageID = strings.TrimSpace(d.ProviderMessageID)
	d.LastError = strings.TrimSpace(d.LastError)

	d.CreatedAt = normalizePayoutNotificationTime(d.CreatedAt)
	if d.CreatedAt.IsZero() || d.CreatedAt.Before(d.PaidAt) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationCreatedAtInvalid
	}

	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = d.CreatedAt
	} else {
		d.UpdatedAt = normalizePayoutNotificationTime(d.UpdatedAt)
	}

	d.NextAttemptAt = normalizePayoutNotificationTimePointer(d.NextAttemptAt)
	d.ProcessingStartedAt = normalizePayoutNotificationTimePointer(d.ProcessingStartedAt)
	d.ProcessingUntil = normalizePayoutNotificationTimePointer(d.ProcessingUntil)
	d.DeliveredAt = normalizePayoutNotificationTimePointer(d.DeliveredAt)
	d.FailedAt = normalizePayoutNotificationTimePointer(d.FailedAt)

	switch d.Status {
	case PayoutNotificationStatusProcessing:
		if d.ProcessingStartedAt == nil ||
			d.ProcessingUntil == nil ||
			!d.ProcessingUntil.After(*d.ProcessingStartedAt) {
			return PayoutNotificationDelivery{}, ErrPayoutNotificationLeaseInvalid
		}

	case PayoutNotificationStatusDelivered:
		if d.DeliveredAt == nil {
			return PayoutNotificationDelivery{}, ErrPayoutNotificationStatusInvalid
		}

	case PayoutNotificationStatusRetryableFailed:
		if d.LastError == "" {
			return PayoutNotificationDelivery{}, ErrPayoutNotificationErrorRequired
		}
		if d.NextAttemptAt == nil {
			return PayoutNotificationDelivery{}, ErrPayoutNotificationNextAttemptInvalid
		}

	case PayoutNotificationStatusFailed:
		if d.LastError == "" {
			return PayoutNotificationDelivery{}, ErrPayoutNotificationErrorRequired
		}
		if d.FailedAt == nil {
			return PayoutNotificationDelivery{}, ErrPayoutNotificationStatusInvalid
		}
	}

	return d, nil
}

func (s PayoutNotificationStatus) IsValid() bool {
	switch s {
	case PayoutNotificationStatusPending,
		PayoutNotificationStatusProcessing,
		PayoutNotificationStatusRetryableFailed,
		PayoutNotificationStatusDelivered,
		PayoutNotificationStatusFailed:
		return true
	default:
		return false
	}
}

func (d PayoutNotificationDelivery) IsTerminal() bool {
	return d.Status == PayoutNotificationStatusDelivered ||
		d.Status == PayoutNotificationStatusFailed
}

func (d PayoutNotificationDelivery) IsDue(
	now time.Time,
) bool {
	now = normalizePayoutNotificationTime(now)

	switch d.Status {
	case PayoutNotificationStatusPending,
		PayoutNotificationStatusRetryableFailed:
		return d.NextAttemptAt == nil ||
			!now.Before(d.NextAttemptAt.UTC())

	case PayoutNotificationStatusProcessing:
		return d.ProcessingUntil != nil &&
			!now.Before(d.ProcessingUntil.UTC())

	default:
		return false
	}
}

func (d PayoutNotificationDelivery) CanClaim(
	now time.Time,
) bool {
	return !d.IsTerminal() &&
		d.AttemptCount < d.MaxAttempts &&
		d.IsDue(now)
}

func (d PayoutNotificationDelivery) Claim(
	now time.Time,
	processingUntil time.Time,
) (PayoutNotificationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return PayoutNotificationDelivery{}, err
	}

	now = normalizePayoutNotificationTime(now)
	processingUntil = normalizePayoutNotificationTime(processingUntil)

	if now.IsZero() ||
		processingUntil.IsZero() ||
		!processingUntil.After(now) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationLeaseInvalid
	}

	if normalized.AttemptCount >= normalized.MaxAttempts {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationAttemptLimit
	}

	if !normalized.CanClaim(now) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationNotClaimable
	}

	normalized.Status = PayoutNotificationStatusProcessing
	normalized.AttemptCount++
	normalized.LastError = ""
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = payoutNotificationTimePointer(now)
	normalized.ProcessingUntil = payoutNotificationTimePointer(processingUntil)
	normalized.DeliveredAt = nil
	normalized.FailedAt = nil
	normalized.UpdatedAt = now

	return normalized.Normalize()
}

func (d PayoutNotificationDelivery) MarkDelivered(
	providerMessageID string,
	deliveredAt time.Time,
) (PayoutNotificationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return PayoutNotificationDelivery{}, err
	}

	if normalized.Status == PayoutNotificationStatusDelivered {
		return normalized, nil
	}

	if normalized.Status != PayoutNotificationStatusProcessing {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationNotClaimable
	}

	deliveredAt = normalizePayoutNotificationTime(deliveredAt)
	if deliveredAt.IsZero() ||
		normalized.ProcessingStartedAt == nil ||
		deliveredAt.Before(*normalized.ProcessingStartedAt) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationStatusInvalid
	}

	normalized.Status = PayoutNotificationStatusDelivered
	normalized.ProviderMessageID = strings.TrimSpace(providerMessageID)
	normalized.LastError = ""
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.DeliveredAt = payoutNotificationTimePointer(deliveredAt)
	normalized.FailedAt = nil
	normalized.UpdatedAt = deliveredAt

	return normalized.Normalize()
}

func (d PayoutNotificationDelivery) MarkRetryableFailed(
	lastError string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) (PayoutNotificationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return PayoutNotificationDelivery{}, err
	}

	if normalized.Status != PayoutNotificationStatusProcessing {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationNotClaimable
	}

	if normalized.AttemptCount >= normalized.MaxAttempts {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationAttemptLimit
	}

	lastError = strings.TrimSpace(lastError)
	if lastError == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationErrorRequired
	}

	failedAt = normalizePayoutNotificationTime(failedAt)
	nextAttemptAt = normalizePayoutNotificationTime(nextAttemptAt)
	if failedAt.IsZero() ||
		nextAttemptAt.IsZero() ||
		!nextAttemptAt.After(failedAt) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationNextAttemptInvalid
	}

	normalized.Status = PayoutNotificationStatusRetryableFailed
	normalized.ProviderMessageID = ""
	normalized.LastError = lastError
	normalized.NextAttemptAt = payoutNotificationTimePointer(nextAttemptAt)
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.DeliveredAt = nil
	normalized.FailedAt = payoutNotificationTimePointer(failedAt)
	normalized.UpdatedAt = failedAt

	return normalized.Normalize()
}

func (d PayoutNotificationDelivery) MarkFailed(
	lastError string,
	failedAt time.Time,
) (PayoutNotificationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return PayoutNotificationDelivery{}, err
	}

	if normalized.Status == PayoutNotificationStatusFailed {
		return normalized, nil
	}

	if normalized.Status != PayoutNotificationStatusProcessing {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationNotClaimable
	}

	lastError = strings.TrimSpace(lastError)
	if lastError == "" {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationErrorRequired
	}

	failedAt = normalizePayoutNotificationTime(failedAt)
	if failedAt.IsZero() ||
		normalized.ProcessingStartedAt == nil ||
		failedAt.Before(*normalized.ProcessingStartedAt) {
		return PayoutNotificationDelivery{}, ErrPayoutNotificationStatusInvalid
	}

	normalized.Status = PayoutNotificationStatusFailed
	normalized.ProviderMessageID = ""
	normalized.LastError = lastError
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.DeliveredAt = nil
	normalized.FailedAt = payoutNotificationTimePointer(failedAt)
	normalized.UpdatedAt = failedAt

	return normalized.Normalize()
}

func normalizePayoutNotificationTime(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return value
	}

	return value.UTC()
}

func normalizePayoutNotificationTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	normalized := normalizePayoutNotificationTime(*value)
	return &normalized
}

func payoutNotificationTimePointer(
	value time.Time,
) *time.Time {
	normalized := normalizePayoutNotificationTime(value)
	return &normalized
}
