// backend/internal/application/productBlueprint/usecase/queries.go
package productBlueprintUsecase

import (
	"context"

	usecase "narratives/internal/application/usecase"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *ProductBlueprintUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, id)
}

// ListByCompanyID
// handler 側の GET /product-blueprints から利用される一覧取得。
// companyId を必須にする（companyId なしの List は廃止済み）。
// テナント境界は repo 側のクエリに委譲しつつ、usecase 側でも二重ガードする。
func (u *ProductBlueprintUsecase) ListByCompanyID(ctx context.Context) ([]productbpdom.ProductBlueprint, error) {
	cid := usecase.CompanyIDFromContext(ctx)
	if cid == "" {
		return nil, productbpdom.ErrInvalidCompanyID
	}

	rows, err := u.repo.ListByCompanyID(ctx, cid)
	if err != nil {
		return nil, err
	}

	// 念のため usecase 側でも companyId をガード
	filtered := make([]productbpdom.ProductBlueprint, 0, len(rows))
	for _, pb := range rows {
		if pb.CompanyID != cid {
			continue
		}
		filtered = append(filtered, pb)
	}

	return filtered, nil
}
