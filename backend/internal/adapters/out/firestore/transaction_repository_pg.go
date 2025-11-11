// backend\internal\adapters\out\firestore\transaction_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	tr "narratives/internal/domain/transaction"
)

type TransactionRepositoryPG struct {
	DB *sql.DB
}

func NewTransactionRepositoryPG(db *sql.DB) *TransactionRepositoryPG {
	return &TransactionRepositoryPG{DB: db}
}

// ==========================
// RepositoryPort impl
// ==========================

func (r *TransactionRepositoryPG) GetAllTransactions(ctx context.Context) ([]*tr.Transaction, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, account_id, brand_name, type, amount, currency, from_account, to_account, timestamp, description
FROM transactions
ORDER BY timestamp DESC, id DESC`
	rows, err := run.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*tr.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, rows.Err()
}

func (r *TransactionRepositoryPG) GetTransactionByID(ctx context.Context, id string) (*tr.Transaction, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, account_id, brand_name, type, amount, currency, from_account, to_account, timestamp, description
FROM transactions
WHERE id = $1
LIMIT 1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	t, err := scanTransaction(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tr.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *TransactionRepositoryPG) GetTransactionsByBrand(ctx context.Context, brandName string) ([]*tr.Transaction, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, account_id, brand_name, type, amount, currency, from_account, to_account, timestamp, description
FROM transactions
WHERE brand_name = $1
ORDER BY timestamp DESC, id DESC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(brandName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*tr.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, rows.Err()
}

func (r *TransactionRepositoryPG) GetTransactionsByAccount(ctx context.Context, accountID string) ([]*tr.Transaction, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, account_id, brand_name, type, amount, currency, from_account, to_account, timestamp, description
FROM transactions
WHERE account_id = $1
ORDER BY timestamp DESC, id DESC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(accountID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*tr.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, rows.Err()
}

func (r *TransactionRepositoryPG) SearchTransactions(ctx context.Context, criteria tr.TransactionSearchCriteria) (txs []*tr.Transaction, total int, err error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTxWhere(criteria)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	// Count
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM transactions "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Sorting
	orderBy := buildTxOrderBy(criteria.Sort)
	if orderBy == "" {
		orderBy = "ORDER BY timestamp DESC, id DESC"
	}

	// Paging
	perPage := 50
	offset := 0
	if criteria.Pagination != nil {
		_, perPage, offset = dbcommon.NormalizePage(criteria.Pagination.Page, criteria.Pagination.PerPage, 50, 200)
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id, account_id, brand_name, type, amount, currency, from_account, to_account, timestamp, description
FROM transactions
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)
	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]*tr.Transaction, 0, perPage)
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, 0, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, total, rows.Err()
}

func (r *TransactionRepositoryPG) CreateTransaction(ctx context.Context, in tr.CreateTransactionInput) (*tr.Transaction, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
INSERT INTO transactions (
  id, account_id, brand_name, type, amount, currency, from_account, to_account, timestamp, description
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING
  id, account_id, brand_name, type, amount, currency, from_account, to_account, timestamp, description
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.AccountID),
		strings.TrimSpace(in.BrandName),
		strings.TrimSpace(string(in.Type)),
		in.Amount,
		strings.ToUpper(strings.TrimSpace(in.Currency)),
		strings.TrimSpace(in.FromAccount),
		strings.TrimSpace(in.ToAccount),
		in.Timestamp.UTC(),
		in.Description,
	)
	t, err := scanTransaction(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, tr.ErrConflict
		}
		return nil, err
	}
	return &t, nil
}

func (r *TransactionRepositoryPG) UpdateTransaction(ctx context.Context, id string, in tr.UpdateTransactionInput) (*tr.Transaction, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	setStr := func(col string, p *string) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, strings.TrimSpace(*p))
			i++
		}
	}
	setEnum := func(col string, p *tr.TransactionType) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, strings.TrimSpace(string(*p)))
			i++
		}
	}
	setInt := func(col string, p *int) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, *p)
			i++
		}
	}
	setTime := func(col string, p *time.Time) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, p.UTC())
			i++
		}
	}

	setStr("account_id", in.AccountID)
	setStr("brand_name", in.BrandName)
	setEnum("type", in.Type)
	setInt("amount", in.Amount)
	if in.Currency != nil {
		sets = append(sets, fmt.Sprintf("currency = $%d", i))
		args = append(args, strings.ToUpper(strings.TrimSpace(*in.Currency)))
		i++
	}
	setStr("from_account", in.FromAccount)
	setStr("to_account", in.ToAccount)
	setTime("timestamp", in.Timestamp)
	setStr("description", in.Description)

	if len(sets) == 0 {
		return r.GetTransactionByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE transactions
SET %s
WHERE id = $%d
RETURNING
  id, account_id, brand_name, type, amount, currency, from_account, to_account, timestamp, description
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	t, err := scanTransaction(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tr.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, tr.ErrConflict
		}
		return nil, err
	}
	return &t, nil
}

func (r *TransactionRepositoryPG) DeleteTransaction(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM transactions WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return tr.ErrNotFound
	}
	return nil
}

func (r *TransactionRepositoryPG) ResetTransactions(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM transactions`)
	return err
}

