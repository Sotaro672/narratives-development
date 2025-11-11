// backend\internal\adapters\out\firestore\invoice_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	invdom "narratives/internal/domain/invoice"
)

// InvoiceRepositoryPG implements invoice.Repository using PostgreSQL.
type InvoiceRepositoryPG struct {
	DB *sql.DB
}

func NewInvoiceRepositoryPG(db *sql.DB) *InvoiceRepositoryPG {
	return &InvoiceRepositoryPG{DB: db}
}

// =======================
// Queries
// =======================

// usecase.InvoiceRepo requires GetByID; our primary key is order_id, so wrap GetByOrderID.
func (r *InvoiceRepositoryPG) GetByID(ctx context.Context, id string) (invdom.Invoice, error) {
	return r.GetByOrderID(ctx, id)
}

func (r *InvoiceRepositoryPG) GetByOrderID(ctx context.Context, orderID string) (invdom.Invoice, error) {
	const q = `
SELECT
  order_id, subtotal, discount_amount, tax_amount, shipping_cost, total_amount,
  currency, created_at, updated_at, billing_address_id
FROM invoices
WHERE order_id = $1
`
	row := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(orderID))
	inv, err := scanInvoice(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return invdom.Invoice{}, invdom.ErrNotFound
		}
		return invdom.Invoice{}, err
	}
	return inv, nil
}

func (r *InvoiceRepositoryPG) Exists(ctx context.Context, orderID string) (bool, error) {
	const q = `SELECT 1 FROM invoices WHERE order_id = $1`
	var one int
	err := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(orderID)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *InvoiceRepositoryPG) Count(ctx context.Context, filter invdom.Filter) (int, error) {
	where, args := buildInvoiceWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM invoices `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *InvoiceRepositoryPG) List(ctx context.Context, filter invdom.Filter, sort invdom.Sort, page invdom.Page) (invdom.PageResult[invdom.Invoice], error) {
	where, args := buildInvoiceWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildInvoiceOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY updated_at DESC, order_id DESC"
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	number := page.Number
	if number <= 0 {
		number = 1
	}
	offset := (number - 1) * perPage

	var total int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM invoices %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return invdom.PageResult[invdom.Invoice]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  order_id, subtotal, discount_amount, tax_amount, shipping_cost, total_amount,
  currency, created_at, updated_at, billing_address_id
FROM invoices
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return invdom.PageResult[invdom.Invoice]{}, err
	}
	defer rows.Close()

	var items []invdom.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return invdom.PageResult[invdom.Invoice]{}, err
		}
		items = append(items, inv)
	}
	if err := rows.Err(); err != nil {
		return invdom.PageResult[invdom.Invoice]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return invdom.PageResult[invdom.Invoice]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *InvoiceRepositoryPG) ListByCursor(ctx context.Context, filter invdom.Filter, _ invdom.Sort, cpage invdom.CursorPage) (invdom.CursorPageResult[invdom.Invoice], error) {
	where, args := buildInvoiceWhere(filter)

	// Cursor by order_id (string) ascending
	if after := strings.TrimSpace(cpage.After); after != "" {
		where = append(where, fmt.Sprintf("order_id > $%d", len(args)+1))
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
  order_id, subtotal, discount_amount, tax_amount, shipping_cost, total_amount,
  currency, created_at, updated_at, billing_address_id
FROM invoices
%s
ORDER BY order_id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return invdom.CursorPageResult[invdom.Invoice]{}, err
	}
	defer rows.Close()

	var items []invdom.Invoice
	var last string
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return invdom.CursorPageResult[invdom.Invoice]{}, err
		}
		items = append(items, inv)
		last = inv.OrderID
	}
	if err := rows.Err(); err != nil {
		return invdom.CursorPageResult[invdom.Invoice]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &last
	}

	return invdom.CursorPageResult[invdom.Invoice]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// =======================
// Mutations
// =======================

func (r *InvoiceRepositoryPG) Create(ctx context.Context, inv invdom.Invoice) (invdom.Invoice, error) {
	const q = `
INSERT INTO invoices (
  order_id, subtotal, discount_amount, tax_amount, shipping_cost, total_amount,
  currency, created_at, updated_at, billing_address_id
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10
)
RETURNING
  order_id, subtotal, discount_amount, tax_amount, shipping_cost, total_amount,
  currency, created_at, updated_at, billing_address_id
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(inv.OrderID),
		inv.Subtotal, inv.DiscountAmount, inv.TaxAmount, inv.ShippingCost, inv.TotalAmount,
		strings.TrimSpace(inv.Currency),
		inv.CreatedAt.UTC(), inv.UpdatedAt.UTC(),
		strings.TrimSpace(inv.BillingAddressID),
	)
	out, err := scanInvoice(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return invdom.Invoice{}, invdom.ErrConflict
		}
		return invdom.Invoice{}, err
	}
	return out, nil
}

