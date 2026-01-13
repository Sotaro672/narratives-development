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
  - productId（= docId 推奨）単位で最新状態を取得できる
  - 同一 productId に対する複数試行（Attempt）を扱える
  - 一覧/件数取得ができる（管理画面/CS用途）
- Firestore 実装では docId=productId とし、
  Attempt はサブコレクション or 配列 or 別ドキュメント（id=productId__attemptN）などで実装してよい。
  （このポートは実装方式に依存しない）
*/

type Filter struct {
	// docId (recommended: productId)
	ID        *string
	ProductID *string
	OrderID   *string
	AvatarID  *string

	// Status filter
	Status *Status

	// ErrorType filter
	ErrorType *ErrorType
}

type Sort struct {
	// Field: "createdAt" | "updatedAt" | "id" | "productId" | ...
	Field string
	Desc  bool
}

// RepositoryPort defines persistence behavior required by domain/usecase.
type RepositoryPort interface {
	// Reads
	GetLatestByProductID(ctx context.Context, productID string) (*Transfer, error)
	GetByProductIDAndAttempt(ctx context.Context, productID string, attempt int) (*Transfer, error)

	// History
	ListByProductID(ctx context.Context, productID string) ([]Transfer, error)

	// Generic list/count
	List(ctx context.Context, filter Filter, sort Sort, page common.Page) (common.PageResult[Transfer], error)
	Count(ctx context.Context, filter Filter) (int, error)

	// Writes
	// CreateAttempt creates a new Transfer attempt for the product.
	// It should allocate next Attempt (>=1) atomically (repo responsibility).
	CreateAttempt(ctx context.Context, t Transfer) (*Transfer, error)

	// Save overwrites/merges the Transfer identified by (productId, attempt).
	Save(ctx context.Context, t Transfer, opts *common.SaveOptions) (*Transfer, error)

	// Patch applies partial update to a specific attempt.
	Patch(ctx context.Context, productID string, attempt int, patch TransferPatch, opts *common.SaveOptions) (*Transfer, error)

	// Delete is optional (mainly for dev/test); production may disallow.
	Delete(ctx context.Context, productID string, attempt int) error

	// Dev/Test
	Reset(ctx context.Context) error
}
