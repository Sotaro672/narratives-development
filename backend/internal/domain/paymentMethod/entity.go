// backend/internal/domain/paymentMethod/entity.go
package paymentMethod

import (
	"errors"
	"regexp"
	"time"
)

// PaymentMethod エンティティ
//
// Stripe 接続を前提とした「保存済み支払い方法」のドメイン表現です。
// このエンティティはクレジットカード番号や CVC を保持しません。
// 保存するのは Stripe が発行した識別子と、UI 表示や既定カード管理に必要な最小限の属性です。
//
// 想定ユースケース:
// - 1 user : N paymentMethods
// - isDefault=true : その user の既定支払い方法
// - isDefault=false: 既定ではない支払い方法
//
// 注意:
//   - 生の cardNumber / cvc は保持しません。
//   - 同一 user に isDefault=true が複数存在しない制約は、
//     単体エンティティではなく usecase / repository 側で担保します。
type PaymentMethod struct {
	ID string `json:"id"`

	// アプリ内ユーザーID
	UserID string `json:"userId"`

	// Stripe Customer ID (例: cus_xxx)
	StripeCustomerID string `json:"stripeCustomerId"`

	// Stripe PaymentMethod ID (例: pm_xxx)
	StripePaymentMethodID string `json:"stripePaymentMethodId"`

	// 表示用カード情報
	Brand    string `json:"brand"`    // visa, mastercard, amex ...
	Last4    string `json:"last4"`    // 下4桁
	ExpMonth int    `json:"expMonth"` // 1..12
	ExpYear  int    `json:"expYear"`  // 4桁年

	// 契約者名義（Stripe billing_details.name 相当）
	CardholderName string `json:"cardholderName"`

	// その user の既定支払い方法か
	IsDefault bool `json:"isDefault"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Errors
var (
	ErrInvalidID                  = errors.New("paymentMethod: invalid id")
	ErrInvalidUserID              = errors.New("paymentMethod: invalid userId")
	ErrInvalidStripeCustomerID    = errors.New("paymentMethod: invalid stripeCustomerId")
	ErrInvalidStripePaymentMethod = errors.New("paymentMethod: invalid stripePaymentMethodId")
	ErrInvalidBrand               = errors.New("paymentMethod: invalid brand")
	ErrInvalidLast4               = errors.New("paymentMethod: invalid last4")
	ErrInvalidExpMonth            = errors.New("paymentMethod: invalid expMonth")
	ErrInvalidExpYear             = errors.New("paymentMethod: invalid expYear")
	ErrInvalidCardholderName      = errors.New("paymentMethod: invalid cardholderName")
	ErrInvalidCreatedAt           = errors.New("paymentMethod: invalid createdAt")
	ErrInvalidUpdatedAt           = errors.New("paymentMethod: invalid updatedAt")
)

var (
	last4Re             = regexp.MustCompile(`^\d{4}$`)
	stripeCustomerIDRe  = regexp.MustCompile(`^cus_[A-Za-z0-9]+$`)
	stripePaymentIDRe   = regexp.MustCompile(`^pm_[A-Za-z0-9]+$`)
	brandAllowedPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{2,32}$`)
)

// ============================================================
// Validation
// ============================================================

