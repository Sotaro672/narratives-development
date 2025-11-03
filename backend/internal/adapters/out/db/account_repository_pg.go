package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"
    "time"

    "narratives/internal/domain/account"
    common "narratives/internal/domain/common"
    dbcommon "narratives/internal/adapters/out/db/common"
)

// AccountRepositoryPG is a PostgreSQL implementation of account.Repository.
type AccountRepositoryPG struct {
    db *sql.DB
}

func NewAccountRepositoryPG(db *sql.DB) *AccountRepositoryPG {
    return &AccountRepositoryPG{db: db}
}

// Scannable abstracts *sql.Row and *sql.Rows for scan helpers.
// [REMOVED] 重複のため削除し、dbcommon.RowScanner を使用します。
// type Scannable interface {
// 	Scan(dest ...any) error
// }

func (r *AccountRepositoryPG) List(ctx context.Context, filter account.Filter, sort common.Sort, page common.Page) (common.PageResult[account.Account], error) {
    where, args := buildAccountWhere(filter)
    orderBy := buildAccountOrderBy(sort)
    if orderBy == "" {
        orderBy = "ORDER BY created_at DESC"
    }

    limit := page.PerPage
    if limit <= 0 {
        limit = 20
    }
    number := page.Number
    if number <= 0 {
        number = 1
    }
    offset := (number - 1) * limit

    q := fmt.Sprintf(`
SELECT
  id, member_id, bank_name, branch_name, account_number, account_type, currency, status,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM accounts
%s
%s
LIMIT %d OFFSET %d
`, where, orderBy, limit, offset)

    rows, err := r.db.QueryContext(ctx, q, args...)
    if err != nil {
        return common.PageResult[account.Account]{}, err
    }
    defer rows.Close()

    var items []account.Account
    for rows.Next() {
        a, err := scanAccountRow(rows)
        if err != nil {
            return common.PageResult[account.Account]{}, err
        }
        items = append(items, a)
    }
    if err := rows.Err(); err != nil {
        return common.PageResult[account.Account]{}, err
    }

    total, err := r.Count(ctx, filter)
    if err != nil {
        return common.PageResult[account.Account]{}, err
    }
    totalPages := (total + limit - 1) / limit

    return common.PageResult[account.Account]{
        Items:      items,
        TotalCount: total,
        TotalPages: totalPages,
        Page:       number,
        PerPage:    limit,
    }, nil
}

func (r *AccountRepositoryPG) ListByCursor(ctx context.Context, filter account.Filter, sort common.Sort, cpage common.CursorPage) (common.CursorPageResult[account.Account], error) {
    // Not enough info about CursorPage shape. Provide minimal, stable stub.
    return common.CursorPageResult[account.Account]{}, errors.New("ListByCursor: not implemented")
}

func (r *AccountRepositoryPG) GetByID(ctx context.Context, id string) (account.Account, error) {
    q := `
SELECT
  id, member_id, bank_name, branch_name, account_number, account_type, currency, status,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM accounts
WHERE id = $1
`
    row := r.db.QueryRowContext(ctx, q, id)
    a, err := scanAccountRow(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return account.Account{}, account.ErrNotFound
        }
        return account.Account{}, err
    }
    return a, nil
}

func (r *AccountRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
    var v int
    if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM accounts WHERE id = $1`, id).Scan(&v); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (r *AccountRepositoryPG) Count(ctx context.Context, filter account.Filter) (int, error) {
    where, args := buildAccountWhere(filter)
    q := fmt.Sprintf(`SELECT COUNT(*) FROM accounts %s`, where)
    var cnt int
    if err := r.db.QueryRowContext(ctx, q, args...).Scan(&cnt); err != nil {
        return 0, err
    }
    return cnt, nil
}

func (r *AccountRepositoryPG) Create(ctx context.Context, a account.Account) (account.Account, error) {
    q := `
INSERT INTO accounts (
  id, member_id, bank_name, branch_name, account_number, account_type, currency, status,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,
  $9,$10,$11,$12,$13,$14
)
RETURNING
  id, member_id, bank_name, branch_name, account_number, account_type, currency, status,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
    var createdBy, updatedBy any
    if a.CreatedBy != nil {
        createdBy = *a.CreatedBy
    }
    if a.UpdatedBy != nil {
        updatedBy = *a.UpdatedBy
    }
    var deletedAt any
    if a.DeletedAt != nil {
        deletedAt = *a.DeletedAt
    }
    var deletedBy any
    if a.DeletedBy != nil {
        deletedBy = *a.DeletedBy
    }

    row := r.db.QueryRowContext(ctx, q,
        a.ID, a.MemberID, a.BankName, a.BranchName, a.AccountNumber, string(a.AccountType), a.Currency, string(a.Status),
        a.CreatedAt, createdBy, a.UpdatedAt, updatedBy, deletedAt, deletedBy,
    )

    out, err := scanAccountRow(row)
    if err != nil {
        if dbcommon.IsUniqueViolation(err) { // ← 共通ユーティリティへ
            return account.Account{}, account.ErrConflict
        }
        return account.Account{}, err
    }
    return out, nil
}

