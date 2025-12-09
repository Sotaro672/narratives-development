// backend\internal\domain\fulfillment\entity.go
package fulfillment

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Fulfillment mirrors web-app/src/shared/types/fulfillment.ts
// TS:
//  id: string
//  orderId: string
//  paymentId: string
//  status: string (ENUM; domain-specific values are validated as non-empty here)
//  createdAt: string (ISO8601)
//  updatedAt: string (ISO8601)

// Status type (ENUM placeholder; validated as non-empty)
type FulfillmentStatus string

func IsValidStatus(s FulfillmentStatus) bool {
	return strings.TrimSpace(string(s)) != ""
}

type Fulfillment struct {
	ID        string            `json:"id"`
	OrderID   string            `json:"orderId"`
	PaymentID string            `json:"paymentId"`
	Status    FulfillmentStatus `json:"status"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// Domain errors
var (
	ErrInvalidID        = errors.New("fulfillment: invalid id")
	ErrInvalidOrderID   = errors.New("fulfillment: invalid orderId")
	ErrInvalidPaymentID = errors.New("fulfillment: invalid paymentId")
	ErrInvalidStatus    = errors.New("fulfillment: invalid status")
	ErrInvalidCreatedAt = errors.New("fulfillment: invalid createdAt")
	ErrInvalidUpdatedAt = errors.New("fulfillment: invalid updatedAt")
)

// Constructors

func New(
	id, orderID, paymentID string,
	status FulfillmentStatus,
	createdAt, updatedAt time.Time,
) (Fulfillment, error) {
	f := Fulfillment{
		ID:        strings.TrimSpace(id),
		OrderID:   strings.TrimSpace(orderID),
		PaymentID: strings.TrimSpace(paymentID),
		Status:    status,
		CreatedAt: createdAt.UTC(),
		UpdatedAt: updatedAt.UTC(),
	}
	if err := f.validate(); err != nil {
		return Fulfillment{}, err
	}
	return f, nil
}

func NewFromStringTimes(
	id, orderID, paymentID string,
	status FulfillmentStatus,
	createdAtStr, updatedAtStr string,
) (Fulfillment, error) {
	ct, err := parseTime(createdAtStr)
	if err != nil {
		return Fulfillment{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ut, err := parseTime(updatedAtStr)
	if err != nil {
		return Fulfillment{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	return New(id, orderID, paymentID, status, ct, ut)
}

// Behavior

func (f *Fulfillment) SetStatus(s FulfillmentStatus, now time.Time) error {
	if !IsValidStatus(s) {
		return ErrInvalidStatus
	}
	f.Status = s
	if now.IsZero() {
		now = time.Now().UTC()
	}
	f.UpdatedAt = now.UTC()
	return nil
}

func (f *Fulfillment) TouchUpdatedAt(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	f.UpdatedAt = now.UTC()
	return nil
}

// Validation

func (f Fulfillment) validate() error {
	if strings.TrimSpace(f.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(f.OrderID) == "" {
		return ErrInvalidOrderID
	}
	if strings.TrimSpace(f.PaymentID) == "" {
		return ErrInvalidPaymentID
	}
	if !IsValidStatus(f.Status) {
		return ErrInvalidStatus
	}
	if f.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if f.UpdatedAt.IsZero() || f.UpdatedAt.Before(f.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// Helpers

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}
