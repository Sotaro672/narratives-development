package usecase

import (
	"context"
	"errors"
	"reflect"
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

// AddItem increments qty for (inventoryId, listId, modelId).
// qty must be >= 1.
func (uc *CartUsecase) AddItem(
	ctx context.Context,
	avatarID, inventoryID, listID, modelID string,
	qty int,
) (*cartdom.Cart, error) {
	aid := strings.TrimSpace(avatarID)
	inv := strings.TrimSpace(inventoryID)
	lid := strings.TrimSpace(listID)
	mid := strings.TrimSpace(modelID)
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
	aid := strings.TrimSpace(avatarID)
	inv := strings.TrimSpace(inventoryID)
	lid := strings.TrimSpace(listID)
	mid := strings.TrimSpace(modelID)
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
	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return ErrCartInvalidArgument
	}
	return uc.repo.DeleteByAvatarID(ctx, aid)
}

// ✅ NEW: EmptyItems empties cart.items but keeps the cart doc.
// - payment 成功後など「カートは残すが中身を空にする」用途
// - cart が存在しない場合は空のカートを作って Upsert（冪等）
func (uc *CartUsecase) EmptyItems(ctx context.Context, avatarID string) error {
	aid := strings.TrimSpace(avatarID)
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

	// best-effort: clear Items field (map/slice), set UpdatedAt if exists
	if ok := clearCartItemsBestEffort(c, now); !ok {
		// fallback: recreate empty cart (schema-safe)
		newCart, err := cartdom.NewCart(aid, nil, now)
		if err != nil {
			return err
		}
		return uc.repo.Upsert(ctx, newCart)
	}

	return uc.repo.Upsert(ctx, c)
}

// clearCartItemsBestEffort tries to set:
// - cart.Items = empty (map/slice)
// - cart.UpdatedAt = &now (if exists and settable)
// Returns true if it changed something.
func clearCartItemsBestEffort(cart any, now time.Time) bool {
	if cart == nil {
		return false
	}

	rv := reflect.ValueOf(cart)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return false
	}
	ev := rv.Elem()
	if !ev.IsValid() || ev.Kind() != reflect.Struct {
		return false
	}

	changed := false

	// Items (map/slice)
	if f := ev.FieldByName("Items"); f.IsValid() && f.CanSet() {
		switch f.Kind() {
		case reflect.Map:
			f.Set(reflect.MakeMap(f.Type()))
			changed = true
		case reflect.Slice:
			f.Set(reflect.MakeSlice(f.Type(), 0, 0))
			changed = true
		}
	}

	// UpdatedAt *time.Time (optional)
	if f := ev.FieldByName("UpdatedAt"); f.IsValid() && f.CanSet() {
		if f.Kind() == reflect.Pointer && f.Type().Elem() == reflect.TypeOf(time.Time{}) {
			t := now.UTC()
			f.Set(reflect.ValueOf(&t))
			changed = true
		}
	}

	return changed
}

// ✅ Ordered フィールド廃止により MarkOrdered は usecase からも削除。
// 以後の「注文確定」は Order を作成するユースケース（例: OrderUsecase）で扱い、
// その中で以下のいずれかを実施してください:
// - 成功後に uc.Clear(ctx, avatarID) でカートを空にする
// - もしくは「注文作成時に items を消す」ドメインメソッドを追加して Upsert する
//
// ※今回の変更ではコンパイルを通すため、MarkOrdered を実装しません。
