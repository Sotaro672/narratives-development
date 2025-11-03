package usecase

import (
    "context"
    "strings"

    txdom "narratives/internal/domain/transaction"
)

// TransactionRepo defines the minimal persistence port needed by TransactionUsecase.
type TransactionRepo interface {
    GetByID(ctx context.Context, id string) (txdom.Transaction, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v txdom.Transaction) (txdom.Transaction, error)
    Save(ctx context.Context, v txdom.Transaction) (txdom.Transaction, error)
    Delete(ctx context.Context, id string) error
}

// TransactionUsecase orchestrates transaction operations.
type TransactionUsecase struct {
    repo TransactionRepo
}

func NewTransactionUsecase(repo TransactionRepo) *TransactionUsecase {
    return &TransactionUsecase{repo: repo}
}

// Queries

func (u *TransactionUsecase) GetByID(ctx context.Context, id string) (txdom.Transaction, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *TransactionUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *TransactionUsecase) Create(ctx context.Context, v txdom.Transaction) (txdom.Transaction, error) {
    return u.repo.Create(ctx, v)
}

func (u *TransactionUsecase) Save(ctx context.Context, v txdom.Transaction) (txdom.Transaction, error) {
    return u.repo.Save(ctx, v)
}

func (u *TransactionUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}