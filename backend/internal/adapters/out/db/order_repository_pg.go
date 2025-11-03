package db

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "time"

    dbcommon "narratives/internal/adapters/out/db/common"
    orderdom "narratives/internal/domain/order"
    uc "narratives/internal/application/usecase" // 追加
)

// PostgreSQL implementation of order.Repository
type OrderRepositoryPG struct {
    DB *sql.DB
}

func NewOrderRepositoryPG(db *sql.DB) *OrderRepositoryPG {
    return &OrderRepositoryPG{DB: db}
}

// ========================
// RepositoryPort impl
// ========================

func (r *OrderRepositoryPG) GetByID(ctx context.Context, id string) (orderdom.Order, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    const q = `
SELECT
  id, order_number, status, user_id, shipping_address_id, billing_address_id, list_id,
  items, invoice_id, payment_id, fulfillment_id, tracking_id, transffered_date,
  created_at, updated_at, updated_by, deleted_at, deleted_by
FROM orders
WHERE id = $1`
    row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
    o, err := scanOrder(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return orderdom.Order{}, orderdom.ErrNotFound
        }
        return orderdom.Order{}, err
    }
    return o, nil
}

func (r *OrderRepositoryPG) List(ctx context.Context, filter orderdom.Filter, sort orderdom.Sort, page orderdom.Page) (orderdom.PageResult, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    where, args := buildOrderWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    orderBy := buildOrderOrderBy(sort)
    if orderBy == "" {
        orderBy = "ORDER BY created_at DESC, id DESC"
    }

    pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

    // Count
    var total int
    if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders "+whereSQL, args...).Scan(&total); err != nil {
        return orderdom.PageResult{}, err
    }

    // Data
    q := fmt.Sprintf(`
SELECT
  id, order_number, status, user_id, shipping_address_id, billing_address_id, list_id,
  items, invoice_id, payment_id, fulfillment_id, tracking_id, transffered_date,
  created_at, updated_at, updated_by, deleted_at, deleted_by
FROM orders
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

    args = append(args, perPage, offset)

    rows, err := run.QueryContext(ctx, q, args...)
    if err != nil {
        return orderdom.PageResult{}, err
    }
    defer rows.Close()

    items := make([]orderdom.Order, 0, perPage)
    for rows.Next() {
        o, err := scanOrder(rows)
        if err != nil {
            return orderdom.PageResult{}, err
        }
        items = append(items, o)
    }
    if err := rows.Err(); err != nil {
        return orderdom.PageResult{}, err
    }

    return orderdom.PageResult{
        Items:      items,
        TotalCount: total,
        TotalPages: dbcommon.ComputeTotalPages(total, perPage),
        Page:       pageNum,
        PerPage:    perPage,
    }, nil
}

func (r *OrderRepositoryPG) ListByCursor(ctx context.Context, filter orderdom.Filter, sort orderdom.Sort, cpage orderdom.CursorPage) (orderdom.CursorPageResult, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    where, args := buildOrderWhere(filter)
    if after := strings.TrimSpace(cpage.After); after != "" {
        where = append(where, fmt.Sprintf("id > $%d", len(args)+1))
        args = append(args, after)
    }
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    limit := cpage.Limit
    if limit <= 0 || limit > 200 {
        limit = 50
    }

    q := fmt.Sprintf(`
SELECT
  id, order_number, status, user_id, shipping_address_id, billing_address_id, list_id,
  items, invoice_id, payment_id, fulfillment_id, tracking_id, transffered_date,
  created_at, updated_at, updated_by, deleted_at, deleted_by
FROM orders
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

    args = append(args, limit+1)

    rows, err := run.QueryContext(ctx, q, args...)
    if err != nil {
        return orderdom.CursorPageResult{}, err
    }
    defer rows.Close()

    var items []orderdom.Order
    var lastID string
    for rows.Next() {
        o, err := scanOrder(rows)
        if err != nil {
            return orderdom.CursorPageResult{}, err
        }
        items = append(items, o)
        lastID = o.ID
    }
    if err := rows.Err(); err != nil {
        return orderdom.CursorPageResult{}, err
    }

    var next *string
    if len(items) > limit {
        items = items[:limit]
        next = &lastID
    }

    return orderdom.CursorPageResult{
        Items:      items,
        NextCursor: next,
        Limit:      limit,
    }, nil
}

