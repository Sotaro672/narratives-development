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
type OrderFilter struct {
    UserID             *string
    Status             *orderdom.LegacyOrderStatus
    CreatedFrom        *time.Time
    CreatedTo          *time.Time
    UpdatedFrom        *time.Time
    UpdatedTo          *time.Time
    TransfferedFrom    *time.Time
    TransfferedTo      *time.Time
    HasTransfferedDate *bool
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
    ID                string
    OrderNumber       string
    Status            orderdom.LegacyOrderStatus
    UserID            string
    ShippingAddressID string
    BillingAddressID  string
    ListID            string
    Items             []string // orderItem primary keys
    InvoiceID         string
    PaymentID         string
    FulfillmentID     string
    TrackingID        *string
    TransfferedDate   *time.Time
    CreatedAt         *time.Time // optional; defaults to now
    UpdatedBy         *string
    DeletedAt         *time.Time
    DeletedBy         *string
}

func (u *OrderUsecase) Create(ctx context.Context, in CreateOrderInput) (orderdom.Order, error) {
    now := u.now().UTC()
    createdAt := now
    if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
        createdAt = in.CreatedAt.UTC()
    }
    // UpdatedAt must be set; initially equal to createdAt
    updatedAt := createdAt

    o, err := orderdom.New(
        strings.TrimSpace(in.ID),
        strings.TrimSpace(in.OrderNumber),
        in.Status,
        strings.TrimSpace(in.UserID),
        strings.TrimSpace(in.ShippingAddressID),
        strings.TrimSpace(in.BillingAddressID),
        strings.TrimSpace(in.ListID),
        in.Items,
        strings.TrimSpace(in.InvoiceID),
        strings.TrimSpace(in.PaymentID),
        strings.TrimSpace(in.FulfillmentID),
        in.TrackingID,
        in.TransfferedDate,
        createdAt,
        updatedAt,
        in.UpdatedBy,
        in.DeletedAt,
        in.DeletedBy,
    )
    if err != nil {
        return orderdom.Order{}, err
    }
    return u.repo.Create(ctx, o)
}

type UpdateOrderInput struct {
    ID string

    OrderNumber       *string
    Status            *orderdom.LegacyOrderStatus
    UserID            *string
    ShippingAddressID *string
    BillingAddressID  *string
    ListID            *string
    TrackingID        *string
    TransfferedDate   *time.Time // if non-nil, set and mark transferred
    UpdatedBy         *string

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

    // Field updates via entity mutators (ensures UpdatedAt coherence)
    if in.OrderNumber != nil {
        o.OrderNumber = strings.TrimSpace(*in.OrderNumber)
        if err := o.Touch(now); err != nil {
            return orderdom.Order{}, err
        }
    }
    if in.Status != nil {
        if err := o.SetLegacyStatus(*in.Status, now); err != nil {
            return orderdom.Order{}, err
        }
    }
    if in.UserID != nil {
        o.UserID = strings.TrimSpace(*in.UserID)
        if err := o.Touch(now); err != nil {
            return orderdom.Order{}, err
        }
    }
    if in.ShippingAddressID != nil {
        if err := o.UpdateShippingAddress(strings.TrimSpace(*in.ShippingAddressID), now); err != nil {
            return orderdom.Order{}, err
        }
    }
    if in.BillingAddressID != nil {
        if err := o.UpdateBillingAddress(strings.TrimSpace(*in.BillingAddressID), now); err != nil {
            return orderdom.Order{}, err
        }
    }
    if in.ListID != nil {
        o.ListID = strings.TrimSpace(*in.ListID)
        if err := o.Touch(now); err != nil {
            return orderdom.Order{}, err
        }
    }
    if in.TrackingID != nil {
        if err := o.SetTracking(in.TrackingID, now); err != nil {
            return orderdom.Order{}, err
        }
    }
    if in.TransfferedDate != nil {
        if err := o.SetTransffered(in.TransfferedDate.UTC(), now); err != nil {
            return orderdom.Order{}, err
        }
    }
    if in.UpdatedBy != nil {
        v := strings.TrimSpace(*in.UpdatedBy)
        if v == "" {
            // Let entity validation handle invalid UpdatedBy when set
            o.UpdatedBy = &v
        } else {
            o.UpdatedBy = &v
        }
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