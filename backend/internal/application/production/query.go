package production

import (
	"context"
	"errors"
	"strings"

	dto "narratives/internal/application/production/dto"
	usecase "narratives/internal/application/usecase"

	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

// ============================
// Queries
// ============================

func (u *ProductionUsecase) GetByID(ctx context.Context, id string) (productiondom.Production, error) {
	p, err := u.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return productiondom.Production{}, err
	}
	if p == nil {
		// RepositoryPort 実装側が nil を返した場合も NotFound 相当として扱う
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	return *p, nil
}

// RepositoryPort に Exists は無いので、GetByID ベースで存在確認する
func (u *ProductionUsecase) Exists(ctx context.Context, id string) (bool, error) {
	_, err := u.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, productiondom.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ★ companyId → productBlueprintId → production のルート以外での list を禁止する
// - companyId が空なら、絶対に repo 側の一覧取得を呼ばない（全社漏洩を防ぐ）
// - companyId から productBlueprintIds を引き、ListByProductBlueprintID のみ使用する
func (u *ProductionUsecase) listByCurrentCompany(ctx context.Context) ([]productiondom.Production, error) {
	// ✅ 方針A: usecase の companyId getter を唯一の真実として利用する
	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		// companyId を持たないユーザーは一覧取得不可（全件漏洩の根本対策）
		return nil, productbpdom.ErrInvalidCompanyID
	}
	if u.pbSvc == nil {
		return nil, productbpdom.ErrInternal
	}

	// 1) companyId → productBlueprintIds
	pbIDs, err := u.pbSvc.ListIDsByCompany(ctx, cid)
	if err != nil {
		return nil, err
	}
	if len(pbIDs) == 0 {
		return []productiondom.Production{}, nil
	}

	// 2) productBlueprintIds → productions
	rows, err := u.repo.ListByProductBlueprintID(ctx, pbIDs)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []productiondom.Production{}, nil
	}

	// 念のため: productBlueprintIds の集合に含まれる production のみを返す（repo 側バグ対策）
	set := make(map[string]struct{}, len(pbIDs))
	for _, id := range pbIDs {
		if tid := strings.TrimSpace(id); tid != "" {
			set[tid] = struct{}{}
		}
	}

	out := make([]productiondom.Production, 0, len(rows))
	for _, p := range rows {
		if _, ok := set[strings.TrimSpace(p.ProductBlueprintID)]; !ok {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// ★ 一覧（素の一覧は削除）
// 必ず companyId → productBlueprintId で絞り込んだ production のみを返す。
func (u *ProductionUsecase) List(ctx context.Context) ([]productiondom.Production, error) {
	return u.listByCurrentCompany(ctx)
}

// ★ 担当者ID から表示名を解決する（NameResolver に委譲）
func (u *ProductionUsecase) ResolveAssigneeName(ctx context.Context, assigneeID string) (string, error) {
	if u.nameResolver == nil {
		return "", nil
	}
	id := strings.TrimSpace(assigneeID)
	if id == "" {
		return "", nil
	}

	name := u.nameResolver.ResolveAssigneeName(ctx, id)
	return strings.TrimSpace(name), nil
}

// ★ productBlueprintId から productName を解決する（NameResolver に委譲）
func (u *ProductionUsecase) ResolveProductName(ctx context.Context, blueprintID string) (string, error) {
	if u.nameResolver == nil {
		return "", nil
	}
	id := strings.TrimSpace(blueprintID)
	if id == "" {
		return "", nil
	}

	name := u.nameResolver.ResolveProductName(ctx, id)
	return strings.TrimSpace(name), nil
}

// ★ productBlueprintId から brandId を解決する
func (u *ProductionUsecase) ResolveBrandID(ctx context.Context, blueprintID string) (string, error) {
	if u.pbSvc == nil {
		return "", nil
	}

	id := strings.TrimSpace(blueprintID)
	if id == "" {
		return "", nil
	}

	brandID, err := u.pbSvc.GetBrandIDByID(ctx, id)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(brandID), nil
}

// ★ brandId から brandName を解決する（NameResolver に委譲）
func (u *ProductionUsecase) ResolveBrandName(ctx context.Context, brandID string) (string, error) {
	if u.nameResolver == nil {
		return "", nil
	}
	id := strings.TrimSpace(brandID)
	if id == "" {
		return "", nil
	}

	name := u.nameResolver.ResolveBrandName(ctx, id)
	return strings.TrimSpace(name), nil
}

// ★ 一覧ページ用 DTO を返却（/productions 用）
// dto.ProductionListItemDTO は backend/internal/application/production/dto/list.go で定義
// ★ 素の一覧は禁止：必ず companyId → productBlueprintId で絞った production のみ返す
func (u *ProductionUsecase) ListWithAssigneeName(ctx context.Context) ([]dto.ProductionListItemDTO, error) {
	list, err := u.listByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]dto.ProductionListItemDTO, 0, len(list))

	for _, p := range list {
		// 担当者名（NameResolver 経由）
		assigneeName, _ := u.ResolveAssigneeName(ctx, p.AssigneeID)

		// productName（NameResolver 経由）
		productName, _ := u.ResolveProductName(ctx, p.ProductBlueprintID)

		// brandId / brandName
		brandID, _ := u.ResolveBrandID(ctx, p.ProductBlueprintID)
		brandName, _ := u.ResolveBrandName(ctx, brandID)

		// 合計数量（Models の Quantity 合計）
		totalQty := 0
		for _, mq := range p.Models {
			if mq.Quantity > 0 {
				totalQty += mq.Quantity
			}
		}

		out = append(out, dto.ProductionListItemDTO{
			Production:    p,
			ProductName:   productName,
			BrandName:     brandName,
			AssigneeName:  assigneeName,
			TotalQuantity: totalQty,
			// PrintedAtLabel / CreatedAtLabel は不要になったため設定しない
		})
	}

	return out, nil
}
