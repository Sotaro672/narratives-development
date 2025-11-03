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

// ========================
// RepositoryPort impl
// ========================

func (r *ShippingAddressRepositoryPG) GetByID(ctx context.Context, id string) (*shipdom.ShippingAddress, error) {
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
            return nil, shipdom.ErrNotFound
        }
        return nil, err
    }
    return &a, nil
}

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

func (r *ShippingAddressRepositoryPG) Create(ctx context.Context, in shipdom.CreateShippingAddressInput) (*shipdom.ShippingAddress, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    // Map DTO to current schema
    street := strings.TrimSpace(in.AddressLine1)
    if in.AddressLine2 != nil {
        al2 := strings.TrimSpace(*in.AddressLine2)
        if al2 != "" {
            if street != "" {
                street = street + " " + al2
            } else {
                street = al2
            }
        }
    }
    city := strings.TrimSpace(in.City)
    state := strings.TrimSpace(in.Prefecture)
    zip := strings.TrimSpace(in.PostalCode)
    country := strings.TrimSpace(in.Country)

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
        strings.TrimSpace(in.UserID),
        street, city, state, zip, country,
    )
    a, err := scanShippingAddress(row)
    if err != nil {
        if dbcommon.IsUniqueViolation(err) {
            return nil, shipdom.ErrConflict
        }
        return nil, err
    }
    return &a, nil
}

func (r *ShippingAddressRepositoryPG) Update(ctx context.Context, id string, in shipdom.UpdateShippingAddressInput) (*shipdom.ShippingAddress, error) {
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
            return nil, shipdom.ErrNotFound
        }
        if dbcommon.IsUniqueViolation(err) {
            return nil, shipdom.ErrConflict
        }
        return nil, err
    }
    return &a, nil
}

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

func (r *ShippingAddressRepositoryPG) Reset(ctx context.Context) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    _, err := run.ExecContext(ctx, `DELETE FROM shipping_addresses`)
    return err
}

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

// ========================
// Helpers
// ========================

func scanShippingAddress(s dbcommon.RowScanner) (shipdom.ShippingAddress, error) {
    var (
        id, userID, street, city, state, zip, country string
        createdAt, updatedAt                           time.Time
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