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
	ErrSetupIntentNotImplemented = errors.New(
		"paymentMethod: setup intent not implemented",
	)
	ErrDevelopmentTestPaymentMethodNotImplemented = errors.New(
		"paymentMethod: development test payment method not implemented",
	)
)

// DevelopmentTestPaymentMethodResultは、development環境で
// Stripeに作成したテスト用PaymentMethodの結果です。
type DevelopmentTestPaymentMethodResult struct {
	StripeCustomerID      string
	StripePaymentMethodID string
	Brand                 string
	Last4                 string
	ExpMonth              int
	ExpYear               int
}

// StripePaymentMethodGatewayは、PaymentMethod登録に必要な
// Stripe Customer、SetupIntentおよびdevelopment用
// テストPaymentMethodの操作を定義します。
//
// cardNumberおよびCVCなどの生カード情報は扱いません。
// 生カード情報はStripe.js / Elementsから直接Stripeへ送信します。
// development用テストPaymentMethodについてもStripeの
// テストトークンを使用し、生カード情報は扱いません。
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

	CreateDevelopmentTestPaymentMethod(
		ctx context.Context,
		userID string,
		cardholderName string,
	) (*DevelopmentTestPaymentMethodResult, error)
}

// PaymentMethodSetupIntentResultは、setup-intent endpoint用の返却値です。
type PaymentMethodSetupIntentResult struct {
	ClientSecret     string `json:"clientSecret"`
	StripeCustomerID string `json:"stripeCustomerId"`
}

// PaymentMethodUsecaseは、PaymentMethodに関するユースケースを提供します。
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

// SetStripeGatewayは、後からStripe gatewayを注入する場合に使用します。
func (u *PaymentMethodUsecase) SetStripeGateway(
	stripeGate StripePaymentMethodGateway,
) {
	u.stripeGate = stripeGate
}

// ============================================================
// Queries
// ============================================================

func (u *PaymentMethodUsecase) GetByID(
	ctx context.Context,
	id string,
) (*pm.PaymentMethod, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *PaymentMethodUsecase) GetByUser(
	ctx context.Context,
	userID string,
) ([]pm.PaymentMethod, error) {
	return u.repo.GetByUser(ctx, userID)
}

func (u *PaymentMethodUsecase) GetDefaultByUser(
	ctx context.Context,
	userID string,
) (*pm.PaymentMethod, error) {
	return u.repo.GetDefaultByUser(ctx, userID)
}

func (u *PaymentMethodUsecase) GetByStripePaymentMethodID(
	ctx context.Context,
	stripePaymentMethodID string,
) (*pm.PaymentMethod, error) {
	return u.repo.GetByStripePaymentMethodID(
		ctx,
		stripePaymentMethodID,
	)
}

// ============================================================
// Stripe setup-intent
// ============================================================

// CreateSetupIntentは、Stripe.js / Elementsでカード情報を登録するための
// SetupIntentを作成します。
//
// 生カード番号およびCVCは、このメソッドには渡しません。
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

	stripeCustomerID, err := u.stripeGate.GetOrCreateCustomer(
		ctx,
		userID,
		cardholderName,
	)
	if err != nil {
		return nil, err
	}
	if stripeCustomerID == "" {
		return nil, pm.ErrInvalidStripeCustomerID
	}

	clientSecret, err := u.stripeGate.CreateSetupIntent(
		ctx,
		stripeCustomerID,
		cardholderName,
	)
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
// Development test PaymentMethod
// ============================================================

