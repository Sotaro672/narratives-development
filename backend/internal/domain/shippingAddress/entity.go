package shippingAddress

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ShippingAddress エンティティ（SNSアプリの配送先入力欄に準拠）
//
// frontend/sns/lib/features/auth/presentation/page/shipping_address.dart
// - 郵便番号: zipCode
// - 都道府県: state
// - 市区町村: city
// - 住所１（番地など）: street
// - 住所２（建物名・部屋番号など）: street2（任意）
type ShippingAddress struct {
	ID      string `json:"id"`
	UserID  string `json:"userId"`
	ZipCode string `json:"zipCode"` // 郵便番号
	State   string `json:"state"`   // 都道府県
	City    string `json:"city"`    // 市区町村
	Street  string `json:"street"`  // 住所１（番地など）
	Street2 string `json:"street2"` // 住所２（建物名・部屋番号など）任意
	Country string `json:"country"` // 国（UI入力が無い場合は実装側で "JP"/"日本" を入れる想定）

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Errors
var (
	ErrInvalidID        = errors.New("shippingAddress: invalid id")
	ErrInvalidUserID    = errors.New("shippingAddress: invalid userId")
	ErrInvalidStreet    = errors.New("shippingAddress: invalid street")
	ErrInvalidStreet2   = errors.New("shippingAddress: invalid street2")
	ErrInvalidCity      = errors.New("shippingAddress: invalid city")
	ErrInvalidState     = errors.New("shippingAddress: invalid state")
	ErrInvalidZipCode   = errors.New("shippingAddress: invalid zipCode")
	ErrInvalidCountry   = errors.New("shippingAddress: invalid country")
	ErrInvalidCreatedAt = errors.New("shippingAddress: invalid createdAt")
	ErrInvalidUpdatedAt = errors.New("shippingAddress: invalid updatedAt")
)

// Validation
func (a ShippingAddress) validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(a.UserID) == "" {
		return ErrInvalidUserID
	}

	if strings.TrimSpace(a.ZipCode) == "" {
		return ErrInvalidZipCode
	}
	if strings.TrimSpace(a.State) == "" {
		return ErrInvalidState
	}
	if strings.TrimSpace(a.City) == "" {
		return ErrInvalidCity
	}
	if strings.TrimSpace(a.Street) == "" {
		return ErrInvalidStreet
	}

	// Street2 は任意（ただし値が入るなら trim して空は不可扱いにする）
	if a.Street2 != "" && strings.TrimSpace(a.Street2) == "" {
		return ErrInvalidStreet2
	}

	if strings.TrimSpace(a.Country) == "" {
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

// Behavior
//
// フロントの入力欄に対応する更新メソッド
func (a *ShippingAddress) UpdateFromForm(
	zipCode, state, city, street, street2, country string,
	now time.Time,
) error {
	zipCode = strings.TrimSpace(zipCode)
	state = strings.TrimSpace(state)
	city = strings.TrimSpace(city)
	street = strings.TrimSpace(street)
	street2 = strings.TrimSpace(street2)
	country = strings.TrimSpace(country)

	if zipCode == "" {
		return ErrInvalidZipCode
	}
	if state == "" {
		return ErrInvalidState
	}
	if city == "" {
		return ErrInvalidCity
	}
	if street == "" {
		return ErrInvalidStreet
	}
	// street2 は任意
	// country は必須（UI入力が無い場合は呼び出し側で "JP"/"日本" を入れる）
	if country == "" {
		return ErrInvalidCountry
	}

	a.ZipCode = zipCode
	a.State = state
	a.City = city
	a.Street = street
	a.Street2 = street2
	a.Country = country

	return a.touch(now)
}

// Helpers
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
	zipCode, state, city, street, street2, country string,
	createdAt, updatedAt time.Time,
) (ShippingAddress, error) {
	a := ShippingAddress{
		ID:        strings.TrimSpace(id),
		UserID:    strings.TrimSpace(userID),
		ZipCode:   strings.TrimSpace(zipCode),
		State:     strings.TrimSpace(state),
		City:      strings.TrimSpace(city),
		Street:    strings.TrimSpace(street),
		Street2:   strings.TrimSpace(street2),
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
	zipCode, state, city, street, street2, country string,
	now time.Time,
) (ShippingAddress, error) {
	now = now.UTC()
	return New(id, userID, zipCode, state, city, street, street2, country, now, now)
}

// From DTO-like strings (createdAt/updatedAt as RFC3339)
func NewFromStringTimes(
	id, userID string,
	zipCode, state, city, street, street2, country string,
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
	return New(id, userID, zipCode, state, city, street, street2, country, ct, ut)
}

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
