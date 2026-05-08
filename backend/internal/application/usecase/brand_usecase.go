// backend/internal/application/usecase/brand_usecase.go
package usecase

import (
	"context"
	"log"
	"time"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
)

type BrandUsecase struct {
	brandRepo  branddom.Repository
	memberRepo memberdom.Repository
	walletSvc  branddom.SolanaBrandWalletService
	now        func() time.Time
}

type BrandUsecaseOption func(*BrandUsecase)

func WithBrandWalletService(svc branddom.SolanaBrandWalletService) BrandUsecaseOption {
	return func(u *BrandUsecase) {
		u.walletSvc = svc
	}
}

func WithNow(now func() time.Time) BrandUsecaseOption {
	return func(u *BrandUsecase) {
		if now != nil {
			u.now = now
		}
	}
}

func NewBrandUsecase(
	brandRepo branddom.Repository,
	memberRepo memberdom.Repository,
	opts ...BrandUsecaseOption,
) *BrandUsecase {
	u := &BrandUsecase{
		brandRepo:  brandRepo,
		memberRepo: memberRepo,
		walletSvc:  nil,
		now:        time.Now,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(u)
		}
	}

	return u
}

func (u *BrandUsecase) GetByID(ctx context.Context, id string) (branddom.Brand, error) {
	return u.brandRepo.GetByID(ctx, id)
}

func (u *BrandUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.brandRepo.Exists(ctx, id)
}

func (u *BrandUsecase) List(
	ctx context.Context,
	f branddom.Filter,
	p branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	if cid := CompanyIDFromContext(ctx); cid != "" {
		f.CompanyID = &cid
	}

	var sort branddom.Sort
	return u.brandRepo.List(ctx, f, sort, p)
}

func (u *BrandUsecase) ListByCursor(
	ctx context.Context,
	f branddom.Filter,
	c branddom.CursorPage,
) (branddom.CursorPageResult[branddom.Brand], error) {
	if cid := CompanyIDFromContext(ctx); cid != "" {
		f.CompanyID = &cid
	}

	return u.brandRepo.ListByCursor(ctx, f, c)
}

func (u *BrandUsecase) ListCurrentCompanyBrandsWithNames(
	ctx context.Context,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	cid := CompanyIDFromContext(ctx)
	if cid == "" {
		return branddom.PageResult[branddom.Brand]{}, nil
	}

	svc := branddom.NewService(u.brandRepo)

	res, err := svc.ListByCompanyID(ctx, cid, page)
	if err != nil {
		return res, err
	}

	for i, b := range res.Items {
		name, err := svc.GetNameByID(ctx, b.ID)
		if err != nil {
			res.Items[i].Name = b.Name
			continue
		}
		res.Items[i].Name = name
	}

	return res, nil
}

func (u *BrandUsecase) GetMemberNameLastFirstByID(
	ctx context.Context,
	memberID string,
) (string, error) {
	svc := memberdom.NewService(u.memberRepo)
	return svc.GetNameLastFirstByID(ctx, memberID)
}

func (u *BrandUsecase) ResolveMemberNameByID(
	ctx context.Context,
	memberID string,
) (string, error) {
	return u.GetMemberNameLastFirstByID(ctx, memberID)
}

func (u *BrandUsecase) Create(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
	if cid := CompanyIDFromContext(ctx); cid != "" {
		b.CompanyID = cid
	}

	if !b.IsActive {
		b.IsActive = true
	}

	if b.CreatedAt.IsZero() {
		b.CreatedAt = u.now().UTC()
	}

	created, err := u.brandRepo.Create(ctx, b)
	if err != nil {
		return created, err
	}

	wa := created.WalletAddress
	if u.walletSvc != nil && (wa == "" || wa == "pending") {
		wallet, werr := u.walletSvc.OpenBrandWallet(ctx, created)
		if werr != nil {
			log.Printf("[BrandUsecase] WARN: failed to open Solana brand wallet (brandId=%s): %v", created.ID, werr)
		} else {
			addr := wallet.Address
			if addr == "" {
				log.Printf("[BrandUsecase] WARN: OpenBrandWallet returned empty address (brandId=%s)", created.ID)
			} else {
				created.WalletAddress = addr
				if saved, errSave := u.brandRepo.Save(ctx, created, nil); errSave != nil {
					log.Printf("[BrandUsecase] WARN: failed to persist brand walletAddress (brandId=%s wallet=%s): %v",
						created.ID, addr, errSave)
				} else {
					created = saved
				}
			}
		}
	}

	if created.ManagerID == nil || *created.ManagerID == "" {
		return created, nil
	}

	managerID := *created.ManagerID

	m, err := u.memberRepo.GetByID(ctx, managerID)
	if err != nil {
		log.Printf("[BrandUsecase] WARN: managerId=%s の Member 取得失敗: %v", managerID, err)
		return created, nil
	}

	brandID := created.ID
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

	if _, err := u.memberRepo.Save(ctx, m, nil); err != nil {
		log.Printf("[BrandUsecase] WARN: Member.assignedBrands 更新失敗 (memberId=%s brandId=%s): %v",
			managerID, brandID, err)
	}

	return created, nil
}

func (u *BrandUsecase) UpdateBrand(
	ctx context.Context,
	id string,
	managerID *string,
	brandName *string,
	description *string,
	websiteURL *string,
	brandIcon *string,
	brandBackgroundImage *string,
	isActive *bool,
) (branddom.Brand, error) {
	if id == "" {
		return branddom.Brand{}, branddom.ErrInvalidID
	}

	patch := branddom.BrandPatch{
		ManagerID:            managerID,
		Name:                 brandName,
		Description:          description,
		URL:                  websiteURL,
		BrandIcon:            brandIcon,
		BrandBackgroundImage: brandBackgroundImage,
		IsActive:             isActive,
		UpdatedAt: func() *time.Time {
			t := u.now().UTC()
			return &t
		}(),
	}

	if cid := CompanyIDFromContext(ctx); cid != "" {
		patch.CompanyID = &cid
	}

	return u.brandRepo.Update(ctx, id, patch)
}

func (u *BrandUsecase) Update(
	ctx context.Context,
	id string,
	patch branddom.BrandPatch,
) (branddom.Brand, error) {
	if id == "" {
		return branddom.Brand{}, branddom.ErrInvalidID
	}

	if cid := CompanyIDFromContext(ctx); cid != "" {
		patch.CompanyID = &cid
	}

	if patch.UpdatedAt == nil {
		t := u.now().UTC()
		patch.UpdatedAt = &t
	}

	return u.brandRepo.Update(ctx, id, patch)
}

func (u *BrandUsecase) Save(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
	if cid := CompanyIDFromContext(ctx); cid != "" {
		b.CompanyID = cid
	}

	if b.CreatedAt.IsZero() {
		b.CreatedAt = u.now().UTC()
	}

	if b.UpdatedAt == nil && b.ID != "" {
		t := u.now().UTC()
		b.UpdatedAt = &t
	}

	return u.brandRepo.Save(ctx, b, nil)
}

func (u *BrandUsecase) Delete(ctx context.Context, id string) error {
	return u.brandRepo.Delete(ctx, id)
}
