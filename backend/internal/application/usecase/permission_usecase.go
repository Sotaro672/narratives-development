package usecase

import (
    "context"
    "strings"

    permissiondom "narratives/internal/domain/permission"
)

// PermissionRepo defines the minimal persistence port needed by PermissionUsecase.
type PermissionRepo interface {
    GetByID(ctx context.Context, id string) (permissiondom.Permission, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v permissiondom.Permission) (permissiondom.Permission, error)
    Save(ctx context.Context, v permissiondom.Permission) (permissiondom.Permission, error)
    Delete(ctx context.Context, id string) error
}

// PermissionUsecase orchestrates permission operations.
type PermissionUsecase struct {
    repo PermissionRepo
}

func NewPermissionUsecase(repo PermissionRepo) *PermissionUsecase {
    return &PermissionUsecase{repo: repo}
}

// Queries

func (u *PermissionUsecase) GetByID(ctx context.Context, id string) (permissiondom.Permission, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *PermissionUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *PermissionUsecase) Create(ctx context.Context, v permissiondom.Permission) (permissiondom.Permission, error) {
    return u.repo.Create(ctx, v)
}

func (u *PermissionUsecase) Save(ctx context.Context, v permissiondom.Permission) (permissiondom.Permission, error) {
    return u.repo.Save(ctx, v)
}

func (u *PermissionUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}