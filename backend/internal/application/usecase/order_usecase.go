// backend\internal\application\usecase\order_usecase.go
package usecase

import (
	"context"
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
// NOTE: kept for existing app/usecase call sites; repository adapters should translate as needed.
type OrderFilter struct {
	UserID *string

	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	UpdatedFrom    *time.Time
	UpdatedTo      *time.Time
	TransferedFrom *time.Time
	TransferedTo   *time.Time
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
	ID     string
	UserID string
	CartID string

	// ✅ Snapshot (required)
	ShippingSnapshot orderdom.ShippingSnapshot
	BillingSnapshot  orderdom.BillingSnapshot // last4 + cardHolderName only

	ListID    string
	Items     []string // orderItem primary keys
	InvoiceID string
	PaymentID string

	TransferedDate *time.Time // optional

	CreatedAt *time.Time // optional; defaults to now
	UpdatedBy *string
}

func (u *OrderUsecase) Create(ctx context.Context, in CreateOrderInput) (orderdom.Order, error) {
	now := u.now().UTC()
	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}
	updatedAt := createdAt

	// normalize snapshots (trim)
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

	o, err := orderdom.New(
		strings.TrimSpace(in.ID),
		strings.TrimSpace(in.UserID),
		strings.TrimSpace(in.CartID),
		ship,
		bill,
		strings.TrimSpace(in.ListID),
		in.Items,
		strings.TrimSpace(in.InvoiceID),
		strings.TrimSpace(in.PaymentID),
		in.TransferedDate,
		createdAt,
		updatedAt,
		in.UpdatedBy,
	)
	if err != nil {
		return orderdom.Order{}, err
	}
	return u.repo.Create(ctx, o)
}

type UpdateOrderInput struct {
	ID string

	UserID *string
	CartID *string

	// ✅ Snapshot updates
	ShippingSnapshot *orderdom.ShippingSnapshot
	BillingSnapshot  *orderdom.BillingSnapshot

	ListID         *string
	InvoiceID      *string
	PaymentID      *string
	TransferedDate *time.Time
	UpdatedBy      *string

	// Items operations (mutually composable)
	ReplaceItems *[]string
	AddItem      *string
	RemoveItem   *string
}

func (u *OrderUsecase) Update(ctx context.Context, in UpdateOrderInput) (orderdom.Order, error) {
	o, err := u.repo.GetByID(ctx, strings.TrimSpace(in.ID))
	if err != nil {
		return orderdom.Order{}, err
	}

	now := u.now().UTC()

	// Simple field updates + Touch
	if in.UserID != nil {
		o.UserID = strings.TrimSpace(*in.UserID)
		if err := o.Touch(now); err != nil {
			return orderdom.Order{}, err
		}
	}
	if in.CartID != nil {
		o.CartID = strings.TrimSpace(*in.CartID)
		if err := o.Touch(now); err != nil {
			return orderdom.Order{}, err
		}
	}

	// ✅ snapshots
	if in.ShippingSnapshot != nil {
		s := orderdom.ShippingSnapshot{
			ZipCode: strings.TrimSpace(in.ShippingSnapshot.ZipCode),
			State:   strings.TrimSpace(in.ShippingSnapshot.State),
			City:    strings.TrimSpace(in.ShippingSnapshot.City),
			Street:  strings.TrimSpace(in.ShippingSnapshot.Street),
			Street2: strings.TrimSpace(in.ShippingSnapshot.Street2),
			Country: strings.TrimSpace(in.ShippingSnapshot.Country),
		}
		if err := o.UpdateShippingSnapshot(s, now); err != nil {
			return orderdom.Order{}, err
		}
	}
	if in.BillingSnapshot != nil {
		b := orderdom.BillingSnapshot{
			Last4:          strings.TrimSpace(in.BillingSnapshot.Last4),
			CardHolderName: strings.TrimSpace(in.BillingSnapshot.CardHolderName),
		}
		if err := o.UpdateBillingSnapshot(b, now); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.ListID != nil {
		o.ListID = strings.TrimSpace(*in.ListID)
		if err := o.Touch(now); err != nil {
			return orderdom.Order{}, err
		}
	}
	if in.InvoiceID != nil {
		if err := o.UpdateInvoice(strings.TrimSpace(*in.InvoiceID), now); err != nil {
			return orderdom.Order{}, err
		}
	}
	if in.PaymentID != nil {
		if err := o.UpdatePayment(strings.TrimSpace(*in.PaymentID), now); err != nil {
			return orderdom.Order{}, err
		}
	}
	if in.TransferedDate != nil {
		if err := o.SetTransfered(in.TransferedDate.UTC(), now); err != nil {
			return orderdom.Order{}, err
		}
	}
	if in.UpdatedBy != nil {
		v := strings.TrimSpace(*in.UpdatedBy)
		// nil/empty validation is handled by entity.validate() when saved (and/or upstream)
		o.UpdatedBy = &v
		if err := o.Touch(now); err != nil {
			return orderdom.Order{}, err
		}
	}

	// Items
	if in.ReplaceItems != nil {
		if err := o.ReplaceItems(*in.ReplaceItems, now); err != nil {
			return orderdom.Order{}, err
		}
	}
	if in.AddItem != nil {
		if err := o.AddItem(*in.AddItem, now); err != nil {
			return orderdom.Order{}, err
		}
	}
	if in.RemoveItem != nil {
		if err := o.RemoveItem(*in.RemoveItem, now); err != nil {
			return orderdom.Order{}, err
		}
	}

	return u.repo.Save(ctx, o, nil)
}

func (u *OrderUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