// 既存の Count(ctx context.Context, filter orderdom.Filter) をこの実装に置き換え
func (r *OrderRepositoryPG) Count(ctx context.Context, _ uc.OrderFilter) (int, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    var total int
    if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&total); err != nil {
        return 0, err
    }
    return total, nil
}

func (r *OrderRepositoryPG) Create(ctx context.Context, o orderdom.Order) (orderdom.Order, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    itemsJSON, err := json.Marshal(o.Items)
    if err != nil {
        return orderdom.Order{}, err
    }

    const q = `
INSERT INTO orders (
  id, order_number, status, user_id, shipping_address_id, billing_address_id, list_id,
  items, invoice_id, payment_id, fulfillment_id, tracking_id, transffered_date,
  created_at, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8::jsonb, $9, $10, $11, $12, $13,
  $14, $15, $16, $17, $18
)
RETURNING
  id, order_number, status, user_id, shipping_address_id, billing_address_id, list_id,
  items, invoice_id, payment_id, fulfillment_id, tracking_id, transffered_date,
  created_at, updated_at, updated_by, deleted_at, deleted_by
`
    row := run.QueryRowContext(ctx, q,
        strings.TrimSpace(o.ID),
        strings.TrimSpace(o.OrderNumber),
        strings.TrimSpace(string(o.Status)),
        strings.TrimSpace(o.UserID),
        strings.TrimSpace(o.ShippingAddressID),
        strings.TrimSpace(o.BillingAddressID),
        strings.TrimSpace(o.ListID),
        string(itemsJSON),
        strings.TrimSpace(o.InvoiceID),
        strings.TrimSpace(o.PaymentID),
        strings.TrimSpace(o.FulfillmentID),
        dbcommon.ToDBText(o.TrackingID),
        dbcommon.ToDBTime(o.TransfferedDate),
        o.CreatedAt.UTC(),
        o.UpdatedAt.UTC(),
        dbcommon.ToDBText(o.UpdatedBy),
        dbcommon.ToDBTime(o.DeletedAt),
        dbcommon.ToDBText(o.DeletedBy),
    )
    out, err := scanOrder(row)
    if err != nil {
        if dbcommon.IsUniqueViolation(err) {
            return orderdom.Order{}, orderdom.ErrConflict
        }
        return orderdom.Order{}, err
    }
    return out, nil
}

