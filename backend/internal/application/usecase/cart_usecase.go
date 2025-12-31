// backend/internal/application/usecase/cart_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	cartdom "narratives/internal/domain/cart"
)

var (
	ErrCartInvalidArgument = errors.New("cart_usecase: invalid argument")
	ErrCartNotFound        = errors.New("cart_usecase: not found")
)

// Clock provides current time (for testability).
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// CartUsecase coordinates cart operations.
type CartUsecase struct {
	repo  cartdom.Repository
	clock Clock
}

func NewCartUsecase(repo cartdom.Repository) *CartUsecase {
	return &CartUsecase{
		repo:  repo,
		clock: systemClock{},
	}
}

// NewCartUsecaseWithClock is useful for tests.
func NewCartUsecaseWithClock(repo cartdom.Repository, clock Clock) *CartUsecase {
	if clock == nil {
		clock = systemClock{}
	}
	return &CartUsecase{repo: repo, clock: clock}
}

// Get returns the cart for avatarID.
// If cart does not exist, returns (nil, ErrCartNotFound).
func (uc *CartUsecase) Get(ctx context.Context, avatarID string) (*cartdom.Cart, error) {
	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return nil, ErrCartInvalidArgument
	}

	c, err := uc.repo.GetByAvatarID(ctx, aid)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCartNotFound
	}
	return c, nil
}

// GetOrCreate returns an existing cart; if absent, creates an empty one and persists it.
func (uc *CartUsecase) GetOrCreate(ctx context.Context, avatarID string) (*cartdom.Cart, error) {
	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return nil, ErrCartInvalidArgument
	}

	c, err := uc.repo.GetByAvatarID(ctx, aid)
	if err != nil {
		return nil, err
	}
	if c != nil {
		return c, nil
	}

	now := uc.clock.Now()
	newCart, err := cartdom.NewCart(aid, nil, now)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Upsert(ctx, newCart); err != nil {
		return nil, err
	}
	return newCart, nil
}

// AddItem increments qty for modelID.
// qty must be >= 1.
func (uc *CartUsecase) AddItem(ctx context.Context, avatarID, modelID string, qty int) (*cartdom.Cart, error) {
	aid := strings.TrimSpace(avatarID)
	mid := strings.TrimSpace(modelID)
	if aid == "" || mid == "" || qty <= 0 {
		return nil, ErrCartInvalidArgument
	}

	now := uc.clock.Now()

	c, err := uc.repo.GetByAvatarID(ctx, aid)
	if err != nil {
		return nil, err
	}
	if c == nil {
		c, err = cartdom.NewCart(aid, nil, now)
		if err != nil {
			return nil, err
		}
	}

	if err := c.Add(mid, qty, now); err != nil {
		return nil, err
	}
	if err := uc.repo.Upsert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// SetItemQty sets qty for modelID.
// If qty <= 0, it removes the item.
func (uc *CartUsecase) SetItemQty(ctx context.Context, avatarID, modelID string, qty int) (*cartdom.Cart, error) {
	aid := strings.TrimSpace(avatarID)
	mid := strings.TrimSpace(modelID)
	if aid == "" || mid == "" {
		return nil, ErrCartInvalidArgument
	}

	c, err := uc.repo.GetByAvatarID(ctx, aid)
	if err != nil {
		return nil, err
	}
	if c == nil {
		// policy: cart absent -> create (then apply)
		now := uc.clock.Now()
		c, err = cartdom.NewCart(aid, nil, now)
		if err != nil {
			return nil, err
		}
	}

	now := uc.clock.Now()
	if err := c.SetQty(mid, qty, now); err != nil {
		return nil, err
	}
	if err := uc.repo.Upsert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RemoveItem removes modelID from cart.
func (uc *CartUsecase) RemoveItem(ctx context.Context, avatarID, modelID string) (*cartdom.Cart, error) {
	return uc.SetItemQty(ctx, avatarID, modelID, 0)
}

// Clear deletes the cart doc (useful for "empty cart" UX).
func (uc *CartUsecase) Clear(ctx context.Context, avatarID string) error {
	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return ErrCartInvalidArgument
	}
	return uc.repo.DeleteByAvatarID(ctx, aid)
}

// MarkOrdered sets Ordered=true for the cart (does not delete).
// (If you prefer "order completes -> cart deleted", call Clear() after this in application flow.)
func (uc *CartUsecase) MarkOrdered(ctx context.Context, avatarID string) (*cartdom.Cart, error) {
	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return nil, ErrCartInvalidArgument
	}

	c, err := uc.repo.GetByAvatarID(ctx, aid)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCartNotFound
	}

	now := uc.clock.Now()
	if err := c.MarkOrdered(now); err != nil {
		return nil, err
	}
	if err := uc.repo.Upsert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
