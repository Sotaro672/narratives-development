// backend/internal/domain/inventory/repository_port.go
package inventory

import "context"

// ------------------------------------------------------
// Repository Port for Inventory (mints コレクション / テーブル)
// ------------------------------------------------------
//
// Hexagonal Architecture における「出力ポート」。
// Firestore などの具体実装は adapters/out 側で実装し、
// ドメイン層からはこのインターフェースのみを参照します。
//
// 対象エンティティ: Mint（backend/internal/domain/inventory/entity.go）
type RepositoryPort interface {
	// Create:
	// - 新しい Mint エンティティを保存します。
	// - m.ID が空の場合、実装側で採番して返却します。
	Create(ctx context.Context, m Mint) (Mint, error)

	// GetByID:
	// - id で 1 件取得します。
	GetByID(ctx context.Context, id string) (Mint, error)

	// Update:
	// - Mint を更新します（updatedAt は実装側で更新してよい）。
	Update(ctx context.Context, m Mint) (Mint, error)

	// Delete:
	// - id の Mint を削除します。
	Delete(ctx context.Context, id string) error

	// ListByTokenBlueprintID:
	// - tokenBlueprintId に紐づく Mint を一覧取得します。
	ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]Mint, error)

	// ListByProductBlueprintID:
	// - productBlueprintId に紐づく Mint を一覧取得します。
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]Mint, error)

	// ListByTokenAndProductBlueprintID:
	// - tokenBlueprintId と productBlueprintId の AND 条件で一覧取得します。
	ListByTokenAndProductBlueprintID(ctx context.Context, tokenBlueprintID, productBlueprintID string) ([]Mint, error)
}
