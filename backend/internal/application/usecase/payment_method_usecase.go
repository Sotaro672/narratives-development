// backend/internal/application/usecase/payment_method_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"
	"unicode"

	pm "narratives/internal/domain/paymentMethod"
)

var (
	ErrSetupIntentNotImplemented = errors.New("paymentMethod: setup intent not implemented")
	ErrInvalidCardNumber         = errors.New("paymentMethod: invalid card number")
	ErrInvalidCVC                = errors.New("paymentMethod: invalid cvc")
)

type StripePaymentMethodGateway interface {
	GetOrCreateCustomer(
		ctx context.Context,
		userID string,
		cardholderName string,
	) (stripeCustomerID string, err error)

	CreateSetupIntent(
		ctx context.Context,
		stripeCustomerID string,
		cardholderName string,
	) (clientSecret string, err error)
}

// PaymentMethodSetupIntentResult は setup-intent endpoint 用の返却値です。
type PaymentMethodSetupIntentResult struct {
	ClientSecret     string `json:"clientSecret"`
	StripeCustomerID string `json:"stripeCustomerId"`
}

type PaymentMethodUsecase struct {
	repo       pm.RepositoryPort
	stripeGate StripePaymentMethodGateway
	now        func() time.Time
}

func NewPaymentMethodUsecase(
	repo pm.RepositoryPort,
	stripeGate StripePaymentMethodGateway,
) *PaymentMethodUsecase {
	return &PaymentMethodUsecase{
		repo:       repo,
		stripeGate: stripeGate,
		now:        time.Now,
	}
}

// SetStripeGateway は後から Stripe gateway を注入したい場合に使います。
func (u *PaymentMethodUsecase) SetStripeGateway(stripeGate StripePaymentMethodGateway) {
	u.stripeGate = stripeGate
}

// ============================================================
// Queries
// ============================================================

func (u *PaymentMethodUsecase) GetByID(ctx context.Context, id string) (*pm.PaymentMethod, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *PaymentMethodUsecase) GetByUser(ctx context.Context, userID string) ([]pm.PaymentMethod, error) {
	return u.repo.GetByUser(ctx, userID)
}

func (u *PaymentMethodUsecase) GetDefaultByUser(ctx context.Context, userID string) (*pm.PaymentMethod, error) {
	return u.repo.GetDefaultByUser(ctx, userID)
}

func (u *PaymentMethodUsecase) GetByStripePaymentMethodID(ctx context.Context, stripePaymentMethodID string) (*pm.PaymentMethod, error) {
	return u.repo.GetByStripePaymentMethodID(ctx, stripePaymentMethodID)
}

// ============================================================
// Stripe setup-intent
// ============================================================

func (u *PaymentMethodUsecase) CreateSetupIntent(
	ctx context.Context,
	userID string,
	cardholderName string,
) (*PaymentMethodSetupIntentResult, error) {
	if u.stripeGate == nil {
		return nil, ErrSetupIntentNotImplemented
	}
	if userID == "" {
		return nil, pm.ErrInvalidUserID
	}
	if cardholderName == "" {
		return nil, pm.ErrInvalidCardholderName
	}

	stripeCustomerID, err := u.stripeGate.GetOrCreateCustomer(ctx, userID, cardholderName)
	if err != nil {
		return nil, err
	}
	if stripeCustomerID == "" {
		return nil, pm.ErrInvalidStripeCustomerID
	}

	clientSecret, err := u.stripeGate.CreateSetupIntent(ctx, stripeCustomerID, cardholderName)
	if err != nil {
		return nil, err
	}
	if clientSecret == "" {
		return nil, ErrSetupIntentNotImplemented
	}

	return &PaymentMethodSetupIntentResult{
		ClientSecret:     clientSecret,
		StripeCustomerID: stripeCustomerID,
	}, nil
}

// ============================================================
// Commands
// ============================================================

func (u *PaymentMethodUsecase) Create(ctx context.Context, in pm.CreatePaymentMethodInput) (*pm.PaymentMethod, error) {
	now := u.now().UTC()

	if in.UserID == "" {
		return nil, pm.ErrInvalidUserID
	}
	if in.CardholderName == "" {
		return nil, pm.ErrInvalidCardholderName
	}
	if in.StripePaymentMethodID == "" {
		return nil, pm.ErrInvalidStripePaymentMethod
	}
	if in.Brand == "" {
		return nil, pm.ErrInvalidBrand
	}

	in.Last4 = normalizeLast4(in.Last4)
	if in.Last4 == "" {
		return nil, pm.ErrInvalidLast4
	}

	if in.CreatedAt == nil || in.CreatedAt.IsZero() {
		t := now
		in.CreatedAt = &t
	}
	if in.UpdatedAt == nil || in.UpdatedAt.IsZero() {
		t := now
		in.UpdatedAt = &t
	}

	// Stripe Customer が未指定なら user 単位で取得または新規作成します。
	if in.StripeCustomerID == "" {
		if u.stripeGate == nil {
			return nil, pm.ErrInvalidStripeCustomerID
		}

		stripeCustomerID, err := u.stripeGate.GetOrCreateCustomer(ctx, in.UserID, in.CardholderName)
		if err != nil {
			return nil, err
		}
		if stripeCustomerID == "" {
			return nil, pm.ErrInvalidStripeCustomerID
		}
		in.StripeCustomerID = stripeCustomerID
	}

	if in.IsDefault {
		if err := u.repo.ClearDefaultByUser(ctx, in.UserID); err != nil {
			return nil, err
		}
	}

	return u.repo.Create(ctx, in)
}

func (u *PaymentMethodUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}

func (u *PaymentMethodUsecase) SetDefault(ctx context.Context, id string, userID string) (*pm.PaymentMethod, error) {
	now := u.now().UTC()

	if err := u.repo.ClearDefaultByUser(ctx, userID); err != nil {
		return nil, err
	}

	return u.repo.SetDefault(ctx, id, userID, now)
}

func normalizeLast4(v string) string {
	digits := make([]rune, 0, len(v))
	for _, r := range v {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}

	if len(digits) <= 4 {
		return string(digits)
	}
	return string(digits[len(digits)-4:])
}

func normalizeDigits(v string) string {
	digits := make([]rune, 0, len(v))
	for _, r := range v {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}
	return string(digits)
}
