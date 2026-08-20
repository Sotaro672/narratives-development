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
// UserIDは配送先住所を登録・所有する認証ユーザーのUIDです。
// CompanyIDは配送先住所が所属するcompanyのdocument IDです。
// Nameは配送先住所・在庫保管場所を識別する名称です。
// CreatedBy / UpdatedByはConsoleから登録・更新された場合のみmember document IDを保持します。
// User側から登録された配送先住所ではCreatedBy / UpdatedByは空文字です。
// 1つのCompanyは複数のShippingAddressを保持できます。
// ID、UserID、CompanyIDはそれぞれ異なる識別子です。
type ShippingAddress struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	CompanyID string `json:"companyId"`
	Name      string `json:"name"`
	ZipCode   string `json:"zipCode"`
	State     string `json:"state"`
	City      string `json:"city"`
	Street    string `json:"street"`
	Street2   string `json:"street2"`
	Country   string `json:"country"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
}

// Errors
var (
	ErrInvalidID        = errors.New("shippingAddress: invalid id")
	ErrInvalidUserID    = errors.New("shippingAddress: invalid userId")
	ErrInvalidCompanyID = errors.New("shippingAddress: invalid companyId")
	ErrInvalidName      = errors.New("shippingAddress: invalid name")
	ErrInvalidStreet    = errors.New("shippingAddress: invalid street")
	ErrInvalidCity      = errors.New("shippingAddress: invalid city")
	ErrInvalidState     = errors.New("shippingAddress: invalid state")
	ErrInvalidZipCode   = errors.New("shippingAddress: invalid zipCode")
	ErrInvalidCountry   = errors.New("shippingAddress: invalid country")
	ErrInvalidCreatedAt = errors.New("shippingAddress: invalid createdAt")
	ErrInvalidCreatedBy = errors.New("shippingAddress: invalid createdBy")
	ErrInvalidUpdatedAt = errors.New("shippingAddress: invalid updatedAt")
	ErrInvalidUpdatedBy = errors.New("shippingAddress: invalid updatedBy")
)

// Domain policy
const (
	DefaultCountry = "JP"

	MaxUserIDLength    = 128
	MaxCompanyIDLength = 128
	MaxMemberIDLength  = 128
	MaxNameLength      = 100
	MaxZipCodeLength   = 32
	MaxStateLength     = 100
	MaxCityLength      = 100
	MaxStreetLength    = 200
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
	switch a.Country {
	case "", "日本":
		a.Country = DefaultCountry
	default:
		a.Country = strings.ToUpper(a.Country)
	}

	a.CreatedAt = a.CreatedAt.UTC()
	a.UpdatedAt = a.UpdatedAt.UTC()
	return a
}

func validateRequiredText(value string, maxLength int, invalidError error) error {
	if value == "" {
		return invalidError
	}
	if len([]rune(value)) > maxLength {
		return invalidError
	}
	return nil
}

func validateOptionalMemberID(value string, invalidError error) error {
	if value == "" {
		return nil
	}
	if len([]rune(value)) > MaxMemberIDLength {
		return invalidError
	}
	return nil
}

func validateUserID(userID string) error {
	return validateRequiredText(userID, MaxUserIDLength, ErrInvalidUserID)
}

func validateCompanyID(companyID string) error {
	return validateRequiredText(companyID, MaxCompanyIDLength, ErrInvalidCompanyID)
}

func validateName(name string) error {
	return validateRequiredText(name, MaxNameLength, ErrInvalidName)
}

func validateZipCode(zipCode string, country string) error {
	if zipCode == "" {
		return ErrInvalidZipCode
	}
	if len([]rune(zipCode)) > MaxZipCodeLength {
		return ErrInvalidZipCode
	}
	if country == DefaultCountry && !japaneseZipCodePattern.MatchString(zipCode) {
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
	if err := validateName(a.Name); err != nil {
		return err
	}
	if err := validateCountry(a.Country); err != nil {
		return err
	}
	if err := validateZipCode(a.ZipCode, a.Country); err != nil {
		return err
	}
	if err := validateRequiredText(a.State, MaxStateLength, ErrInvalidState); err != nil {
		return err
	}
	if err := validateRequiredText(a.City, MaxCityLength, ErrInvalidCity); err != nil {
		return err
	}
	if err := validateRequiredText(a.Street, MaxStreetLength, ErrInvalidStreet); err != nil {
		return err
	}

	return nil
}

func (a ShippingAddress) validateAuditFields() error {
	if err := validateOptionalMemberID(a.CreatedBy, ErrInvalidCreatedBy); err != nil {
		return err
	}
	if err := validateOptionalMemberID(a.UpdatedBy, ErrInvalidUpdatedBy); err != nil {
		return err
	}
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
	if err := validateCompanyID(a.CompanyID); err != nil {
		return err
	}
	if err := a.validateAddressFields(); err != nil {
		return err
	}
	if err := a.validateAuditFields(); err != nil {
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

// UpdateFromFormは、User側のフォームから受け取った住所情報を更新します。
//
// ID、UserID、CompanyID、CreatedAt、CreatedBy、UpdatedByは変更しません。
// UpdatedAtのみ更新します。
// Street2は任意項目です。
func (a *ShippingAddress) UpdateFromForm(
	name string,
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
		CompanyID: a.CompanyID,
		Name:      name,
		ZipCode:   zipCode,
		State:     state,
		City:      city,
		Street:    street,
		Street2:   street2,
		Country:   country,
		CreatedAt: a.CreatedAt,
		CreatedBy: a.CreatedBy,
		UpdatedAt: now,
		UpdatedBy: a.UpdatedBy,
	}.normalizeFields()

	if err := next.validate(); err != nil {
		return err
	}

	*a = next
	return nil
}

// UpdateFromCompanyFormは、Consoleから受け取った在庫保管場所情報を更新します。
//
// ID、UserID、CompanyID、CreatedAt、CreatedByは変更しません。
// UpdatedByには更新を実行したmember document IDを設定します。
// UpdatedAtはserver時刻へ更新します。
func (a *ShippingAddress) UpdateFromCompanyForm(
	name string,
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	updatedBy string,
	now time.Time,
) error {
	if a == nil {
		return ErrInvalidID
	}

	if err := validateRequiredText(
		updatedBy,
		MaxMemberIDLength,
		ErrInvalidUpdatedBy,
	); err != nil {
		return err
	}

	next := ShippingAddress{
		ID:        a.ID,
		UserID:    a.UserID,
		CompanyID: a.CompanyID,
		Name:      name,
		ZipCode:   zipCode,
		State:     state,
		City:      city,
		Street:    street,
		Street2:   street2,
		Country:   country,
		CreatedAt: a.CreatedAt,
		CreatedBy: a.CreatedBy,
		UpdatedAt: now,
		UpdatedBy: updatedBy,
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

func newShippingAddress(
	id string,
	userID string,
	companyID string,
	name string,
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	createdAt time.Time,
	createdBy string,
	updatedAt time.Time,
	updatedBy string,
) (ShippingAddress, error) {
	a := ShippingAddress{
		ID:        id,
		UserID:    userID,
		CompanyID: companyID,
		Name:      name,
		ZipCode:   zipCode,
		State:     state,
		City:      city,
		Street:    street,
		Street2:   street2,
		Country:   country,
		CreatedAt: createdAt,
		CreatedBy: createdBy,
		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
	}.normalizeFields()

	if err := a.validate(); err != nil {
		return ShippingAddress{}, err
	}

	return a, nil
}

// Newは、User側など監査member IDを持たないShippingAddressを生成します。
func New(
	id string,
	userID string,
	companyID string,
	name string,
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	createdAt time.Time,
	updatedAt time.Time,
) (ShippingAddress, error) {
	return newShippingAddress(
		id,
		userID,
		companyID,
		name,
		zipCode,
		state,
		city,
		street,
		street2,
		country,
		createdAt,
		"",
		updatedAt,
		"",
	)
}

// NewWithAuditは、Console用の監査member IDを含むShippingAddressを生成します。
func NewWithAudit(
	id string,
	userID string,
	companyID string,
	name string,
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	createdAt time.Time,
	createdBy string,
	updatedAt time.Time,
	updatedBy string,
) (ShippingAddress, error) {
	return newShippingAddress(
		id,
		userID,
		companyID,
		name,
		zipCode,
		state,
		city,
		street,
		street2,
		country,
		createdAt,
		createdBy,
		updatedAt,
		updatedBy,
	)
}

// NewWithNowは、User側など監査member IDを持たないShippingAddressを生成します。
func NewWithNow(
	id string,
	userID string,
	companyID string,
	name string,
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
		companyID,
		name,
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

// NewWithNowAndAuditは、Consoleから作成する在庫保管場所を生成します。
//
// CreatedByとUpdatedByには同じmember document IDを設定します。
func NewWithNowAndAudit(
	id string,
	userID string,
	companyID string,
	name string,
	zipCode string,
	state string,
	city string,
	street string,
	street2 string,
	country string,
	createdBy string,
	now time.Time,
) (ShippingAddress, error) {
	now = now.UTC()

	if err := validateRequiredText(
		createdBy,
		MaxMemberIDLength,
		ErrInvalidCreatedBy,
	); err != nil {
		return ShippingAddress{}, err
	}

	return NewWithAudit(
		id,
		userID,
		companyID,
		name,
		zipCode,
		state,
		city,
		street,
		street2,
		country,
		now,
		createdBy,
		now,
		createdBy,
	)
}

// NewForCreateWithNowは、ID採番前のShippingAddressを生成します.
//
// IDは空文字で生成し、UsecaseがUUIDを採番した後、
// NewまたはNewWithNowを使用してIDを含む完全なEntityを確定します。
// User側の作成用途なのでCreatedBy / UpdatedByは設定しません。
func NewForCreateWithNow(
	userID string,
	companyID string,
	name string,
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
		CompanyID: companyID,
		Name:      name,
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
