package usecase

import (
    "context"
    "strings"

    tokopdom "narratives/internal/domain/tokenOperation"
)

// TokenOperationRepo defines the minimal persistence port needed by TokenOperationUsecase.
type TokenOperationRepo interface {
    GetByID(ctx context.Context, id string) (tokopdom.TokenOperation, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v tokopdom.TokenOperation) (tokopdom.TokenOperation, error)
    Save(ctx context.Context, v tokopdom.TokenOperation) (tokopdom.TokenOperation, error)
    Delete(ctx context.Context, id string) error
}

// TokenOperationUsecase orchestrates tokenOperation operations.
type TokenOperationUsecase struct {
    repo TokenOperationRepo
}

func NewTokenOperationUsecase(repo TokenOperationRepo) *TokenOperationUsecase {
    return &TokenOperationUsecase{repo: repo}
}

// Queries

func (u *TokenOperationUsecase) GetByID(ctx context.Context, id string) (tokopdom.TokenOperation, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *TokenOperationUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *TokenOperationUsecase) Create(ctx context.Context, v tokopdom.TokenOperation) (tokopdom.TokenOperation, error) {
    return u.repo.Create(ctx, v)
}

func (u *TokenOperationUsecase) Save(ctx context.Context, v tokopdom.TokenOperation) (tokopdom.TokenOperation, error) {
    return u.repo.Save(ctx, v)
}

func (u *TokenOperationUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}