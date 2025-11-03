package usecase

import (
    "context"
    "strings"
    "time"

    inventorydom "narratives/internal/domain/inventory"
)

// InventoryRepo defines the minimal persistence port needed by InventoryUsecase.
type InventoryRepo interface {
    GetByID(ctx context.Context, id string) (inventorydom.Inventory, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, inv inventorydom.Inventory) (inventorydom.Inventory, error)
    Save(ctx context.Context, inv inventorydom.Inventory) (inventorydom.Inventory, error)
    Delete(ctx context.Context, id string) error
}

// InventoryUsecase orchestrates inventory operations.
type InventoryUsecase struct {
    repo InventoryRepo
    now  func() time.Time
}

func NewInventoryUsecase(repo InventoryRepo) *InventoryUsecase {
    return &InventoryUsecase{
        repo: repo,
        now:  time.Now,
    }
}

// Queries

func (u *InventoryUsecase) GetByID(ctx context.Context, id string) (inventorydom.Inventory, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *InventoryUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

// Create accepts a fully-formed entity. If CreatedAt is zero, it is set to now (UTC).
func (u *InventoryUsecase) Create(ctx context.Context, inv inventorydom.Inventory) (inventorydom.Inventory, error) {
    if inv.CreatedAt.IsZero() {
        inv.CreatedAt = u.now().UTC()
    }
    return u.repo.Create(ctx, inv)
}

// Save performs upsert. If CreatedAt is zero, it is set to now (UTC).
func (u *InventoryUsecase) Save(ctx context.Context, inv inventorydom.Inventory) (inventorydom.Inventory, error) {
    if inv.CreatedAt.IsZero() {
        inv.CreatedAt = u.now().UTC()
    }
    return u.repo.Save(ctx, inv)
}

func (u *InventoryUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}