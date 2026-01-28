package productBlueprintUsecase

import (
	"context"
	"strings"

	usecase "narratives/internal/application/usecase"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) Create(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}
	v.CompanyID = cid

	created, err := u.repo.Create(ctx, v)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, created); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return created, nil
}

// ★ printed フラグを true（印刷済み）に更新するユースケース
// Handler から /product-blueprints/{id}/mark-printed などで呼ばれる想定。
func (u *ProductBlueprintUsecase) MarkPrinted(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	updated, err := u.repo.MarkPrinted(ctx, id)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, updated); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return updated, nil
}

func (u *ProductBlueprintUsecase) Save(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}
	v.CompanyID = cid

	return u.repo.Save(ctx, v)
}

func (u *ProductBlueprintUsecase) Update(ctx context.Context, v productbpdom.ProductBlueprint) (productbpdom.ProductBlueprint, error) {
	id := strings.TrimSpace(v.ID)
	if id == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidID
	}

	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}
	v.CompanyID = cid

	if _, err := u.repo.GetByID(ctx, id); err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	updated, err := u.repo.Save(ctx, v)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, updated); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return updated, nil
}
