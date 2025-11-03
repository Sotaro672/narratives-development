package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	paymentdom "narratives/internal/domain/payment"
)

type PaymentRepositoryPG struct {
	DB *sql.DB
}

func NewPaymentRepositoryPG(db *sql.DB) *PaymentRepositoryPG {
	return &PaymentRepositoryPG{DB: db}
}

// ========================
// RepositoryPort impl
// ========================

func (r *PaymentRepositoryPG) GetByID(ctx context.Context, id string) (*paymentdom.Payment, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, invoice_id, billing_address_id, amount, status, error_type,
  created_at, updated_at, deleted_at
FROM payments
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	p, err := scanPayment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, paymentdom.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepositoryPG) GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, invoice_id, billing_address_id, amount, status, error_type,
  created_at, updated_at, deleted_at
FROM payments
WHERE invoice_id = $1
ORDER BY created_at ASC, id ASC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(invoiceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []paymentdom.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PaymentRepositoryPG) List(ctx context.Context, filter paymentdom.Filter, sort paymentdom.Sort, page paymentdom.Page) (paymentdom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildPaymentWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildPaymentOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY created_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM payments "+whereSQL, args...).Scan(&total); err != nil {
		return paymentdom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id, invoice_id, billing_address_id, amount, status, error_type,
  created_at, updated_at, deleted_at
FROM payments
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return paymentdom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]paymentdom.Payment, 0, perPage)
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return paymentdom.PageResult{}, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return paymentdom.PageResult{}, err
	}

	return paymentdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *PaymentRepositoryPG) Count(ctx context.Context, filter paymentdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildPaymentWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM payments "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *PaymentRepositoryPG) Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO payments (
  id, invoice_id, billing_address_id, amount, status, error_type,
  created_at, updated_at, deleted_at
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4, $5,
  NOW(), NOW(), NULL
)
RETURNING
  id, invoice_id, billing_address_id, amount, status, error_type,
  created_at, updated_at, deleted_at
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.InvoiceID),
		strings.TrimSpace(in.BillingAddressID),
		in.Amount,
		strings.TrimSpace(string(in.Status)),
		dbcommon.ToDBText(in.ErrorType),
	)
	p, err := scanPayment(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, paymentdom.ErrConflict
		}
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepositoryPG) Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	setText := func(col string, p *string) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, strings.TrimSpace(*p))
			i++
		}
	}
	if patch.InvoiceID != nil {
		setText("invoice_id", patch.InvoiceID)
	}
	if patch.BillingAddressID != nil {
		setText("billing_address_id", patch.BillingAddressID)
	}
	if patch.Amount != nil {
		sets = append(sets, fmt.Sprintf("amount = $%d", i))
		args = append(args, *patch.Amount)
		i++
	}
	if patch.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", i))
		args = append(args, strings.TrimSpace(string(*patch.Status)))
		i++
	}
	if patch.ErrorType != nil {
		v := strings.TrimSpace(*patch.ErrorType)
		if v == "" {
			sets = append(sets, "error_type = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("error_type = $%d", i))
			args = append(args, v)
			i++
		}
	}

	// Touch updated_at
	sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
	args = append(args, time.Now().UTC())
	i++

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE payments
SET %s
WHERE id = $%d
RETURNING
  id, invoice_id, billing_address_id, amount, status, error_type,
  created_at, updated_at, deleted_at
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	p, err := scanPayment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, paymentdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, paymentdom.ErrConflict
		}
		return nil, err
	}
	return &p, nil
}

func (r *PaymentRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM payments WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return paymentdom.ErrNotFound
	}
	return nil
}

func (r *PaymentRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM payments`)
	return err
}

// ========================
// Helpers
// ========================

func scanPayment(s dbcommon.RowScanner) (paymentdom.Payment, error) {
	var (
		id, invoiceID, billingAddressID string
		amount                          int
		status                          string
		errorTypeNS                     sql.NullString
		createdAt, updatedAt            time.Time
		deletedAtNS                     sql.NullTime
	)
	if err := s.Scan(
		&id, &invoiceID, &billingAddressID, &amount, &status, &errorTypeNS,
		&createdAt, &updatedAt, &deletedAtNS,
	); err != nil {
		return paymentdom.Payment{}, err
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
	toTimePtr := func(nt sql.NullTime) *time.Time {
		if nt.Valid {
			t := nt.Time.UTC()
			return &t
		}
		return nil
	}

	return paymentdom.Payment{
		ID:               strings.TrimSpace(id),
		InvoiceID:        strings.TrimSpace(invoiceID),
		BillingAddressID: strings.TrimSpace(billingAddressID),
		Amount:           amount,
		Status:           paymentdom.PaymentStatus(strings.TrimSpace(status)),
		ErrorType:        toStrPtr(errorTypeNS),
		CreatedAt:        createdAt.UTC(),
		UpdatedAt:        updatedAt.UTC(),
		DeletedAt:        toTimePtr(deletedAtNS),
	}, nil
}

func buildPaymentWhere(f paymentdom.Filter) ([]string, []any) {
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
	addEq("invoice_id", f.InvoiceID)
	addEq("billing_address_id", f.BillingAddressID)

	if len(f.Statuses) > 0 {
		base := len(args)
		ph := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			args = append(args, strings.TrimSpace(string(s)))
			ph[i] = fmt.Sprintf("$%d", base+i+1)
		}
		where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(ph, ",")))
	}

	if v := strings.TrimSpace(f.ErrorType); v != "" {
		where = append(where, fmt.Sprintf("error_type = $%d", len(args)+1))
		args = append(args, v)
	}

	// Amount range
	if f.MinAmount != nil {
		where = append(where, fmt.Sprintf("amount >= $%d", len(args)+1))
		args = append(args, *f.MinAmount)
	}
	if f.MaxAmount != nil {
		where = append(where, fmt.Sprintf("amount <= $%d", len(args)+1))
		args = append(args, *f.MaxAmount)
	}

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
	if f.DeletedFrom != nil {
		where = append(where, fmt.Sprintf("(deleted_at IS NOT NULL AND deleted_at >= $%d)", len(args)+1))
		args = append(args, f.DeletedFrom.UTC())
	}
	if f.DeletedTo != nil {
		where = append(where, fmt.Sprintf("(deleted_at IS NOT NULL AND deleted_at < $%d)", len(args)+1))
		args = append(args, f.DeletedTo.UTC())
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			where = append(where, "deleted_at IS NOT NULL")
		} else {
			where = append(where, "deleted_at IS NULL")
		}
	}

	return where, args
}

func buildPaymentOrderBy(sort paymentdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	switch col {
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	case "amount":
		col = "amount"
	case "status":
		col = "status"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