func (r *InvoiceRepositoryPG) Update(ctx context.Context, orderID string, patch invdom.InvoicePatch) (invdom.Invoice, error) {
	sets := []string{}
	args := []any{}
	i := 1

	setInt := func(col string, p *int) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, *p)
			i++
		}
	}

	setStr := func(col string, p *string) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, strings.TrimSpace(*p))
			i++
		}
	}

	setInt("subtotal", patch.Subtotal)
	setInt("discount_amount", patch.DiscountAmount)
	setInt("tax_amount", patch.TaxAmount)
	setInt("shipping_cost", patch.ShippingCost)
	setInt("total_amount", patch.TotalAmount)

	setStr("currency", patch.Currency)
	setStr("billing_address_id", patch.BillingAddressID)

	// updated_at explicit or NOW()
	if patch.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, patch.UpdatedAt.UTC())
		i++
	} else if len(sets) > 0 {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().UTC())
		i++
	}

	if len(sets) == 0 {
		return r.GetByOrderID(ctx, orderID)
	}

	args = append(args, strings.TrimSpace(orderID))
	q := fmt.Sprintf(`
UPDATE invoices
SET %s
WHERE order_id = $%d
RETURNING
  order_id, subtotal, discount_amount, tax_amount, shipping_cost, total_amount,
  currency, created_at, updated_at, billing_address_id
`, strings.Join(sets, ", "), i)

	row := r.DB.QueryRowContext(ctx, q, args...)
	out, err := scanInvoice(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return invdom.Invoice{}, invdom.ErrNotFound
		}
		return invdom.Invoice{}, err
	}
	return out, nil
}

func (r *InvoiceRepositoryPG) Delete(ctx context.Context, orderID string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM invoices WHERE order_id = $1`, strings.TrimSpace(orderID))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return invdom.ErrNotFound
	}
	return nil
}

// Save must match usecase.InvoiceRepo: no extra options parameter.
func (r *InvoiceRepositoryPG) Save(ctx context.Context, inv invdom.Invoice) (invdom.Invoice, error) {
	const q = `
INSERT INTO invoices (
  order_id, subtotal, discount_amount, tax_amount, shipping_cost, total_amount,
  currency, created_at, updated_at, billing_address_id
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10
)
ON CONFLICT (order_id) DO UPDATE SET
  subtotal          = EXCLUDED.subtotal,
  discount_amount   = EXCLUDED.discount_amount,
  tax_amount        = EXCLUDED.tax_amount,
  shipping_cost     = EXCLUDED.shipping_cost,
  total_amount      = EXCLUDED.total_amount,
  currency          = EXCLUDED.currency,
  created_at        = LEAST(invoices.created_at, EXCLUDED.created_at),
  updated_at        = COALESCE(EXCLUDED.updated_at, NOW()),
  billing_address_id= EXCLUDED.billing_address_id
RETURNING
  order_id, subtotal, discount_amount, tax_amount, shipping_cost, total_amount,
  currency, created_at, updated_at, billing_address_id
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(inv.OrderID),
		inv.Subtotal, inv.DiscountAmount, inv.TaxAmount, inv.ShippingCost, inv.TotalAmount,
		strings.TrimSpace(inv.Currency),
		inv.CreatedAt.UTC(), inv.UpdatedAt.UTC(),
		strings.TrimSpace(inv.BillingAddressID),
	)
	out, err := scanInvoice(row)
	if err != nil {
		return invdom.Invoice{}, err
	}
	return out, nil
}

// =======================
// Order item level invoices
// =======================

func (r *InvoiceRepositoryPG) GetOrderItemInvoiceByOrderItemID(ctx context.Context, orderItemID string) (invdom.OrderItemInvoice, error) {
	const q = `
SELECT id, order_item_id, unit_price, total_price, created_at, updated_at
FROM order_item_invoices
WHERE order_item_id = $1
ORDER BY updated_at DESC
LIMIT 1
`
	row := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(orderItemID))
	oi, err := scanOrderItemInvoice(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return invdom.OrderItemInvoice{}, invdom.ErrNotFound
		}
		return invdom.OrderItemInvoice{}, err
	}
	return oi, nil
}

