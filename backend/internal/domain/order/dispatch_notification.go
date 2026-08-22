// backend/internal/domain/order/dispatch_notification.go
package order

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const DefaultDispatchNotificationMaxAttempts = 5

type DispatchNotificationStatus string

const (
	DispatchNotificationStatusPending         DispatchNotificationStatus = "pending"
	DispatchNotificationStatusProcessing      DispatchNotificationStatus = "processing"
	DispatchNotificationStatusRetryableFailed DispatchNotificationStatus = "retryable_failed"
	DispatchNotificationStatusDelivered       DispatchNotificationStatus = "delivered"
	DispatchNotificationStatusFailed          DispatchNotificationStatus = "failed"
)

var (
	ErrDispatchNotificationDeliveryIDRequired  = errors.New("order: dispatch notification deliveryID is required")
	ErrDispatchNotificationDeliveryIDInvalid   = errors.New("order: dispatch notification deliveryID is invalid")
	ErrDispatchNotificationOrderIDRequired     = errors.New("order: dispatch notification orderID is required")
	ErrDispatchNotificationCompanyIDRequired   = errors.New("order: dispatch notification companyID is required")
	ErrDispatchNotificationUserIDRequired      = errors.New("order: dispatch notification userID is required")
	ErrDispatchNotificationItemsRequired       = errors.New("order: dispatch notification items are required")
	ErrDispatchNotificationItemInvalid         = errors.New("order: dispatch notification item is invalid")
	ErrDispatchNotificationStatusInvalid       = errors.New("order: dispatch notification status is invalid")
	ErrDispatchNotificationAttemptCountInvalid = errors.New(
		"order: dispatch notification attempt count is invalid",
	)
	ErrDispatchNotificationMaxAttemptsInvalid = errors.New(
		"order: dispatch notification max attempts is invalid",
	)
	ErrDispatchNotificationAttemptLimit = errors.New(
		"order: dispatch notification attempt limit reached",
	)
	ErrDispatchNotificationNotClaimable = errors.New(
		"order: dispatch notification is not claimable",
	)
	ErrDispatchNotificationLeaseInvalid = errors.New(
		"order: dispatch notification lease is invalid",
	)
	ErrDispatchNotificationErrorRequired = errors.New(
		"order: dispatch notification error is required",
	)
	ErrDispatchNotificationNextAttemptInvalid = errors.New(
		"order: dispatch notification next attempt is invalid",
	)
)

type DispatchNotificationItem struct {
	InventoryID        string
	ListID             string
	ProductBlueprintID string
	TokenBlueprintID   string
	Qty                int
}

type DispatchNotificationDelivery struct {
	ID        string
	OrderID   string
	CompanyID string
	UserID    string

	Items []DispatchNotificationItem

	Status       DispatchNotificationStatus
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

func BuildDispatchNotificationDeliveryID(
	orderID string,
	companyID string,
) (string, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return "", ErrDispatchNotificationOrderIDRequired
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return "", ErrDispatchNotificationCompanyIDRequired
	}

	sum := sha256.Sum256(
		[]byte(orderID + "\x00" + companyID),
	)

	return "dispatch_" + hex.EncodeToString(sum[:]), nil
}

func NewDispatchNotificationItem(
	inventoryID string,
	listID string,
	productBlueprintID string,
	tokenBlueprintID string,
	qty int,
) (DispatchNotificationItem, error) {
	item := DispatchNotificationItem{
		InventoryID:        inventoryID,
		ListID:             listID,
		ProductBlueprintID: productBlueprintID,
		TokenBlueprintID:   tokenBlueprintID,
		Qty:                qty,
	}

	return item.Normalize()
}

func (item DispatchNotificationItem) Normalize() (
	DispatchNotificationItem,
	error,
) {
	item.InventoryID = strings.TrimSpace(item.InventoryID)
	item.ListID = strings.TrimSpace(item.ListID)
	item.ProductBlueprintID = strings.TrimSpace(item.ProductBlueprintID)
	item.TokenBlueprintID = strings.TrimSpace(item.TokenBlueprintID)

	if item.InventoryID == "" ||
		item.ListID == "" ||
		item.ProductBlueprintID == "" ||
		item.TokenBlueprintID == "" ||
		item.Qty <= 0 {
		return DispatchNotificationItem{}, ErrDispatchNotificationItemInvalid
	}

	return item, nil
}

func NewDispatchNotificationDelivery(
	orderID string,
	companyID string,
	userID string,
	items []DispatchNotificationItem,
	createdAt time.Time,
	maxAttempts int,
) (DispatchNotificationDelivery, error) {
	deliveryID, err := BuildDispatchNotificationDeliveryID(
		orderID,
		companyID,
	)
	if err != nil {
		return DispatchNotificationDelivery{}, err
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationUserIDRequired
	}

	normalizedItems, err := normalizeDispatchNotificationItems(items)
	if err != nil {
		return DispatchNotificationDelivery{}, err
	}

	if maxAttempts == 0 {
		maxAttempts = DefaultDispatchNotificationMaxAttempts
	}

	if maxAttempts < 1 {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationMaxAttemptsInvalid
	}

	createdAt = normalizeDispatchNotificationTime(createdAt)

	return DispatchNotificationDelivery{
		ID:            deliveryID,
		OrderID:       strings.TrimSpace(orderID),
		CompanyID:     strings.TrimSpace(companyID),
		UserID:        userID,
		Items:         normalizedItems,
		Status:        DispatchNotificationStatusPending,
		AttemptCount:  0,
		MaxAttempts:   maxAttempts,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
		NextAttemptAt: dispatchNotificationTimePointer(createdAt),
	}, nil
}

