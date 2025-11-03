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

// ============================================================
// PaymentRepo (usecase層が要求するPort) の実装
// ============================================================

// GetByID implements PaymentRepo.GetByID.
// 返り値は値型 (paymentdom.Payment) に揃える。
func (r *PaymentRepositoryPG) GetByID(ctx context.Context, id string) (paymentdom.Payment, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
FROM payments
WHERE id = $1
`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))

	p, err := scanPayment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return paymentdom.Payment{}, paymentdom.ErrNotFound
		}
		return paymentdom.Payment{}, err
	}
	return p, nil
}

// Exists implements PaymentRepo.Exists.
func (r *PaymentRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `SELECT 1 FROM payments WHERE id = $1`
	var one int
	err := run.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create implements PaymentRepo.Create.
//
// usecase層は paymentdom.Payment を渡してくる想定。
// ここでは v.ID が空なら DB 側で gen_random_uuid() 生成、
// 入っていればそれを使います。
func (r *PaymentRepositoryPG) Create(ctx context.Context, v paymentdom.Payment) (paymentdom.Payment, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	hasID := strings.TrimSpace(v.ID) != ""

	// 作成時刻・更新時刻を整える
	nowUTC := time.Now().UTC()
	createdAt := v.CreatedAt
	if createdAt.IsZero() {
		createdAt = nowUTC
	}
	updatedAt := v.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = nowUTC
	}

	if hasID {
		// IDを呼び出し側が決めているパターン
		const q = `
INSERT INTO payments (
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9
)
RETURNING
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
`
		row := run.QueryRowContext(ctx, q,
			strings.TrimSpace(v.ID),
			strings.TrimSpace(v.InvoiceID),
			strings.TrimSpace(v.BillingAddressID),
			v.Amount,
			strings.TrimSpace(string(v.Status)),
			dbcommon.ToDBText(v.ErrorType),
			createdAt.UTC(),
			updatedAt.UTC(),
			dbcommon.ToDBTime(v.DeletedAt),
		)

		out, err := scanPayment(row)
		if err != nil {
			if dbcommon.IsUniqueViolation(err) {
				return paymentdom.Payment{}, paymentdom.ErrConflict
			}
			return paymentdom.Payment{}, err
		}
		return out, nil
	}

	// IDなしならDB側でUUID生成
	const qNoID = `
INSERT INTO payments (
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
) VALUES (
  gen_random_uuid()::text,
  $1,$2,$3,$4,$5,
  $6,$7,$8
)
RETURNING
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
`
	row := run.QueryRowContext(ctx, qNoID,
		strings.TrimSpace(v.InvoiceID),
		strings.TrimSpace(v.BillingAddressID),
		v.Amount,
		strings.TrimSpace(string(v.Status)),
		dbcommon.ToDBText(v.ErrorType),
		createdAt.UTC(),
		updatedAt.UTC(),
		dbcommon.ToDBTime(v.DeletedAt),
	)

	out, err := scanPayment(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return paymentdom.Payment{}, paymentdom.ErrConflict
		}
		return paymentdom.Payment{}, err
	}
	return out, nil
}

// Save implements PaymentRepo.Save.
//
// upsert 的な挙動: INSERT ... ON CONFLICT(id) DO UPDATE
// ・created_at / created_by 系の取り扱いは最低限にしている
// ・updated_at はここで必ず now() 相当を反映する
func (r *PaymentRepositoryPG) Save(ctx context.Context, v paymentdom.Payment) (paymentdom.Payment, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	nowUTC := time.Now().UTC()
	createdAt := v.CreatedAt
	if createdAt.IsZero() {
		createdAt = nowUTC
	}
	updatedAt := v.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = nowUTC
	}

	const q = `
INSERT INTO payments (
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9
)
ON CONFLICT (id) DO UPDATE SET
  invoice_id         = EXCLUDED.invoice_id,
  billing_address_id = EXCLUDED.billing_address_id,
  amount             = EXCLUDED.amount,
  status             = EXCLUDED.status,
  error_type         = EXCLUDED.error_type,
  -- created_at は過去最古を維持
  created_at         = LEAST(payments.created_at, EXCLUDED.created_at),
  -- updated_at は新しい方を採用
  updated_at         = GREATEST(payments.updated_at, EXCLUDED.updated_at),
  deleted_at         = EXCLUDED.deleted_at
RETURNING
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.InvoiceID),
		strings.TrimSpace(v.BillingAddressID),
		v.Amount,
		strings.TrimSpace(string(v.Status)),
		dbcommon.ToDBText(v.ErrorType),
		createdAt.UTC(),
		updatedAt.UTC(),
		dbcommon.ToDBTime(v.DeletedAt),
	)

	out, err := scanPayment(row)
	if err != nil {
		return paymentdom.Payment{}, err
	}
	return out, nil
}

