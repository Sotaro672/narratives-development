// backend/internal/domain/tracking/entity.go
package tracking

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	domcommon "narratives/internal/domain/common"
)

// Entity (mirror web-app/src/shared/types/tracking.ts)
type Tracking struct {
	ID                  string
	OrderID             string
	TrackingNumber      string
	Carrier             string
	SpecialInstructions *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Errors
var (
	ErrInvalidID                  = errors.New("tracking: invalid id")
	ErrInvalidOrderID             = errors.New("tracking: invalid orderId")
	ErrInvalidTrackingNumber      = errors.New("tracking: invalid trackingNumber")
	ErrInvalidCarrier             = errors.New("tracking: invalid carrier")
	ErrInvalidSpecialInstructions = errors.New("tracking: invalid specialInstructions")
	ErrInvalidCreatedAt           = errors.New("tracking: invalid createdAt")
	ErrInvalidUpdatedAt           = errors.New("tracking: invalid updatedAt")
)

// Policy (align with shared/constants/trackingConstants.ts)
var (
	// Optional ID prefix enforcement
	IDPrefix        = ""
	EnforceIDPrefix = false
	MaxIDLength     = 128

	// Basic length limits (0 disables upper-bound checks)
	MinTrackingNumberLength = 1
	MaxTrackingNumberLength = 128

	MinCarrierLength = 1
	MaxCarrierLength = 80

	MaxSpecialInstructionsLength = 2000

	// Optional pattern checks (nil disables)
	TrackingNumberRe = regexp.MustCompile(`^[A-Za-z0-9\-_.]+$`)

	// Optional allow-list for carriers (empty map = allow all)
	AllowedCarriers = map[string]struct{}{}
)

// Constructors

func New(
	id, orderID, trackingNumber, carrier string,
	specialInstructions *string,
	createdAt, updatedAt time.Time,
) (Tracking, error) {
	t := Tracking{
		ID:                  strings.TrimSpace(id),
		OrderID:             strings.TrimSpace(orderID),
		TrackingNumber:      strings.TrimSpace(trackingNumber),
		Carrier:             strings.TrimSpace(carrier),
		SpecialInstructions: domcommon.NormalizeStringPtr(specialInstructions),
		CreatedAt:           createdAt.UTC(),
		UpdatedAt:           updatedAt.UTC(),
	}
	if err := t.validate(); err != nil {
		return Tracking{}, err
	}
	return t, nil
}

func NewWithNow(
	id, orderID, trackingNumber, carrier string,
	specialInstructions *string,
	now time.Time,
) (Tracking, error) {
	now = now.UTC()
	return New(id, orderID, trackingNumber, carrier, specialInstructions, now, now)
}

func NewFromStringTimes(
	id, orderID, trackingNumber, carrier string,
	specialInstructions *string,
	createdAt, updatedAt string,
) (Tracking, error) {
	ct, err := domcommon.ParseTime(createdAt)
	if err != nil {
		return Tracking{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ut, err := domcommon.ParseTime(updatedAt)
	if err != nil {
		return Tracking{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	return New(id, orderID, trackingNumber, carrier, specialInstructions, ct, ut)
}

// Behavior

func (t *Tracking) Touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	t.UpdatedAt = now.UTC()
	return nil
}

func (t *Tracking) SetTrackingNumber(v string, now time.Time) error {
	v = strings.TrimSpace(v)
	if !withinLen(v, MinTrackingNumberLength, MaxTrackingNumberLength) {
		return ErrInvalidTrackingNumber
	}
	if TrackingNumberRe != nil && !TrackingNumberRe.MatchString(v) {
		return ErrInvalidTrackingNumber
	}
	t.TrackingNumber = v
	return t.Touch(now)
}

func (t *Tracking) SetCarrier(v string, now time.Time) error {
	v = strings.TrimSpace(v)
	if !withinLen(v, MinCarrierLength, MaxCarrierLength) {
		return ErrInvalidCarrier
	}
	if len(AllowedCarriers) > 0 {
		if _, ok := AllowedCarriers[v]; !ok {
			return ErrInvalidCarrier
		}
	}
	t.Carrier = v
	return t.Touch(now)
}

func (t *Tracking) SetSpecialInstructions(v *string, now time.Time) error {
	n := domcommon.NormalizeStringPtr(v)
	if n != nil && MaxSpecialInstructionsLength > 0 && len([]rune(*n)) > MaxSpecialInstructionsLength {
		return ErrInvalidSpecialInstructions
	}
	t.SpecialInstructions = n
	return t.Touch(now)
}

// Validation

func (t Tracking) validate() error {
	// id
	if t.ID == "" {
		return ErrInvalidID
	}
	if EnforceIDPrefix && IDPrefix != "" && !strings.HasPrefix(t.ID, IDPrefix) {
		return ErrInvalidID
	}
	if MaxIDLength > 0 && len([]rune(t.ID)) > MaxIDLength {
		return ErrInvalidID
	}
	// orderId
	if t.OrderID == "" {
		return ErrInvalidOrderID
	}
	// trackingNumber
	if !withinLen(t.TrackingNumber, MinTrackingNumberLength, MaxTrackingNumberLength) {
		return ErrInvalidTrackingNumber
	}
	if TrackingNumberRe != nil && !TrackingNumberRe.MatchString(t.TrackingNumber) {
		return ErrInvalidTrackingNumber
	}
	// carrier
	if !withinLen(t.Carrier, MinCarrierLength, MaxCarrierLength) {
		return ErrInvalidCarrier
	}
	if len(AllowedCarriers) > 0 {
		if _, ok := AllowedCarriers[t.Carrier]; !ok {
			return ErrInvalidCarrier
		}
	}
	// specialInstructions
	if t.SpecialInstructions != nil && MaxSpecialInstructionsLength > 0 && len([]rune(*t.SpecialInstructions)) > MaxSpecialInstructionsLength {
		return ErrInvalidSpecialInstructions
	}
	// times
	if t.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if t.UpdatedAt.IsZero() || t.UpdatedAt.Before(t.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// Helpers

func withinLen(s string, min, max int) bool {
	n := len([]rune(s))
	if min > 0 && n < min {
		return false
	}
	if max > 0 && n > max {
		return false
	}
	return true
}
