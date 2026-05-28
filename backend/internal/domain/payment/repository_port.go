// backend/internal/domain/payment/repository_port.go
package payment

import (
	"context"
	"errors"
)

// CreatePaymentInput - 支払い作成入力（ドメイン契約）
//
// PaymentID is the Firestore payment document ID.
// It must be the same value as order.ID.
//
// 正の Firestore payment record schema:
//
//	amount
//	createdAt
//	paymentMethodId
//	status
//	stripeCustomerId
//	stripePaymentIntentId
//	stripePaymentMethodId
//
// paymentId itself is NOT stored as a field in the payment document.
type CreatePaymentInput struct {
	PaymentID string `json:"paymentId"`

	PaymentMethodID string `json:"paymentMethodId"`

	StripeCustomerID      string `json:"stripeCustomerId"`
	StripePaymentMethodID string `json:"stripePaymentMethodId"`
	StripePaymentIntentID string `json:"stripePaymentIntentId"`

	Amount int `json:"amount"`

	Status PaymentStatus `json:"status"`

	ErrorType *string `json:"errorType,omitempty"`
	ErrorCode *string `json:"errorCode,omitempty"`
	ErrorMsg  *string `json:"errorMsg,omitempty"`
}

// UpdatePaymentInput - Payment部分更新（nilは未更新）
//
// PaymentID is not included here because update target is selected by paymentID,
// which must be the same value as order.ID.
type UpdatePaymentInput struct {
	PaymentMethodID *string `json:"paymentMethodId,omitempty"`

	StripeCustomerID      *string `json:"stripeCustomerId,omitempty"`
	StripePaymentMethodID *string `json:"stripePaymentMethodId,omitempty"`
	StripePaymentIntentID *string `json:"stripePaymentIntentId,omitempty"`

	Amount *int `json:"amount,omitempty"`

	Status *PaymentStatus `json:"status,omitempty"`

	ErrorType *string `json:"errorType,omitempty"`
	ErrorCode *string `json:"errorCode,omitempty"`
	ErrorMsg  *string `json:"errorMsg,omitempty"`
}

// RepositoryPort - ドメインのリポジトリ契約
type RepositoryPort interface {
	// 取得
	GetByPaymentID(ctx context.Context, paymentID string) (*Payment, error)

	// 作成
	Create(ctx context.Context, in CreatePaymentInput) (*Payment, error)

	// 更新
	UpdateByPaymentID(ctx context.Context, paymentID string, patch UpdatePaymentInput) (*Payment, error)
}

// 共通エラー
var (
	ErrNotFound = errors.New("payment: not found")
	ErrConflict = errors.New("payment: conflict")
)
