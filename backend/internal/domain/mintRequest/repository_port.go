// backend/internal/domain/mintRequest/repository_port.go
package mintrequest

import (
	"context"
	"errors"
)

// Repository は mintRequest ドメインの永続化ポートです。
// アダプタ層（Firestore など）はこのインターフェースを実装します。
type Repository interface {
	// GetByID は mintRequest を ID で 1 件取得します。
	// 見つからない場合は ErrNotFound を返します。
	GetByID(ctx context.Context, id string) (MintRequest, error)

	// ListByProductionIDs:
	// 指定された複数の productionId のいずれかに紐づく
	// すべての MintRequest を取得します。
	// Firestore の Query.Where("productionId", "in", productionIDs) を想定。
	ListByProductionIDs(ctx context.Context, productionIDs []string) ([]MintRequest, error)

	// Update は既存の MintRequest を保存します。
	// 対象が存在しない場合は ErrNotFound を返す実装にして構いません。
	Update(ctx context.Context, mr MintRequest) (MintRequest, error)
}

// ErrNotFound は指定された条件の MintRequest が存在しないことを表します。
var ErrNotFound = errors.New("mintRequest: not found")
