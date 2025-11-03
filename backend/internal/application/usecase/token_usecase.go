package usecase

import (
    "context"
    "strings"

    tokendom "narratives/internal/domain/token"
)

// TokenRepo defines the minimal persistence port needed by TokenUsecase.
type TokenRepo interface {
    GetByID(ctx context.Context, id string) (tokendom.Token, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v tokendom.Token) (tokendom.Token, error)
    Save(ctx context.Context, v tokendom.Token) (tokendom.Token, error)
    Delete(ctx context.Context, id string) error
}

// TokenUsecase orchestrates token operations.
type TokenUsecase struct {
    repo TokenRepo
}

func NewTokenUsecase(repo TokenRepo) *TokenUsecase {
    return &TokenUsecase{repo: repo}
}

// Queries

func (u *TokenUsecase) GetByID(ctx context.Context, id string) (tokendom.Token, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *TokenUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *TokenUsecase) Create(ctx context.Context, v tokendom.Token) (tokendom.Token, error) {
    return u.repo.Create(ctx, v)
}

func (u *TokenUsecase) Save(ctx context.Context, v tokendom.Token) (tokendom.Token, error) {
    return u.repo.Save(ctx, v)
}

func (u *TokenUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}