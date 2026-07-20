// backend/internal/domain/shippingAddress/entity.go
package shippingAddress

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ShippingAddressは配送先住所を表すドメインエンティティです。
//
// IDは配送先住所documentのUUIDです。
// UserIDは配送先住所を所有する認証ユーザーのUIDです。
// IDとUserIDは異なる値です。
type ShippingAddress struct {
	ID      string `json:"id"`
	UserID  string `json:"userId"`
	ZipCode string `json:"zipCode"`
	State   string `json:"state"`
	City    string `json:"city"`
	Street  string `json:"street"`
	Street2 string `json:"street2"`
	Country string `json:"country"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Errors
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

// Domain policy
const (
	DefaultCountry = "JP"

	MaxUserIDLength  = 128
	MaxZipCodeLength = 32
	MaxStateLength   = 100
	MaxCityLength    = 100
	MaxStreetLength  = 200
)

// 日本の郵便番号は、1234567または123-4567を許可します。
var japaneseZipCodePattern = regexp.MustCompile(`^[0-9]{3}-?[0-9]{4}$`)

// 国コードはISO 3166-1 alpha-2形式を使用します。
var countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

// normalizeFieldsは、住所入力をDomainの保存形式へ正規化します。
//
// countryが未指定の場合はJPを使用します。
// 既存クライアントとの互換性のため「日本」もJPへ変換します。
func (a ShippingAddress) normalizeFields() ShippingAddress {
	a.ID = strings.TrimSpace(a.ID)
	a.UserID = strings.TrimSpace(a.UserID)
	a.ZipCode = strings.TrimSpace(a.ZipCode)
	a.State = strings.TrimSpace(a.State)
	a.City = strings.TrimSpace(a.City)
	a.Street = strings.TrimSpace(a.Street)
	a.Street2 = strings.TrimSpace(a.Street2)

	country := strings.TrimSpace(a.Country)
	switch country {
	case "", "日本":
		a.Country = DefaultCountry
	default:
		a.Country = strings.ToUpper(country)
	}

	a.CreatedAt = a.CreatedAt.UTC()
	a.UpdatedAt = a.UpdatedAt.UTC()

	return a
}

func validateRequiredText(
	value string,
	maxLength int,
	invalidError error,
) error {
	if value == "" {
		return invalidError
	}

	if len([]rune(value)) > maxLength {
		return invalidError
	}

	return nil
}

func validateUserID(userID string) error {
	return validateRequiredText(
		userID,
		MaxUserIDLength,
		ErrInvalidUserID,
	)
}

func validateZipCode(zipCode string, country string) error {
	if zipCode == "" {
		return ErrInvalidZipCode
	}

	if len([]rune(zipCode)) > MaxZipCodeLength {
		return ErrInvalidZipCode
	}

	if country == DefaultCountry &&
		!japaneseZipCodePattern.MatchString(zipCode) {
		return ErrInvalidZipCode
	}

	return nil
}

func validateCountry(country string) error {
	if !countryCodePattern.MatchString(country) {
		return ErrInvalidCountry
	}

	return nil
}

func (a ShippingAddress) validateAddressFields() error {
	if err := validateCountry(a.Country); err != nil {
		return err
	}

	if err := validateZipCode(a.ZipCode, a.Country); err != nil {
		return err
	}

	if err := validateRequiredText(
		a.State,
		MaxStateLength,
		ErrInvalidState,
	); err != nil {
		return err
	}

	if err := validateRequiredText(
		a.City,
		MaxCityLength,
		ErrInvalidCity,
	); err != nil {
		return err
	}

	if err := validateRequiredText(
		a.Street,
		MaxStreetLength,
		ErrInvalidStreet,
	); err != nil {
		return err
	}

	// Street2は任意項目なので、空文字を許可します。
	// ErrInvalidStreet2は定義しません。

	return nil
}

func (a ShippingAddress) validateTimestamps() error {
	if a.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if a.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	return nil
}

func (a ShippingAddress) validateCommon() error {
	if err := validateUserID(a.UserID); err != nil {
		return err
	}

	if err := a.validateAddressFields(); err != nil {
		return err
	}

	return a.validateTimestamps()
}

// validateは、ID採番済みのShippingAddressを検証します。
func (a ShippingAddress) validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}

	if _, err := uuid.Parse(a.ID); err != nil {
		return ErrInvalidID
	}

	return a.validateCommon()
}

// validateForCreateは、ID採番前のShippingAddressを検証します。
//
// IDはUsecaseで採番するため、空文字を許可します。
func (a ShippingAddress) validateForCreate() error {
	if a.ID != "" {
		if _, err := uuid.Parse(a.ID); err != nil {
			return ErrInvalidID
		}
	}

	return a.validateCommon()
}

// UpdateFromFormは、フォームから受け取った住所情報を更新します。
//
// ID、UserIDおよびCreatedAtは変更しません。
// Countryが空の場合はDomain規則に従ってJPへ正規化します。
// Street2は任意項目です。
func (a *ShippingAddress) UpdateFromForm(
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	now time.Time,
) error {
	if a == nil {
		return ErrInvalidID
	}

	next := ShippingAddress{
		ID:        a.ID,
		UserID:    a.UserID,
		ZipCode:   zipCode,
		State:     state,
		City:      city,
		Street:    street,
		Street2:   street2,
		Country:   country,
		CreatedAt: a.CreatedAt,
		UpdatedAt: now,
	}.normalizeFields()

	if err := next.validate(); err != nil {
		return err
	}

	*a = next

	return nil
}

// touchはUpdatedAtを更新します。
func (a *ShippingAddress) touch(now time.Time) error {
	if a == nil {
		return ErrInvalidID
	}

	now = now.UTC()
	if now.IsZero() || now.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	a.UpdatedAt = now

	return nil
}

// Newは、ID採番済みのShippingAddressを生成します。
func New(
	id string,
	userID string,
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	createdAt time.Time,
	updatedAt time.Time,
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
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}.normalizeFields()

	if err := a.validate(); err != nil {
		return ShippingAddress{}, err
	}

	return a, nil
}

// NewWithNowは、CreatedAtとUpdatedAtに同じserver時刻を設定して、
// ID採番済みのShippingAddressを生成します。
func NewWithNow(
	id string,
	userID string,
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	now time.Time,
) (ShippingAddress, error) {
	now = now.UTC()

	return New(
		id,
		userID,
		zipCode,
		state,
		city,
		street,
		street2,
		country,
		now,
		now,
	)
}

// NewForCreateWithNowは、ID採番前のShippingAddressを生成します。
//
// IDは空文字で生成し、UsecaseがUUIDを採番した後、
// NewまたはNewWithNowを使用してIDを含む完全なEntityを確定します。
func NewForCreateWithNow(
	userID string,
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	now time.Time,
) (ShippingAddress, error) {
	now = now.UTC()

	a := ShippingAddress{
		UserID:    userID,
		ZipCode:   zipCode,
		State:     state,
		City:      city,
		Street:    street,
		Street2:   street2,
		Country:   country,
		CreatedAt: now,
		UpdatedAt: now,
	}.normalizeFields()

	if err := a.validateForCreate(); err != nil {
		return ShippingAddress{}, err
	}

	return a, nil
}