func (r *OrderRepositoryPG) Save(ctx context.Context, o orderdom.Order, _ *orderdom.SaveOptions) (orderdom.Order, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    itemsJSON, err := json.Marshal(o.Items)
    if err != nil {
        return orderdom.Order{}, err
    }

    const q = `
INSERT INTO orders (
  id, order_number, status, user_id, shipping_address_id, billing_address_id, list_id,
  items, invoice_id, payment_id, fulfillment_id, tracking_id, transffered_date,
  created_at, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8::jsonb, $9, $10, $11, $12, $13,
  $14, $15, $16, $17, $18
)
ON CONFLICT (id) DO UPDATE SET
  order_number       = EXCLUDED.order_number,
  status             = EXCLUDED.status,
  user_id            = EXCLUDED.user_id,
  shipping_address_id= EXCLUDED.shipping_address_id,
  billing_address_id = EXCLUDED.billing_address_id,
  list_id            = EXCLUDED.list_id,
  items              = EXCLUDED.items,
  invoice_id         = EXCLUDED.invoice_id,
  payment_id         = EXCLUDED.payment_id,
  fulfillment_id     = EXCLUDED.fulfillment_id,
  tracking_id        = EXCLUDED.tracking_id,
  transffered_date   = EXCLUDED.transffered_date,
  created_at         = EXCLUDED.created_at,
  updated_at         = EXCLUDED.updated_at,
  updated_by         = EXCLUDED.updated_by,
  deleted_at         = EXCLUDED.deleted_at,
  deleted_by         = EXCLUDED.deleted_by
RETURNING
  id, order_number, status, user_id, shipping_address_id, billing_address_id, list_id,
  items, invoice_id, payment_id, fulfillment_id, tracking_id, transffered_date,
  created_at, updated_at, updated_by, deleted_at, deleted_by
`
    row := run.QueryRowContext(ctx, q,
        strings.TrimSpace(o.ID),
        strings.TrimSpace(o.OrderNumber),
        strings.TrimSpace(string(o.Status)),
        strings.TrimSpace(o.UserID),
        strings.TrimSpace(o.ShippingAddressID),
        strings.TrimSpace(o.BillingAddressID),
        strings.TrimSpace(o.ListID),
        string(itemsJSON),
        strings.TrimSpace(o.InvoiceID),
        strings.TrimSpace(o.PaymentID),
        strings.TrimSpace(o.FulfillmentID),
        dbcommon.ToDBText(o.TrackingID),
        dbcommon.ToDBTime(o.TransfferedDate),
        o.CreatedAt.UTC(),
        o.UpdatedAt.UTC(),
        dbcommon.ToDBText(o.UpdatedBy),
        dbcommon.ToDBTime(o.DeletedAt),
        dbcommon.ToDBText(o.DeletedBy),
    )
    out, err := scanOrder(row)
    if err != nil {
        return orderdom.Order{}, err
    }
    return out, nil
}

func (r *OrderRepositoryPG) Delete(ctx context.Context, id string) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    res, err := run.ExecContext(ctx, `DELETE FROM orders WHERE id = $1`, strings.TrimSpace(id))
    if err != nil {
        return err
    }
    aff, _ := res.RowsAffected()
    if aff == 0 {
        return orderdom.ErrNotFound
    }
    return nil
}

func (r *OrderRepositoryPG) Reset(ctx context.Context) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    _, err := run.ExecContext(ctx, `DELETE FROM orders`)
    return err
}

// ========================
// Helpers
// ========================

func scanOrder(s dbcommon.RowScanner) (orderdom.Order, error) {
    var (
        id, orderNumber, status                         string
        userID, shippingAddressID, billingAddressID     string
        listID, invoiceID, paymentID, fulfillmentID     string
        itemsRaw                                        []byte
        trackingIDNS, updatedByNS, deletedByNS          sql.NullString
        transfferedDateNS, deletedAtNS                  sql.NullTime
        createdAt, updatedAt                            time.Time
    )
    if err := s.Scan(
        &id, &orderNumber, &status, &userID, &shippingAddressID, &billingAddressID, &listID,
        &itemsRaw, &invoiceID, &paymentID, &fulfillmentID, &trackingIDNS, &transfferedDateNS,
        &createdAt, &updatedAt, &updatedByNS, &deletedAtNS, &deletedByNS,
    ); err != nil {
        return orderdom.Order{}, err
    }

    var items []string
    if len(itemsRaw) > 0 {
        if err := json.Unmarshal(itemsRaw, &items); err != nil {
            return orderdom.Order{}, err
        }
    }

    toTimePtr := func(nt sql.NullTime) *time.Time {
        if nt.Valid {
            t := nt.Time.UTC()
            return &t
        }
        return nil
    }
    toStrPtr := func(ns sql.NullString) *string {
        if ns.Valid {
            v := strings.TrimSpace(ns.String)
            if v == "" {
                return nil
            }
            return &v
        }
        return nil
    }

    return orderdom.Order{
        ID:                strings.TrimSpace(id),
        OrderNumber:       strings.TrimSpace(orderNumber),
        Status:            orderdom.LegacyOrderStatus(strings.TrimSpace(status)),
        UserID:            strings.TrimSpace(userID),
        ShippingAddressID: strings.TrimSpace(shippingAddressID),
        BillingAddressID:  strings.TrimSpace(billingAddressID),
        ListID:            strings.TrimSpace(listID),
        Items:             items,
        InvoiceID:         strings.TrimSpace(invoiceID),
        PaymentID:         strings.TrimSpace(paymentID),
        FulfillmentID:     strings.TrimSpace(fulfillmentID),
        TrackingID:        toStrPtr(trackingIDNS),
        TransfferedDate:   toTimePtr(transfferedDateNS),
        CreatedAt:         createdAt.UTC(),
        UpdatedAt:         updatedAt.UTC(),
        UpdatedBy:         toStrPtr(updatedByNS),
        DeletedAt:         toTimePtr(deletedAtNS),
        DeletedBy:         toStrPtr(deletedByNS),
    }, nil
}

