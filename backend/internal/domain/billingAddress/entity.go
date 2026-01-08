// backend/internal/domain/billingAddress/entity.go
package billingAddress

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// BillingAddress エンティティ（Mallアプリの billing_address.dart の入力欄に準拠）
//
// frontend/mall/lib/features/auth/presentation/page/billing_address.dart
// - クレジットカード番号: cardNumber
// - 契約者名義: cardholderName
// - 裏の3桁コード: cvc
//
// 注意:
// - これは「入力欄に一致するドメイン表現」です（トークン化/ブランド/有効期限などは今は扱わない）。
// - 本番では cardNumber/cvc を保存しない（PCI DSS）ことが一般的。ここでは要件に合わせて“いったん保持可能”にしています。
type BillingAddress struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	CardNumber     string `json:"cardNumber"`     // 入力欄: クレジットカード番号
	CardholderName string `json:"cardholderName"` // 入力欄: 契約者名義
	CVC            string `json:"cvc"`            // 入力欄: 裏の3桁コード（AMEX等は4桁の可能性あり）

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Errors
var (
	ErrInvalidID             = errors.New("billingAddress: invalid id")
	ErrInvalidUserID         = errors.New("billingAddress: invalid userId")
	ErrInvalidCardNumber     = errors.New("billingAddress: invalid cardNumber")
	ErrInvalidCardholderName = errors.New("billingAddress: invalid cardholderName")
	ErrInvalidCVC            = errors.New("billingAddress: invalid cvc")
	ErrInvalidCreatedAt      = errors.New("billingAddress: invalid createdAt")
	ErrInvalidUpdatedAt      = errors.New("billingAddress: invalid updatedAt")
)

var (
	// 既存ポリシーを踏襲（UUID）
	uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

	// CVC: 3桁（要件どおり）。※将来 AMEX(4桁) 対応するなら {3,4} にする。
	cvc3Re = regexp.MustCompile(`^\d{3}$`)
)

// ============================================================
// Validation
// ============================================================

func (b BillingAddress) validate() error {
	if strings.TrimSpace(b.ID) == "" || !uuidRe.MatchString(strings.TrimSpace(b.ID)) {
		return ErrInvalidID
	}
	if strings.TrimSpace(b.UserID) == "" {
		return ErrInvalidUserID
	}

	// cardNumber: 数字のみ（空白/ハイフンは許容して正規化する）
	n := normalizeCardNumber(b.CardNumber)
	if n == "" {
		return ErrInvalidCardNumber
	}
	// ざっくり長さチェック（一般的な範囲 12-19）
	if ln := len(n); ln < 12 || ln > 19 {
		return ErrInvalidCardNumber
	}
	// Luhn チェック（一般的なカード番号検証）
	if !luhnValid(n) {
		return ErrInvalidCardNumber
	}

	if strings.TrimSpace(b.CardholderName) == "" {
		return ErrInvalidCardholderName
	}

	c := normalizeDigits(b.CVC)
	if !cvc3Re.MatchString(c) {
		return ErrInvalidCVC
	}

	if b.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if b.UpdatedAt.IsZero() || b.UpdatedAt.Before(b.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// ============================================================
// Behavior
// ============================================================

// UpdateFromForm は billing_address.dart の入力欄に対応する更新メソッドです。
func (b *BillingAddress) UpdateFromForm(cardNumber, cardholderName, cvc string, now time.Time) error {
	cardNumber = strings.TrimSpace(cardNumber)
	cardholderName = strings.TrimSpace(cardholderName)
	cvc = strings.TrimSpace(cvc)

	// 正規化してから保持
	n := normalizeCardNumber(cardNumber)
	if n == "" {
		return ErrInvalidCardNumber
	}
	if ln := len(n); ln < 12 || ln > 19 {
		return ErrInvalidCardNumber
	}
	if !luhnValid(n) {
		return ErrInvalidCardNumber
	}

	if cardholderName == "" {
		return ErrInvalidCardholderName
	}

	cc := normalizeDigits(cvc)
	if !cvc3Re.MatchString(cc) {
		return ErrInvalidCVC
	}

	b.CardNumber = n
	b.CardholderName = cardholderName
	b.CVC = cc

	return b.touch(now)
}

// ============================================================
// Constructors
// ============================================================

func New(
	id string,
	userID string,
	cardNumber string,
	cardholderName string,
	cvc string,
	createdAt, updatedAt time.Time,
) (BillingAddress, error) {
	ba := BillingAddress{
		ID:             strings.TrimSpace(id),
		UserID:         strings.TrimSpace(userID),
		CardNumber:     strings.TrimSpace(cardNumber),
		CardholderName: strings.TrimSpace(cardholderName),
		CVC:            strings.TrimSpace(cvc),
		CreatedAt:      createdAt.UTC(),
		UpdatedAt:      updatedAt.UTC(),
	}
	// 保存用の正規化（バリデーション前に整形）
	ba.CardNumber = normalizeCardNumber(ba.CardNumber)
	ba.CVC = normalizeDigits(ba.CVC)

	if err := ba.validate(); err != nil {
		return BillingAddress{}, err
	}
	return ba, nil
}

func NewWithNow(
	id string,
	userID string,
	cardNumber string,
	cardholderName string,
	cvc string,
	now time.Time,
) (BillingAddress, error) {
	now = now.UTC()
	return New(id, userID, cardNumber, cardholderName, cvc, now, now)
}

func NewFromStringTimes(
	id string,
	userID string,
	cardNumber string,
	cardholderName string,
	cvc string,
	createdAtStr, updatedAtStr string,
) (BillingAddress, error) {
	ca, err := parseTime(createdAtStr)
	if err != nil {
		return BillingAddress{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ua, err := parseTime(updatedAtStr)
	if err != nil {
		return BillingAddress{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	return New(id, userID, cardNumber, cardholderName, cvc, ca, ua)
}

// ============================================================
// Helpers
// ============================================================

func (b *BillingAddress) touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	b.UpdatedAt = now.UTC()
	return nil
}

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

// normalizeDigits: 数字以外を除去
func normalizeDigits(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizeCardNumber: 空白/ハイフン等を除去して数字のみへ
func normalizeCardNumber(s string) string {
	return normalizeDigits(s)
}

// luhnValid performs Luhn checksum validation.
func luhnValid(number string) bool {
	// number must be digits only
	if number == "" {
		return false
	}
	sum := 0
	alt := false
	for i := len(number) - 1; i >= 0; i-- {
		c := number[i]
		if c < '0' || c > '9' {
			return false
		}
		n := int(c - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

// ============================================================
// Patch type (partial update; nil means "no change")
// ============================================================

type BillingAddressPatch struct {
	CardNumber     *string
	CardholderName *string
	CVC            *string

	UpdatedAt *time.Time
}
