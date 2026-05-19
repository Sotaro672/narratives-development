// backend/internal/application/productBlueprint/usecase/ports.go
package productBlueprintUsecase

import (
	"context"
	"time"

	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepo defines the minimal persistence port needed by ProductBlueprintUsecase.
type ProductBlueprintRepo interface {
	// Read (single)
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
	Exists(ctx context.Context, id string) (bool, error)

	// Read (multi) - company スコープ必須
	ListByCompanyID(ctx context.Context, companyID string) ([]productbpdom.ProductBlueprint, error)

	// companyId 単位で productBlueprint の ID 一覧を取得（MintRequest chain 等）
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// printed フラグを true（印刷済み）に更新する
	MarkPrinted(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)

	// Write
	Create(ctx context.Context, in productbpdom.CreateInput) (productbpdom.ProductBlueprint, error)
	Update(ctx context.Context, id string, patch productbpdom.Patch) (productbpdom.ProductBlueprint, error)

	// Delete (physical)
	Delete(ctx context.Context, id string) error

	// 起票後に modelRefs を追記（updatedAt / updatedBy を更新しない）
	// - refs は displayOrder を含む（usecase 側で採番して渡す）
	// - repo 実装側で modelRefs フィールドのみ部分更新し、updatedAt/updatedBy を触らないこと
	AppendModelRefsWithoutTouch(ctx context.Context, id string, refs []productbpdom.ModelRef) (productbpdom.ProductBlueprint, error)
}

// ------------------------------------------------------------
// ProductBlueprintReview initializer port
// ------------------------------------------------------------
//
// ProductBlueprint 起票時に、レビュー側の「商品単位初期化ドキュメント」等を作成するためのポート。
// Firestore実装で、例えば
// - productBlueprintReviewAggregates/{productBlueprintId} を作る
// - もしくは初期状態のドキュメントを作る
// などに使う想定。
type ProductBlueprintReviewInitializer interface {
	InitForProductBlueprint(
		ctx context.Context,
		productBlueprintID string,
		companyID string,
		createdAt time.Time,
		createdBy *string,
	) error
}
