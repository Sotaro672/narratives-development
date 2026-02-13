package usecase

import (
	"context"
	"strings"
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
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *CompanyUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

// Create accepts a fully-formed entity. If CreatedAt is zero, it is set to now (UTC).
func (u *CompanyUsecase) Create(ctx context.Context, c companydom.Company) (companydom.Company, error) {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = u.now().UTC()
	}
	return u.repo.Create(ctx, c)
}

// Save performs upsert. If CreatedAt is zero, it is set to now (UTC).
func (u *CompanyUsecase) Save(ctx context.Context, c companydom.Company) (companydom.Company, error) {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = u.now().UTC()
	}
	return u.repo.Save(ctx, c, nil)
}

func (u *CompanyUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
