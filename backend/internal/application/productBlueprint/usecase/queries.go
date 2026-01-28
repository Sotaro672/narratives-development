// backend/internal/application/productBlueprint/usecase/queries.go
package productBlueprintUsecase

import (
	"context"
	"strings"

	usecase "narratives/internal/application/usecase"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductBlueprintUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// ListByCompanyID
// handler 側の GET /product-blueprints から利用される一覧取得。
// ★ companyId を必須にする（companyId なしの List は廃止済み）。
// テナント境界は repo 側のクエリに委譲しつつ、usecase 側でも二重ガードする。
func (u *ProductBlueprintUsecase) ListByCompanyID(ctx context.Context) ([]productbpdom.ProductBlueprint, error) {
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return nil, productbpdom.ErrInvalidCompanyID
	}

	rows, err := u.repo.ListByCompanyID(ctx, cid)
	if err != nil {
		return nil, err
	}

	// 念のため usecase 側でも companyId と deleted をガード
	filtered := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.DeletedAt != nil {
			continue
		}
		if strings.TrimSpace(pb.CompanyID) != cid {
			continue
		}
		filtered = append(filtered, pb)
	}

	return filtered, nil
}

// ListDeletedByCompanyID
// ★ 論理削除済みのみの一覧（companyId 必須）
func (u *ProductBlueprintUsecase) ListDeletedByCompanyID(ctx context.Context) ([]productbpdom.ProductBlueprint, error) {
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return nil, productbpdom.ErrInvalidCompanyID
	}

	rows, err := u.repo.ListDeletedByCompanyID(ctx, cid)
	if err != nil {
		return nil, err
	}

	// 念のため usecase 側でも DeletedAt != nil と companyId を保証
	deleted := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.DeletedAt == nil {
			continue
		}
		if strings.TrimSpace(pb.CompanyID) != cid {
			continue
		}
		deleted = append(deleted, pb)
	}

	return deleted, nil
}

// ListHistory
// ★ 履歴一覧取得（LogCard 用）
func (u *ProductBlueprintUsecase) ListHistory(ctx context.Context, productBlueprintID string) ([]productbpdom.ProductBlueprint, error) {
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, productbpdom.ErrInvalidID
	}
	if u.historyRepo == nil {
		return nil, productbpdom.ErrInternal
	}
	return u.historyRepo.ListByProductBlueprintID(ctx, productBlueprintID)
}
