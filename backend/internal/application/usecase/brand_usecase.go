// backend/internal/application/usecase/brand_usecase.go
package usecase

import (
	"context"
	"log"
	"strings"
	"time"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
)

type BrandUsecase struct {
	brandRepo  branddom.Repository
	memberRepo memberdom.Repository
	now        func() time.Time
}

func NewBrandUsecase(
	brandRepo branddom.Repository,
	memberRepo memberdom.Repository,
) *BrandUsecase {
	return &BrandUsecase{
		brandRepo:  brandRepo,
		memberRepo: memberRepo,
		now:        time.Now,
	}
}

// ==============================
// Queries
// ==============================

func (u *BrandUsecase) GetByID(ctx context.Context, id string) (branddom.Brand, error) {
	return u.brandRepo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *BrandUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.brandRepo.Exists(ctx, strings.TrimSpace(id))
}

// ★ Count はドメイン側から削除したので、Usecase からも削除済み

// List: currentMember と同じ companyId に絞って一覧取得（ソート指定は廃止）
func (u *BrandUsecase) List(
	ctx context.Context,
	f branddom.Filter,
	p branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	// currentMember と同じ companyId の Brand のみを list
	if cid := companyIDFromContext(ctx); cid != "" {
		f.CompanyID = &cid
	}
	return u.brandRepo.List(ctx, f, p)
}

// ListByCursor: currentMember と同じ companyId に絞ってカーソル一覧取得（ソート指定は廃止）
func (u *BrandUsecase) ListByCursor(
	ctx context.Context,
	f branddom.Filter,
	c branddom.CursorPage,
) (branddom.CursorPageResult[branddom.Brand], error) {
	// currentMember と同じ companyId に制限
	if cid := companyIDFromContext(ctx); cid != "" {
		f.CompanyID = &cid
	}
	return u.brandRepo.ListByCursor(ctx, f, c)
}

// ★ 追加: ListByCompanyID → GetNameByID をセットで組み立てるヘルパ
// currentMember の companyId を使って Brand を一覧取得し、
// brand.Service.GetNameByID で Name を正規化した結果を Items に反映して返す。
func (u *BrandUsecase) ListCurrentCompanyBrandsWithNames(
	ctx context.Context,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	cid := companyIDFromContext(ctx)
	if strings.TrimSpace(cid) == "" {
		// companyId が無い場合は空を返す（必要であれば ErrInvalidID にしてもよい）
		return branddom.PageResult[branddom.Brand]{}, nil
	}

	// brand.Service を使って ListByCompanyID → GetNameByID を組み立てる
	svc := branddom.NewService(u.brandRepo)

	// 1. companyId に紐づく Brand 一覧を取得
	res, err := svc.ListByCompanyID(ctx, cid, page)
	if err != nil {
		return res, err
	}

	// 2. 各 Brand について GetNameByID を呼び出し、Name を正規化して上書き
	for i, b := range res.Items {
		name, err := svc.GetNameByID(ctx, b.ID)
		if err != nil {
			// 取得に失敗した場合は、既存の Name を trim だけして使う
			res.Items[i].Name = strings.TrimSpace(b.Name)
			continue
		}
		res.Items[i].Name = strings.TrimSpace(name)
	}

	return res, nil
}

// ==============================
// Commands
// ==============================

// Create
// Brand を作成し、その後 ManagerID を元に member.assignedBrands を更新する
func (u *BrandUsecase) Create(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
	// context の companyId を優先して強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		b.CompanyID = strings.TrimSpace(cid)
	}

	if b.CreatedAt.IsZero() {
		b.CreatedAt = u.now().UTC()
	}

	// 1. Brand を作成
	created, err := u.brandRepo.Create(ctx, b)
	if err != nil {
		return created, err
	}

	// 2. ManagerID が存在しない場合はここで終了
	if created.ManagerID == nil || strings.TrimSpace(*created.ManagerID) == "" {
		return created, nil
	}

	managerID := strings.TrimSpace(*created.ManagerID)

	// 3. Member を取得
	m, err := u.memberRepo.GetByID(ctx, managerID)
	if err != nil {
		log.Printf("[BrandUsecase] WARN: managerId=%s の Member 取得失敗: %v", managerID, err)
		return created, nil // Brand 作成は成功扱い
	}

	// 4. assignedBrands に brandId を追加（重複チェックあり）
	brandID := strings.TrimSpace(created.ID)
	found := false
	for _, bid := range m.AssignedBrands {
		if bid == brandID {
			found = true
			break
		}
	}
	if !found {
		m.AssignedBrands = append(m.AssignedBrands, brandID)
	}

	// 5. Member を保存
	if _, err := u.memberRepo.Save(ctx, m, nil); err != nil {
		log.Printf("[BrandUsecase] WARN: Member.assignedBrands 更新失敗 (memberId=%s brandId=%s): %v",
			managerID, brandID, err)
		// Brand 作成自体は成功済みなので return created
	}

	return created, nil
}

// Update via BrandPatch
func (u *BrandUsecase) Update(
	ctx context.Context,
	id string,
	patch branddom.BrandPatch,
) (branddom.Brand, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return branddom.Brand{}, branddom.ErrInvalidID
	}
	return u.brandRepo.Update(ctx, id, patch)
}

// Save (upsert)
func (u *BrandUsecase) Save(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
	// context の companyId を優先して強制適用
	if cid := companyIDFromContext(ctx); cid != "" {
		b.CompanyID = strings.TrimSpace(cid)
	}

	if b.CreatedAt.IsZero() {
		b.CreatedAt = u.now().UTC()
	}
	return u.brandRepo.Save(ctx, b, nil)
}

func (u *BrandUsecase) Delete(ctx context.Context, id string) error {
	return u.brandRepo.Delete(ctx, strings.TrimSpace(id))
}
