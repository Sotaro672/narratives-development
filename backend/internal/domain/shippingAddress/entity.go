package shippingAddress

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ShippingAddress エンティティ（TSの ShippingAddressSchema に準拠）
type ShippingAddress struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Street    string    `json:"street"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	ZipCode   string    `json:"zipCode"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Errors: moved to error.go
// Errors (moved from entity.go)
var (
	ErrInvalidID        = errors.New("shippingAddress: invalid id")
	ErrInvalidUserID    = errors.New("shippingAddress: invalid userId")
	ErrInvalidStreet    = errors.New("shippingAddress: invalid street")
	ErrInvalidCity      = errors.New("shippingAddress: invalid city")
	ErrInvalidState     = errors.New("shippingAddress: invalid state")
	ErrInvalidZipCode   = errors.New("shippingAddress: invalid zipCode")
	ErrInvalidCountry   = errors.New("shippingAddress: invalid country")
	ErrInvalidCreatedAt = errors.New("shippingAddress: invalid createdAt")
	ErrInvalidUpdatedAt = errors.New("shippingAddress: invalid updatedAt")
)

// Validation (moved from entity.go)
func (a ShippingAddress) validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}
	if a.UserID == "" {
		return ErrInvalidUserID
	}
	if a.Street == "" {
		return ErrInvalidStreet
	}
	if a.City == "" {
		return ErrInvalidCity
	}
	if a.State == "" {
		return ErrInvalidState
	}
	if a.ZipCode == "" {
		return ErrInvalidZipCode
	}
	if a.Country == "" {
		return ErrInvalidCountry
	}
	if a.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if a.UpdatedAt.IsZero() || a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// Behavior (moved from entity.go)

func (a *ShippingAddress) UpdateLines(street, city, state, zip, country string, now time.Time) error {
	street = strings.TrimSpace(street)
	city = strings.TrimSpace(city)
	state = strings.TrimSpace(state)
	zip = strings.TrimSpace(zip)
	country = strings.TrimSpace(country)

	if street == "" {
		return ErrInvalidStreet
	}
	if city == "" {
		return ErrInvalidCity
	}
	if state == "" {
		return ErrInvalidState
	}
	if zip == "" {
		return ErrInvalidZipCode
	}
	if country == "" {
		return ErrInvalidCountry
	}
	a.Street, a.City, a.State, a.ZipCode, a.Country = street, city, state, zip, country
	return a.touch(now)
}

// Helpers (moved from entity.go)

func (a *ShippingAddress) touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	a.UpdatedAt = now.UTC()
	return nil
}

// Constructors

func New(
	id, userID string,
	street, city, state, zip, country string,
	createdAt, updatedAt time.Time,
) (ShippingAddress, error) {
	a := ShippingAddress{
		ID:        strings.TrimSpace(id),
		UserID:    strings.TrimSpace(userID),
		Street:    strings.TrimSpace(street),
		City:      strings.TrimSpace(city),
		State:     strings.TrimSpace(state),
		ZipCode:   strings.TrimSpace(zip),
		Country:   strings.TrimSpace(country),
		CreatedAt: createdAt.UTC(),
		UpdatedAt: updatedAt.UTC(),
	}
	if err := a.validate(); err != nil {
		return ShippingAddress{}, err
	}
	return a, nil
}

func NewWithNow(
	id, userID string,
	street, city, state, zip, country string,
	now time.Time,
) (ShippingAddress, error) {
	now = now.UTC()
	return New(id, userID, street, city, state, zip, country, now, now)
}

// From DTO-like strings (createdAt/updatedAt as RFC3339)
func NewFromStringTimes(
	id, userID string,
	street, city, state, zip, country string,
	createdAt, updatedAt string,
) (ShippingAddress, error) {
	ct, err := parseTime(createdAt)
	if err != nil {
		return ShippingAddress{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ut, err := parseTime(updatedAt)
	if err != nil {
		return ShippingAddress{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	return New(id, userID, street, city, state, zip, country, ct, ut)
}

// Helpers
// touch moved to error.go
// parseTime is kept here for constructors that parse timestamps.
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

// ShippingAddressesTableDDL defines the SQL for the shipping_addresses table migration.
const ShippingAddressesTableDDL = `
-- Migration: Initialize ShippingAddress domain
-- Mirrors backend/internal/domain/shippingAddress/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS shipping_addresses (
  id              TEXT        PRIMARY KEY,
  user_id         TEXT        NOT NULL,
  street          TEXT        NOT NULL,
  city            TEXT        NOT NULL,
  state           TEXT        NOT NULL,
  zip_code        TEXT        NOT NULL,
  country         TEXT        NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL,
  updated_at      TIMESTAMPTZ NOT NULL,

  -- Non-empty checks
  CONSTRAINT chk_shipping_addresses_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(user_id)) > 0
    AND char_length(trim(street)) > 0
    AND char_length(trim(city)) > 0
    AND char_length(trim(state)) > 0
    AND char_length(trim(zip_code)) > 0
    AND char_length(trim(country)) > 0
  ),

  -- time order
  CONSTRAINT chk_shipping_addresses_time_order CHECK (updated_at >= created_at)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_shipping_addresses_user_id     ON shipping_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_shipping_addresses_updated_at  ON shipping_addresses(updated_at);

COMMIT;
`
