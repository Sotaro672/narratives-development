// backend\internal\domain\tracking\entity.go
package tracking

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
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
		SpecialInstructions: normalizePtr(specialInstructions),
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
	ct, err := parseTime(createdAt, ErrInvalidCreatedAt)
	if err != nil {
		return Tracking{}, err
	}
	ut, err := parseTime(updatedAt, ErrInvalidUpdatedAt)
	if err != nil {
		return Tracking{}, err
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
	n := normalizePtr(v)
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

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
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
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}

// TrackingsTableDDL defines the SQL for the trackings table migration.
const TrackingsTableDDL = `
-- Migration: Initialize Tracking domain
-- Mirrors backend/internal/domain/tracking/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS trackings (
  id                     TEXT        PRIMARY KEY,
  order_id               TEXT        NOT NULL,
  tracking_number        TEXT        NOT NULL,
  carrier                TEXT        NOT NULL,
  special_instructions   TEXT,
  created_at             TIMESTAMPTZ NOT NULL,
  updated_at             TIMESTAMPTZ NOT NULL,

  -- Non-empty checks
  CONSTRAINT chk_trackings_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(order_id)) > 0
    AND char_length(trim(tracking_number)) > 0
    AND char_length(trim(carrier)) > 0
  ),

  -- tracking_number format (matches ^[A-Za-z0-9\-_.]+$)
  CONSTRAINT chk_trackings_tracking_number_format CHECK (tracking_number ~ '^[A-Za-z0-9\\-_.]+$'),

  -- time order
  CHECK (updated_at >= created_at)
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_trackings_order_id         ON trackings(order_id);
CREATE INDEX IF NOT EXISTS idx_trackings_created_at       ON trackings(created_at);
CREATE INDEX IF NOT EXISTS idx_trackings_updated_at       ON trackings(updated_at);
CREATE INDEX IF NOT EXISTS idx_trackings_tracking_number  ON trackings(tracking_number);
CREATE INDEX IF NOT EXISTS idx_trackings_carrier          ON trackings(carrier);

COMMIT;
`
