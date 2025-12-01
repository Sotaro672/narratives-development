// backend/internal/application/usecase/production_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

// ============================================================
// Ports
// ============================================================

// ProductionRepo defines the minimal persistence port needed by ProductionUsecase.
type ProductionRepo interface {
	GetByID(ctx context.Context, id string) (productiondom.Production, error)
	Exists(ctx context.Context, id string) (bool, error)

	// ★ 一覧取得
	List(ctx context.Context) ([]productiondom.Production, error)

	Create(ctx context.Context, p productiondom.Production) (productiondom.Production, error)
	Save(ctx context.Context, p productiondom.Production) (productiondom.Production, error)
	Delete(ctx context.Context, id string) error
}

// ★ フロント用 DTO: 担当者名 + productName + brandId/brandName 付き Production
type ProductionWithAssigneeName struct {
	productiondom.Production
	AssigneeName string `json:"assigneeName"`
	ProductName  string `json:"productName"`
	BrandID      string `json:"brandId"`
	BrandName    string `json:"brandName"`
}

// ============================================================
// Usecase
// ============================================================

// ProductionUsecase orchestrates production operations.
type ProductionUsecase struct {
	repo      ProductionRepo
	memberSvc *memberdom.Service

	// ★ productBlueprintId → ProductName / BrandID 解決用
	pbSvc *productbpdom.Service

	// ★ brandId → BrandName 解決用
	brandSvc *branddom.Service

	now func() time.Time
}

func NewProductionUsecase(
	repo ProductionRepo,
	memberSvc *memberdom.Service,
	pbSvc *productbpdom.Service,
	brandSvc *branddom.Service,
) *ProductionUsecase {
	return &ProductionUsecase{
		repo:      repo,
		memberSvc: memberSvc,
		pbSvc:     pbSvc,
		brandSvc:  brandSvc,
		now:       time.Now,
	}
}

// ============================
// Queries
// ============================

func (u *ProductionUsecase) GetByID(ctx context.Context, id string) (productiondom.Production, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductionUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// ★ 素の一覧（必要なところ向けに残す）
func (u *ProductionUsecase) List(ctx context.Context) ([]productiondom.Production, error) {
	return u.repo.List(ctx)
}

// ★ 担当者ID から表示名を解決する resolver 的メソッド
//   - memberSvc が設定されていない場合や ID が空文字の場合は "" を返す
//   - memberSvc 側（member.Service.GetNameLastFirstByID）からのエラーはそのまま返却
func (u *ProductionUsecase) ResolveAssigneeName(ctx context.Context, assigneeID string) (string, error) {
	if u.memberSvc == nil {
		// 名前解決サービスが DI されていない場合は何も表示しない前提
		return "", nil
	}

	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return "", nil
	}

	return u.memberSvc.GetNameLastFirstByID(ctx, assigneeID)
}

// ★ productBlueprintId から productName を解決する resolver 的メソッド
//   - pbSvc が設定されていない場合や ID が空文字の場合は "" を返す
func (u *ProductionUsecase) ResolveProductName(ctx context.Context, blueprintID string) (string, error) {
	if u.pbSvc == nil {
		return "", nil
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return "", nil
	}

	name, err := u.pbSvc.GetProductNameByID(ctx, blueprintID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

// ★ productBlueprintId から brandId を解決する resolver 的メソッド
//   - pbSvc が設定されていない場合や ID が空文字の場合は "" を返す
func (u *ProductionUsecase) ResolveBrandID(ctx context.Context, blueprintID string) (string, error) {
	if u.pbSvc == nil {
		return "", nil
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return "", nil
	}

	brandID, err := u.pbSvc.GetBrandIDByID(ctx, blueprintID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(brandID), nil
}

// ★ brandId から brandName を解決する resolver 的メソッド
//   - brandSvc が設定されていない場合や ID が空文字の場合は "" を返す
func (u *ProductionUsecase) ResolveBrandName(ctx context.Context, brandID string) (string, error) {
	if u.brandSvc == nil {
		return "", nil
	}

	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return "", nil
	}

	name, err := u.brandSvc.GetNameByID(ctx, brandID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

// ★ 担当者名 + productName + brandId/brandName 付き一覧（/productions 用）
func (u *ProductionUsecase) ListWithAssigneeName(ctx context.Context) ([]ProductionWithAssigneeName, error) {
	list, err := u.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ProductionWithAssigneeName, 0, len(list))

	for _, p := range list {
		// 担当者名
		name := ""
		if u.memberSvc != nil && strings.TrimSpace(p.AssigneeID) != "" {
			if n, err := u.memberSvc.GetNameLastFirstByID(ctx, p.AssigneeID); err == nil {
				name = n
			}
		}

		// productName & brandId / brandName
		productName := ""
		brandID := ""
		brandName := ""

		if u.pbSvc != nil && strings.TrimSpace(p.ProductBlueprintID) != "" {
			if n, err := u.pbSvc.GetProductNameByID(ctx, p.ProductBlueprintID); err == nil {
				productName = strings.TrimSpace(n)
			}
			if bID, err := u.pbSvc.GetBrandIDByID(ctx, p.ProductBlueprintID); err == nil {
				brandID = strings.TrimSpace(bID)

				// brandId → brandName
				if u.brandSvc != nil && brandID != "" {
					if bName, err := u.brandSvc.GetNameByID(ctx, brandID); err == nil {
						brandName = strings.TrimSpace(bName)
					}
				}
			}
		}

		out = append(out, ProductionWithAssigneeName{
			Production:   p,
			AssigneeName: name,
			ProductName:  productName,
			BrandID:      brandID,
			BrandName:    brandName,
		})
	}

	return out, nil
}

// ============================
// Commands
// ============================

// Create accepts a fully-formed entity. If CreatedAt is zero, it is set to now (UTC).
func (u *ProductionUsecase) Create(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
	// Best-effort normalization of timestamps commonly present on entities
	if p.CreatedAt.IsZero() {
		p.CreatedAt = u.now().UTC()
	}
	return u.repo.Create(ctx, p)
}

// Save performs upsert. If CreatedAt is zero, it is set to now (UTC).
func (u *ProductionUsecase) Save(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = u.now().UTC()
	}
	return u.repo.Save(ctx, p)
}

// Update updates only quantity (models) and assigneeId.
//   - 他の項目は既存値を維持する。
//   - assigneeId: patch.AssigneeID が非空なら上書き
//   - quantity: patch.Models が与えられていれば、それを現在の Models として保存（数量更新用想定）
func (u *ProductionUsecase) Update(ctx context.Context, id string, patch productiondom.Production) (productiondom.Production, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return productiondom.Production{}, productiondom.ErrInvalidID
	}

	// 既存データを取得
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productiondom.Production{}, err
	}

	// ★ assigneeId のみ更新（非空なら上書き）
	if strings.TrimSpace(patch.AssigneeID) != "" {
		current.AssigneeID = strings.TrimSpace(patch.AssigneeID)
	}

	// ★ quantity（Models）のみ更新
	//    フロントからは「既存モデルの数量更新用」の Models が渡される想定。
	//    ここでは Models 全体を差し替えるが、他フィールドはフロント側で
	//    既存値を維持したまま送ってもらう設計。
	if len(patch.Models) > 0 {
		current.Models = patch.Models
	}

	// 更新日時を更新
	current.UpdatedAt = u.now().UTC()

	// 他の項目は current をそのまま再保存
	return u.repo.Save(ctx, current)
}

func (u *ProductionUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
