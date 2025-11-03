package usecase

import (
    "context"
    "strings"

    saledom "narratives/internal/domain/sale"
)

// SaleRepo defines the minimal persistence port needed by SaleUsecase.
type SaleRepo interface {
    GetByID(ctx context.Context, id string) (saledom.Sale, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v saledom.Sale) (saledom.Sale, error)
    Save(ctx context.Context, v saledom.Sale) (saledom.Sale, error)
    Delete(ctx context.Context, id string) error
}

// SaleUsecase orchestrates sale operations.
type SaleUsecase struct {
    repo SaleRepo
}

func NewSaleUsecase(repo SaleRepo) *SaleUsecase {
    return &SaleUsecase{repo: repo}
}

// Queries

func (u *SaleUsecase) GetByID(ctx context.Context, id string) (saledom.Sale, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *SaleUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *SaleUsecase) Create(ctx context.Context, v saledom.Sale) (saledom.Sale, error) {
    return u.repo.Create(ctx, v)
}

func (u *SaleUsecase) Save(ctx context.Context, v saledom.Sale) (saledom.Sale, error) {
    return u.repo.Save(ctx, v)
}

func (u *SaleUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}