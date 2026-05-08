// backend/internal/domain/paymentMethod/repository_port.go
package paymentMethod

import (
	"context"
	"errors"
	"time"
)

// 契約（インターフェース）のみ定義。実装はインフラ層に委譲します。

// Create input
//
// 本番運用では create 時に Stripe へ渡すための入力値も扱います。
// Firestore 等の永続化では最終的に以下の保存値へ正規化します。
// - stripeCustomerId
// - stripePaymentMethodId
// - brand
// - last4
// - expMonth
// - expYear
// - cardholderName
// - isDefault
//
// 追加で、Stripe 作成用の生入力として以下を受け取ります。
// - cardNumber
// - cvc
//
// 想定フロー:
// 1. cardNumber / expMonth / expYear / cvc / cardholderName / brand を Stripe に渡す
// 2. Stripe から stripePaymentMethodId / last4 等を取得する
// 3. 最終的な PaymentMethod を保存する
type CreatePaymentMethodInput struct {
	UserID string `json:"userId"`

	// Stripe 作成・連携用
	CardNumber     string `json:"cardNumber"`
	CVC            string `json:"cvc"`
	Brand          string `json:"brand"`
	ExpMonth       int    `json:"expMonth"`
	ExpYear        int    `json:"expYear"`
	CardholderName string `json:"cardholderName"`

	// Stripe / 保存結果
	StripeCustomerID      string `json:"stripeCustomerId"`
	StripePaymentMethodID string `json:"stripePaymentMethodId"`
	Last4                 string `json:"last4"`

	// その他
	IsDefault bool       `json:"isDefault"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type RepositoryPort interface {
	// 取得系
	GetByID(ctx context.Context, id string) (*PaymentMethod, error)

	// user に紐づく paymentMethod 一覧を返す
	GetByUser(ctx context.Context, userID string) ([]PaymentMethod, error)

	// user の既定 paymentMethod を返す
	GetDefaultByUser(ctx context.Context, userID string) (*PaymentMethod, error)

	// Stripe PaymentMethod ID で取得する
	GetByStripePaymentMethodID(ctx context.Context, stripePaymentMethodID string) (*PaymentMethod, error)

	// 変更系
	Create(ctx context.Context, in CreatePaymentMethodInput) (*PaymentMethod, error)
	Delete(ctx context.Context, id string) error

	// user 内の既定 paymentMethod 切替補助
	ClearDefaultByUser(ctx context.Context, userID string) error
	SetDefault(ctx context.Context, id string, userID string, updatedAt time.Time) (*PaymentMethod, error)
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("paymentMethod: not found")
	ErrConflict = errors.New("paymentMethod: conflict")
)