func (p PaymentMethod) validate() error {
	if p.ID == "" {
		return ErrInvalidID
	}
	if p.UserID == "" {
		return ErrInvalidUserID
	}
	if p.StripeCustomerID == "" || !stripeCustomerIDRe.MatchString(p.StripeCustomerID) {
		return ErrInvalidStripeCustomerID
	}
	if p.StripePaymentMethodID == "" || !stripePaymentIDRe.MatchString(p.StripePaymentMethodID) {
		return ErrInvalidStripePaymentMethod
	}
	if p.Brand == "" || !brandAllowedPattern.MatchString(p.Brand) {
		return ErrInvalidBrand
	}
	if !last4Re.MatchString(p.Last4) {
		return ErrInvalidLast4
	}
	if p.ExpMonth < 1 || p.ExpMonth > 12 {
		return ErrInvalidExpMonth
	}
	if p.ExpYear < 2000 || p.ExpYear > 9999 {
		return ErrInvalidExpYear
	}
	if p.CardholderName == "" {
		return ErrInvalidCardholderName
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if p.UpdatedAt.IsZero() || p.UpdatedAt.Before(p.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// ============================================================
// Behavior
// ============================================================

// UpdateDisplayInfo は Stripe から取得した表示用情報の更新に使います。
// 生カード情報は扱わず、brand / last4 / exp / cardholderName のみを更新します。
func (p *PaymentMethod) UpdateDisplayInfo(
	brand string,
	last4 string,
	expMonth int,
	expYear int,
	cardholderName string,
	now time.Time,
) error {
	if brand == "" || !brandAllowedPattern.MatchString(brand) {
		return ErrInvalidBrand
	}
	if !last4Re.MatchString(last4) {
		return ErrInvalidLast4
	}
	if expMonth < 1 || expMonth > 12 {
		return ErrInvalidExpMonth
	}
	if expYear < 2000 || expYear > 9999 {
		return ErrInvalidExpYear
	}
	if cardholderName == "" {
		return ErrInvalidCardholderName
	}

	p.Brand = brand
	p.Last4 = last4
	p.ExpMonth = expMonth
	p.ExpYear = expYear
	p.CardholderName = cardholderName

	return p.touch(now)
}

// SetDefault はこの支払い方法を既定にします。
// 同一 user の他 paymentMethods を false にする責務は usecase / repository 側で担保します。
func (p *PaymentMethod) SetDefault(now time.Time) error {
	p.IsDefault = true
	return p.touch(now)
}

// UnsetDefault はこの支払い方法を既定ではない状態にします。
func (p *PaymentMethod) UnsetDefault(now time.Time) error {
	p.IsDefault = false
	return p.touch(now)
}

// ============================================================
// Constructors
// ============================================================

func New(
	id string,
	userID string,
	stripeCustomerID string,
	stripePaymentMethodID string,
	brand string,
	last4 string,
	expMonth int,
	expYear int,
	cardholderName string,
	isDefault bool,
	createdAt, updatedAt time.Time,
) (PaymentMethod, error) {
	pm := PaymentMethod{
		ID:                    id,
		UserID:                userID,
		StripeCustomerID:      stripeCustomerID,
		StripePaymentMethodID: stripePaymentMethodID,
		Brand:                 brand,
		Last4:                 last4,
		ExpMonth:              expMonth,
		ExpYear:               expYear,
		CardholderName:        cardholderName,
		IsDefault:             isDefault,
		CreatedAt:             createdAt.UTC(),
		UpdatedAt:             updatedAt.UTC(),
	}

	if err := pm.validate(); err != nil {
		return PaymentMethod{}, err
	}
	return pm, nil
}

func NewWithNow(
	id string,
	userID string,
	stripeCustomerID string,
	stripePaymentMethodID string,
	brand string,
	last4 string,
	expMonth int,
	expYear int,
	cardholderName string,
	isDefault bool,
	now time.Time,
) (PaymentMethod, error) {
	now = now.UTC()
	return New(
		id,
		userID,
		stripeCustomerID,
		stripePaymentMethodID,
		brand,
		last4,
		expMonth,
		expYear,
		cardholderName,
		isDefault,
		now,
		now,
	)
}

// ============================================================
// Helpers
// ============================================================

func (p *PaymentMethod) touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	p.UpdatedAt = now.UTC()
	return nil
}

// ============================================================
// Patch type (partial update; nil means "no change")
// ============================================================

type PaymentMethodPatch struct {
	Brand          *string
	Last4          *string
	ExpMonth       *int
	ExpYear        *int
	CardholderName *string
	IsDefault      *bool

	UpdatedAt *time.Time
}
