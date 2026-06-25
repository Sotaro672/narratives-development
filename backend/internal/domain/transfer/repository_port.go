// backend/internal/domain/transfer/repository_port.go
package transfer

import (
	"context"

	common "narratives/internal/domain/common"
)

/*
責任と機能:
- Transfer エンティティの永続化に必要な最小ポート（RepositoryPort）を定義する。
- カスタマーサポート/監査/再実行を想定し、以下を満たす:
  - productId 単位で最新状態を取得できる
  - 同一 productId に対する複数試行（Attempt）を扱える
  - mintAddress から transfer 実行日時 transferredAt を取得できる
- Firestore 実装は docId="<productId>__<attempt>" のフラット保存を推奨。
  （このポートは実装方式に依存しない）
- entity.go を正として:
  - Transfer に "ID" は存在しない
  - MintAddress を保持する
  - Patch で MintAddress / ToWalletAddress（任意）を更新できる

補足:
- Transfer entity には transferredAt を持たせない。
- mintAddress から注文特定に必要な transfer 実行日時を取得したい場合は、
  ResolveTransferredAtByMintAddressResult の TransferredAt として返す。
*/

// RepositoryPort defines persistence behavior required by domain/usecase.
type RepositoryPort interface {
	// Reads

	// Latest attempt for a productId (highest attempt, or latest createdAt in repo policy).
	GetLatestByProductID(ctx context.Context, productID string) (*Transfer, error)

	// Specific attempt
	GetByProductIDAndAttempt(ctx context.Context, productID string, attempt int) (*Transfer, error)

	// History (all attempts for a productId)
	ListByProductID(ctx context.Context, productID string) ([]Transfer, error)

	// ResolveTransferredAtByMintAddress resolves transfer execution time by mintAddress.
	//
	// Firestore 実装では transfers collection を mintAddress で検索し、
	// 該当 document の transferredAt を返す想定。
	//
	// Firestore 側に typo の transferedAt が残っている場合は、
	// adapter 層で transferredAt / transferedAt の両方を吸収する。
	ResolveTransferredAtByMintAddress(
		ctx context.Context,
		mintAddress string,
	) (ResolveTransferredAtByMintAddressResult, error)

	// Writes

	// CreateAttempt creates a new Transfer attempt for the product.
	// It should allocate next Attempt (>=1) atomically (repo responsibility),
	// and persist the resulting Transfer.
	CreateAttempt(ctx context.Context, t Transfer) (*Transfer, error)

	// Save overwrites/merges the Transfer identified by (productId, attempt).
	Save(ctx context.Context, t Transfer, opts *common.SaveOptions) (*Transfer, error)

	// Patch applies partial update to a specific attempt.
	Patch(ctx context.Context, productID string, attempt int, patch TransferPatch, opts *common.SaveOptions) (*Transfer, error)
}
