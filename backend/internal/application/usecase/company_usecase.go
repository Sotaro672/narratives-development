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

func (u *CompanyUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
