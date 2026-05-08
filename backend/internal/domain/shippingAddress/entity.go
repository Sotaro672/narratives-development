// backend/internal/domain/shippingAddress/entity.go
package shippingAddress

import (
	"errors"
	"time"
)

// ShippingAddress エンティティ（Mallアプリの配送先入力欄に準拠）
//
// frontend/mall/lib/features/auth/presentation/page/shipping_address.dart
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

// Validation (for existing/update: ID required)
func (a ShippingAddress) validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}
	if a.UserID == "" {
		return ErrInvalidUserID
	}

	if a.ZipCode == "" {
		return ErrInvalidZipCode
	}
	if a.State == "" {
		return ErrInvalidState
	}
	if a.City == "" {
		return ErrInvalidCity
	}
	if a.Street == "" {
		return ErrInvalidStreet
	}

	// Street2 は任意
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

// Validation for Create (ID is assigned by usecase/repository)
func (a ShippingAddress) validateForCreate() error {
	// IDはまだ空でOK（usecaseが採番して埋める）
	if a.UserID == "" {
		return ErrInvalidUserID
	}

	if a.ZipCode == "" {
		return ErrInvalidZipCode
	}
	if a.State == "" {
		return ErrInvalidState
	}
	if a.City == "" {
		return ErrInvalidCity
	}
	if a.Street == "" {
		return ErrInvalidStreet
	}

	// Street2 は任意
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

// Behavior
//
// フロントの入力欄に対応する更新メソッド
func (a *ShippingAddress) UpdateFromForm(
	zipCode, state, city, street, street2, country string,
	now time.Time,
) error {
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
		ID:        id,
		UserID:    userID,
		ZipCode:   zipCode,
		State:     state,
		City:      city,
		Street:    street,
		Street2:   street2,
		Country:   country,
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

// Create constructor: ID is not required (assigned later)
func NewForCreateWithNow(
	userID string,
	zipCode, state, city, street, street2, country string,
	now time.Time,
) (ShippingAddress, error) {
	now = now.UTC()

	a := ShippingAddress{
		ID:        "",
		UserID:    userID,
		ZipCode:   zipCode,
		State:     state,
		City:      city,
		Street:    street,
		Street2:   street2,
		Country:   country,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := a.validateForCreate(); err != nil {
		return ShippingAddress{}, err
	}
	return a, nil
}
