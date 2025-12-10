// backend/internal/application/production/usecase.go
package production

import (
	"context"
	"errors"
	"strings"
	"time"

	dto "narratives/internal/application/production/dto"
	resolver "narratives/internal/application/resolver"

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

// ============================================================
// Usecase
// ============================================================

// ProductionUsecase orchestrates production operations.
type ProductionUsecase struct {
	repo ProductionRepo

	// ★ productBlueprintId → BrandID 解決用
	pbSvc *productbpdom.Service

	// ★ ID→名前解決ヘルパ
	nameResolver *resolver.NameResolver

	now func() time.Time
}

func NewProductionUsecase(
	repo ProductionRepo,
	pbSvc *productbpdom.Service,
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

// ★ 素の一覧（必要なところ向けに残す）
// RepositoryPort 側の ListAll を利用する
func (u *ProductionUsecase) List(ctx context.Context) ([]productiondom.Production, error) {
	return u.repo.ListAll(ctx)
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
//   - BrandID 自体は NameResolver にはないため、従来どおり pbSvc を利用
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
func (u *ProductionUsecase) ListWithAssigneeName(ctx context.Context) ([]dto.ProductionListItemDTO, error) {
	list, err := u.repo.ListAll(ctx)
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
			Production:     p,            // ← Production をそのまま埋め込む
			ProductName:    productName,  // 名前解決済み
			BrandName:      brandName,    // 名前解決済み
			AssigneeName:   assigneeName, // 名前解決済み
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
		// Repository 実装が nil を返すのは異常ケースなので一応 NotFound 相当として扱う
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
//
// - 通常の編集画面からの更新:
//   - models / assigneeId を更新
//
// - 印刷完了シグナル(notifyPrintLogCompleted)からの更新:
//   - status / printedAt / printedBy を更新
//
// いずれも「patch に値が入っているフィールドだけを current に上書き」する。
func (u *ProductionUsecase) Update(
	ctx context.Context,
	id string,
	patch productiondom.Production,
) (productiondom.Production, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return productiondom.Production{}, productiondom.ErrInvalidID
	}

	// 既存データを取得（RepositoryPort.GetByID は *Production を返す）
	currentPtr, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productiondom.Production{}, err
	}
	if currentPtr == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	current := *currentPtr

	// --------------------------------------------------
	// 1. assigneeId の更新（非空なら上書き）
	// --------------------------------------------------
	if strings.TrimSpace(patch.AssigneeID) != "" {
		current.AssigneeID = strings.TrimSpace(patch.AssigneeID)
	}

	// --------------------------------------------------
	// 2. quantity（Models）の更新
	//    フロントからは「既存モデルの数量更新用」の Models が渡される想定。
	//    配列が渡されているときだけ差し替える。
	// --------------------------------------------------
	if len(patch.Models) > 0 {
		current.Models = patch.Models
	}

	// --------------------------------------------------
	// 3. 印刷関連: status / printedAt / printedBy
	//    notifyPrintLogCompleted からの PUT でここに入る。
	// --------------------------------------------------

	// status: 空でなければ上書き（"printed" など）
	if patch.Status != "" {
		current.Status = patch.Status
	}

	// printedAt: patch.PrintedAt が非 nil なら上書き（UTC 正規化）
	if patch.PrintedAt != nil {
		t := patch.PrintedAt.UTC() // time.Time
		current.PrintedAt = &t     // *time.Time に代入
	}

	// printedBy: patch.PrintedBy が非 nil なら上書き
	if patch.PrintedBy != nil {
		v := strings.TrimSpace(*patch.PrintedBy)
		if v == "" {
			// 空文字なら nil とみなす
			current.PrintedBy = nil
		} else {
			// 新しい string を確保してポインタで保持
			vCopy := v
			current.PrintedBy = &vCopy
		}
	}

	// --------------------------------------------------
	// 4. updatedAt を現在時刻で更新
	// --------------------------------------------------
	current.UpdatedAt = u.now().UTC()

	// 他の項目は current をそのまま再保存
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