// Delete implements PaymentRepo.Delete.
func (r *PaymentRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)

	res, err := run.ExecContext(ctx,
		`DELETE FROM payments WHERE id = $1`,
		strings.TrimSpace(id),
	)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return paymentdom.ErrNotFound
	}
	return nil
}

// ============================================================
// 追加機能（ユースケースPort外）: List/Count/Update等
// これらは使い勝手のために残してOK。インターフェース実装には影響しない。
// ============================================================

func (r *PaymentRepositoryPG) GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
FROM payments
WHERE invoice_id = $1
ORDER BY created_at ASC, id ASC
`
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

func (r *PaymentRepositoryPG) List(
	ctx context.Context,
	filter paymentdom.Filter,
	sort paymentdom.Sort,
	page paymentdom.Page,
) (paymentdom.PageResult, error) {

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

	// count
	var total int
	if err := run.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM payments "+whereSQL,
		args...,
	).Scan(&total); err != nil {
		return paymentdom.PageResult{}, err
	}

	// page data
	q := fmt.Sprintf(`
SELECT
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
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
	if err := run.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM payments "+whereSQL,
		args...,
	).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// Update は便利関数として残す（ユースケースの Port には含めていない）
func (r *PaymentRepositoryPG) Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (paymentdom.Payment, error) {
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

	// always bump updated_at
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
  id,
  invoice_id,
  billing_address_id,
  amount,
  status,
  error_type,
  created_at,
  updated_at,
  deleted_at
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	out, err := scanPayment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return paymentdom.Payment{}, paymentdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return paymentdom.Payment{}, paymentdom.ErrConflict
		}
		return paymentdom.Payment{}, err
	}
	return out, nil
}

// Reset はテスト用ユーティリティ
func (r *PaymentRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM payments`)
	return err
}

// ============================================================
// 共通ヘルパー
// ============================================================

func scanPayment(s dbcommon.RowScanner) (paymentdom.Payment, error) {
	var (
		id               string
		invoiceID        string
		billingAddressID string
		amount           int
		status           string
		errorTypeNS      sql.NullString
		createdAt        time.Time
		updatedAt        time.Time
		deletedAtNS      sql.NullTime
	)
	if err := s.Scan(
		&id,
		&invoiceID,
		&billingAddressID,
		&amount,
		&status,
		&errorTypeNS,
		&createdAt,
		&updatedAt,
		&deletedAtNS,
	); err != nil {
		return paymentdom.Payment{}, err
	}

	toStrPtr := func(ns sql.NullString) *string {
		if ns.Valid {
			v := strings.TrimSpace(ns.String)
			if v != "" {
				return &v
			}
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
		where = append(where,
			fmt.Sprintf("(deleted_at IS NOT NULL AND deleted_at >= $%d)", len(args)+1),
		)
		args = append(args, f.DeletedFrom.UTC())
	}
	if f.DeletedTo != nil {
		where = append(where,
			fmt.Sprintf("(deleted_at IS NOT NULL AND deleted_at < $%d)", len(args)+1),
		)
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
	// created_at DESC, id DESC のように複合で並べたいので id も合わせる
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