// EnsureDevelopmentDefaultPaymentMethodは、development環境で
// ユーザーの既定PaymentMethodが存在しない場合に限り、
// Stripeのテスト用PaymentMethodを作成して既定として保存します。
//
// 既に既定PaymentMethodが存在する場合は新しいPaymentMethodを
// 作成せず、その既定PaymentMethodを返します。
//
// development環境でのみ呼び出すことを前提とし、
// Stripe側でもtest secret keyであることを検証します。
func (u *PaymentMethodUsecase) EnsureDevelopmentDefaultPaymentMethod(
	ctx context.Context,
	userID string,
	cardholderName string,
) (*pm.PaymentMethod, error) {
	if u == nil || u.repo == nil {
		return nil, ErrDevelopmentTestPaymentMethodNotImplemented
	}

	if u.stripeGate == nil {
		return nil, ErrDevelopmentTestPaymentMethodNotImplemented
	}

	if userID == "" {
		return nil, pm.ErrInvalidUserID
	}

	if cardholderName == "" {
		return nil, pm.ErrInvalidCardholderName
	}

	existing, err := u.repo.GetDefaultByUser(
		ctx,
		userID,
	)
	if err == nil && existing != nil {
		return existing, nil
	}

	if err != nil && !errors.Is(err, pm.ErrNotFound) {
		return nil, err
	}

	result, err := u.stripeGate.CreateDevelopmentTestPaymentMethod(
		ctx,
		userID,
		cardholderName,
	)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, ErrDevelopmentTestPaymentMethodNotImplemented
	}

	if result.StripeCustomerID == "" {
		return nil, pm.ErrInvalidStripeCustomerID
	}

	if result.StripePaymentMethodID == "" {
		return nil, pm.ErrInvalidStripePaymentMethod
	}

	if result.Brand == "" {
		return nil, pm.ErrInvalidBrand
	}

	if result.Last4 == "" {
		return nil, pm.ErrInvalidLast4
	}

	if result.ExpMonth <= 0 {
		return nil, pm.ErrInvalidExpMonth
	}

	if result.ExpYear <= 0 {
		return nil, pm.ErrInvalidExpYear
	}

	return u.Create(
		ctx,
		pm.CreatePaymentMethodInput{
			UserID:                userID,
			StripeCustomerID:      result.StripeCustomerID,
			StripePaymentMethodID: result.StripePaymentMethodID,
			Brand:                 result.Brand,
			Last4:                 result.Last4,
			ExpMonth:              result.ExpMonth,
			ExpYear:               result.ExpYear,
			CardholderName:        cardholderName,
			IsDefault:             true,
		},
	)
}

// ============================================================
// Commands
// ============================================================

// Createは、Stripeで作成・確認済みのPaymentMethodを保存します。
//
// StripePaymentMethodIDやカード表示情報はStripe.js / Elementsによる
// 登録完了後の値を使用し、生カード番号およびCVCは受け取りません。
//
// in.IsDefaultがtrueの場合の「既存の既定解除＋新規カード作成」は、
// Repository実装が同一Transaction内で原子的に処理します。
func (u *PaymentMethodUsecase) Create(
	ctx context.Context,
	in pm.CreatePaymentMethodInput,
) (*pm.PaymentMethod, error) {
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
		createdAt := now
		in.CreatedAt = &createdAt
	}
	if in.UpdatedAt == nil || in.UpdatedAt.IsZero() {
		updatedAt := now
		in.UpdatedAt = &updatedAt
	}

	// StripeCustomerIDが未指定の場合は、
	// ユーザー単位でStripe Customerを取得または作成します。
	if in.StripeCustomerID == "" {
		if u.stripeGate == nil {
			return nil, pm.ErrInvalidStripeCustomerID
		}

		stripeCustomerID, err := u.stripeGate.GetOrCreateCustomer(
			ctx,
			in.UserID,
			in.CardholderName,
		)
		if err != nil {
			return nil, err
		}
		if stripeCustomerID == "" {
			return nil, pm.ErrInvalidStripeCustomerID
		}

		in.StripeCustomerID = stripeCustomerID
	}

	// IsDefault=trueの場合でも、Usecaseから既存の既定カードを
	// 個別に解除しません。
	//
	// RepositoryのCreateが、既存の既定解除と新規カード作成を
	// 同一Transaction内で原子的に処理します。
	return u.repo.Create(ctx, in)
}

func (u *PaymentMethodUsecase) Delete(
	ctx context.Context,
	id string,
) error {
	return u.repo.Delete(ctx, id)
}

// SetDefaultは、指定PaymentMethodをユーザーの既定に設定します。
//
// 既存の既定解除と対象カードの既定化は、RepositoryのSetDefaultが
// 同一Transaction内で原子的に処理します。
func (u *PaymentMethodUsecase) SetDefault(
	ctx context.Context,
	id string,
	userID string,
) (*pm.PaymentMethod, error) {
	now := u.now().UTC()

	// Usecaseから既存の既定カードを個別に解除しません。
	//
	// RepositoryのSetDefaultが、既存の既定解除と対象カードの更新を
	// 同一Transaction内で原子的に処理します。
	return u.repo.SetDefault(
		ctx,
		id,
		userID,
		now,
	)
}

// normalizeLast4は数字以外を除去し、末尾4桁を返します。
func normalizeLast4(value string) string {
	digits := make([]rune, 0, len(value))

	for _, character := range value {
		if unicode.IsDigit(character) {
			digits = append(digits, character)
		}
	}

	if len(digits) <= 4 {
		return string(digits)
	}

	return string(digits[len(digits)-4:])
}
