package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	udom "narratives/internal/domain/user"
)

type UserRepositoryPG struct {
	DB *sql.DB
}

func NewUserRepositoryPG(db *sql.DB) *UserRepositoryPG {
	return &UserRepositoryPG{DB: db}
}

// ======================================================================
// UserRepo facade for usecase.UserRepo
// (Make UserRepositoryPG satisfy the interface expected by UserUsecase.)
// ======================================================================

// GetByID(ctx, id) (udom.User, error)
// NOTE: return value (not pointer) to match usecase.UserRepo.
func (r *UserRepositoryPG) GetByID(ctx context.Context, id string) (udom.User, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, first_name, first_name_kana, last_name_kana, last_name,
  email, phone_number, created_at, updated_at, deleted_at
FROM users
WHERE id = $1
LIMIT 1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return udom.User{}, udom.ErrNotFound
		}
		return udom.User{}, err
	}
	return u, nil
}

// Exists(ctx, id) (bool, error)
func (r *UserRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `SELECT 1 FROM users WHERE id = $1 LIMIT 1`
	var dummy int
	err := run.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create(ctx, v udom.User) (udom.User, error)
// Wraps lower-level INSERT logic, mapping domain User -> CreateUserInput.
func (r *UserRepositoryPG) Create(ctx context.Context, v udom.User) (udom.User, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	// timestamps
	now := time.Now().UTC()
	createdAt := now
	if !v.CreatedAt.IsZero() {
		createdAt = v.CreatedAt.UTC()
	}
	updatedAt := now
	if !v.UpdatedAt.IsZero() {
		updatedAt = v.UpdatedAt.UTC()
	}
	// NOTE: deleted_at is NOT NULL in table.
	deletedAt := createdAt
	if !v.DeletedAt.IsZero() {
		deletedAt = v.DeletedAt.UTC()
		if deletedAt.Before(createdAt) {
			deletedAt = createdAt
		}
	}

	const q = `
INSERT INTO users (
  id, first_name, first_name_kana, last_name_kana, last_name,
  email, phone_number, created_at, updated_at, deleted_at
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4,
  $5, $6, $7, $8, $9
)
RETURNING
  id, first_name, first_name_kana, last_name_kana, last_name,
  email, phone_number, created_at, updated_at, deleted_at
`
	row := run.QueryRowContext(
		ctx,
		q,
		dbcommon.NullableTrim(v.FirstName),
		dbcommon.NullableTrim(v.FirstNameKana),
		dbcommon.NullableTrim(v.LastNameKana),
		dbcommon.NullableTrim(v.LastName),
		nullableEmail(v.Email),
		dbcommon.NullableTrim(v.PhoneNumber),
		createdAt,
		updatedAt,
		deletedAt,
	)

	u, err := scanUser(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return udom.User{}, udom.ErrConflict
		}
		return udom.User{}, err
	}
	return u, nil
}

// Save(ctx, v udom.User) (udom.User, error)
// Upsert-y wrapper: if ID empty or missing -> Create; otherwise -> Update.
func (r *UserRepositoryPG) Save(ctx context.Context, v udom.User) (udom.User, error) {
	id := strings.TrimSpace(v.ID)
	if id == "" {
		// no ID? treat as brand new
		return r.Create(ctx, v)
	}

	exists, err := r.Exists(ctx, id)
	if err != nil {
		return udom.User{}, err
	}
	if !exists {
		// caller gave ID but record doesn't exist -> treat as create (DB will assign new ID)
		return r.Create(ctx, v)
	}

	// map domain User -> UpdateUserInput
	patch := udom.UpdateUserInput{
		FirstName:     strPtrIfNonEmptyPtr(v.FirstName),
		FirstNameKana: strPtrIfNonEmptyPtr(v.FirstNameKana),
		LastNameKana:  strPtrIfNonEmptyPtr(v.LastNameKana),
		LastName:      strPtrIfNonEmptyPtr(v.LastName),
		Email:         strPtrIfNonEmptyPtr(v.Email),
		PhoneNumber:   strPtrIfNonEmptyPtr(v.PhoneNumber),
		// UpdatedAt: prefer v.UpdatedAt if it's non-zero
		UpdatedAt: func(t time.Time) *time.Time {
			if t.IsZero() {
				return nil
			}
			tt := t.UTC()
			return &tt
		}(v.UpdatedAt),
		// DeletedAt: we treat zero as "do not touch"
		DeletedAt: func(t time.Time) *time.Time {
			if t.IsZero() {
				return nil
			}
			tt := t.UTC()
			return &tt
		}(v.DeletedAt),
	}

	updatedPtr, err := r.Update(ctx, id, patch)
	if err != nil {
		return udom.User{}, err
	}
	if updatedPtr == nil {
		return udom.User{}, udom.ErrNotFound
	}
	return *updatedPtr, nil
}

// Delete(ctx, id) error
// (already matches the interface signature, keep as is below.)

// helper: if ptr is nil or *ptr == "" -> nil, else trimmed copy
func strPtrIfNonEmptyPtr(src *string) *string {
	if src == nil {
		return nil
	}
	v := strings.TrimSpace(*src)
	if v == "" {
		return nil
	}
	return &v
}

// ======================================================================
// Lower-level / richer query methods
// (These are still useful to handlers, etc.)
// ======================================================================

