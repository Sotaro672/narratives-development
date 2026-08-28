// backend/internal/application/port/stripe_payment_method_gateway.go
package port

import "context"

// DevelopmentTestPaymentMethodResult は、development 環境で
// Stripe に作成したテスト用 PaymentMethod の結果です。
type DevelopmentTestPaymentMethodResult struct {
	StripeCustomerID      string
	StripePaymentMethodID string
	Brand                 string
	Last4                 string
	ExpMonth              int
	ExpYear               int
}

// StripePaymentMethodGateway は、PaymentMethod 登録に必要な
// Stripe Customer、SetupIntent および development 用
// テスト PaymentMethod の操作を定義します.
//
// cardNumber および CVC などの生カード情報は扱いません。
// 生カード情報は Stripe.js / Elements から直接 Stripe へ送信します。
// development 用テスト PaymentMethod についても Stripe の
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
