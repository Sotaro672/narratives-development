package usecase

import (
    "context"
    "strings"

    paymentdom "narratives/internal/domain/payment"
)

// PaymentRepo defines the minimal persistence port needed by PaymentUsecase.
type PaymentRepo interface {
    GetByID(ctx context.Context, id string) (paymentdom.Payment, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v paymentdom.Payment) (paymentdom.Payment, error)
    Save(ctx context.Context, v paymentdom.Payment) (paymentdom.Payment, error)
    Delete(ctx context.Context, id string) error
}

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
    repo PaymentRepo
}

func NewPaymentUsecase(repo PaymentRepo) *PaymentUsecase {
    return &PaymentUsecase{repo: repo}
}

// Queries

func (u *PaymentUsecase) GetByID(ctx context.Context, id string) (paymentdom.Payment, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *PaymentUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *PaymentUsecase) Create(ctx context.Context, v paymentdom.Payment) (paymentdom.Payment, error) {
    return u.repo.Create(ctx, v)
}

func (u *PaymentUsecase) Save(ctx context.Context, v paymentdom.Payment) (paymentdom.Payment, error) {
    return u.repo.Save(ctx, v)
}

func (u *PaymentUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}