func buildOrderWhere(f orderdom.Filter) ([]string, []any) {
    where := []string{}
    args := []any{}

    addEq := func(col, v string) {
        v = strings.TrimSpace(v)
        if v != "" {
            where = append(where, fmt.Sprintf("%s = $%d", col, len(args)+1))
            args = append(args, v)
        }
    }
    addPtrText := func(col string, p *string) {
        if p != nil {
            v := strings.TrimSpace(*p)
            if v != "" {
                where = append(where, fmt.Sprintf("%s = $%d", col, len(args)+1))
                args = append(args, v)
            }
        }
    }

    if v := strings.TrimSpace(f.ID); v != "" {
        addEq("id", v)
    }
    if v := strings.TrimSpace(f.OrderNumberLike); v != "" {
        where = append(where, fmt.Sprintf("order_number ILIKE $%d", len(args)+1))
        args = append(args, "%"+v+"%")
    }
    if len(f.Statuses) > 0 {
        base := len(args)
        ph := make([]string, len(f.Statuses))
        for i, s := range f.Statuses {
            args = append(args, strings.TrimSpace(string(s)))
            ph[i] = fmt.Sprintf("$%d", base+i+1)
        }
        where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(ph, ",")))
    }
    addEq("user_id", f.UserID)
    addEq("list_id", f.ListID)
    addEq("invoice_id", f.InvoiceID)
    addEq("payment_id", f.PaymentID)
    addEq("fulfillment_id", f.FulfillmentID)
    addPtrText("tracking_id", f.TrackingID)

    // Time ranges
    if f.CreatedFrom != nil {
        where = append(where, fmt.Sprintf("created_at >= $%d", len(args)+1))
        args = append(args, f.CreatedFrom.UTC())
    }
    if f.CreatedTo != nil {
        where = append(where, fmt.Sprintf("created_at < $%d", len(args)+1))
        args = append(args, f.CreatedTo.UTC())
    }
    if f.UpdatedFrom != nil {
        where = append(where, fmt.Sprintf("updated_at >= $%d", len(args)+1))
        args = append(args, f.UpdatedFrom.UTC())
    }
    if f.UpdatedTo != nil {
        where = append(where, fmt.Sprintf("updated_at < $%d", len(args)+1))
        args = append(args, f.UpdatedTo.UTC())
    }
    if f.TransfferedFrom != nil {
        where = append(where, fmt.Sprintf("(transffered_date IS NOT NULL AND transffered_date >= $%d)", len(args)+1))
        args = append(args, f.TransfferedFrom.UTC())
    }
    if f.TransfferedTo != nil {
        where = append(where, fmt.Sprintf("(transffered_date IS NOT NULL AND transffered_date < $%d)", len(args)+1))
        args = append(args, f.TransfferedTo.UTC())
    }

    // Deleted filter
    if f.Deleted != nil {
        if *f.Deleted {
            where = append(where, "deleted_at IS NOT NULL")
        } else {
            where = append(where, "deleted_at IS NULL")
        }
    }

    return where, args
}

func buildOrderOrderBy(sort orderdom.Sort) string {
    col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
    switch col {
    case "createdat", "created_at":
        col = "created_at"
    case "updatedat", "updated_at":
        col = "updated_at"
    case "ordernumber", "order_number":
        col = "order_number"
    case "transffereddate", "transffered_date":
        col = "transffered_date"
    default:
        return ""
    }
    dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
    if dir != "ASC" && dir != "DESC" {
        dir = "DESC"
    }
    return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}