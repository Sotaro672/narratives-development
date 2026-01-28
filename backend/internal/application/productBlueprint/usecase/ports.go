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
	Create(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)
	Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error)
	Delete(ctx context.Context, id string) error
}
