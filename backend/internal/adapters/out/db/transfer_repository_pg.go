package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	trdom "narratives/internal/domain/transfer"
)

// TransferRepositoryPG implements transfer.RepositoryPort with PostgreSQL.
type TransferRepositoryPG struct {
	DB *sql.DB
}

func NewTransferRepositoryPG(db *sql.DB) *TransferRepositoryPG {
	return &TransferRepositoryPG{DB: db}
}

// ===============================
// RepositoryPort impl
// ===============================

func (r *TransferRepositoryPG) GetByID(ctx context.Context, id string) (*trdom.Transfer, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, mint_address, from_address, to_address,
  requested_at, transferred_at, status, error_type
FROM transfers
WHERE id = $1
LIMIT 1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	tr, err := scanTransfer(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, trdom.ErrNotFound
		}
		return nil, err
	}
	return &tr, nil
}

func (r *TransferRepositoryPG) List(ctx context.Context, filter trdom.Filter, sort trdom.Sort, page trdom.Page) (trdom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTransferWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildTransferOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY requested_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM transfers "+whereSQL, args...).Scan(&total); err != nil {
		return trdom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id, mint_address, from_address, to_address,
  requested_at, transferred_at, status, error_type
FROM transfers
%s
%s
LIMIT $%d OFFSET $%d`, whereSQL, orderBy, len(args)+1, len(args)+2)
	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return trdom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]trdom.Transfer, 0, perPage)
	for rows.Next() {
		tr, err := scanTransfer(rows)
		if err != nil {
			return trdom.PageResult{}, err
		}
		items = append(items, tr)
	}
	if err := rows.Err(); err != nil {
		return trdom.PageResult{}, err
	}

	return trdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TransferRepositoryPG) Count(ctx context.Context, filter trdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	where, args := buildTransferWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM transfers "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *TransferRepositoryPG) Create(ctx context.Context, in trdom.CreateTransferInput) (*trdom.Transfer, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
INSERT INTO transfers (
  id, mint_address, from_address, to_address,
  requested_at, transferred_at, status, error_type
) VALUES (
  gen_random_uuid()::text, $1, $2, $3,
  $4, NULL, 'requested', NULL
)
RETURNING
  id, mint_address, from_address, to_address,
  requested_at, transferred_at, status, error_type`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.MintAddress),
		strings.TrimSpace(in.FromAddress),
		strings.TrimSpace(in.ToAddress),
		in.RequestedAt.UTC(),
	)
	tr, err := scanTransfer(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, trdom.ErrConflict
		}
		return nil, err
	}
	return &tr, nil
}

func (r *TransferRepositoryPG) Update(ctx context.Context, id string, in trdom.UpdateTransferInput) (*trdom.Transfer, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	// status
	if in.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", i))
		args = append(args, strings.TrimSpace(string(*in.Status)))
		i++
	}
	// error_type: if provided and empty string => NULL
	if in.ErrorType != nil {
		v := strings.TrimSpace(string(*in.ErrorType))
		if v == "" {
			sets = append(sets, "error_type = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("error_type = $%d", i))
			args = append(args, v)
			i++
		}
	}
	// transferred_at: if provided and zero => NULL, else set
	if in.TransferredAt != nil {
		if in.TransferredAt.IsZero() {
			sets = append(sets, "transferred_at = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("transferred_at = $%d", i))
			args = append(args, in.TransferredAt.UTC())
			i++
		}
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE transfers
SET %s
WHERE id = $%d
RETURNING
  id, mint_address, from_address, to_address,
  requested_at, transferred_at, status, error_type
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	tr, err := scanTransfer(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, trdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, trdom.ErrConflict
		}
		return nil, err
	}
	return &tr, nil
}

func (r *TransferRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM transfers WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return trdom.ErrNotFound
	}
	return nil
}

func (r *TransferRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

func (r *TransferRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM transfers`)
	return err
}

// ===============================
// Compatibility methods
// ===============================

func (r *TransferRepositoryPG) GetAllTransfers(ctx context.Context) ([]*trdom.Transfer, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, mint_address, from_address, to_address,
  requested_at, transferred_at, status, error_type
FROM transfers
ORDER BY requested_at DESC, id DESC`
	rows, err := run.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*trdom.Transfer
	for rows.Next() {
		t, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}
		tt := trdom.Transfer(t)
		out = append(out, &tt)
	}
	return out, rows.Err()
}

func (r *TransferRepositoryPG) GetTransferByID(ctx context.Context, id string) (*trdom.Transfer, error) {
	tr, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	tt := trdom.Transfer(*tr)
	return &tt, nil
}

func (r *TransferRepositoryPG) GetTransfersByFromAddress(ctx context.Context, fromAddress string) ([]*trdom.Transfer, error) {
	return r.getTransfersByColumn(ctx, "from_address", strings.TrimSpace(fromAddress))
}

func (r *TransferRepositoryPG) GetTransfersByToAddress(ctx context.Context, toAddress string) ([]*trdom.Transfer, error) {
	return r.getTransfersByColumn(ctx, "to_address", strings.TrimSpace(toAddress))
}

func (r *TransferRepositoryPG) GetTransfersByMintAddress(ctx context.Context, mintAddress string) ([]*trdom.Transfer, error) {
	return r.getTransfersByColumn(ctx, "mint_address", strings.TrimSpace(mintAddress))
}