func (r *InvoiceRepositoryPG) ListOrderItemInvoicesByOrderItemIDs(ctx context.Context, orderItemIDs []string) ([]invdom.OrderItemInvoice, error) {
	ids := make([]string, 0, len(orderItemIDs))
	for _, id := range orderItemIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return []invdom.OrderItemInvoice{}, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i := range ids {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = ids[i]
	}

	q := fmt.Sprintf(`
SELECT id, order_item_id, unit_price, total_price, created_at, updated_at
FROM order_item_invoices
WHERE order_item_id IN (%s)
ORDER BY order_item_id ASC, updated_at DESC
`, strings.Join(ph, ","))

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []invdom.OrderItemInvoice
	for rows.Next() {
		oi, err := scanOrderItemInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, oi)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// =======================
// Helpers
// =======================

func scanInvoice(s dbcommon.RowScanner) (invdom.Invoice, error) {
	var (
		orderIDNS, currencyNS, billingAddrNS sql.NullString
		subtotal, discount, tax, shipping    int
		total                                int
		createdAt, updatedAt                 time.Time
	)
	if err := s.Scan(
		&orderIDNS, &subtotal, &discount, &tax, &shipping, &total,
		&currencyNS, &createdAt, &updatedAt, &billingAddrNS,
	); err != nil {
		return invdom.Invoice{}, err
	}
	return invdom.Invoice{
		OrderID:           strings.TrimSpace(orderIDNS.String),
		OrderItemInvoices: nil, // not joined here
		Subtotal:          subtotal,
		DiscountAmount:    discount,
		TaxAmount:         tax,
		ShippingCost:      shipping,
		TotalAmount:       total,
		Currency:          strings.TrimSpace(currencyNS.String),
		CreatedAt:         createdAt.UTC(),
		UpdatedAt:         updatedAt.UTC(),
		BillingAddressID:  strings.TrimSpace(billingAddrNS.String),
	}, nil
}

func scanOrderItemInvoice(s dbcommon.RowScanner) (invdom.OrderItemInvoice, error) {
	var (
		idNS, orderItemIDNS  sql.NullString
		unitPrice, total     int
		createdAt, updatedAt time.Time
	)
	if err := s.Scan(
		&idNS, &orderItemIDNS, &unitPrice, &total, &createdAt, &updatedAt,
	); err != nil {
		return invdom.OrderItemInvoice{}, err
	}
	return invdom.OrderItemInvoice{
		ID:          strings.TrimSpace(idNS.String),
		OrderItemID: strings.TrimSpace(orderItemIDNS.String),
		UnitPrice:   unitPrice,
		TotalPrice:  total,
		CreatedAt:   createdAt.UTC(),
		UpdatedAt:   updatedAt.UTC(),
	}, nil
}

func buildInvoiceWhere(f invdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// Free text search
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(where, fmt.Sprintf("(order_id ILIKE $%d OR currency ILIKE $%d)", len(args)+1, len(args)+1))
		args = append(args, "%"+sq+"%")
	}

	// OrderID equals
	if f.OrderID != nil && strings.TrimSpace(*f.OrderID) != "" {
		where = append(where, fmt.Sprintf("order_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.OrderID))
	}
	// OrderIDs IN (...)
	if len(f.OrderIDs) > 0 {
		ph := []string{}
		for _, v := range f.OrderIDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "order_id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// Currency exact
	if f.Currency != nil && strings.TrimSpace(*f.Currency) != "" {
		where = append(where, fmt.Sprintf("currency = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.Currency))
	}

	// Amount ranges
	addRange := func(col string, min, max *int) {
		if min != nil {
			where = append(where, fmt.Sprintf("%s >= $%d", col, len(args)+1))
			args = append(args, *min)
		}
		if max != nil {
			where = append(where, fmt.Sprintf("%s <= $%d", col, len(args)+1))
			args = append(args, *max)
		}
	}
	addRange("subtotal", f.MinSubtotal, f.MaxSubtotal)
	addRange("discount_amount", f.MinDiscountAmount, f.MaxDiscountAmount)
	addRange("tax_amount", f.MinTaxAmount, f.MaxTaxAmount)
	addRange("shipping_cost", f.MinShippingCost, f.MaxShippingCost)
	addRange("total_amount", f.MinTotalAmount, f.MaxTotalAmount)

	// Date ranges
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

	return where, args
}

func buildInvoiceOrderBy(sort invdom.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "orderid", "order_id":
		col = "order_id"
	case "subtotal":
		col = "subtotal"
	case "discountamount", "discount_amount":
		col = "discount_amount"
	case "taxamount", "tax_amount":
		col = "tax_amount"
	case "shippingcost", "shipping_cost":
		col = "shipping_cost"
	case "totalamount", "total_amount":
		col = "total_amount"
	case "currency":
		col = "currency"
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	default:
		return ""
	}
	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}
