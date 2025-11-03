package usecase

import (
    "context"
    "strings"
    "time"

    productiondom "narratives/internal/domain/production"
)

// ProductionRepo defines the minimal persistence port needed by ProductionUsecase.
type ProductionRepo interface {
    GetByID(ctx context.Context, id string) (productiondom.Production, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, p productiondom.Production) (productiondom.Production, error)
    Save(ctx context.Context, p productiondom.Production) (productiondom.Production, error)
    Delete(ctx context.Context, id string) error
}

// ProductionUsecase orchestrates production operations.
type ProductionUsecase struct {
    repo ProductionRepo
    now  func() time.Time
}

func NewProductionUsecase(repo ProductionRepo) *ProductionUsecase {
    return &ProductionUsecase{
        repo: repo,
        now:  time.Now,
    }
}

// Queries

func (u *ProductionUsecase) GetByID(ctx context.Context, id string) (productiondom.Production, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductionUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

// Create accepts a fully-formed entity. If CreatedAt is zero, it is set to now (UTC).
func (u *ProductionUsecase) Create(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
    // Best-effort normalization of timestamps commonly present on entities
    if p.CreatedAt.IsZero() {
        p.CreatedAt = u.now().UTC()
    }
    return u.repo.Create(ctx, p)
}

// Save performs upsert. If CreatedAt is zero, it is set to now (UTC).
func (u *ProductionUsecase) Save(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
    if p.CreatedAt.IsZero() {
        p.CreatedAt = u.now().UTC()
    }
    return u.repo.Save(ctx, p)
}

func (u *ProductionUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}