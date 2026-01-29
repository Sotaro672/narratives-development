// backend/internal/application/productBlueprint/usecase/ports.go
package productBlueprintUsecase

import (
	"context"

	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepo defines the minimal persistence port needed by ProductBlueprintUsecase.
//
// 方針:
// - companyId を伴わない List は廃止（全件取得を防止）
// - ListPrinted は廃止（不要）
// - ListDeleted は ListDeletedByCompanyID に改名し、companyId を必須化
type ProductBlueprintRepo interface {
	// Read (single)
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
	Exists(ctx context.Context, id string) (bool, error)

	// Read (multi) - company スコープ必須
	ListByCompanyID(ctx context.Context, companyID string) ([]productbpdom.ProductBlueprint, error)

	// Read (deleted only) - company スコープ必須
	ListDeletedByCompanyID(ctx context.Context, companyID string) ([]productbpdom.ProductBlueprint, error)

	// companyId 単位で productBlueprint の ID 一覧を取得（MintRequest chain 等）
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

	// printed フラグを true（印刷済み）に更新する
	MarkPrinted(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)

	// Write
	//
	// ✅ Create は CreateInput を受け取る（repo の contract に合わせる）
	Create(ctx context.Context, in productbpdom.CreateInput) (productbpdom.ProductBlueprint, error)

	// ✅ Update は Patch を受け取る（repo の contract に合わせる）
	Update(ctx context.Context, id string, patch productbpdom.Patch) (productbpdom.ProductBlueprint, error)

	// ✅ Save は lifecycle（SoftDelete/Restore）等で entity 全体を永続化するために残す（互換用）
	// NOTE: 将来的に Patch へ寄せるなら削除可能。ただしその場合は Patch に deleted 系/ttl 系を追加するか別portが必要。
	Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)

	// Delete (physical)
	Delete(ctx context.Context, id string) error

	// ★ 追加: 起票後に modelRefs を追記（updatedAt / updatedBy を更新しない）
	// - refs は displayOrder を含む（usecase 側で採番して渡す）
	// - repo 実装側で modelRefs フィールドのみ部分更新し、updatedAt/updatedBy を触らないこと
	AppendModelRefsWithoutTouch(ctx context.Context, id string, refs []productbpdom.ModelRef) (productbpdom.ProductBlueprint, error)
}