func (r *AccountRepositoryPG) Update(ctx context.Context, id string, patch account.AccountPatch) (account.Account, error) {
    set := make([]string, 0, 10)
    args := make([]any, 0, 12)
    i := 1

    if patch.BankName != nil {
        set = append(set, fmt.Sprintf("bank_name = $%d", i))
        args = append(args, strings.TrimSpace(*patch.BankName))
        i++
    }
    if patch.BranchName != nil {
        set = append(set, fmt.Sprintf("branch_name = $%d", i))
        args = append(args, strings.TrimSpace(*patch.BranchName))
        i++
    }
    if patch.AccountNumber != nil {
        set = append(set, fmt.Sprintf("account_number = $%d", i))
        args = append(args, *patch.AccountNumber)
        i++
    }
    if patch.AccountType != nil {
        set = append(set, fmt.Sprintf("account_type = $%d", i))
        args = append(args, string(*patch.AccountType))
        i++
    }
    if patch.Currency != nil {
        set = append(set, fmt.Sprintf("currency = $%d", i))
        args = append(args, strings.TrimSpace(*patch.Currency))
        i++
    }
    if patch.Status != nil {
        set = append(set, fmt.Sprintf("status = $%d", i))
        args = append(args, string(*patch.Status))
        i++
    }
    if patch.UpdatedBy != nil {
        set = append(set, fmt.Sprintf("updated_by = $%d", i))
        args = append(args, strings.TrimSpace(*patch.UpdatedBy))
        i++
    }
    if patch.DeletedAt != nil {
        set = append(set, fmt.Sprintf("deleted_at = $%d", i))
        args = append(args, (*patch.DeletedAt).UTC())
        i++
    }
    if patch.DeletedBy != nil {
        set = append(set, fmt.Sprintf("deleted_by = $%d", i))
        args = append(args, strings.TrimSpace(*patch.DeletedBy))
        i++
    }

    // Always bump updated_at
    set = append(set, fmt.Sprintf("updated_at = $%d", i))
    now := time.Now().UTC()
    args = append(args, now)
    i++

    if len(set) == 0 {
        // Nothing to update; return current row
        return r.GetByID(ctx, id)
    }

    q := fmt.Sprintf(`
UPDATE accounts
SET %s
WHERE id = $%d
RETURNING
  id, member_id, bank_name, branch_name, account_number, account_type, currency, status,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(set, ", "), i)
    args = append(args, id)

    row := r.db.QueryRowContext(ctx, q, args...)
    a, err := scanAccountRow(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return account.Account{}, account.ErrNotFound
        }
        return account.Account{}, err
    }
    return a, nil
}

func (r *AccountRepositoryPG) Delete(ctx context.Context, id string) error {
    res, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1`, id)
    if err != nil {
        return err
    }
    aff, _ := res.RowsAffected()
    if aff == 0 {
        return account.ErrNotFound
    }
    return nil
}

