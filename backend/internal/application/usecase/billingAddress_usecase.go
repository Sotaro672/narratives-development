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

// 互換: entity.go には default 概念がないが、RepositoryPort 互換のため残す
func (u *BillingAddressUsecase) GetDefaultByUser(ctx context.Context, userID string) (*baddr.BillingAddress, error) {
	return u.repo.GetDefaultByUser(ctx, strings.TrimSpace(userID))
}

func (u *BillingAddressUsecase) Count(ctx context.Context, f baddr.Filter) (int, error) {
	return u.repo.Count(ctx, f)
}

func (u *BillingAddressUsecase) List(ctx context.Context, f baddr.Filter, s baddr.Sort, p baddr.Page) (baddr.PageResult, error) {
	return u.repo.List(ctx, f, s, p)
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

	// Normalize (entity.go 準拠)
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

// 互換: entity.go には default 概念がないが、RepositoryPort 互換のため残す
func (u *BillingAddressUsecase) SetDefault(ctx context.Context, id string) error {
	return u.repo.SetDefault(ctx, strings.TrimSpace(id))
}

// ============================================================
// Utilities (dev/testing)
// ============================================================

func (u *BillingAddressUsecase) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return u.repo.WithTx(ctx, fn)
}

func (u *BillingAddressUsecase) Reset(ctx context.Context) error {
	return u.repo.Reset(ctx)
}

// Helpers は common_usecase.go に移動しました（trimPtr を使用してください）.
