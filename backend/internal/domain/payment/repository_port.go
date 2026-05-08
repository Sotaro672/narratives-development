// backend/internal/domain/payment/repository_port.go
package payment

import (
	"context"
	"errors"

	common "narratives/internal/domain/common"
)

// Filter - 検索条件
type Filter struct {
	common.FilterCommon

	// PaymentID is the Firestore payment document ID.
	// It must be the same value as order.ID.
	PaymentID string `json:"paymentId"`

	PaymentMethodID string `json:"paymentMethodId"`

	StripeCustomerID      string `json:"stripeCustomerId"`
	StripePaymentMethodID string `json:"stripePaymentMethodId"`
	StripePaymentIntentID string `json:"stripePaymentIntentId"`

	Statuses  []PaymentStatus `json:"statuses"`
	ErrorType string          `json:"errorType"`
	ErrorCode string          `json:"errorCode"`

	MinAmount *int `json:"minAmount"`
	MaxAmount *int `json:"maxAmount"`
}

// Sort / Page / Result aliases
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult = common.PageResult[Payment]
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Payment]
type SaveOptions = common.SaveOptions

const (
	SortAsc  SortOrder = common.SortAsc
	SortDesc SortOrder = common.SortDesc
)

type SortColumn string

const (
	SortByCreatedAt SortColumn = "createdAt"
	SortByAmount    SortColumn = "amount"
	SortByStatus    SortColumn = "status"
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
// PaymentID is not included here because update target is selected by paymentId,
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

	// 一覧
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult, error)

	// 変更
	Create(ctx context.Context, in CreatePaymentInput) (*Payment, error)
	UpdateByPaymentID(ctx context.Context, paymentID string, patch UpdatePaymentInput) (*Payment, error)
	DeleteByPaymentID(ctx context.Context, paymentID string) error

	// Save stores the whole payment entity.
	//
	// entity.PaymentID is used as the Firestore payment document ID.
	// paymentId itself should not be saved as a document field unless an adapter
	// explicitly needs it for compatibility.
	Save(ctx context.Context, entity Payment, opts *SaveOptions) (Payment, error)
}

const (
	PaymentStatusPending        PaymentStatus = "pending"
	PaymentStatusRequiresAction PaymentStatus = "requires_action"
	PaymentStatusProcessing     PaymentStatus = "processing"
	PaymentStatusSucceeded      PaymentStatus = "succeeded"
	PaymentStatusFailed         PaymentStatus = "failed"
	PaymentStatusCanceled       PaymentStatus = "canceled"
)

// 共通エラー
var (
	ErrNotFound = errors.New("payment: not found")
	ErrConflict = errors.New("payment: conflict")
)
