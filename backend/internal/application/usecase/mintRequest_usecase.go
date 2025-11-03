package usecase

import (
    "context"
    "strings"

    mintreqdom "narratives/internal/domain/mintRequest"
)

// MintRequestRepo defines the minimal persistence port needed by MintRequestUsecase.
type MintRequestRepo interface {
    GetByID(ctx context.Context, id string) (mintreqdom.MintRequest, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v mintreqdom.MintRequest) (mintreqdom.MintRequest, error)
    Save(ctx context.Context, v mintreqdom.MintRequest) (mintreqdom.MintRequest, error)
    Delete(ctx context.Context, id string) error
}

// MintRequestUsecase orchestrates mintRequest operations.
type MintRequestUsecase struct {
    repo MintRequestRepo
}

func NewMintRequestUsecase(repo MintRequestRepo) *MintRequestUsecase {
    return &MintRequestUsecase{repo: repo}
}

// Queries

func (u *MintRequestUsecase) GetByID(ctx context.Context, id string) (mintreqdom.MintRequest, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *MintRequestUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *MintRequestUsecase) Create(ctx context.Context, v mintreqdom.MintRequest) (mintreqdom.MintRequest, error) {
    return u.repo.Create(ctx, v)
}

func (u *MintRequestUsecase) Save(ctx context.Context, v mintreqdom.MintRequest) (mintreqdom.MintRequest, error) {
    return u.repo.Save(ctx, v)
}

func (u *MintRequestUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}