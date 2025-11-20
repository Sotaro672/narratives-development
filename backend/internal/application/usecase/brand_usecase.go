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

func (u *BrandUsecase) Count(ctx context.Context, f branddom.Filter) (int, error) {
	return u.brandRepo.Count(ctx, f)
}

func (u *BrandUsecase) List(
	ctx context.Context,
	f branddom.Filter,
	s branddom.Sort,
	p branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	return u.brandRepo.List(ctx, f, s, p)
}

func (u *BrandUsecase) ListByCursor(
	ctx context.Context,
	f branddom.Filter,
	s branddom.Sort,
	c branddom.CursorPage,
) (branddom.CursorPageResult[branddom.Brand], error) {
	return u.brandRepo.ListByCursor(ctx, f, s, c)
}

// ==============================
// Commands
// ==============================

// Create
// Brand を作成し、その後 ManagerID を元に member.assignedBrands を更新する
func (u *BrandUsecase) Create(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
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
	if b.CreatedAt.IsZero() {
		b.CreatedAt = u.now().UTC()
	}
	return u.brandRepo.Save(ctx, b, nil)
}

func (u *BrandUsecase) Delete(ctx context.Context, id string) error {
	return u.brandRepo.Delete(ctx, strings.TrimSpace(id))
}
