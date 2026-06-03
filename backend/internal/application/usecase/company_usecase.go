// backend/internal/application/usecase/company_usecase.go
package usecase

import (
	"context"
	"time"

	companydom "narratives/internal/domain/company"
)

type CompanyUsecase struct {
	repo companydom.Repository
	now  func() time.Time
}

func NewCompanyUsecase(repo companydom.Repository) *CompanyUsecase {
	return &CompanyUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// Queries

func (u *CompanyUsecase) GetByID(ctx context.Context, id string) (companydom.Company, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *CompanyUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, id)
}

// Commands

// Create accepts a fully-formed entity.
// If CreatedAt is zero, it is set to now UTC.
// If UpdatedAt is zero, it is set to CreatedAt.
func (u *CompanyUsecase) Create(ctx context.Context, c companydom.Company) (companydom.Company, error) {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = u.now().UTC()
	}

	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}

	return u.repo.Create(ctx, c)
}

// Update applies partial updates through repository port.
// Save は使わず、Repository.Update に統一する。
func (u *CompanyUsecase) Update(ctx context.Context, id string, patch companydom.CompanyPatch) (companydom.Company, error) {
	if patch.UpdatedAt == nil {
		now := u.now().UTC()
		patch.UpdatedAt = &now
	}

	return u.repo.Update(ctx, id, patch)
}

// UpdateFromEntity converts a Company entity into CompanyPatch and updates it.
// 既存の Save(ctx, company) 的な呼び出しを置き換えるための入口。
func (u *CompanyUsecase) UpdateFromEntity(ctx context.Context, c companydom.Company) (companydom.Company, error) {
	updatedAt := c.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = u.now().UTC()
	}

	patch := companydom.CompanyPatch{
		Name:      &c.Name,
		Admin:     &c.Admin,
		IsActive:  &c.IsActive,
		UpdatedAt: &updatedAt,
		UpdatedBy: &c.UpdatedBy,
		DeletedAt: c.DeletedAt,
		DeletedBy: c.DeletedBy,
	}

	return u.repo.Update(ctx, c.ID, patch)
}

func (u *CompanyUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
