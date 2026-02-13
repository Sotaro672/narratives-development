// backend/internal/application/usecase/billing_address_usecase.go
package usecase

import (
	"context"
	"time"

	baddr "narratives/internal/domain/billingAddress"
)

type BillingAddressUsecase struct {
	repo baddr.RepositoryPort
	now  func() time.Time
}

func NewBillingAddressUsecase(repo baddr.RepositoryPort) *BillingAddressUsecase {
	return &BillingAddressUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// ============================================================
// Queries
// ============================================================

func (u *BillingAddressUsecase) GetByID(ctx context.Context, id string) (*baddr.BillingAddress, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *BillingAddressUsecase) GetByUser(ctx context.Context, userID string) ([]baddr.BillingAddress, error) {
	return u.repo.GetByUser(ctx, userID)
}

// ============================================================
// Commands
// ============================================================

func (u *BillingAddressUsecase) Create(ctx context.Context, in baddr.CreateBillingAddressInput) (*baddr.BillingAddress, error) {
	now := u.now().UTC()

	// Defaults
	if in.CreatedAt == nil || in.CreatedAt.IsZero() {
		t := now
		in.CreatedAt = &t
	}
	if in.UpdatedAt == nil || in.UpdatedAt.IsZero() {
		t := now
		in.UpdatedAt = &t
	}

	return u.repo.Create(ctx, in)
}

func (u *BillingAddressUsecase) Update(ctx context.Context, id string, in baddr.UpdateBillingAddressInput) (*baddr.BillingAddress, error) {
	now := u.now().UTC()

	// UpdatedAt default
	if in.UpdatedAt == nil || in.UpdatedAt.IsZero() {
		t := now
		in.UpdatedAt = &t
	}

	return u.repo.Update(ctx, id, in)
}

func (u *BillingAddressUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