func (r *UserRepositoryPG) GetByEmail(ctx context.Context, email string) (*udom.User, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, first_name, first_name_kana, last_name_kana, last_name,
  email, phone_number, created_at, updated_at, deleted_at
FROM users
WHERE email = $1
LIMIT 1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(email))
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, udom.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepositoryPG) List(ctx context.Context, filter udom.Filter, sort udom.Sort, page udom.Page) (udom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildUserWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildUserOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY created_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM users "+whereSQL, args...).Scan(&total); err != nil {
		return udom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id, first_name, first_name_kana, last_name_kana, last_name,
  email, phone_number, created_at, updated_at, deleted_at
FROM users
%s
%s
LIMIT $%d OFFSET $%d`, whereSQL, orderBy, len(args)+1, len(args)+2)
	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return udom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]udom.User, 0, perPage)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return udom.PageResult{}, err
		}
		items = append(items, u)
	}
	if err := rows.Err(); err != nil {
		return udom.PageResult{}, err
	}

	return udom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *UserRepositoryPG) Count(ctx context.Context, filter udom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	where, args := buildUserWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM users "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// Update is the lower-level UPDATE ... RETURNING.
// This is used by Save().
func (r *UserRepositoryPG) Update(ctx context.Context, id string, in udom.UpdateUserInput) (*udom.User, error) {
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
	setEmail := func(p *string) {
		if p != nil {
			v := strings.TrimSpace(*p)
			if v == "" {
				sets = append(sets, "email = NULL")
			} else {
				// optional basic validation; we don't block if invalid
				if _, err := mail.ParseAddress(v); err != nil {
					// ignore err here
				}
				sets = append(sets, fmt.Sprintf("email = $%d", i))
				args = append(args, v)
				i++
			}
		}
	}
	setTime := func(col string, p *time.Time) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, p.UTC())
			i++
		}
	}

	setStr("first_name", in.FirstName)
	setStr("first_name_kana", in.FirstNameKana)
	setStr("last_name_kana", in.LastNameKana)
	setStr("last_name", in.LastName)
	setEmail(in.Email)
	setStr("phone_number", in.PhoneNumber)

	// Always update updated_at.
	if in.UpdatedAt != nil {
		setTime("updated_at", in.UpdatedAt)
	} else {
		sets = append(sets, "updated_at = NOW()")
	}

	// deleted_at is optional
	if in.DeletedAt != nil {
		setTime("deleted_at", in.DeletedAt)
	}

	if len(sets) == 0 {
		// No-op update: just reload
		got, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return &got, nil
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE users
SET %s
WHERE id = $%d
RETURNING
  id, first_name, first_name_kana, last_name_kana, last_name,
  email, phone_number, created_at, updated_at, deleted_at
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, udom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, udom.ErrConflict
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return udom.ErrNotFound
	}
	return nil
}

// OPTIONAL extra helpers
func (r *UserRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM users`)
	return err
}

func (r *UserRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

// ======================================================================
// Helpers
// ======================================================================

func scanUser(s dbcommon.RowScanner) (udom.User, error) {
	var (
		id        string
		fnNS      sql.NullString
		fnkNS     sql.NullString
		lnkNS     sql.NullString
		lnNS      sql.NullString
		emailNS   sql.NullString
		phoneNS   sql.NullString
		createdAt time.Time
		updatedAt time.Time
		deletedAt time.Time
	)
	if err := s.Scan(
		&id, &fnNS, &fnkNS, &lnkNS, &lnNS,
		&emailNS, &phoneNS, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return udom.User{}, err
	}

	return udom.User{
		ID:            strings.TrimSpace(id),
		FirstName:     nsToPtr(fnNS),
		FirstNameKana: nsToPtr(fnkNS),
		LastNameKana:  nsToPtr(lnkNS),
		LastName:      nsToPtr(lnNS),
		Email:         nsToPtr(emailNS),
		PhoneNumber:   nsToPtr(phoneNS),
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
		DeletedAt:     deletedAt.UTC(),
	}, nil
}

func buildUserWhere(f udom.Filter) ([]string, []any) {
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

	// Arrays
	addIn("id", f.IDs)
	addIn("email", f.Emails)

	// Likes
	if v := strings.TrimSpace(f.FirstNameLike); v != "" {
		where = append(where, fmt.Sprintf("first_name ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}
	if v := strings.TrimSpace(f.LastNameLike); v != "" {
		where = append(where, fmt.Sprintf("last_name ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}
	if v := strings.TrimSpace(f.NameLike); v != "" {
		or := []string{
			fmt.Sprintf("first_name ILIKE $%d", len(args)+1),
			fmt.Sprintf("last_name ILIKE $%d", len(args)+2),
		}
		args = append(args, "%"+v+"%", "%"+v+"%")
		where = append(where, "("+strings.Join(or, " OR ")+")")
	}
	if v := strings.TrimSpace(f.PhoneLike); v != "" {
		where = append(where, fmt.Sprintf("phone_number ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
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
		where = append(where, fmt.Sprintf("deleted_at >= $%d", len(args)+1))
		args = append(args, f.DeletedFrom.UTC())
	}
	if f.DeletedTo != nil {
		where = append(where, fmt.Sprintf("deleted_at < $%d", len(args)+1))
		args = append(args, f.DeletedTo.UTC())
	}

	return where, args
}

func buildUserOrderBy(s udom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	case "deletedat", "deleted_at":
		col = "deleted_at"
	case "first_name", "firstname":
		col = "first_name"
	case "last_name", "lastname":
		col = "last_name"
	case "email":
		col = "email"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}

func nsToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := strings.TrimSpace(ns.String)
	if v == "" {
		return nil
	}
	return &v
}

func nullableEmail(p *string) any {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	// best-effort syntax check, but don't block
	if _, err := mail.ParseAddress(v); err != nil {
		// ignore validation error; domain layer should enforce
	}
	return v
}
