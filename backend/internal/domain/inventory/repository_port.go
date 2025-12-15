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

	// ------------------------------------------------------
	// Accumulation operations
	// ------------------------------------------------------
	//
	// 目的:
	// - mint テーブルの products の個数ぶん在庫(accumulation)を増やす
	// - order テーブルの orderItem の個数ぶん在庫(accumulation)を減らす
	//
	// 実装要件:
	// - 永続化層で原子的(atomic)に加算・減算できること（Firestore なら Transaction / Increment）
	// - 戻り値は更新後の最新 Mint を返す

	// IncrementAccumulation:
	// - accumulation を delta 分だけ増減します（delta は正でも負でも可）。
	// - 例: mint.products 件数だけ増やす → delta = +len(products)
	// - 例: orderItems 件数だけ減らす → delta = -len(orderItems)
	IncrementAccumulation(ctx context.Context, id string, delta int) (Mint, error)

	// IncrementAccumulationByMintProducts:
	// - 指定 mint(id) の products 件数を数えて accumulation を増加させます。
	// - 例: mint 実行完了時に在庫を増やす用途
	IncrementAccumulationByMintProducts(ctx context.Context, id string) (Mint, error)

	// DecrementAccumulationByOrderItemsCount:
	// - accumulation を orderItemsCount 分だけ減少させます（常に減算）。
	// - 例: 注文確定時に在庫を引き当てる用途
	DecrementAccumulationByOrderItemsCount(ctx context.Context, id string, orderItemsCount int) (Mint, error)
}
