// backend/internal/domain/transfer/repository_port.go
package transfer

import (
	"context"
	"time"
)

/*
責任と機能:
- Transferエンティティの永続化・参照に必要な唯一のポートを定義する。
- カスタマーサポート、監査、再実行を想定し、以下を満たす:
  - productId単位で最新Attemptを取得できる
  - productIdとAttemptを指定して個別取得できる
  - productId単位で全試行履歴を取得できる
  - assetIdから成功したtransferの実行日時を取得できる
  - operationId単位で同一の論理transferを識別できる
  - 次のAttempt採番とpending Transfer作成を原子的に実行できる
  - 同一operationIdの再実行では新しいAttemptを作成せず既存Transferを返す
- Firestore実装ではdocId="<productId>__<attempt>"のフラット保存を想定するが、
  RepositoryPort自体は永続化方式に依存しない。

設計方針:
- Transferの永続化・参照契約はRepositoryPortへ統一する。
- Application層に同等のTransferRepo interfaceを再定義しない。
- TransferにはIDフィールドを持たせない。
- TransferにはtransferredAtを持たせない。
- transferredAtはResolveTransferredAtByAssetIDResultとして返す。
- Firestoreでは正規フィールド名"assetId"と"transferredAt"だけを使用する。
- OperationIDは1回の論理transferを識別するidempotency keyとして扱う。
- 同一OperationIDに対して複数のTransfer attemptを作成しない。
*/

// CreateAttemptInput represents the data required before an Attempt number is
// allocated.
//
// Attempt is not included because its allocation is the repository's
// responsibility. This avoids passing an invalid Transfer whose Attempt is
// zero to CreateAttempt.
//
// OperationID is the stable idempotency key for one logical transfer.
// The same logical retry must always reuse the exact same OperationID.
type CreateAttemptInput struct {
	ProductID       string
	OperationID     string
	OrderID         string
	AvatarID        string
	ToWalletAddress string
	AssetID         string
	CreatedAt       time.Time
}

// Validate validates the input before repository processing.
func (in CreateAttemptInput) Validate() error {
	if in.ProductID == "" {
		return ErrInvalidProductID
	}
	if in.OperationID == "" {
		return ErrInvalidOperationID
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
	if in.AssetID == "" {
		return ErrInvalidAssetID
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
		in.OperationID,
		in.OrderID,
		in.AvatarID,
		in.ToWalletAddress,
		in.AssetID,
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

	// ResolveTransferredAtByAssetID returns the latest successful Transfer
	// execution time for assetId.
	//
	// The repository must query the canonical "assetId" field and read the
	// canonical "transferredAt" field.
	//
	// It returns ErrNotFound when no successful Transfer exists.
	ResolveTransferredAtByAssetID(
		ctx context.Context,
		assetID string,
	) (ResolveTransferredAtByAssetIDResult, error)

	// CreateAttempt creates or reuses one Transfer attempt identified by
	// OperationID.
	//
	// When OperationID has not been used:
	// - atomically allocate the next Attempt number for ProductID
	// - create a pending Transfer
	// - persist the Transfer and OperationID mapping in the same transaction
	//
	// When the same OperationID already exists:
	// - do not allocate a new Attempt
	// - do not create another Transfer
	// - return the existing Transfer
	//
	// Attempt allocation, Transfer persistence, and OperationID reservation
	// must be completed atomically. If persistence fails, neither the Attempt
	// counter nor the OperationID reservation may be advanced.
	CreateAttempt(
		ctx context.Context,
		in CreateAttemptInput,
	) (*Transfer, error)

	// Save persists the complete Transfer identified by productId and Attempt.
	//
	// The Transfer must be valid before it is written.
	// Save must not allocate or change Attempt.
	// Save must not change OperationID.
	Save(
		ctx context.Context,
		t Transfer,
	) (*Transfer, error)

	// Patch applies specified fields to the Transfer identified by productId
	// and Attempt, validates the resulting Transfer, and returns the updated
	// entity.
	//
	// A nil field in TransferPatch means no change.
	// OperationID is immutable and cannot be changed through Patch.
	Patch(
		ctx context.Context,
		productID string,
		attempt int,
		patch TransferPatch,
	) (*Transfer, error)
}
