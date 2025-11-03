package usecase

import (
    "context"
    "strings"
    "time"

    branddom "narratives/internal/domain/brand"
)

type BrandUsecase struct {
    repo branddom.Repository
    now  func() time.Time
}

func NewBrandUsecase(repo branddom.Repository) *BrandUsecase {
    return &BrandUsecase{
        repo: repo,
        now:  time.Now,
    }
}

// Queries

func (u *BrandUsecase) GetByID(ctx context.Context, id string) (branddom.Brand, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *BrandUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

func (u *BrandUsecase) Count(ctx context.Context, f branddom.Filter) (int, error) {
    return u.repo.Count(ctx, f)
}

func (u *BrandUsecase) List(ctx context.Context, f branddom.Filter, s branddom.Sort, p branddom.Page) (branddom.PageResult[branddom.Brand], error) {
    return u.repo.List(ctx, f, s, p)
}

func (u *BrandUsecase) ListByCursor(ctx context.Context, f branddom.Filter, s branddom.Sort, c branddom.CursorPage) (branddom.CursorPageResult[branddom.Brand], error) {
    return u.repo.ListByCursor(ctx, f, s, c)
}

// Commands

// Create accepts a fully-formed entity. If CreatedAt is zero, it is set to now (UTC).
func (u *BrandUsecase) Create(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
    if b.CreatedAt.IsZero() {
        b.CreatedAt = u.now().UTC()
    }
    return u.repo.Create(ctx, b)
}

// Save performs upsert. If CreatedAt is zero, it is set to now (UTC).
func (u *BrandUsecase) Save(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
    if b.CreatedAt.IsZero() {
        b.CreatedAt = u.now().UTC()
    }
    return u.repo.Save(ctx, b, nil)
}

func (u *BrandUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}