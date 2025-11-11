// backend\internal\adapters\out\firestore\orderItem_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbcommon "narratives/internal/adapters/out/db/common"
	orderitemdom "narratives/internal/domain/orderItem"
)

type OrderItemRepositoryPG struct {
	DB *sql.DB
}

func NewOrderItemRepositoryPG(db *sql.DB) *OrderItemRepositoryPG {
	return &OrderItemRepositoryPG{DB: db}
}

// ========================
// RepositoryPort impl
// ========================

func (r *OrderItemRepositoryPG) GetByID(ctx context.Context, id string) (*orderitemdom.OrderItem, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT id, model_id, sale_id, inventory_id, quantity
FROM order_items
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	oi, err := scanOrderItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, orderitemdom.ErrNotFound
		}
		return nil, err
	}
	return &oi, nil
}

func (r *OrderItemRepositoryPG) List(ctx context.Context, filter orderitemdom.Filter, sort orderitemdom.Sort, page orderitemdom.Page) (orderitemdom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildOrderItemWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildOrderItemOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY id ASC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM order_items "+whereSQL, args...).Scan(&total); err != nil {
		return orderitemdom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT id, model_id, sale_id, inventory_id, quantity
FROM order_items
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return orderitemdom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]orderitemdom.OrderItem, 0, perPage)
	for rows.Next() {
		oi, err := scanOrderItem(rows)
		if err != nil {
			return orderitemdom.PageResult{}, err
		}
		items = append(items, oi)
	}
	if err := rows.Err(); err != nil {
		return orderitemdom.PageResult{}, err
	}

	return orderitemdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *OrderItemRepositoryPG) Count(ctx context.Context, filter orderitemdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildOrderItemWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM order_items "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *OrderItemRepositoryPG) Create(ctx context.Context, in orderitemdom.CreateOrderItemInput) (*orderitemdom.OrderItem, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO order_items (id, model_id, sale_id, inventory_id, quantity)
VALUES (gen_random_uuid()::text, $1, $2, $3, $4)
RETURNING id, model_id, sale_id, inventory_id, quantity
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.ModelID),
		strings.TrimSpace(in.SaleID),
		strings.TrimSpace(in.InventoryID),
		in.Quantity,
	)
	oi, err := scanOrderItem(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, orderitemdom.ErrConflict
		}
		return nil, err
	}
	return &oi, nil
}

func (r *OrderItemRepositoryPG) Update(ctx context.Context, id string, patch orderitemdom.UpdateOrderItemInput) (*orderitemdom.OrderItem, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	if patch.ModelID != nil {
		sets = append(sets, fmt.Sprintf("model_id = $%d", i))
		args = append(args, strings.TrimSpace(*patch.ModelID))
		i++
	}
	if patch.SaleID != nil {
		sets = append(sets, fmt.Sprintf("sale_id = $%d", i))
		args = append(args, strings.TrimSpace(*patch.SaleID))
		i++
	}
	if patch.InventoryID != nil {
		sets = append(sets, fmt.Sprintf("inventory_id = $%d", i))
		args = append(args, strings.TrimSpace(*patch.InventoryID))
		i++
	}
	if patch.Quantity != nil {
		sets = append(sets, fmt.Sprintf("quantity = $%d", i))
		args = append(args, *patch.Quantity)
		i++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE order_items
SET %s
WHERE id = $%d
RETURNING id, model_id, sale_id, inventory_id, quantity
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	oi, err := scanOrderItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, orderitemdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, orderitemdom.ErrConflict
		}
		return nil, err
	}
	return &oi, nil
}

func (r *OrderItemRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM order_items WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return orderitemdom.ErrNotFound
	}
	return nil
}

func (r *OrderItemRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM order_items`)
	return err
}

// ========================
// Helpers
// ========================

func scanOrderItem(s dbcommon.RowScanner) (orderitemdom.OrderItem, error) {
	var id, modelID, saleID, inventoryID string
	var quantity int
	if err := s.Scan(&id, &modelID, &saleID, &inventoryID, &quantity); err != nil {
		return orderitemdom.OrderItem{}, err
	}
	return orderitemdom.OrderItem{
		ID:          strings.TrimSpace(id),
		ModelID:     strings.TrimSpace(modelID),
		SaleID:      strings.TrimSpace(saleID),
		InventoryID: strings.TrimSpace(inventoryID),
		Quantity:    quantity,
	}, nil
}

func buildOrderItemWhere(f orderitemdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	addEq := func(col, v string) {
		v = strings.TrimSpace(v)
		if v != "" {
			where = append(where, fmt.Sprintf("%s = $%d", col, len(args)+1))
			args = append(args, v)
		}
	}

	addEq("id", f.ID)
	addEq("model_id", f.ModelID)
	addEq("sale_id", f.SaleID)
	addEq("inventory_id", f.InventoryID)

	if f.MinQuantity != nil {
		where = append(where, fmt.Sprintf("quantity >= $%d", len(args)+1))
		args = append(args, *f.MinQuantity)
	}
	if f.MaxQuantity != nil {
		where = append(where, fmt.Sprintf("quantity <= $%d", len(args)+1))
		args = append(args, *f.MaxQuantity)
	}

	return where, args
}

func buildOrderItemOrderBy(s orderitemdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "id":
		col = "id"
	case "quantity":
		col = "quantity"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
