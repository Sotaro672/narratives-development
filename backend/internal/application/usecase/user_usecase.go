package usecase

import (
    "context"
    "strings"

    userdom "narratives/internal/domain/user"
)

// UserRepo defines the minimal persistence port needed by UserUsecase.
type UserRepo interface {
    GetByID(ctx context.Context, id string) (userdom.User, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v userdom.User) (userdom.User, error)
    Save(ctx context.Context, v userdom.User) (userdom.User, error)
    Delete(ctx context.Context, id string) error
}

// UserUsecase orchestrates user operations.
type UserUsecase struct {
    repo UserRepo
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
    return &UserUsecase{repo: repo}
}

// Queries

func (u *UserUsecase) GetByID(ctx context.Context, id string) (userdom.User, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *UserUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *UserUsecase) Create(ctx context.Context, v userdom.User) (userdom.User, error) {
    return u.repo.Create(ctx, v)
}

func (u *UserUsecase) Save(ctx context.Context, v userdom.User) (userdom.User, error) {
    return u.repo.Save(ctx, v)
}

func (u *UserUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}