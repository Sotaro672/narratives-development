package usecase

import (
    "context"
    "strings"
    "time"

    fulfillmentdom "narratives/internal/domain/fulfillment"
)

// FulfillmentRepo defines the minimal persistence port needed by FulfillmentUsecase.
type FulfillmentRepo interface {
    GetByID(ctx context.Context, id string) (fulfillmentdom.Fulfillment, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, f fulfillmentdom.Fulfillment) (fulfillmentdom.Fulfillment, error)
    Save(ctx context.Context, f fulfillmentdom.Fulfillment) (fulfillmentdom.Fulfillment, error)
    Delete(ctx context.Context, id string) error
}

// FulfillmentUsecase orchestrates fulfillment operations.
type FulfillmentUsecase struct {
    repo FulfillmentRepo
    now  func() time.Time
}

func NewFulfillmentUsecase(repo FulfillmentRepo) *FulfillmentUsecase {
    return &FulfillmentUsecase{
        repo: repo,
        now:  time.Now,
    }
}

// Queries

func (u *FulfillmentUsecase) GetByID(ctx context.Context, id string) (fulfillmentdom.Fulfillment, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *FulfillmentUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

// Create accepts a fully-formed entity. If CreatedAt is zero, it is set to now (UTC).
func (u *FulfillmentUsecase) Create(ctx context.Context, f fulfillmentdom.Fulfillment) (fulfillmentdom.Fulfillment, error) {
    if f.CreatedAt.IsZero() {
        f.CreatedAt = u.now().UTC()
    }
    return u.repo.Create(ctx, f)
}

// Save performs upsert. If CreatedAt is zero, it is set to now (UTC).
func (u *FulfillmentUsecase) Save(ctx context.Context, f fulfillmentdom.Fulfillment) (fulfillmentdom.Fulfillment, error) {
    if f.CreatedAt.IsZero() {
        f.CreatedAt = u.now().UTC()
    }
    return u.repo.Save(ctx, f)
}

func (u *FulfillmentUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}