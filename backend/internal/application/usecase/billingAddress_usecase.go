// backend/internal/application/usecase/billing_address_usecase.go
package usecase

import (
	"context"
	"strings"
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
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *BillingAddressUsecase) GetByUser(ctx context.Context, userID string) ([]baddr.BillingAddress, error) {
	return u.repo.GetByUser(ctx, strings.TrimSpace(userID))
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

	// Normalize
	in.UserID = strings.TrimSpace(in.UserID)
	in.CardNumber = strings.TrimSpace(in.CardNumber)
	in.CardholderName = strings.TrimSpace(in.CardholderName)
	in.CVC = strings.TrimSpace(in.CVC)

	return u.repo.Create(ctx, in)
}

func (u *BillingAddressUsecase) Update(ctx context.Context, id string, in baddr.UpdateBillingAddressInput) (*baddr.BillingAddress, error) {
	now := u.now().UTC()

	// Normalize
	id = strings.TrimSpace(id)
	in.CardNumber = trimPtr(in.CardNumber)
	in.CardholderName = trimPtr(in.CardholderName)
	in.CVC = trimPtr(in.CVC)

	// UpdatedAt default
	if in.UpdatedAt == nil || in.UpdatedAt.IsZero() {
		t := now
		in.UpdatedAt = &t
	}

	return u.repo.Update(ctx, id, in)
}

func (u *BillingAddressUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
