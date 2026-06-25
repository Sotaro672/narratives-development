// backend/internal/domain/transfer/entity.go
package transfer

import (
	"context"
	"errors"
	"time"
)

/*
責任と機能:
- Token transfer（移転申請/実行）の結果を永続化するためのドメインエンティティ。
- カスタマーサポート/監査/再実行のために、成功/失敗・エラー種別・tx署名・対象識別子を保持する。
- docId は "<productId>__<attempt>" のフラット保存を想定（repo側で組み立てる）。
  そのため entity 内での "ID" フィールドは不要。
- 同一 productId で複数回試行があり得るため、Attempt（連番）も保持して履歴性を担保する。

方針:
- NewPending の引数順を "attempt -> productId -> ..." に統一し、呼び出し側の型不一致を防ぐ。
  （usecase 側で attempt(int) を先に渡せるようにする）
- transferredAt は Transfer entity には持たせず、mintAddress 逆引き用の read result として返す。
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
)

// TransferQueryPort defines read-only query behavior for transfer lookup.
//
// RepositoryPort は Transfer の永続化用、TransferQueryPort は画面/query 用の逆引き専用。
// mintAddress から transferredAt を取得し、注文特定に使う。
type TransferQueryPort interface {
	ResolveTransferredAtByMintAddress(
		ctx context.Context,
		mintAddress string,
	) (ResolveTransferredAtByMintAddressResult, error)
}

// Transfer represents one attempt of token transfer for a specific product item.
type Transfer struct {
	// Attempt is a monotonically increasing number for the same ProductID.
	Attempt int `json:"attempt"`

	// Identifiers
	ProductID string `json:"productId"`
	OrderID   string `json:"orderId"`
	AvatarID  string `json:"avatarId"`

	// Token info (audit)
	MintAddress string `json:"mintAddress"`

	// Destination / execution
	ToWalletAddress string  `json:"toWalletAddress"`
	TxSignature     *string `json:"txSignature,omitempty"`

	// Result
	Status    Status     `json:"status"`
	ErrorType *ErrorType `json:"errorType,omitempty"`
	ErrorMsg  *string    `json:"errorMsg,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
}

// ResolveTransferredAtByMintAddressResult represents a lookup result for order identification.
//
// Transfer entity には transferredAt を持たせない方針のため、
// mintAddress から transfer 実行日時を引きたい query では、この read result として返す。
// transferredAt のみを正とし、transferedAt などの typo field は扱わない。
type ResolveTransferredAtByMintAddressResult struct {
	ProductID     string    `json:"productId"`
	Attempt       int       `json:"attempt"`
	AvatarID      string    `json:"avatarId"`
	MintAddress   string    `json:"mintAddress"`
	TransferredAt time.Time `json:"transferredAt"`
}

// TransferPatch represents partial updates.
// A nil field means "no change".
type TransferPatch struct {
	Status       *Status
	ErrorType    *ErrorType
	ErrorMsg     *string
	TxSignature  *string
	MintAddress  *string
	ToWalletAddr *string // optional: allow patching destination if needed
}

// NewPending creates a new Transfer in pending status.
//
// NOTE:
// 引数順を "attempt -> productId -> ..." に統一することで、usecase 側の呼び出しを自然にし、
// 型不一致（attempt を string 扱い等）を防ぐ。
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
		Attempt:   attempt,
		ProductID: productID,
		OrderID:   orderID,
		AvatarID:  avatarID,

		MintAddress: mintAddress,

		ToWalletAddress: toWalletAddress,

		Status:      StatusPending,
		ErrorType:   nil,
		ErrorMsg:    nil,
		TxSignature: nil,

		CreatedAt: createdAt.UTC(),
	}

	if err := t.validate(); err != nil {
		return Transfer{}, err
	}
	return t, nil
}

// MarkSucceeded marks transfer as succeeded and stores tx signature.
func (t *Transfer) MarkSucceeded(txSig string) error {
	if t == nil {
		return nil
	}
	if txSig == "" {
		return errors.New("transfer: txSignature is empty")
	}
	t.Status = StatusSucceeded
	t.TxSignature = &txSig
	t.ErrorType = nil
	t.ErrorMsg = nil
	return nil
}

// MarkFailed marks transfer as failed with error type and message (optional).
func (t *Transfer) MarkFailed(errType ErrorType, msg string) error {
	if t == nil {
		return nil
	}
	et := errType
	if string(et) == "" {
		et = ErrorTypeUnknown
	}
	t.Status = StatusFailed
	t.ErrorType = &et

	m := msg
	if m == "" {
		t.ErrorMsg = nil
	} else {
		t.ErrorMsg = &m
	}
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
		m := *p.ErrorMsg
		if m == "" {
			t.ErrorMsg = nil
		} else {
			t.ErrorMsg = &m
		}
	}
	if p.TxSignature != nil {
		s := *p.TxSignature
		if s == "" {
			t.TxSignature = nil
		} else {
			t.TxSignature = &s
		}
	}
	if p.MintAddress != nil {
		s := *p.MintAddress
		if s != "" {
			t.MintAddress = s
		}
	}
	if p.ToWalletAddr != nil {
		s := *p.ToWalletAddr
		if s != "" {
			t.ToWalletAddress = s
		}
	}
}

// validate enforces domain invariants.
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
