// backend/internal/application/usecase/brand_usecase.go
package usecase

import (
	"context"
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

func WithBrandWalletService(
	svc branddom.SolanaBrandWalletService,
) BrandUsecaseOption {
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

func (u *BrandUsecase) Create(
	ctx context.Context,
	b branddom.Brand,
) (branddom.Brand, error) {
	if cid := CompanyIDFromContext(ctx); cid != "" {
		b.CompanyID = cid
	}

	b.IsActive = true

	if b.CreatedAt.IsZero() {
		b.CreatedAt = u.now().UTC()
	}

	var (
		managerID     string
		managerRecord memberdom.Record
		hasManager    bool
	)

	if b.ManagerID != nil &&
		*b.ManagerID != "" &&
		u.memberRepo != nil {
		managerID = *b.ManagerID

		rec, err := u.memberRepo.GetByID(
			ctx,
			managerID,
		)
		if err != nil {
			return branddom.Brand{}, err
		}

		if rec.Member.CompanyID != b.CompanyID {
			return branddom.Brand{}, memberdom.ErrNotFound
		}

		managerRecord = rec
		hasManager = true
	}

	created, err := u.brandRepo.Create(ctx, b)
	if err != nil {
		return created, err
	}

	if hasManager {
		brandID := created.ID
		found := false

		for _, assignedBrandID := range managerRecord.Member.AssignedBrands {
			if assignedBrandID == brandID {
				found = true
				break
			}
		}

		if !found {
			assignedBrands := append(
				[]string(nil),
				managerRecord.Member.AssignedBrands...,
			)

			assignedBrands = append(
				assignedBrands,
				brandID,
			)

			updatedAt := u.now().UTC()

			_, err = u.memberRepo.Update(
				ctx,
				managerID,
				memberdom.MemberPatch{
					AssignedBrands: &assignedBrands,
					UpdatedAt:      &updatedAt,
				},
			)
			if err != nil {
				return created, err
			}
		}
	}

	wa := created.WalletAddress

	if u.walletSvc != nil &&
		(wa == "" || wa == "pending") {
		wallet, walletErr :=
			u.walletSvc.OpenBrandWallet(
				ctx,
				created,
			)

		if walletErr == nil && wallet.Address != "" {
			walletAddress := wallet.Address
			updatedAt := u.now().UTC()

			updated, updateErr :=
				u.brandRepo.Update(
					ctx,
					created.ID,
					branddom.BrandPatch{
						WalletAddress: &walletAddress,
						UpdatedAt:     &updatedAt,
					},
				)

			if updateErr == nil {
				created = updated
			} else {
				created.WalletAddress =
					walletAddress
			}
		}
	}

	return created, nil
}

func (u *BrandUsecase) Update(
	ctx context.Context,
	id string,
	patch branddom.BrandPatch,
) (branddom.Brand, error) {
	if id == "" {
		return branddom.Brand{},
			branddom.ErrInvalidID
	}

	if err := patch.Validate(); err != nil {
		return branddom.Brand{}, err
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

func (u *BrandUsecase) Delete(
	ctx context.Context,
	id string,
) error {
	if id == "" {
		return branddom.ErrInvalidID
	}

	return u.brandRepo.Delete(ctx, id)
}
