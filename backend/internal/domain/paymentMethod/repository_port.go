// backend/internal/domain/paymentMethod/repository_port.go
package paymentMethod

import (
	"context"
	"errors"
	"time"
)

// CreatePaymentMethodInputは、Stripeで作成・確認済みの
// PaymentMethodを永続化するための入力です。
//
// cardNumberおよびcvcなどの生カード情報は扱いません。
// 生カード情報はStripe.js / Elementsから直接Stripeへ送信し、
// バックエンドにはStripeが発行した識別子とカード表示情報のみを渡します。
//
// Firestoreなどの永続化では、次の値を保存します。
//
//   - userId
//   - stripeCustomerId
//   - stripePaymentMethodId
//   - brand
//   - last4
//   - expMonth
//   - expYear
//   - cardholderName
//   - isDefault
//   - createdAt
//   - updatedAt
type CreatePaymentMethodInput struct {
	UserID string `json:"userId"`

	// Stripeで作成・確認済みの識別子
	StripeCustomerID      string `json:"stripeCustomerId"`
	StripePaymentMethodID string `json:"stripePaymentMethodId"`

	// Stripeから取得したカード表示情報
	Brand          string `json:"brand"`
	Last4          string `json:"last4"`
	ExpMonth       int    `json:"expMonth"`
	ExpYear        int    `json:"expYear"`
	CardholderName string `json:"cardholderName"`

	// その他
	IsDefault bool       `json:"isDefault"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// RepositoryPortは、PaymentMethodの永続化操作を定義します。
//
// 既定カードの一意性維持と切替は、Repository実装の責務です。
// Application層から既存の既定カードを個別に解除してはいけません。
//
// Repository実装は、既定カードの解除と新しい既定カードの設定を
// 同一Transaction内で原子的に処理しなければなりません。
type RepositoryPort interface {
	// GetByIDは、PaymentMethod IDで取得します。
	GetByID(
		ctx context.Context,
		id string,
	) (*PaymentMethod, error)

	// GetByUserは、ユーザーに紐づくPaymentMethod一覧を返します。
	GetByUser(
		ctx context.Context,
		userID string,
	) ([]PaymentMethod, error)

	// GetDefaultByUserは、ユーザーの既定PaymentMethodを返します。
	GetDefaultByUser(
		ctx context.Context,
		userID string,
	) (*PaymentMethod, error)

	// GetByStripePaymentMethodIDは、
	// Stripe PaymentMethod IDで取得します。
	GetByStripePaymentMethodID(
		ctx context.Context,
		stripePaymentMethodID string,
	) (*PaymentMethod, error)

	// Createは、Stripeで作成・確認済みのPaymentMethodを保存します。
	//
	// in.IsDefaultがfalseの場合は、PaymentMethodの作成だけを行います。
	//
	// in.IsDefaultがtrueの場合、Repository実装は次の処理を
	// 同一Transaction内で原子的に行わなければなりません。
	//
	//  1. 同じユーザーに属する既存PaymentMethodを取得する
	//  2. 既存のisDefault=trueをfalseへ更新する
	//  3. 新しいPaymentMethodをisDefault=trueで作成する
	//
	// いずれかの処理が失敗した場合は、既存の既定設定を含む
	// すべての変更をロールバックします。
	Create(
		ctx context.Context,
		in CreatePaymentMethodInput,
	) (*PaymentMethod, error)

	// Deleteは、PaymentMethodを削除します。
	Delete(
		ctx context.Context,
		id string,
	) error

	// SetDefaultは、指定PaymentMethodをユーザーの既定に設定します。
	//
	// Repository実装は次の処理を同一Transaction内で
	// 原子的に行わなければなりません。
	//
	//  1. 指定PaymentMethodを取得する
	//  2. 指定PaymentMethodがuserIDの所有物であることを確認する
	//  3. 同じユーザーに属する既存PaymentMethodを取得する
	//  4. 既存のisDefault=trueをfalseへ更新する
	//  5. 指定PaymentMethodのisDefaultをtrueへ更新する
	//  6. 指定PaymentMethodのupdatedAtを更新する
	//
	// 対象が存在しない場合、またはuserIDの所有物でない場合は
	// ErrNotFoundを返します。
	//
	// いずれかの処理が失敗した場合は、既存の既定設定を含む
	// すべての変更をロールバックします。
	SetDefault(
		ctx context.Context,
		id string,
		userID string,
		updatedAt time.Time,
	) (*PaymentMethod, error)
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("paymentMethod: not found")
	ErrConflict = errors.New("paymentMethod: conflict")
)
