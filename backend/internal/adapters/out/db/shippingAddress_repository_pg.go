package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	shipdom "narratives/internal/domain/shippingAddress"
)

type ShippingAddressRepositoryPG struct {
	DB *sql.DB
}

func NewShippingAddressRepositoryPG(db *sql.DB) *ShippingAddressRepositoryPG {
	return &ShippingAddressRepositoryPG{DB: db}
}

// ======================================================================
// Facade to satisfy usecase.ShippingAddressRepo
// ======================================================================

// GetByID returns value (not *ptr) to match usecase interface.
func (r *ShippingAddressRepositoryPG) GetByID(ctx context.Context, id string) (shipdom.ShippingAddress, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  id, user_id, street, city, state, zip_code, country, created_at, updated_at
FROM shipping_addresses
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))

	a, err := scanShippingAddress(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shipdom.ShippingAddress{}, shipdom.ErrNotFound
		}
		return shipdom.ShippingAddress{}, err
	}
	return a, nil
}

// Exists checks if an address with given ID exists.
func (r *ShippingAddressRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `SELECT 1 FROM shipping_addresses WHERE id = $1`
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

// Create inserts using full domain entity and returns the created row.
func (r *ShippingAddressRepositoryPG) Create(ctx context.Context, v shipdom.ShippingAddress) (shipdom.ShippingAddress, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO shipping_addresses (
  id, user_id, street, city, state, zip_code, country, created_at, updated_at
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4, $5, $6, NOW(), NOW()
)
RETURNING
  id, user_id, street, city, state, zip_code, country, created_at, updated_at
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(v.UserID),
		strings.TrimSpace(v.Street),
		strings.TrimSpace(v.City),
		strings.TrimSpace(v.State),
		strings.TrimSpace(v.ZipCode),
		strings.TrimSpace(v.Country),
	)

	created, err := scanShippingAddress(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return shipdom.ShippingAddress{}, shipdom.ErrConflict
		}
		return shipdom.ShippingAddress{}, err
	}
	return created, nil
}

// Save provides an upsert-like behavior:
// - if v.ID == "" -> Create
// - if v.ID exists -> Update existing row
// - if v.ID doesn't exist -> Create new row (ignoring provided ID)
func (r *ShippingAddressRepositoryPG) Save(ctx context.Context, v shipdom.ShippingAddress) (shipdom.ShippingAddress, error) {
	id := strings.TrimSpace(v.ID)
	if id == "" {
		// no ID -> just Create
		return r.Create(ctx, v)
	}

	exists, err := r.Exists(ctx, id)
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}

	if !exists {
		// caller gave an ID that isn't in DB -> treat as Create (fresh ID)
		return r.Create(ctx, v)
	}

	// exists -> Update using UpdateShippingAddressInput
	patch := shipdom.UpdateShippingAddressInput{
		// We only have flattened fields in the domain entity.
		AddressLine1: optString(v.Street),
		// no AddressLine2 info in v -> leave nil
		City:       optString(v.City),
		Prefecture: optString(v.State),
		PostalCode: optString(v.ZipCode),
		Country:    optString(v.Country),
		// UpdatedAt bump is handled in updateInternal()
	}

	updated, err := r.updateInternal(ctx, id, patch)
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}
	return updated, nil
}

// Delete matches the interface signature.
func (r *ShippingAddressRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM shipping_addresses WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return shipdom.ErrNotFound
	}
	return nil
}

// ======================================================================
// Richer / internal helpers (list, count, updateInternal, tx helpers, etc.)
// ======================================================================

func (r *ShippingAddressRepositoryPG) List(ctx context.Context, filter shipdom.Filter, sort shipdom.Sort, page shipdom.Page) (shipdom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildAddrWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildAddrOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY updated_at DESC, id DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM shipping_addresses "+whereSQL, args...).Scan(&total); err != nil {
		return shipdom.PageResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  id, user_id, street, city, state, zip_code, country, created_at, updated_at
FROM shipping_addresses
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return shipdom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]shipdom.ShippingAddress, 0, perPage)
	for rows.Next() {
		a, err := scanShippingAddress(rows)
		if err != nil {
			return shipdom.PageResult{}, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return shipdom.PageResult{}, err
	}

	return shipdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ShippingAddressRepositoryPG) Count(ctx context.Context, filter shipdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildAddrWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM shipping_addresses "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// updateInternal is the old Update() logic, made private so Save() can call it.
// It returns value, not pointer.
func (r *ShippingAddressRepositoryPG) updateInternal(ctx context.Context, id string, in shipdom.UpdateShippingAddressInput) (shipdom.ShippingAddress, error) {
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

	// Map DTO fields to schema
	setStr("street", in.AddressLine1) // AddressLine2 appended if provided below
	setStr("city", in.City)
	setStr("state", in.Prefecture)
	setStr("zip_code", in.PostalCode)
	setStr("country", in.Country)

	// If AddressLine2 provided, append to street
	if in.AddressLine2 != nil {
		sets = append(sets, fmt.Sprintf("street = CONCAT(street, CASE WHEN $%d <> '' THEN ' ' || $%d ELSE '' END)", i, i))
		args = append(args, strings.TrimSpace(*in.AddressLine2))
		i++
	}

	// Always bump updated_at
	sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
	args = append(args, time.Now().UTC())
	i++

	if len(sets) == 0 {
		// no change but exists -> just fetch current
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE shipping_addresses
SET %s
WHERE id = $%d
RETURNING
  id, user_id, street, city, state, zip_code, country, created_at, updated_at
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	a, err := scanShippingAddress(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shipdom.ShippingAddress{}, shipdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return shipdom.ShippingAddress{}, shipdom.ErrConflict
		}
		return shipdom.ShippingAddress{}, err
	}
	return a, nil
}

// WithTx for transactional usage
func (r *ShippingAddressRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *ShippingAddressRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM shipping_addresses`)
	return err
}

// ======================================================================
// Helpers
// ======================================================================

func scanShippingAddress(s dbcommon.RowScanner) (shipdom.ShippingAddress, error) {
	var (
		id, userID, street, city, state, zip, country string
		createdAt, updatedAt                          time.Time
	)
	if err := s.Scan(&id, &userID, &street, &city, &state, &zip, &country, &createdAt, &updatedAt); err != nil {
		return shipdom.ShippingAddress{}, err
	}
	return shipdom.ShippingAddress{
		ID:        strings.TrimSpace(id),
		UserID:    strings.TrimSpace(userID),
		Street:    strings.TrimSpace(street),
		City:      strings.TrimSpace(city),
		State:     strings.TrimSpace(state),
		ZipCode:   strings.TrimSpace(zip),
		Country:   strings.TrimSpace(country),
		CreatedAt: createdAt.UTC(),
		UpdatedAt: updatedAt.UTC(),
	}, nil
}

func buildAddrWhere(f shipdom.Filter) ([]string, []any) {
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
	addEq("user_id", f.UserID)
	addEq("city", f.City)
	addEq("state", f.State)
	addEq("zip_code", f.ZipCode)
	addEq("country", f.Country)

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

func buildAddrOrderBy(s shipdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	case "city":
		col = "city"
	case "state":
		col = "state"
	case "zipcode", "zip_code":
		col = "zip_code"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}

// optString: local helper for Save().
// This does NOT clash with strPtrOrNil in other db/*.go files.
func optString(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
