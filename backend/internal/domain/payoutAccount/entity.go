// backend/internal/domain/payoutAccount/entity.go

package payoutAccount

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidUserID          = errors.New("payoutAccount: invalid userId")
	ErrInvalidStripeAccountID = errors.New("payoutAccount: invalid stripeAccountId")
	ErrInvalidBankName        = errors.New("payoutAccount: invalid bankName")
	ErrInvalidBankLast4       = errors.New("payoutAccount: invalid bankLast4")
	ErrInvalidCreatedAt       = errors.New("payoutAccount: invalid createdAt")
	ErrInvalidUpdatedAt       = errors.New("payoutAccount: invalid updatedAt")
)

var (
	MaxUserIDLength          = 128
	MaxStripeAccountIDLength = 255
	MaxBankNameLength        = 100
)

// PayoutAccount represents the Stripe Connect payout destination for a Mall user.
//
// Persistence policy:
//   - Firestore document path: payoutAccounts/{userId}
//   - One User has at most one PayoutAccount.
//   - StripeAccountID identifies the Stripe Connected Account.
//   - Full bank account numbers, branch numbers, and routing numbers are not stored.
//
// Stripe state:
//   - DetailsSubmitted indicates that the user has submitted the currently
//     required onboarding information.
//   - PayoutsEnabled indicates that the Connected Account can receive transfers.
//   - BankName and BankLast4 are display-only snapshots obtained from Stripe.
//
// Resale settlement resolves:
//
//	Resale.AvatarID
//	    -> Avatar.UserID
//	    -> PayoutAccount
//	    -> StripeAccountID
type PayoutAccount struct {
	UserID          string `json:"userId" firestore:"userId"`
	StripeAccountID string `json:"stripeAccountId" firestore:"stripeAccountId"`

	DetailsSubmitted bool `json:"detailsSubmitted" firestore:"detailsSubmitted"`
	PayoutsEnabled   bool `json:"payoutsEnabled" firestore:"payoutsEnabled"`

	BankName  string `json:"bankName,omitempty" firestore:"bankName,omitempty"`
	BankLast4 string `json:"bankLast4,omitempty" firestore:"bankLast4,omitempty"`

	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// New creates a persisted PayoutAccount.
//
// userID is also used as the Firestore document ID by the repository.
func New(
	userID string,
	stripeAccountID string,
	detailsSubmitted bool,
	payoutsEnabled bool,
	bankName string,
	bankLast4 string,
	createdAt time.Time,
	updatedAt time.Time,
) (PayoutAccount, error) {
	account := PayoutAccount{
		UserID:           strings.TrimSpace(userID),
		StripeAccountID:  strings.TrimSpace(stripeAccountID),
		DetailsSubmitted: detailsSubmitted,
		PayoutsEnabled:   payoutsEnabled,
		BankName:         strings.TrimSpace(bankName),
		BankLast4:        strings.TrimSpace(bankLast4),
		CreatedAt:        createdAt.UTC(),
		UpdatedAt:        updatedAt.UTC(),
	}

	if err := account.Validate(); err != nil {
		return PayoutAccount{}, err
	}

	return account, nil
}

// NewWithNow creates a new PayoutAccount using the same timestamp for
// CreatedAt and UpdatedAt.
//
// A newly created Stripe Connected Account normally starts with onboarding
// incomplete and payouts disabled.
func NewWithNow(
	userID string,
	stripeAccountID string,
	now time.Time,
) (PayoutAccount, error) {
	return New(
		userID,
		stripeAccountID,
		false,
		false,
		"",
		"",
		now,
		now,
	)
}

// ApplyStripeState synchronizes the Stripe Connected Account state.
//
// BankName and BankLast4 are display-only values. An empty value is allowed
// when Stripe has not returned a payout bank account yet.
func (a *PayoutAccount) ApplyStripeState(
	detailsSubmitted bool,
	payoutsEnabled bool,
	bankName string,
	bankLast4 string,
	now time.Time,
) error {
	if a == nil {
		return ErrInvalidUserID
	}

	bankName = strings.TrimSpace(bankName)
	bankLast4 = strings.TrimSpace(bankLast4)

	if err := validateBankName(bankName); err != nil {
		return err
	}

	if err := validateBankLast4(bankLast4); err != nil {
		return err
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	a.DetailsSubmitted = detailsSubmitted
	a.PayoutsEnabled = payoutsEnabled
	a.BankName = bankName
	a.BankLast4 = bankLast4
	a.UpdatedAt = now.UTC()

	return nil
}

// SetStripeAccountID associates the PayoutAccount with a Stripe Connected Account.
//
// In normal operation StripeAccountID should be fixed after initial creation.
// This method exists for controlled recovery or migration paths.
func (a *PayoutAccount) SetStripeAccountID(
	stripeAccountID string,
	now time.Time,
) error {
	if a == nil {
		return ErrInvalidUserID
	}

	stripeAccountID = strings.TrimSpace(stripeAccountID)

	if err := validateStripeAccountID(stripeAccountID); err != nil {
		return err
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	a.StripeAccountID = stripeAccountID
	a.UpdatedAt = now.UTC()

	return nil
}

// Validate validates the persisted PayoutAccount state.
func (a PayoutAccount) Validate() error {
	if err := validateUserID(a.UserID); err != nil {
		return err
	}

	if err := validateStripeAccountID(a.StripeAccountID); err != nil {
		return err
	}

	if err := validateBankName(a.BankName); err != nil {
		return err
	}

	if err := validateBankLast4(a.BankLast4); err != nil {
		return err
	}

	if a.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if a.UpdatedAt.IsZero() || a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	return nil
}

func validateUserID(userID string) error {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return ErrInvalidUserID
	}

	if MaxUserIDLength > 0 &&
		len([]rune(userID)) > MaxUserIDLength {
		return ErrInvalidUserID
	}

	return nil
}

func validateStripeAccountID(stripeAccountID string) error {
	stripeAccountID = strings.TrimSpace(stripeAccountID)

	if stripeAccountID == "" ||
		!strings.HasPrefix(stripeAccountID, "acct_") {
		return ErrInvalidStripeAccountID
	}

	if MaxStripeAccountIDLength > 0 &&
		len([]rune(stripeAccountID)) > MaxStripeAccountIDLength {
		return ErrInvalidStripeAccountID
	}

	return nil
}

func validateBankName(bankName string) error {
	bankName = strings.TrimSpace(bankName)

	if bankName == "" {
		return nil
	}

	if MaxBankNameLength > 0 &&
		len([]rune(bankName)) > MaxBankNameLength {
		return ErrInvalidBankName
	}

	return nil
}

func validateBankLast4(bankLast4 string) error {
	bankLast4 = strings.TrimSpace(bankLast4)

	if bankLast4 == "" {
		return nil
	}

	if len(bankLast4) != 4 {
		return ErrInvalidBankLast4
	}

	for _, r := range bankLast4 {
		if r < '0' || r > '9' {
			return ErrInvalidBankLast4
		}
	}

	return nil
}
