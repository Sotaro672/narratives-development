// backend/internal/application/production/usecase.go
package production

import (
	"context"
	"errors"
	"strings"
	"time"

	dto "narratives/internal/application/production/dto"
	resolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

// ============================================================
// Ports
// ============================================================

// ProductionRepo は domain 側の RepositoryPort をそのまま利用しつつ、
// Usecase からはシンプルな CRUD として扱うためのポートです。
type ProductionRepo interface {
	productiondom.RepositoryPort
}

// ★ companyId → productBlueprintId 解決に必要な最小インターフェース
// （*productBlueprint.Service をそのまま渡せる想定）
type ProductBlueprintService interface {
	// productBlueprintId → brandId 解決
	GetBrandIDByID(ctx context.Context, blueprintID string) (string, error)

	// companyId 単位で productBlueprint の ID 一覧を取得
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
}

// ============================================================
// Usecase
// ============================================================

// ProductionUsecase orchestrates production operations.
type ProductionUsecase struct {
	repo ProductionRepo

	// ★ companyId → productBlueprintIds / productBlueprintId → BrandID 解決用
	pbSvc ProductBlueprintService

	// ★ ID→名前解決ヘルパ
	nameResolver *resolver.NameResolver

	now func() time.Time
}

func NewProductionUsecase(
	repo ProductionRepo,
	pbSvc ProductBlueprintService,
	nameResolver *resolver.NameResolver,
) *ProductionUsecase {
	return &ProductionUsecase{
		repo:         repo,
		pbSvc:        pbSvc,
		nameResolver: nameResolver,
		now:          time.Now,
	}
}

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

		// ラベル用日時
		printedAtLabel := ""
		if p.PrintedAt != nil && !p.PrintedAt.IsZero() {
			printedAtLabel = p.PrintedAt.In(time.Local).Format("2006/01/02 15:04")
		}

		createdAtLabel := ""
		if !p.CreatedAt.IsZero() {
			createdAtLabel = p.CreatedAt.In(time.Local).Format("2006/01/02 15:04")
		}

		out = append(out, dto.ProductionListItemDTO{
			Production:     p,
			ProductName:    productName,
			BrandName:      brandName,
			AssigneeName:   assigneeName,
			TotalQuantity:  totalQty,
			PrintedAtLabel: printedAtLabel,
			CreatedAtLabel: createdAtLabel,
		})
	}

	return out, nil
}

// ============================
// Commands
// ============================

// Create accepts a fully-formed entity.
// RepositoryPort の Create(CreateProductionInput) を呼び出す形にブリッジする。
func (u *ProductionUsecase) Create(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
	// Best-effort normalization of timestamps commonly present on entities
	if p.CreatedAt.IsZero() {
		p.CreatedAt = u.now().UTC()
	}

	// Production → CreateProductionInput へ変換
	var statusPtr *productiondom.ProductionStatus
	if p.Status != "" {
		s := p.Status
		statusPtr = &s
	}

	in := productiondom.CreateProductionInput{
		ProductBlueprintID: p.ProductBlueprintID,
		AssigneeID:         p.AssigneeID,
		Models:             p.Models,
		Status:             statusPtr,
		PrintedAt:          p.PrintedAt,
		CreatedBy:          p.CreatedBy,
	}

	// CreatedAt があればポインタで渡す
	if !p.CreatedAt.IsZero() {
		t := p.CreatedAt
		in.CreatedAt = &t
	}

	created, err := u.repo.Create(ctx, in)
	if err != nil {
		return productiondom.Production{}, err
	}
	if created == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	return *created, nil
}

// Save performs upsert. If CreatedAt is zero, it is set to now (UTC).
// RepositoryPort.Save(Production) を利用。
func (u *ProductionUsecase) Save(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = u.now().UTC()
	}
	saved, err := u.repo.Save(ctx, p)
	if err != nil {
		return productiondom.Production{}, err
	}
	if saved == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	return *saved, nil
}

// Update updates Production partially.
func (u *ProductionUsecase) Update(
	ctx context.Context,
	id string,
	patch productiondom.Production,
) (productiondom.Production, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return productiondom.Production{}, productiondom.ErrInvalidID
	}

	currentPtr, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productiondom.Production{}, err
	}
	if currentPtr == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	current := *currentPtr

	if strings.TrimSpace(patch.AssigneeID) != "" {
		current.AssigneeID = strings.TrimSpace(patch.AssigneeID)
	}

	if len(patch.Models) > 0 {
		current.Models = patch.Models
	}

	if patch.Status != "" {
		current.Status = patch.Status
	}

	if patch.PrintedAt != nil {
		t := patch.PrintedAt.UTC()
		current.PrintedAt = &t
	}

	if patch.PrintedBy != nil {
		v := strings.TrimSpace(*patch.PrintedBy)
		if v == "" {
			current.PrintedBy = nil
		} else {
			vCopy := v
			current.PrintedBy = &vCopy
		}
	}

	current.UpdatedAt = u.now().UTC()

	saved, err := u.repo.Save(ctx, current)
	if err != nil {
		return productiondom.Production{}, err
	}
	if saved == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	return *saved, nil
}

func (u *ProductionUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
