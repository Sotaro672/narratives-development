// backend/internal/adapters/out/db/fulfillment_repository_pg.go
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	fdom "narratives/internal/domain/fulfillment"
)

type txKey struct{}

// FulfillmentRepositoryPG implements fulfillment.RepositoryPort using PostgreSQL.
type FulfillmentRepositoryPG struct {
	DB *sql.DB
}

func NewFulfillmentRepositoryPG(db *sql.DB) *FulfillmentRepositoryPG {
	return &FulfillmentRepositoryPG{DB: db}
}

// sqlRunner is the common interface satisfied by *sql.DB and *sql.Tx.
type sqlRunner interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (r *FulfillmentRepositoryPG) runner(ctx context.Context) sqlRunner {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return r.DB
}

// =======================
// Queries
// =======================

func (r *FulfillmentRepositoryPG) GetByID(ctx context.Context, id string) (*fdom.Fulfillment, error) {
	const q = `
SELECT id, order_id, payment_id, status, created_at, updated_at
FROM fulfillments
WHERE id = $1
`
	row := r.runner(ctx).QueryRowContext(ctx, q, id)
	f, err := scanFulfillment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fdom.ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *FulfillmentRepositoryPG) GetByOrderID(ctx context.Context, orderID string) ([]fdom.Fulfillment, error) {
	const q = `
SELECT id, order_id, payment_id, status, created_at, updated_at
FROM fulfillments
WHERE order_id = $1
ORDER BY created_at ASC, id ASC
`
	rows, err := r.runner(ctx).QueryContext(ctx, q, strings.TrimSpace(orderID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []fdom.Fulfillment
	for rows.Next() {
		f, err := scanFulfillment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *FulfillmentRepositoryPG) GetLatestByOrderID(ctx context.Context, orderID string) (*fdom.Fulfillment, error) {
	const q = `
SELECT id, order_id, payment_id, status, created_at, updated_at
FROM fulfillments
WHERE order_id = $1
ORDER BY updated_at DESC, created_at DESC, id DESC
LIMIT 1
`
	row := r.runner(ctx).QueryRowContext(ctx, q, strings.TrimSpace(orderID))
	f, err := scanFulfillment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fdom.ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *FulfillmentRepositoryPG) List(ctx context.Context, filter fdom.Filter, sort fdom.Sort, page fdom.Page) (fdom.PageResult, error) {
	where, args := buildFulfillmentWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildFulfillmentOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY created_at DESC, id DESC"
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM fulfillments %s", whereSQL)
	if err := r.runner(ctx).QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return fdom.PageResult{}, err
	}

	q := fmt.Sprintf(`
SELECT id, order_id, payment_id, status, created_at, updated_at
FROM fulfillments
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)
	args = append(args, perPage, offset)

	rows, err := r.runner(ctx).QueryContext(ctx, q, args...)
	if err != nil {
		return fdom.PageResult{}, err
	}
	defer rows.Close()

	var items []fdom.Fulfillment
	for rows.Next() {
		f, err := scanFulfillment(rows)
		if err != nil {
			return fdom.PageResult{}, err
		}
		items = append(items, f)
	}
	if err := rows.Err(); err != nil {
		return fdom.PageResult{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return fdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *FulfillmentRepositoryPG) Count(ctx context.Context, filter fdom.Filter) (int, error) {
	where, args := buildFulfillmentWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.runner(ctx).QueryRowContext(ctx, "SELECT COUNT(*) FROM fulfillments "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// =======================
// Mutations
// =======================

func (r *FulfillmentRepositoryPG) Create(ctx context.Context, in fdom.CreateFulfillmentInput) (*fdom.Fulfillment, error) {
	createdAt := time.Now().UTC()
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}
	updatedAt := createdAt
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	const q = `
INSERT INTO fulfillments (
  id, order_id, payment_id, status, created_at, updated_at
) VALUES (
  gen_random_uuid(), $1, $2, $3, $4, $5
)
RETURNING id, order_id, payment_id, status, created_at, updated_at
`
	row := r.runner(ctx).QueryRowContext(ctx, q,
		strings.TrimSpace(in.OrderID),
		strings.TrimSpace(in.PaymentID),
		strings.TrimSpace(string(in.Status)),
		createdAt,
		updatedAt,
	)
	f, err := scanFulfillment(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, fdom.ErrConflict
		}
		return nil, err
	}
	return &f, nil
}

func (r *FulfillmentRepositoryPG) Update(ctx context.Context, id string, in fdom.UpdateFulfillmentInput) (*fdom.Fulfillment, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if in.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", i))
		args = append(args, strings.TrimSpace(string(*in.Status)))
		i++
	}
	// updated_at: explicit or NOW() if something changed
	if in.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, in.UpdatedAt.UTC())
		i++
	} else if len(sets) > 0 {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().UTC())
		i++
	}

	if len(sets) == 0 {
		// nothing to update; return current entity if exists
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(`
UPDATE fulfillments
SET %s
WHERE id = $%d
RETURNING id, order_id, payment_id, status, created_at, updated_at
`, strings.Join(sets, ", "), i)

	row := r.runner(ctx).QueryRowContext(ctx, q, args...)
	f, err := scanFulfillment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fdom.ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *FulfillmentRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.runner(ctx).ExecContext(ctx, `DELETE FROM fulfillments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return fdom.ErrNotFound
	}
	return nil
}

// WithTx runs fn within a transaction boundary.
func (r *FulfillmentRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ctxTx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(ctxTx); err != nil {
		return err
	}
	return tx.Commit()
}

// Reset clears the table (for maintenance/testing).
func (r *FulfillmentRepositoryPG) Reset(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, `TRUNCATE TABLE fulfillments RESTART IDENTITY`)
	return err
}

// =======================
// Helpers
// =======================

func scanFulfillment(s dbcommon.RowScanner) (fdom.Fulfillment, error) {
	var (
		idNS, orderIDNS, paymentIDNS, statusNS sql.NullString
		createdAt, updatedAt                   time.Time
	)
	if err := s.Scan(&idNS, &orderIDNS, &paymentIDNS, &statusNS, &createdAt, &updatedAt); err != nil {
		return fdom.Fulfillment{}, err
	}
	return fdom.Fulfillment{
		ID:        strings.TrimSpace(idNS.String),
		OrderID:   strings.TrimSpace(orderIDNS.String),
		PaymentID: strings.TrimSpace(paymentIDNS.String),
		Status:    fdom.FulfillmentStatus(strings.TrimSpace(statusNS.String)),
		CreatedAt: createdAt.UTC(),
		UpdatedAt: updatedAt.UTC(),
	}, nil
}

func buildFulfillmentWhere(f fdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// IDs
	if len(f.IDs) > 0 {
		ph := []string{}
		for _, v := range f.IDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "id IN ("+strings.Join(ph, ",")+")")
		}
	}
	// OrderIDs
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
	// PaymentIDs
	if len(f.PaymentIDs) > 0 {
		ph := []string{}
		for _, v := range f.PaymentIDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "payment_id IN ("+strings.Join(ph, ",")+")")
		}
	}
	// Statuses
	if len(f.Statuses) > 0 {
		ph := []string{}
		for _, st := range f.Statuses {
			v := strings.TrimSpace(string(st))
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "status IN ("+strings.Join(ph, ",")+")")
		}
	}
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

func buildFulfillmentOrderBy(sort fdom.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case strings.ToLower(string(fdom.SortByCreatedAt)):
		col = "created_at"
	case strings.ToLower(string(fdom.SortByUpdatedAt)):
		col = "updated_at"
	case strings.ToLower(string(fdom.SortByStatus)):
		col = "status"
	default:
		return ""
	}
	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}
