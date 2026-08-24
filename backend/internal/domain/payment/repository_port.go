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
// StripePaymentIntentID and TransferGroup are required.
// A Stripe PaymentIntent must be created before RepositoryPort.Create is called.
// This requirement applies to every payment status, including pending.
//
// StripeChargeID may be empty until Stripe has created a Charge.
//
// 正規のFirestore payment record schema:
//
//	amount
//	createdAt
//	paymentMethodId
//	status
//	stripeChargeId
//	stripeCustomerId
//	stripePaymentIntentId
//	stripePaymentMethodId
//	transferGroup
//
// paymentId itself is NOT stored as a field in the payment document.
type CreatePaymentInput struct {
	PaymentID string `json:"paymentId"`

	PaymentMethodID string `json:"paymentMethodId"`

	StripeCustomerID      string `json:"stripeCustomerId"`
	StripePaymentMethodID string `json:"stripePaymentMethodId"`

	// StripePaymentIntentID is required for every status,
	// including StatusPending.
	StripePaymentIntentID string `json:"stripePaymentIntentId"`

	// StripeChargeID identifies the Stripe Charge used as the source
	// transaction for later Stripe Connect settlement transfers.
	// It may be empty until Stripe has created a Charge.
	StripeChargeID string `json:"stripeChargeId,omitempty"`

	// TransferGroup associates this PaymentIntent with the later
	// Stripe Connect settlement Transfers for the same Order.
	TransferGroup string `json:"transferGroup"`

	Amount int `json:"amount"`

	Status PaymentStatus `json:"status"`

	ErrorType *string `json:"errorType,omitempty"`
	ErrorCode *string `json:"errorCode,omitempty"`
	ErrorMsg  *string `json:"errorMsg,omitempty"`
}

// UpdatePaymentInput - Payment部分更新（nilは未更新）
//
// PaymentID is not included here because the update target is selected
// by paymentID, which must be the same value as order.ID.
//
// StripePaymentIntentID and TransferGroup are already required when a payment
// is created. When either field is specified in an update, it must not be empty.
//
// StripeChargeID may be added later when Stripe creates the Charge.
type UpdatePaymentInput struct {
	PaymentMethodID *string `json:"paymentMethodId,omitempty"`

	StripeCustomerID      *string `json:"stripeCustomerId,omitempty"`
	StripePaymentMethodID *string `json:"stripePaymentMethodId,omitempty"`
	StripePaymentIntentID *string `json:"stripePaymentIntentId,omitempty"`
	StripeChargeID        *string `json:"stripeChargeId,omitempty"`

	TransferGroup *string `json:"transferGroup,omitempty"`

	Amount *int `json:"amount,omitempty"`

	Status *PaymentStatus `json:"status,omitempty"`

	ErrorType *string `json:"errorType,omitempty"`
	ErrorCode *string `json:"errorCode,omitempty"`
	ErrorMsg  *string `json:"errorMsg,omitempty"`
}

// RepositoryPort - ドメインのリポジトリ契約
type RepositoryPort interface {
	// GetByPaymentID returns a payment by its Firestore document ID.
	GetByPaymentID(
		ctx context.Context,
		paymentID string,
	) (*Payment, error)

	// Create creates a payment document.
	//
	// The caller must create the Stripe PaymentIntent first and pass its
	// non-empty ID through CreatePaymentInput.StripePaymentIntentID.
	//
	// The caller must also pass a non-empty TransferGroup.
	//
	// StripeChargeID may be empty until Stripe has created a Charge.
	//
	// Implementations must reject an empty StripePaymentIntentID or
	// TransferGroup, including when Status is StatusPending.
	Create(
		ctx context.Context,
		in CreatePaymentInput,
	) (*Payment, error)

	// UpdateByPaymentID partially updates a payment document.
	//
	// A nil field means that the field is not updated.
	// If StripePaymentIntentID, StripeChargeID, or TransferGroup is non-nil,
	// its value must not be empty.
	UpdateByPaymentID(
		ctx context.Context,
		paymentID string,
		patch UpdatePaymentInput,
	) (*Payment, error)
}

// 共通エラー
var (
	ErrNotFound = errors.New("payment: not found")
	ErrConflict = errors.New("payment: conflict")
)
