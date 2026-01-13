// backend/internal/domain/transfer/entity.go
package transfer

import (
	"errors"
	"strings"
	"time"
)

/*
責任と機能:
- Token transfer（移転申請/実行）の結果を永続化するためのドメインエンティティ。
- カスタマーサポート/監査/再実行のために、成功/失敗・エラー種別・tx署名・対象識別子を保持する。
- docId は productId を推奨（要件: 1 order に複数商品があるため、item単位で追えるようにする）。
  ただし同一 productId で複数回試行があり得るため、Attempt（連番）も保持して一意性/履歴性を担保する。
*/

type Status string

const (
	StatusPending   Status = "pending"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type ErrorType string

const (
	// Common
	ErrorTypeUnknown ErrorType = "unknown"
	ErrorTypeInvalid ErrorType = "invalid"

	// Eligibility / business rules
	ErrorTypeNotPaid            ErrorType = "not_paid"
	ErrorTypeAlreadyTransferred ErrorType = "already_transferred"
	ErrorTypeOrderNotFound      ErrorType = "order_not_found"
	ErrorTypeItemNotFound       ErrorType = "item_not_found"
	ErrorTypeMismatch           ErrorType = "mismatch" // scanned info mismatch etc

	// Dependencies
	ErrorTypeBrandWalletNotFound ErrorType = "brand_wallet_not_found"
	ErrorTypeSecretNotFound      ErrorType = "secret_not_found"
	ErrorTypeSecretAccessDenied  ErrorType = "secret_access_denied"
	ErrorTypeSecretInvalid       ErrorType = "secret_invalid"

	// Blockchain / execution
	ErrorTypeTransferRejected ErrorType = "transfer_rejected"
	ErrorTypeTransferTimeout  ErrorType = "transfer_timeout"
	ErrorTypeTransferFailed   ErrorType = "transfer_failed"
)

var (
	ErrInvalidID              = errors.New("transfer: invalid id")
	ErrInvalidProductID       = errors.New("transfer: invalid productId")
	ErrInvalidOrderID         = errors.New("transfer: invalid orderId")
	ErrInvalidAvatarID        = errors.New("transfer: invalid avatarId")
	ErrInvalidToWalletAddress = errors.New("transfer: invalid toWalletAddress")
	ErrInvalidStatus          = errors.New("transfer: invalid status")
	ErrInvalidCreatedAt       = errors.New("transfer: invalid createdAt")
)

// Transfer represents one attempt of token transfer for a specific product item.
type Transfer struct {
	// ID is the document id (recommended: productId).
	// If your persistence needs strict uniqueness per attempt, use ID=productId and Attempt to separate attempts.
	ID string `json:"id"`

	// Attempt is a monotonically increasing number for the same ID (productId).
	// First attempt should be 1.
	Attempt int `json:"attempt"`

	// Identifiers
	ProductID string `json:"productId"`
	OrderID   string `json:"orderId"`
	AvatarID  string `json:"avatarId"`

	// Destination / execution
	ToWalletAddress string  `json:"toWalletAddress"`
	TxSignature     *string `json:"txSignature,omitempty"`

	// Result
	Status    Status     `json:"status"`
	ErrorType *ErrorType `json:"errorType,omitempty"`
	ErrorMsg  *string    `json:"errorMsg,omitempty"`

	// Timestamps
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// TransferPatch represents partial updates.
// A nil field means "no change".
type TransferPatch struct {
	Status      *Status
	ErrorType   *ErrorType
	ErrorMsg    *string
	TxSignature *string

	UpdatedAt *time.Time
}

// NewPending creates a new Transfer in pending status.
// id and productId are both required; recommended: id == productId.
func NewPending(
	id string,
	attempt int,
	productID string,
	orderID string,
	avatarID string,
	toWalletAddress string,
	createdAt time.Time,
) (Transfer, error) {
	t := Transfer{
		ID:        strings.TrimSpace(id),
		Attempt:   attempt,
		ProductID: strings.TrimSpace(productID),
		OrderID:   strings.TrimSpace(orderID),
		AvatarID:  strings.TrimSpace(avatarID),

		ToWalletAddress: strings.TrimSpace(toWalletAddress),

		Status:      StatusPending,
		ErrorType:   nil,
		ErrorMsg:    nil,
		TxSignature: nil,

		CreatedAt: createdAt.UTC(),
		UpdatedAt: nil,
	}

	if err := t.validate(); err != nil {
		return Transfer{}, err
	}
	return t, nil
}

// MarkSucceeded marks transfer as succeeded and stores tx signature.
func (t *Transfer) MarkSucceeded(txSig string, at time.Time) error {
	if t == nil {
		return nil
	}
	txSig = strings.TrimSpace(txSig)
	if txSig == "" {
		return errors.New("transfer: txSignature is empty")
	}
	t.Status = StatusSucceeded
	t.TxSignature = &txSig
	t.ErrorType = nil
	t.ErrorMsg = nil

	u := at.UTC()
	t.UpdatedAt = &u
	return nil
}

// MarkFailed marks transfer as failed with error type and message (optional).
func (t *Transfer) MarkFailed(errType ErrorType, msg string, at time.Time) error {
	if t == nil {
		return nil
	}
	et := errType
	if strings.TrimSpace(string(et)) == "" {
		et = ErrorTypeUnknown
	}
	t.Status = StatusFailed
	t.ErrorType = &et

	m := strings.TrimSpace(msg)
	if m == "" {
		t.ErrorMsg = nil
	} else {
		t.ErrorMsg = &m
	}

	u := at.UTC()
	t.UpdatedAt = &u
	return nil
}

// ApplyPatch applies partial updates (for repositories/usecases).
func (t *Transfer) ApplyPatch(p TransferPatch) {
	if t == nil {
		return
	}
	if p.Status != nil {
		t.Status = *p.Status
	}
	if p.ErrorType != nil {
		et := *p.ErrorType
		t.ErrorType = &et
	}
	if p.ErrorMsg != nil {
		m := strings.TrimSpace(*p.ErrorMsg)
		if m == "" {
			t.ErrorMsg = nil
		} else {
			t.ErrorMsg = &m
		}
	}
	if p.TxSignature != nil {
		s := strings.TrimSpace(*p.TxSignature)
		if s == "" {
			t.TxSignature = nil
		} else {
			t.TxSignature = &s
		}
	}
	if p.UpdatedAt != nil && !p.UpdatedAt.IsZero() {
		u := p.UpdatedAt.UTC()
		t.UpdatedAt = &u
	}
}

// validate enforces domain invariants.
func (t Transfer) validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(t.ProductID) == "" {
		return ErrInvalidProductID
	}
	if strings.TrimSpace(t.OrderID) == "" {
		return ErrInvalidOrderID
	}
	if strings.TrimSpace(t.AvatarID) == "" {
		return ErrInvalidAvatarID
	}
	if strings.TrimSpace(t.ToWalletAddress) == "" {
		return ErrInvalidToWalletAddress
	}
	if t.Attempt <= 0 {
		return errors.New("transfer: attempt must be >= 1")
	}
	switch t.Status {
	case StatusPending, StatusSucceeded, StatusFailed:
		// ok
	default:
		return ErrInvalidStatus
	}
	if t.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	return nil
}
