// backend/internal/domain/transfer/entity.go
package transfer

import (
	"errors"
	"time"
)

/*
責任と機能:
- Token transferの実行結果を永続化するためのドメインエンティティ。
- カスタマーサポート、監査、再実行のために、成功・失敗、エラー種別、
  tx署名、対象識別子を保持する。
- FirestoreのdocIdは"<productId>__<attempt>"を想定し、
  Transfer自体にはIDフィールドを持たせない。
- 同一productIdに対する複数試行を扱うため、Attemptを保持する。

方針:
- 永続化および参照の契約はrepository_port.goのRepositoryPortへ統一する。
- TransferにはtransferredAtを持たせない。
- mintAddressから転送実行日時を取得する場合は、
  ResolveTransferredAtByMintAddressResultとして返す。
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
	ErrorTypeMismatch           ErrorType = "mismatch"

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
	ErrNotFound = errors.New("transfer: not found")

	ErrInvalidProductID       = errors.New("transfer: invalid productId")
	ErrInvalidOrderID         = errors.New("transfer: invalid orderId")
	ErrInvalidAvatarID        = errors.New("transfer: invalid avatarId")
	ErrInvalidToWalletAddress = errors.New("transfer: invalid toWalletAddress")
	ErrInvalidMintAddress     = errors.New("transfer: invalid mintAddress")
	ErrInvalidTransferredAt   = errors.New("transfer: invalid transferredAt")
	ErrInvalidStatus          = errors.New("transfer: invalid status")
	ErrInvalidCreatedAt       = errors.New("transfer: invalid createdAt")
	ErrInvalidAttempt         = errors.New("transfer: attempt must be >= 1")
	ErrEmptyTxSignature       = errors.New("transfer: txSignature is empty")
)

// Transfer represents one token transfer attempt for a specific product.
type Transfer struct {
	// Attempt is monotonically increased for the same ProductID.
	Attempt int `json:"attempt"`

	// Identifiers
	ProductID string `json:"productId"`
	OrderID   string `json:"orderId"`
	AvatarID  string `json:"avatarId"`

	// Token information
	MintAddress string `json:"mintAddress"`

	// Destination and execution result
	ToWalletAddress string  `json:"toWalletAddress"`
	TxSignature     *string `json:"txSignature,omitempty"`

	// Status and error details
	Status    Status     `json:"status"`
	ErrorType *ErrorType `json:"errorType,omitempty"`
	ErrorMsg  *string    `json:"errorMsg,omitempty"`

	// CreatedAt is the time when this transfer attempt was created.
	CreatedAt time.Time `json:"createdAt"`
}

// ResolveTransferredAtByMintAddressResult represents the result of resolving
// a successful transfer by mintAddress.
//
// Transfer does not contain TransferredAt. It is returned only by the
// repository query that resolves the successful transfer execution time.
//
// Firestore must use the correctly spelled "transferredAt" field.
// Legacy spellings such as "transferedAt" are not supported.
type ResolveTransferredAtByMintAddressResult struct {
	ProductID     string    `json:"productId"`
	Attempt       int       `json:"attempt"`
	AvatarID      string    `json:"avatarId"`
	MintAddress   string    `json:"mintAddress"`
	TransferredAt time.Time `json:"transferredAt"`
}

// TransferPatch represents a partial Transfer update.
// A nil field means no change.
type TransferPatch struct {
	Status          *Status
	ErrorType       *ErrorType
	ErrorMsg        *string
	TxSignature     *string
	MintAddress     *string
	ToWalletAddress *string
}

// NewPending creates a validated pending Transfer.
func NewPending(
	attempt int,
	productID string,
	orderID string,
	avatarID string,
	toWalletAddress string,
	mintAddress string,
	createdAt time.Time,
) (Transfer, error) {
	t := Transfer{
		Attempt:         attempt,
		ProductID:       productID,
		OrderID:         orderID,
		AvatarID:        avatarID,
		MintAddress:     mintAddress,
		ToWalletAddress: toWalletAddress,
		TxSignature:     nil,
		Status:          StatusPending,
		ErrorType:       nil,
		ErrorMsg:        nil,
		CreatedAt:       createdAt.UTC(),
	}

	if err := t.validate(); err != nil {
		return Transfer{}, err
	}

	return t, nil
}

// MarkSucceeded marks the Transfer as succeeded and stores its transaction
// signature.
func (t *Transfer) MarkSucceeded(txSignature string) error {
	if t == nil {
		return nil
	}
	if txSignature == "" {
		return ErrEmptyTxSignature
	}

	t.Status = StatusSucceeded
	t.TxSignature = &txSignature
	t.ErrorType = nil
	t.ErrorMsg = nil

	return nil
}

// MarkFailed marks the Transfer as failed.
func (t *Transfer) MarkFailed(
	errorType ErrorType,
	message string,
) error {
	if t == nil {
		return nil
	}

	if errorType == "" {
		errorType = ErrorTypeUnknown
	}

	t.Status = StatusFailed
	t.ErrorType = &errorType

	if message == "" {
		t.ErrorMsg = nil
	} else {
		t.ErrorMsg = &message
	}

	return nil
}

// ApplyPatch applies a partial update.
//
// ApplyPatch does not silently ignore invalid non-nil values. Validation is
// performed after all specified fields have been applied.
func (t *Transfer) ApplyPatch(
	patch TransferPatch,
) error {
	if t == nil {
		return nil
	}

	if patch.Status != nil {
		t.Status = *patch.Status
	}
	if patch.ErrorType != nil {
		errorType := *patch.ErrorType
		t.ErrorType = &errorType
	}
	if patch.ErrorMsg != nil {
		message := *patch.ErrorMsg
		if message == "" {
			t.ErrorMsg = nil
		} else {
			t.ErrorMsg = &message
		}
	}
	if patch.TxSignature != nil {
		txSignature := *patch.TxSignature
		if txSignature == "" {
			t.TxSignature = nil
		} else {
			t.TxSignature = &txSignature
		}
	}
	if patch.MintAddress != nil {
		t.MintAddress = *patch.MintAddress
	}
	if patch.ToWalletAddress != nil {
		t.ToWalletAddress = *patch.ToWalletAddress
	}

	return t.validate()
}

// Validate verifies the Transfer's domain invariants.
func (t Transfer) Validate() error {
	return t.validate()
}

// validate enforces Transfer domain invariants.
func (t Transfer) validate() error {
	if t.ProductID == "" {
		return ErrInvalidProductID
	}
	if t.OrderID == "" {
		return ErrInvalidOrderID
	}
	if t.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if t.ToWalletAddress == "" {
		return ErrInvalidToWalletAddress
	}
	if t.MintAddress == "" {
		return ErrInvalidMintAddress
	}
	if t.Attempt <= 0 {
		return ErrInvalidAttempt
	}

	switch t.Status {
	case StatusPending, StatusSucceeded, StatusFailed:
	default:
		return ErrInvalidStatus
	}

	if t.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if t.Status == StatusSucceeded {
		if t.TxSignature == nil || *t.TxSignature == "" {
			return ErrEmptyTxSignature
		}
	}

	return nil
}
