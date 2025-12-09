// backend\internal\domain\billingAddress\entity.go
package billingAddress

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// BillingAddress mirrors web-app/src/shared/types/billingAddress.ts
type BillingAddress struct {
	ID            string  `json:"id"`            // UUID
	UserID        string  `json:"userId"`        // user_id (UUID or TEXT)
	NameOnAccount *string `json:"nameOnAccount"` // optional

	BillingType  string  `json:"billingType"` // NOT NULL
	CardBrand    *string `json:"cardBrand"`
	CardLast4    *string `json:"cardLast4"`
	CardExpMonth *int    `json:"cardExpMonth"`
	CardExpYear  *int    `json:"cardExpYear"`
	CardToken    *string `json:"cardToken"`

	PostalCode *int    `json:"postalCode"` // optional INTEGER
	State      *string `json:"state"`
	City       *string `json:"city"`
	Street     *string `json:"street"`
	Country    *string `json:"country"`

	IsDefault bool      `json:"isDefault"` // NOT NULL
	CreatedAt time.Time `json:"createdAt"` // NOT NULL
	UpdatedAt time.Time `json:"updatedAt"` // NOT NULL
}

// Errors
var (
	ErrInvalidID           = errors.New("billingAddress: invalid id")
	ErrInvalidUserID       = errors.New("billingAddress: invalid userId")
	ErrInvalidBillingType  = errors.New("billingAddress: invalid billingType")
	ErrInvalidCardLast4    = errors.New("billingAddress: invalid cardLast4")
	ErrInvalidCardExpMonth = errors.New("billingAddress: invalid cardExpMonth")
	ErrInvalidCardExpYear  = errors.New("billingAddress: invalid cardExpYear")
	ErrInvalidPostalCode   = errors.New("billingAddress: invalid postalCode")
	ErrInvalidCreatedAt    = errors.New("billingAddress: invalid createdAt")
	ErrInvalidUpdatedAt    = errors.New("billingAddress: invalid updatedAt")
)

// Basic policies/regex
var (
	uuidRe     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	last4Re    = regexp.MustCompile(`^\d{4}$`)
	minExpYear = 2000
	maxExpYear = 2100
)

// Constructors

func New(
	id string,
	userID string,
	billingType string,
	isDefault bool,
	createdAt, updatedAt time.Time,
	// optional fields
	nameOnAccount, cardBrand, cardLast4, cardToken, state, city, street, country *string,
	cardExpMonth, cardExpYear, postalCode *int,
) (BillingAddress, error) {
	ba := BillingAddress{
		ID:            strings.TrimSpace(id),
		UserID:        strings.TrimSpace(userID),
		NameOnAccount: trimPtr(nameOnAccount),
		BillingType:   strings.TrimSpace(billingType),
		CardBrand:     trimPtr(cardBrand),
		CardLast4:     trimPtr(cardLast4),
		CardExpMonth:  cardExpMonth,
		CardExpYear:   cardExpYear,
		CardToken:     trimPtr(cardToken),
		PostalCode:    postalCode,
		State:         trimPtr(state),
		City:          trimPtr(city),
		Street:        trimPtr(street),
		Country:       trimPtr(country),
		IsDefault:     isDefault,
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
	}
	if err := ba.validate(); err != nil {
		return BillingAddress{}, err
	}
	return ba, nil
}

func NewFromStringTimes(
	id string,
	userID string,
	billingType string,
	isDefault bool,
	createdAtStr, updatedAtStr string,
	// optional fields
	nameOnAccount, cardBrand, cardLast4, cardToken, state, city, street, country *string,
	cardExpMonth, cardExpYear, postalCode *int,
) (BillingAddress, error) {
	ca, err := parseTime(createdAtStr)
	if err != nil {
		return BillingAddress{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ua, err := parseTime(updatedAtStr)
	if err != nil {
		return BillingAddress{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	return New(
		id, userID, billingType, isDefault, ca, ua,
		nameOnAccount, cardBrand, cardLast4, cardToken, state, city, street, country,
		cardExpMonth, cardExpYear, postalCode,
	)
}

// Validation

func (b BillingAddress) validate() error {
	if strings.TrimSpace(b.ID) == "" || !uuidRe.MatchString(b.ID) {
		return ErrInvalidID
	}
	if strings.TrimSpace(b.UserID) == "" {
		return ErrInvalidUserID
	}
	if strings.TrimSpace(b.BillingType) == "" {
		return ErrInvalidBillingType
	}
	if b.CardLast4 != nil && !last4Re.MatchString(*b.CardLast4) {
		return ErrInvalidCardLast4
	}
	if b.CardExpMonth != nil {
		if *b.CardExpMonth < 1 || *b.CardExpMonth > 12 {
			return ErrInvalidCardExpMonth
		}
	}
	if b.CardExpYear != nil {
		if *b.CardExpYear < minExpYear || *b.CardExpYear > maxExpYear {
			return ErrInvalidCardExpYear
		}
	}
	if b.PostalCode != nil && *b.PostalCode < 0 {
		return ErrInvalidPostalCode
	}
	if b.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if b.UpdatedAt.IsZero() || b.UpdatedAt.Before(b.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// Helpers

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
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

// 部分更新用のパッチ型（nilは未変更を意味します）
type BillingAddressPatch struct {
	FullName   *string
	Company    *string
	Country    *string
	PostalCode *string
	State      *string
	City       *string
	Address1   *string
	Address2   *string
	Phone      *string

	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}
