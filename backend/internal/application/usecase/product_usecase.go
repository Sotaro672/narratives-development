package usecase

import (
    "context"
    "strings"

    productdom "narratives/internal/domain/product"
)

// ProductRepo defines the minimal persistence port needed by ProductUsecase.
type ProductRepo interface {
    GetByID(ctx context.Context, id string) (productdom.Product, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, p productdom.Product) (productdom.Product, error)
    Save(ctx context.Context, p productdom.Product) (productdom.Product, error)
    Delete(ctx context.Context, id string) error
}

// ProductUsecase orchestrates product operations.
type ProductUsecase struct {
    repo ProductRepo
}

func NewProductUsecase(repo ProductRepo) *ProductUsecase {
    return &ProductUsecase{repo: repo}
}

// Queries

func (u *ProductUsecase) GetByID(ctx context.Context, id string) (productdom.Product, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ProductUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *ProductUsecase) Create(ctx context.Context, p productdom.Product) (productdom.Product, error) {
    return u.repo.Create(ctx, p)
}

func (u *ProductUsecase) Save(ctx context.Context, p productdom.Product) (productdom.Product, error) {
    return u.repo.Save(ctx, p)
}

func (u *ProductUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}