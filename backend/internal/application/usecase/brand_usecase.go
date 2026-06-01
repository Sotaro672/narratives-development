// backend/internal/application/usecase/brand_usecase.go
package usecase

import (
	"context"
	"time"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
)

type BrandUsecase struct {
	brandRepo  branddom.RepositoryPort
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
	brandRepo branddom.RepositoryPort,
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
	if id == "" {
		return branddom.Brand{}, branddom.ErrInvalidID
	}

	return u.brandRepo.GetByID(ctx, id)
}

func (u *BrandUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	if companyID == "" {
		return branddom.PageResult[branddom.Brand]{}, branddom.ErrInvalidID
	}

	return u.brandRepo.ListByCompanyID(ctx, companyID, page)
}

func (u *BrandUsecase) ListCurrentCompanyBrandsWithNames(
	ctx context.Context,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	cid := CompanyIDFromContext(ctx)
	if cid == "" {
		return branddom.PageResult[branddom.Brand]{}, nil
	}

	return u.brandRepo.ListByCompanyID(ctx, cid, page)
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
		if werr == nil && wallet.Address != "" {
			walletAddress := wallet.Address
			updatedAt := u.now().UTC()

			updated, errUpdate := u.brandRepo.Update(ctx, created.ID, branddom.BrandPatch{
				WalletAddress: &walletAddress,
				UpdatedAt:     &updatedAt,
			})
			if errUpdate == nil {
				created = updated
			} else {
				created.WalletAddress = walletAddress
			}
		}
	}

	if created.ManagerID == nil || *created.ManagerID == "" {
		return created, nil
	}

	managerID := *created.ManagerID

	m, err := u.memberRepo.GetByID(ctx, managerID)
	if err != nil {
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

	_, _ = u.memberRepo.Save(ctx, m, nil)

	return created, nil
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

func (u *BrandUsecase) Delete(ctx context.Context, id string) error {
	if id == "" {
		return branddom.ErrInvalidID
	}

	return u.brandRepo.Delete(ctx, id)
}