func (d DispatchNotificationDelivery) Normalize() (
	DispatchNotificationDelivery,
	error,
) {
	d.ID = strings.TrimSpace(d.ID)
	if d.ID == "" {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationDeliveryIDRequired
	}

	d.OrderID = strings.TrimSpace(d.OrderID)
	if d.OrderID == "" {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationOrderIDRequired
	}

	d.CompanyID = strings.TrimSpace(d.CompanyID)
	if d.CompanyID == "" {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationCompanyIDRequired
	}

	expectedID, err := BuildDispatchNotificationDeliveryID(
		d.OrderID,
		d.CompanyID,
	)
	if err != nil {
		return DispatchNotificationDelivery{}, err
	}

	if d.ID != expectedID {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationDeliveryIDInvalid
	}

	d.UserID = strings.TrimSpace(d.UserID)
	if d.UserID == "" {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationUserIDRequired
	}

	d.Items, err = normalizeDispatchNotificationItems(d.Items)
	if err != nil {
		return DispatchNotificationDelivery{}, err
	}

	d.Status = DispatchNotificationStatus(
		strings.TrimSpace(string(d.Status)),
	)

	if d.Status == "" {
		d.Status = DispatchNotificationStatusPending
	}

	if !d.Status.IsValid() {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationStatusInvalid
	}

	if d.AttemptCount < 0 {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationAttemptCountInvalid
	}

	if d.MaxAttempts == 0 {
		d.MaxAttempts = DefaultDispatchNotificationMaxAttempts
	}

	if d.MaxAttempts < 1 ||
		d.AttemptCount > d.MaxAttempts {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationMaxAttemptsInvalid
	}

	d.ProviderMessageID = strings.TrimSpace(d.ProviderMessageID)
	d.LastError = strings.TrimSpace(d.LastError)
	d.CreatedAt = normalizeDispatchNotificationTime(d.CreatedAt)

	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = d.CreatedAt
	} else {
		d.UpdatedAt = d.UpdatedAt.UTC()
	}

	d.NextAttemptAt = normalizeDispatchNotificationTimePointer(d.NextAttemptAt)
	d.ProcessingStartedAt = normalizeDispatchNotificationTimePointer(d.ProcessingStartedAt)
	d.ProcessingUntil = normalizeDispatchNotificationTimePointer(d.ProcessingUntil)
	d.DeliveredAt = normalizeDispatchNotificationTimePointer(d.DeliveredAt)
	d.FailedAt = normalizeDispatchNotificationTimePointer(d.FailedAt)

	switch d.Status {
	case DispatchNotificationStatusProcessing:
		if d.ProcessingUntil == nil {
			return DispatchNotificationDelivery{}, ErrDispatchNotificationLeaseInvalid
		}

	case DispatchNotificationStatusDelivered:
		if d.DeliveredAt == nil {
			return DispatchNotificationDelivery{}, ErrDispatchNotificationStatusInvalid
		}

	case DispatchNotificationStatusRetryableFailed:
		if d.LastError == "" {
			return DispatchNotificationDelivery{}, ErrDispatchNotificationErrorRequired
		}

		if d.NextAttemptAt == nil {
			return DispatchNotificationDelivery{}, ErrDispatchNotificationNextAttemptInvalid
		}

	case DispatchNotificationStatusFailed:
		if d.LastError == "" {
			return DispatchNotificationDelivery{}, ErrDispatchNotificationErrorRequired
		}

		if d.FailedAt == nil {
			return DispatchNotificationDelivery{}, ErrDispatchNotificationStatusInvalid
		}
	}

	return d, nil
}

func (s DispatchNotificationStatus) IsValid() bool {
	switch s {
	case DispatchNotificationStatusPending,
		DispatchNotificationStatusProcessing,
		DispatchNotificationStatusRetryableFailed,
		DispatchNotificationStatusDelivered,
		DispatchNotificationStatusFailed:
		return true

	default:
		return false
	}
}

func (d DispatchNotificationDelivery) IsTerminal() bool {
	return d.Status == DispatchNotificationStatusDelivered ||
		d.Status == DispatchNotificationStatusFailed
}

func (d DispatchNotificationDelivery) IsDue(
	now time.Time,
) bool {
	now = normalizeDispatchNotificationTime(now)

	switch d.Status {
	case DispatchNotificationStatusPending,
		DispatchNotificationStatusRetryableFailed:
		return d.NextAttemptAt == nil ||
			!now.Before(d.NextAttemptAt.UTC())

	case DispatchNotificationStatusProcessing:
		return d.ProcessingUntil != nil &&
			!now.Before(d.ProcessingUntil.UTC())

	default:
		return false
	}
}

