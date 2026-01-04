// backend/internal/application/usecase/order_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// OrderRepo is the persistence port required by OrderUsecase.
type OrderRepo interface {
	// Queries
	GetByID(ctx context.Context, id string) (orderdom.Order, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter OrderFilter) (int, error)
	List(ctx context.Context, filter OrderFilter, sort common.Sort, page common.Page) (common.PageResult[orderdom.Order], error)
	ListByCursor(ctx context.Context, filter OrderFilter, sort common.Sort, cpage common.CursorPage) (common.CursorPageResult[orderdom.Order], error)

	// Commands
	Create(ctx context.Context, o orderdom.Order) (orderdom.Order, error)
	Save(ctx context.Context, o orderdom.Order, opts *common.SaveOptions) (orderdom.Order, error)
	Delete(ctx context.Context, id string) error
}

// OrderFilter provides basic filtering for listing orders.
// ✅ entity.go を正として、CreatedAt のみ
type OrderFilter struct {
	ID     string
	UserID *string
	CartID *string

	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// OrderUsecase orchestrates order operations.
type OrderUsecase struct {
	repo OrderRepo
	now  func() time.Time
}

func NewOrderUsecase(repo OrderRepo) *OrderUsecase {
	return &OrderUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// =======================
// Queries
// =======================

func (u *OrderUsecase) GetByID(ctx context.Context, id string) (orderdom.Order, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *OrderUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

func (u *OrderUsecase) Count(ctx context.Context, f OrderFilter) (int, error) {
	return u.repo.Count(ctx, f)
}

func (u *OrderUsecase) List(ctx context.Context, f OrderFilter, s common.Sort, p common.Page) (common.PageResult[orderdom.Order], error) {
	return u.repo.List(ctx, f, s, p)
}

func (u *OrderUsecase) ListByCursor(ctx context.Context, f OrderFilter, s common.Sort, c common.CursorPage) (common.CursorPageResult[orderdom.Order], error) {
	return u.repo.ListByCursor(ctx, f, s, c)
}

// =======================
// Commands
// =======================

type CreateOrderInput struct {
	ID       string
	UserID   string
	AvatarID string
	CartID   string

	ShippingSnapshot orderdom.ShippingSnapshot
	BillingSnapshot  orderdom.BillingSnapshot
	Items            []orderdom.OrderItemSnapshot

	CreatedAt *time.Time // optional; defaults to now
}

func (u *OrderUsecase) Create(ctx context.Context, in CreateOrderInput) (orderdom.Order, error) {
	now := u.now().UTC()
	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	// ✅ IDはdomainで必須。未指定ならここで生成してからNewする。
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = u.newOrderID(now)
	}

	ship := orderdom.ShippingSnapshot{
		ZipCode: strings.TrimSpace(in.ShippingSnapshot.ZipCode),
		State:   strings.TrimSpace(in.ShippingSnapshot.State),
		City:    strings.TrimSpace(in.ShippingSnapshot.City),
		Street:  strings.TrimSpace(in.ShippingSnapshot.Street),
		Street2: strings.TrimSpace(in.ShippingSnapshot.Street2),
		Country: strings.TrimSpace(in.ShippingSnapshot.Country),
	}
	bill := orderdom.BillingSnapshot{
		Last4:          strings.TrimSpace(in.BillingSnapshot.Last4),
		CardHolderName: strings.TrimSpace(in.BillingSnapshot.CardHolderName),
	}

	// normalize items (trim strings)
	items := make([]orderdom.OrderItemSnapshot, 0, len(in.Items))
	for _, it := range in.Items {
		items = append(items, orderdom.OrderItemSnapshot{
			ModelID:     strings.TrimSpace(it.ModelID),
			InventoryID: strings.TrimSpace(it.InventoryID),
			Qty:         it.Qty,
			Price:       it.Price,
		})
	}

	// ✅ entity.go の New(...) に合わせる（avatarId を含む）
	o, err := orderdom.New(
		id,
		strings.TrimSpace(in.UserID),
		strings.TrimSpace(in.AvatarID),
		strings.TrimSpace(in.CartID),
		ship,
		bill,
		items,
		createdAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}
	return u.repo.Create(ctx, o)
}

type UpdateOrderInput struct {
	ID string

	UserID   *string
	AvatarID *string
	CartID   *string

	ShippingSnapshot *orderdom.ShippingSnapshot
	BillingSnapshot  *orderdom.BillingSnapshot

	ReplaceItems *[]orderdom.OrderItemSnapshot
}

func (u *OrderUsecase) Update(ctx context.Context, in UpdateOrderInput) (orderdom.Order, error) {
	o, err := u.repo.GetByID(ctx, strings.TrimSpace(in.ID))
	if err != nil {
		return orderdom.Order{}, err
	}

	// ✅ CreatedAt は entity.go の必須想定。ゼロなら now を補完して整合させる
	if o.CreatedAt.IsZero() {
		o.CreatedAt = u.now().UTC()
	}

	if in.UserID != nil {
		o.UserID = strings.TrimSpace(*in.UserID)
	}
	if in.AvatarID != nil {
		o.AvatarID = strings.TrimSpace(*in.AvatarID)
	}
	if in.CartID != nil {
		o.CartID = strings.TrimSpace(*in.CartID)
	}

	if in.ShippingSnapshot != nil {
		s := orderdom.ShippingSnapshot{
			ZipCode: strings.TrimSpace(in.ShippingSnapshot.ZipCode),
			State:   strings.TrimSpace(in.ShippingSnapshot.State),
			City:    strings.TrimSpace(in.ShippingSnapshot.City),
			Street:  strings.TrimSpace(in.ShippingSnapshot.Street),
			Street2: strings.TrimSpace(in.ShippingSnapshot.Street2),
			Country: strings.TrimSpace(in.ShippingSnapshot.Country),
		}
		if err := o.UpdateShippingSnapshot(s); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.BillingSnapshot != nil {
		b := orderdom.BillingSnapshot{
			Last4:          strings.TrimSpace(in.BillingSnapshot.Last4),
			CardHolderName: strings.TrimSpace(in.BillingSnapshot.CardHolderName),
		}
		if err := o.UpdateBillingSnapshot(b); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.ReplaceItems != nil {
		items := make([]orderdom.OrderItemSnapshot, 0, len(*in.ReplaceItems))
		for _, it := range *in.ReplaceItems {
			items = append(items, orderdom.OrderItemSnapshot{
				ModelID:     strings.TrimSpace(it.ModelID),
				InventoryID: strings.TrimSpace(it.InventoryID),
				Qty:         it.Qty,
				Price:       it.Price,
			})
		}
		if err := o.ReplaceItems(items); err != nil {
			return orderdom.Order{}, err
		}
	}

	// ✅ 最後に New で再検証してから保存（avatarId を含む）
	checked, err := orderdom.New(
		strings.TrimSpace(o.ID),
		strings.TrimSpace(o.UserID),
		strings.TrimSpace(o.AvatarID),
		strings.TrimSpace(o.CartID),
		o.ShippingSnapshot,
		o.BillingSnapshot,
		o.Items,
		o.CreatedAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	return u.repo.Save(ctx, checked, nil)
}

func (u *OrderUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

// ------------------------------------------------------------
// ID generation
// ------------------------------------------------------------

// newOrderID generates an order id when client didn't specify one.
// Firestore auto-idに依存せず、domain.Newの必須ID条件を満たすためにここで採番する。
func (u *OrderUsecase) newOrderID(t time.Time) string {
	return fmt.Sprintf("ord_%d", t.UTC().UnixNano())
}