func (r *AccountRepositoryPG) Save(ctx context.Context, a account.Account, opts *common.SaveOptions) (account.Account, error) {
    // Simple UPSERT by id
    q := `
INSERT INTO accounts (
  id, member_id, bank_name, branch_name, account_number, account_type, currency, status,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,
  $9,$10,$11,$12,$13,$14
)
ON CONFLICT (id) DO UPDATE SET
  member_id = EXCLUDED.member_id,
  bank_name = EXCLUDED.bank_name,
  branch_name = EXCLUDED.branch_name,
  account_number = EXCLUDED.account_number,
  account_type = EXCLUDED.account_type,
  currency = EXCLUDED.currency,
  status = EXCLUDED.status,
  created_at = LEAST(accounts.created_at, EXCLUDED.created_at),
  created_by = COALESCE(EXCLUDED.created_by, accounts.created_by),
  updated_at = EXCLUDED.updated_at,
  updated_by = COALESCE(EXCLUDED.updated_by, accounts.updated_by),
  deleted_at = EXCLUDED.deleted_at,
  deleted_by = EXCLUDED.deleted_by
RETURNING
  id, member_id, bank_name, branch_name, account_number, account_type, currency, status,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
    var createdBy, updatedBy any
    if a.CreatedBy != nil {
        createdBy = *a.CreatedBy
    }
    if a.UpdatedBy != nil {
        updatedBy = *a.UpdatedBy
    }
    var deletedAt any
    if a.DeletedAt != nil {
        deletedAt = *a.DeletedAt
    }
    var deletedBy any
    if a.DeletedBy != nil {
        deletedBy = *a.DeletedBy
    }

    row := r.db.QueryRowContext(ctx, q,
        a.ID, a.MemberID, a.BankName, a.BranchName, a.AccountNumber, string(a.AccountType), a.Currency, string(a.Status),
        a.CreatedAt, createdBy, a.UpdatedAt, updatedBy, deletedAt, deletedBy,
    )
    out, err := scanAccountRow(row)
    if err != nil {
        return account.Account{}, err
    }
    return out, nil
}

// --- Helpers ---

func scanAccountRow(s dbcommon.RowScanner) (account.Account, error) { // ← 共通の RowScanner を使用
    var (
        id, memberID, bankName, branchName          string
        accountNumber                                int
        accountTypeStr, currency, statusStr          string
        createdAt, updatedAt                          time.Time
        createdByNS, updatedByNS, deletedByNS         sql.NullString
        deletedAtNT                                   sql.NullTime
    )

    err := s.Scan(
        &id, &memberID, &bankName, &branchName, &accountNumber, &accountTypeStr, &currency, &statusStr,
        &createdAt, &createdByNS, &updatedAt, &updatedByNS, &deletedAtNT, &deletedByNS,
    )
    if err != nil {
        return account.Account{}, err
    }

    // Build domain entity via constructor for validation.
    a, err := account.New(
        id, memberID, bankName, branchName, accountNumber,
        account.AccountType(accountTypeStr), currency, account.AccountStatus(statusStr),
        createdAt, updatedAt,
    )
    if err != nil {
        return account.Account{}, err
    }
    if createdByNS.Valid {
        cb := createdByNS.String
        a.CreatedBy = &cb
    }
    if updatedByNS.Valid {
        ub := updatedByNS.String
        a.UpdatedBy = &ub
    }
    if deletedAtNT.Valid {
        dt := deletedAtNT.Time.UTC()
        a.DeletedAt = &dt
    }
    if deletedByNS.Valid {
        db := deletedByNS.String
        a.DeletedBy = &db
    }
    return a, nil
}

func buildAccountWhere(f account.Filter) (string, []any) {
    var conds []string
    var args []any
    i := 1

    // Search across id/bank_name/branch_name/currency
    if qs := strings.TrimSpace(f.SearchQuery); qs != "" {
        like := "%" + qs + "%"
        conds = append(conds, fmt.Sprintf("(id ILIKE $%d OR bank_name ILIKE $%d OR branch_name ILIKE $%d OR currency ILIKE $%d)", i, i, i, i))
        args = append(args, like)
        i++
    }

    if f.MemberID != nil && strings.TrimSpace(*f.MemberID) != "" {
        conds = append(conds, fmt.Sprintf("member_id = $%d", i))
        args = append(args, strings.TrimSpace(*f.MemberID))
        i++
    }

    if f.Currency != nil && strings.TrimSpace(*f.Currency) != "" {
        conds = append(conds, fmt.Sprintf("currency = $%d", i))
        args = append(args, strings.TrimSpace(*f.Currency))
        i++
    }

    if len(f.Statuses) > 0 {
        ph := make([]string, len(f.Statuses))
        for idx, s := range f.Statuses {
            ph[idx] = fmt.Sprintf("$%d", i)
            args = append(args, string(s))
            i++
        }
        conds = append(conds, fmt.Sprintf("status IN (%s)", strings.Join(ph, ",")))
    }

    if len(f.Types) > 0 {
        ph := make([]string, len(f.Types))
        for idx, t := range f.Types {
            ph[idx] = fmt.Sprintf("$%d", i)
            args = append(args, string(t))
            i++
        }
        conds = append(conds, fmt.Sprintf("account_type IN (%s)", strings.Join(ph, ",")))
    }

    if f.AccountNumberMin != nil {
        conds = append(conds, fmt.Sprintf("account_number >= $%d", i))
        args = append(args, *f.AccountNumberMin)
        i++
    }
    if f.AccountNumberMax != nil {
        conds = append(conds, fmt.Sprintf("account_number <= $%d", i))
        args = append(args, *f.AccountNumberMax)
        i++
    }

    if f.CreatedFrom != nil {
        conds = append(conds, fmt.Sprintf("created_at >= $%d", i))
        args = append(args, (*f.CreatedFrom).UTC())
        i++
    }
    if f.CreatedTo != nil {
        conds = append(conds, fmt.Sprintf("created_at <= $%d", i))
        args = append(args, (*f.CreatedTo).UTC())
        i++
    }
    if f.UpdatedFrom != nil {
        conds = append(conds, fmt.Sprintf("updated_at >= $%d", i))
        args = append(args, (*f.UpdatedFrom).UTC())
        i++
    }
    if f.UpdatedTo != nil {
        conds = append(conds, fmt.Sprintf("updated_at <= $%d", i))
        args = append(args, (*f.UpdatedTo).UTC())
        i++
    }

    if f.Deleted != nil {
        if *f.Deleted {
            conds = append(conds, "deleted_at IS NOT NULL")
        } else {
            conds = append(conds, "deleted_at IS NULL")
        }
    }

    where := ""
    if len(conds) > 0 {
        where = "WHERE " + strings.Join(conds, " AND ")
    }
    return where, args
}

func buildAccountOrderBy(_ common.Sort) string {
    // Unknown Sort shape. Default to created_at DESC for stability.
    return "ORDER BY created_at DESC, id DESC"
}