package usecase

import (
    "context"
    "strings"

    shipaddrdom "narratives/internal/domain/shippingAddress"
)

// ShippingAddressRepo defines the minimal persistence port needed by ShippingAddressUsecase.
type ShippingAddressRepo interface {
    GetByID(ctx context.Context, id string) (shipaddrdom.ShippingAddress, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v shipaddrdom.ShippingAddress) (shipaddrdom.ShippingAddress, error)
    Save(ctx context.Context, v shipaddrdom.ShippingAddress) (shipaddrdom.ShippingAddress, error)
    Delete(ctx context.Context, id string) error
}

// ShippingAddressUsecase orchestrates shippingAddress operations.
type ShippingAddressUsecase struct {
    repo ShippingAddressRepo
}

func NewShippingAddressUsecase(repo ShippingAddressRepo) *ShippingAddressUsecase {
    return &ShippingAddressUsecase{repo: repo}
}

// Queries

func (u *ShippingAddressUsecase) GetByID(ctx context.Context, id string) (shipaddrdom.ShippingAddress, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ShippingAddressUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *ShippingAddressUsecase) Create(ctx context.Context, v shipaddrdom.ShippingAddress) (shipaddrdom.ShippingAddress, error) {
    return u.repo.Create(ctx, v)
}

func (u *ShippingAddressUsecase) Save(ctx context.Context, v shipaddrdom.ShippingAddress) (shipaddrdom.ShippingAddress, error) {
    return u.repo.Save(ctx, v)
}

func (u *ShippingAddressUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}