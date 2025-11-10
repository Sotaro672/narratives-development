package usecase

import (
	"context"
	"strings"

	txdom "narratives/internal/domain/transaction"
)

// TransferRepo defines the minimal persistence port needed by TransferUsecase.
type TransferRepo interface {
	GetByID(ctx context.Context, id string) (txdom.Transaction, error)
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, v txdom.Transaction) (txdom.Transaction, error)
	Save(ctx context.Context, v txdom.Transaction) (txdom.Transaction, error)
	Delete(ctx context.Context, id string) error
}

// TransferUsecase orchestrates transfer-related transaction operations.
type TransferUsecase struct {
	repo TransferRepo
}

func NewTransferUsecase(repo TransferRepo) *TransferUsecase {
	return &TransferUsecase{repo: repo}
}

// Queries

func (u *TransferUsecase) GetByID(ctx context.Context, id string) (txdom.Transaction, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *TransferUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *TransferUsecase) Create(ctx context.Context, v txdom.Transaction) (txdom.Transaction, error) {
	return u.repo.Create(ctx, v)
}

func (u *TransferUsecase) Save(ctx context.Context, v txdom.Transaction) (txdom.Transaction, error) {
	return u.repo.Save(ctx, v)
}

func (u *TransferUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
