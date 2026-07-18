// backend/internal/domain/transfer/repository_port.go
package transfer

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
)

/*
責任と機能:
- Transferエンティティの永続化・参照に必要な唯一のポートを定義する。
- カスタマーサポート、監査、再実行を想定し、以下を満たす:
  - productId単位で最新Attemptを取得できる
  - productIdとAttemptを指定して個別取得できる
  - productId単位で全試行履歴を取得できる
  - mintAddressから成功したtransferの実行日時を取得できる
  - 次のAttempt採番とpending Transfer作成を原子的に実行できる
- Firestore実装ではdocId="<productId>__<attempt>"のフラット保存を想定するが、
  RepositoryPort自体は永続化方式に依存しない。

設計方針:
- Transferの永続化・参照契約はRepositoryPortへ統一する。
- Application層に同等のTransferRepo interfaceを再定義しない。
- TransferにはIDフィールドを持たせない。
- TransferにはtransferredAtを持たせない。
- transferredAtはResolveTransferredAtByMintAddressResultとして返す。
- Firestoreでは正規フィールド名"transferredAt"だけを使用する。
- "transferedAt"などの旧表記やtypoは吸収しない。
*/

// CreateAttemptInput represents the data required before an Attempt number is
// allocated.
//
// Attempt is not included because its allocation is the repository's
// responsibility. This avoids passing an invalid Transfer whose Attempt is
// zero to CreateAttempt.
type CreateAttemptInput struct {
	ProductID       string
	OrderID         string
	AvatarID        string
	ToWalletAddress string
	MintAddress     string
	CreatedAt       time.Time
}

// Validate validates the input before repository processing.
func (in CreateAttemptInput) Validate() error {
	if in.ProductID == "" {
		return ErrInvalidProductID
	}
	if in.OrderID == "" {
		return ErrInvalidOrderID
	}
	if in.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if in.ToWalletAddress == "" {
		return ErrInvalidToWalletAddress
	}
	if in.MintAddress == "" {
		return ErrInvalidMintAddress
	}
	if in.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	return nil
}

// NewTransfer creates a validated pending Transfer after the repository has
// allocated an Attempt number.
func (in CreateAttemptInput) NewTransfer(
	attempt int,
) (Transfer, error) {
	if err := in.Validate(); err != nil {
		return Transfer{}, err
	}

	return NewPending(
		attempt,
		in.ProductID,
		in.OrderID,
		in.AvatarID,
		in.ToWalletAddress,
		in.MintAddress,
		in.CreatedAt,
	)
}

// RepositoryPort defines all persistence and lookup behavior required for
// Transfer.
type RepositoryPort interface {
	// GetLatestByProductID returns the Transfer with the highest Attempt for
	// the specified productId.
	//
	// It returns ErrNotFound when no Transfer exists.
	GetLatestByProductID(
		ctx context.Context,
		productID string,
	) (*Transfer, error)

	// GetByProductIDAndAttempt returns one Transfer identified by productId
	// and Attempt.
	//
	// It returns ErrNotFound when the Transfer does not exist.
	GetByProductIDAndAttempt(
		ctx context.Context,
		productID string,
		attempt int,
	) (*Transfer, error)

	// ListByProductID returns all Transfer attempts for productId in ascending
	// Attempt order.
	//
	// It returns an empty slice when no Transfer exists.
	ListByProductID(
		ctx context.Context,
		productID string,
	) ([]Transfer, error)

	// ResolveTransferredAtByMintAddress returns the latest successful Transfer
	// execution time for mintAddress.
	//
	// The repository must read the canonical "transferredAt" field.
	// Legacy or misspelled fields are not supported.
	//
	// It returns ErrNotFound when no successful Transfer exists.
	ResolveTransferredAtByMintAddress(
		ctx context.Context,
		mintAddress string,
	) (ResolveTransferredAtByMintAddressResult, error)

	// CreateAttempt atomically allocates the next Attempt number, creates a
	// pending Transfer, and persists it.
	//
	// Attempt allocation and Transfer persistence must be completed in the
	// same transaction. If persistence fails, the Attempt counter must not be
	// advanced.
	CreateAttempt(
		ctx context.Context,
		in CreateAttemptInput,
	) (*Transfer, error)

	// Save persists the complete Transfer identified by productId and Attempt.
	//
	// The Transfer must be valid before it is written.
	// Save must not allocate or change Attempt.
	Save(
		ctx context.Context,
		t Transfer,
		opts *common.SaveOptions,
	) (*Transfer, error)

	// Patch applies specified fields to the Transfer identified by productId
	// and Attempt, validates the resulting Transfer, and returns the updated
	// entity.
	//
	// A nil field in TransferPatch means no change.
	Patch(
		ctx context.Context,
		productID string,
		attempt int,
		patch TransferPatch,
		opts *common.SaveOptions,
	) (*Transfer, error)
}