func (d DispatchNotificationDelivery) CanClaim(
	now time.Time,
) bool {
	return !d.IsTerminal() &&
		d.AttemptCount < d.MaxAttempts &&
		d.IsDue(now)
}

func (d DispatchNotificationDelivery) Claim(
	now time.Time,
	processingUntil time.Time,
) (DispatchNotificationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return DispatchNotificationDelivery{}, err
	}

	now = normalizeDispatchNotificationTime(now)
	processingUntil = processingUntil.UTC()

	if !processingUntil.After(now) {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationLeaseInvalid
	}

	if normalized.AttemptCount >= normalized.MaxAttempts {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationAttemptLimit
	}

	if !normalized.CanClaim(now) {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationNotClaimable
	}

	normalized.Status = DispatchNotificationStatusProcessing
	normalized.AttemptCount++
	normalized.LastError = ""
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = dispatchNotificationTimePointer(now)
	normalized.ProcessingUntil = dispatchNotificationTimePointer(processingUntil)
	normalized.UpdatedAt = now

	return normalized, nil
}

func (d DispatchNotificationDelivery) MarkDelivered(
	providerMessageID string,
	deliveredAt time.Time,
) (DispatchNotificationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return DispatchNotificationDelivery{}, err
	}

	if normalized.Status == DispatchNotificationStatusDelivered {
		return normalized, nil
	}

	if normalized.Status != DispatchNotificationStatusProcessing {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationNotClaimable
	}

	deliveredAt = normalizeDispatchNotificationTime(deliveredAt)

	normalized.Status = DispatchNotificationStatusDelivered
	normalized.ProviderMessageID = strings.TrimSpace(providerMessageID)
	normalized.LastError = ""
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.DeliveredAt = dispatchNotificationTimePointer(deliveredAt)
	normalized.FailedAt = nil
	normalized.UpdatedAt = deliveredAt

	return normalized, nil
}

func (d DispatchNotificationDelivery) MarkRetryableFailed(
	lastError string,
	nextAttemptAt time.Time,
	failedAt time.Time,
) (DispatchNotificationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return DispatchNotificationDelivery{}, err
	}

	if normalized.Status != DispatchNotificationStatusProcessing {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationNotClaimable
	}

	if normalized.AttemptCount >= normalized.MaxAttempts {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationAttemptLimit
	}

	lastError = strings.TrimSpace(lastError)
	if lastError == "" {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationErrorRequired
	}

	failedAt = normalizeDispatchNotificationTime(failedAt)
	nextAttemptAt = nextAttemptAt.UTC()

	if !nextAttemptAt.After(failedAt) {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationNextAttemptInvalid
	}

	normalized.Status = DispatchNotificationStatusRetryableFailed
	normalized.LastError = lastError
	normalized.NextAttemptAt = dispatchNotificationTimePointer(nextAttemptAt)
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.FailedAt = nil
	normalized.UpdatedAt = failedAt

	return normalized, nil
}

func (d DispatchNotificationDelivery) MarkFailed(
	lastError string,
	failedAt time.Time,
) (DispatchNotificationDelivery, error) {
	normalized, err := d.Normalize()
	if err != nil {
		return DispatchNotificationDelivery{}, err
	}

	if normalized.Status == DispatchNotificationStatusDelivered {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationStatusInvalid
	}

	if normalized.Status == DispatchNotificationStatusFailed {
		return normalized, nil
	}

	lastError = strings.TrimSpace(lastError)
	if lastError == "" {
		return DispatchNotificationDelivery{}, ErrDispatchNotificationErrorRequired
	}

	failedAt = normalizeDispatchNotificationTime(failedAt)

	normalized.Status = DispatchNotificationStatusFailed
	normalized.LastError = lastError
	normalized.NextAttemptAt = nil
	normalized.ProcessingStartedAt = nil
	normalized.ProcessingUntil = nil
	normalized.FailedAt = dispatchNotificationTimePointer(failedAt)
	normalized.UpdatedAt = failedAt

	return normalized, nil
}

func normalizeDispatchNotificationItems(
	items []DispatchNotificationItem,
) ([]DispatchNotificationItem, error) {
	if len(items) == 0 {
		return nil, ErrDispatchNotificationItemsRequired
	}

	normalized := make(
		[]DispatchNotificationItem,
		0,
		len(items),
	)

	for _, item := range items {
		normalizedItem, err := item.Normalize()
		if err != nil {
			return nil, err
		}

		normalized = append(
			normalized,
			normalizedItem,
		)
	}

	return normalized, nil
}

func normalizeDispatchNotificationTime(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}

func normalizeDispatchNotificationTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil ||
		value.IsZero() {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}

func dispatchNotificationTimePointer(
	value time.Time,
) *time.Time {
	normalized := value.UTC()
	return &normalized
}