func (r *TransferRepositoryPG) GetTransfersByStatus(ctx context.Context, status string) ([]*trdom.Transfer, error) {
	return r.getTransfersByColumn(ctx, "status", strings.TrimSpace(status))
}

func (r *TransferRepositoryPG) CreateTransfer(ctx context.Context, in trdom.CreateTransferInput) (*trdom.Transfer, error) {
	tr, err := r.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	tt := trdom.Transfer(*tr)
	return &tt, nil
}

func (r *TransferRepositoryPG) UpdateTransfer(ctx context.Context, id string, in trdom.UpdateTransferInput) (*trdom.Transfer, error) {
	tr, err := r.Update(ctx, id, in)
	if err != nil {
		return nil, err
	}
	tt := trdom.Transfer(*tr)
	return &tt, nil
}

func (r *TransferRepositoryPG) DeleteTransfer(ctx context.Context, id string) error {
	return r.Delete(ctx, id)
}

func (r *TransferRepositoryPG) ResetTransfers(ctx context.Context) error {
	return r.Reset(ctx)
}

// helper for compatibility listing by one column
func (r *TransferRepositoryPG) getTransfersByColumn(ctx context.Context, col, val string) ([]*trdom.Transfer, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	q := fmt.Sprintf(`
SELECT
  id, mint_address, from_address, to_address,
  requested_at, transferred_at, status, error_type
FROM transfers
WHERE %s = $1
ORDER BY requested_at DESC, id DESC`, col)
	rows, err := run.QueryContext(ctx, q, val)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*trdom.Transfer
	for rows.Next() {
		t, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}
		tt := trdom.Transfer(t)
		out = append(out, &tt)
	}
	return out, rows.Err()
}

// ===============================
// Scanners and builders
// ===============================

func scanTransfer(s dbcommon.RowScanner) (trdom.Transfer, error) {
	var (
		id, mint, from, to, status string
		errTypeNS                  sql.NullString
		reqAt                      time.Time
		trAtNS                     sql.NullTime
	)
	if err := s.Scan(&id, &mint, &from, &to, &reqAt, &trAtNS, &status, &errTypeNS); err != nil {
		return trdom.Transfer{}, err
	}

	var trAt *time.Time
	if trAtNS.Valid {
		t := trAtNS.Time.UTC()
		trAt = &t
	}
	var eType *trdom.TransferErrorType
	if errTypeNS.Valid {
		v := strings.TrimSpace(errTypeNS.String)
		if v != "" {
			x := trdom.TransferErrorType(v)
			eType = &x
		}
	}

	return trdom.Transfer{
		ID:            strings.TrimSpace(id),
		MintAddress:   strings.TrimSpace(mint),
		FromAddress:   strings.TrimSpace(from),
		ToAddress:     strings.TrimSpace(to),
		RequestedAt:   reqAt.UTC(),
		TransferredAt: trAt,
		Status:        trdom.TransferStatus(strings.TrimSpace(status)),
		ErrorType:     eType,
	}, nil
}

func buildTransferWhere(f trdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	addInStatus := func(col string, vals []trdom.TransferStatus) {
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
	addInErrType := func(col string, vals []trdom.TransferErrorType) {
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

	// Single ID filters
	if v := strings.TrimSpace(f.ID); v != "" {
		where = append(where, fmt.Sprintf("id = $%d", len(args)+1))
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.MintAddress); v != "" {
		where = append(where, fmt.Sprintf("mint_address = $%d", len(args)+1))
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.FromAddress); v != "" {
		where = append(where, fmt.Sprintf("from_address = $%d", len(args)+1))
		args = append(args, v)
	}
	if v := strings.TrimSpace(f.ToAddress); v != "" {
		where = append(where, fmt.Sprintf("to_address = $%d", len(args)+1))
		args = append(args, v)
	}

	// Arrays
	addInStatus("status", f.Statuses)
	addInErrType("error_type", f.ErrorTypes)

	// hasError flag
	if f.HasError != nil {
		if *f.HasError {
			where = append(where, "error_type IS NOT NULL")
		} else {
			where = append(where, "error_type IS NULL")
		}
	}

	// time ranges
	if f.RequestedFrom != nil {
		where = append(where, fmt.Sprintf("requested_at >= $%d", len(args)+1))
		args = append(args, f.RequestedFrom.UTC())
	}
	if f.RequestedTo != nil {
		where = append(where, fmt.Sprintf("requested_at < $%d", len(args)+1))
		args = append(args, f.RequestedTo.UTC())
	}
	if f.TransferedFrom != nil {
		where = append(where, fmt.Sprintf("transferred_at >= $%d", len(args)+1))
		args = append(args, f.TransferedFrom.UTC())
	}
	if f.TransferedTo != nil {
		where = append(where, fmt.Sprintf("transferred_at < $%d", len(args)+1))
		args = append(args, f.TransferedTo.UTC())
	}

	return where, args
}

func buildTransferOrderBy(s trdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "requestedat", "requested_at":
		col = "requested_at"
	case "transferredat", "transferred_at":
		col = "transferred_at"
	case "status":
		col = "status"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}
