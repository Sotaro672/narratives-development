// backend/internal/application/usecase/cart_usecase.go
package usecase

import (
	"context"
	"errors"
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
	aid := avatarID
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
	aid := avatarID
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

// AddItem increments qty for (inventoryId, listId, modelId).
// qty must be >= 1.
func (uc *CartUsecase) AddItem(
	ctx context.Context,
	avatarID, inventoryID, listID, modelID string,
	qty int,
) (*cartdom.Cart, error) {
	aid := avatarID
	inv := inventoryID
	lid := listID
	mid := modelID
	if aid == "" || inv == "" || lid == "" || mid == "" || qty <= 0 {
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

	if err := c.Add(inv, lid, mid, qty, now); err != nil {
		return nil, err
	}
	if err := uc.repo.Upsert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// SetItemQty sets qty for (inventoryId, listId, modelId).
// If qty <= 0, it removes the item.
func (uc *CartUsecase) SetItemQty(
	ctx context.Context,
	avatarID, inventoryID, listID, modelID string,
	qty int,
) (*cartdom.Cart, error) {
	aid := avatarID
	inv := inventoryID
	lid := listID
	mid := modelID
	if aid == "" || inv == "" || lid == "" || mid == "" {
		return nil, ErrCartInvalidArgument
	}

	c, err := uc.repo.GetByAvatarID(ctx, aid)
	if err != nil {
		return nil, err
	}

	now := uc.clock.Now()

	if c == nil {
		// policy: cart absent -> create (then apply)
		c, err = cartdom.NewCart(aid, nil, now)
		if err != nil {
			return nil, err
		}
	}

	if err := c.SetQty(inv, lid, mid, qty, now); err != nil {
		return nil, err
	}
	if err := uc.repo.Upsert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// RemoveItem removes an item (inventoryId, listId, modelId) from cart.
func (uc *CartUsecase) RemoveItem(
	ctx context.Context,
	avatarID, inventoryID, listID, modelID string,
) (*cartdom.Cart, error) {
	return uc.SetItemQty(ctx, avatarID, inventoryID, listID, modelID, 0)
}

// Clear deletes the cart doc (useful for "empty cart" UX).
func (uc *CartUsecase) Clear(ctx context.Context, avatarID string) error {
	aid := avatarID
	if aid == "" {
		return ErrCartInvalidArgument
	}
	return uc.repo.DeleteByAvatarID(ctx, aid)
}

// EmptyItems empties cart.items but keeps the cart doc.
// - payment 成功後など「カートは残すが中身を空にする」用途
// - cart が存在しない場合は空のカートを作って Upsert（冪等）
func (uc *CartUsecase) EmptyItems(ctx context.Context, avatarID string) error {
	aid := avatarID
	if aid == "" {
		return ErrCartInvalidArgument
	}

	now := uc.clock.Now()

	c, err := uc.repo.GetByAvatarID(ctx, aid)
	if err != nil {
		return err
	}

	// absent -> create empty and upsert (idempotent)
	if c == nil {
		newCart, err := cartdom.NewCart(aid, nil, now)
		if err != nil {
			return err
		}
		return uc.repo.Upsert(ctx, newCart)
	}

	if _, err := c.ConsumeAll(now); err != nil {
		return err
	}

	return uc.repo.Upsert(ctx, c)
}
