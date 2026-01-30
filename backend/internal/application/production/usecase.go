// backend\internal\application\production\usecase.go
package production

import (
	"context"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"

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

	// ★ list/query は read side に委譲する（CompanyProductionQueryService）
	listQuery ProductionListQuery

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
// Optional dependency injection
// ============================

// SetListQuery injects the query service that backs List/ListWithAssigneeName.
// This is required for console endpoints that call list APIs.
func (u *ProductionUsecase) SetListQuery(q ProductionListQuery) {
	if u == nil {
		return
	}
	u.listQuery = q
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
	in := productiondom.CreateProductionInput{
		ProductBlueprintID: p.ProductBlueprintID,
		AssigneeID:         p.AssigneeID,
		Models:             p.Models,
		PrintedAt:          p.PrintedAt,
		Printed:            &p.Printed,
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

	// Printed（bool）はゼロ値判定ができないので、ここでは patch.Printed を常に反映せず
	// 「PrintedAt が来たら printed=true」 or 「明示的に false にしたい場合は別API/DTOで受ける」等が必要。
	// ただし、現在のシグネチャは Production を patch として使っているため、ここでは以下の方針にする：
	// - patch.PrintedAt != nil なら current.Printed=true
	// - patch.PrintedBy の更新があるなら current.Printed=true（担当者更新は印刷済み前提）
	// - patch.Printed の変更を許容したい場合は、呼び出し側で patch.Printed を意図的にセットして渡す前提で常に反映する
	//
	// → 要件に合わせ、ここでは patch.Printed を常に反映する（PrintedAt/PrintedBy 整合も補正）
	current.Printed = patch.Printed

	if patch.PrintedAt != nil {
		t := patch.PrintedAt.UTC()
		current.PrintedAt = &t
		// PrintedAt をセットするなら printed=true に寄せる
		current.Printed = true
	}

	if patch.PrintedBy != nil {
		v := strings.TrimSpace(*patch.PrintedBy)
		if v == "" {
			current.PrintedBy = nil
		} else {
			vCopy := v
			current.PrintedBy = &vCopy
			// PrintedBy をセットするなら printed=true に寄せる
			current.Printed = true
		}
	}

	// printed=false の場合は整合性として printedAt/printedBy を落とす
	if !current.Printed {
		current.PrintedAt = nil
		current.PrintedBy = nil
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