func (r *TransactionRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txCtx := dbcommon.CtxWithTx(ctx, tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ==========================
// Helpers
// ==========================

func scanTransaction(s dbcommon.RowScanner) (tr.Transaction, error) {
	var (
		id, accountID, brandName, typStr, currency, fromAcc, toAcc, desc string
		amount                                                           int64
		ts                                                               time.Time
	)
	if err := s.Scan(
		&id, &accountID, &brandName, &typStr, &amount, &currency, &fromAcc, &toAcc, &ts, &desc,
	); err != nil {
		return tr.Transaction{}, err
	}
	return tr.Transaction{
		ID:          strings.TrimSpace(id),
		AccountID:   strings.TrimSpace(accountID),
		BrandName:   strings.TrimSpace(brandName),
		Type:        tr.TransactionType(strings.TrimSpace(typStr)),
		Amount:      int(amount),
		Currency:    strings.ToUpper(strings.TrimSpace(currency)),
		FromAccount: strings.TrimSpace(fromAcc),
		ToAccount:   strings.TrimSpace(toAcc),
		Timestamp:   ts.UTC(),
		Description: desc,
	}, nil
}

func buildTxWhere(c tr.TransactionSearchCriteria) ([]string, []any) {
	where := []string{}
	args := []any{}

	addIn := func(col string, vals []string) {
		clean := make([]string, 0, len(vals))
		for _, v := range vals {
			if v = strings.TrimSpace(v); v != "" {
				clean = append(clean, v)
			}
		}
		if len(clean) == 0 {
			return
		}
		base := len(args)
		ph := make([]string, len(clean))
		for i, v := range clean {
			args = append(args, v)
			ph[i] = fmt.Sprintf("$%d", base+i+1)
		}
		where = append(where, fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ",")))
	}
	addInEnum := func(col string, vals []tr.TransactionType) {
		if len(vals) == 0 {
			return
		}
		base := len(args)
		ph := make([]string, len(vals))
		for i, v := range vals {
			args = append(args, strings.TrimSpace(string(v)))
			ph[i] = fmt.Sprintf("$%d", base+i+1)
		}
		where = append(where, fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ",")))
	}

	// Arrays
	addIn("account_id", c.Filters.AccountIDs)
	addIn("brand_name", c.Filters.Brands)
	addIn("currency", c.Filters.Currencies)
	addIn("from_account", c.Filters.FromAccounts)
	addIn("to_account", c.Filters.ToAccounts)
	addInEnum("type", c.Filters.Types)

	// Ranges
	if c.Filters.DateFrom != nil {
		where = append(where, fmt.Sprintf("timestamp >= $%d", len(args)+1))
		args = append(args, c.Filters.DateFrom.UTC())
	}
	if c.Filters.DateTo != nil {
		where = append(where, fmt.Sprintf("timestamp < $%d", len(args)+1))
		args = append(args, c.Filters.DateTo.UTC())
	}
	if c.Filters.AmountMin != nil {
		where = append(where, fmt.Sprintf("amount >= $%d", len(args)+1))
		args = append(args, *c.Filters.AmountMin)
	}
	if c.Filters.AmountMax != nil {
		where = append(where, fmt.Sprintf("amount <= $%d", len(args)+1))
		args = append(args, *c.Filters.AmountMax)
	}

	// Description like
	if v := strings.TrimSpace(c.Filters.DescriptionLike); v != "" {
		where = append(where, fmt.Sprintf("description ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}

	// SearchTerm across several columns
	if v := strings.TrimSpace(c.SearchTerm); v != "" {
		or := []string{
			fmt.Sprintf("brand_name ILIKE $%d", len(args)+1),
			fmt.Sprintf("currency ILIKE $%d", len(args)+2),
			fmt.Sprintf("from_account ILIKE $%d", len(args)+3),
			fmt.Sprintf("to_account ILIKE $%d", len(args)+4),
			fmt.Sprintf("description ILIKE $%d", len(args)+5),
		}
		for i := 0; i < 5; i++ {
			args = append(args, "%"+v+"%")
		}
		where = append(where, "("+strings.Join(or, " OR ")+")")
	}

	return where, args
}

func buildTxOrderBy(s tr.TransactionSort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "timestamp":
		col = "timestamp"
	case "amount":
		col = "amount"
	case "brandname", "brand_name":
		col = "brand_name"
	case "accountid", "account_id":
		col = "account_id"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
