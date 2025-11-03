package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"
    "time"

    dbcommon "narratives/internal/adapters/out/db/common"
    badom "narratives/internal/domain/billingAddress"
)

// BillingAddressRepositoryPG is the PostgreSQL implementation of billingAddress.RepositoryPort.
type BillingAddressRepositoryPG struct {
    DB *sql.DB
}

func NewBillingAddressRepositoryPG(db *sql.DB) *BillingAddressRepositoryPG {
    return &BillingAddressRepositoryPG{DB: db}
}

// ========== Public API (implements badom.RepositoryPort) ==========

func (r *BillingAddressRepositoryPG) GetByID(ctx context.Context, id string) (*badom.BillingAddress, error) {
    const q = `
SELECT
  id, user_id, name_on_account, billing_type, card_brand, card_last4, card_exp_month, card_exp_year, card_token,
  postal_code, state, city, street, country, is_default, created_at, updated_at
FROM billing_addresses
WHERE id = $1
`
    row := getQ(r, ctx).QueryRowContext(ctx, q, id)
    ba, err := scanBillingAddress(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, badom.ErrNotFound
        }
        return nil, err
    }
    return &ba, nil
}

func (r *BillingAddressRepositoryPG) GetByUser(ctx context.Context, userID string) ([]badom.BillingAddress, error) {
    const q = `
SELECT
  id, user_id, name_on_account, billing_type, card_brand, card_last4, card_exp_month, card_exp_year, card_token,
  postal_code, state, city, street, country, is_default, created_at, updated_at
FROM billing_addresses
WHERE user_id = $1
ORDER BY is_default DESC, updated_at DESC, id DESC
`
    rows, err := getQ(r, ctx).QueryContext(ctx, q, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []badom.BillingAddress
    for rows.Next() {
        ba, err := scanBillingAddress(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, ba)
    }
    return list, rows.Err()
}

func (r *BillingAddressRepositoryPG) GetDefaultByUser(ctx context.Context, userID string) (*badom.BillingAddress, error) {
    const q = `
SELECT
  id, user_id, name_on_account, billing_type, card_brand, card_last4, card_exp_month, card_exp_year, card_token,
  postal_code, state, city, street, country, is_default, created_at, updated_at
FROM billing_addresses
WHERE user_id = $1 AND is_default = TRUE
ORDER BY updated_at DESC, id DESC
LIMIT 1
`
    row := getQ(r, ctx).QueryRowContext(ctx, q, userID)
    ba, err := scanBillingAddress(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, badom.ErrNotFound
        }
        return nil, err
    }
    return &ba, nil
}

func (r *BillingAddressRepositoryPG) List(ctx context.Context, filter badom.Filter, sort badom.Sort, page badom.Page) (badom.PageResult, error) {
    where, args := buildBillingAddressWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    orderBy := buildBillingAddressOrderBy(sort)
    if orderBy == "" {
        orderBy = "ORDER BY updated_at DESC, id DESC"
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
    countSQL := fmt.Sprintf("SELECT COUNT(*) FROM billing_addresses %s", whereSQL)
    if err := getQ(r, ctx).QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
        return badom.PageResult{}, err
    }

    q := fmt.Sprintf(`
SELECT
  id, user_id, name_on_account, billing_type, card_brand, card_last4, card_exp_month, card_exp_year, card_token,
  postal_code, state, city, street, country, is_default, created_at, updated_at
FROM billing_addresses
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

    args = append(args, perPage, offset)

    rows, err := getQ(r, ctx).QueryContext(ctx, q, args...)
    if err != nil {
        return badom.PageResult{}, err
    }
    defer rows.Close()

    var items []badom.BillingAddress
    for rows.Next() {
        ba, err := scanBillingAddress(rows)
        if err != nil {
            return badom.PageResult{}, err
        }
        items = append(items, ba)
    }
    if err := rows.Err(); err != nil {
        return badom.PageResult{}, err
    }

    totalPages := (total + perPage - 1) / perPage
    return badom.PageResult{
        Items:      items,
        TotalCount: total,
        TotalPages: totalPages,
        Page:       number,
        PerPage:    perPage,
    }, nil
}

func (r *BillingAddressRepositoryPG) Count(ctx context.Context, filter badom.Filter) (int, error) {
    where, args := buildBillingAddressWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }
    var total int
    if err := getQ(r, ctx).QueryRowContext(ctx, `SELECT COUNT(*) FROM billing_addresses `+whereSQL, args...).Scan(&total); err != nil {
        return 0, err
    }
    return total, nil
}

func (r *BillingAddressRepositoryPG) Create(ctx context.Context, in badom.CreateBillingAddressInput) (*badom.BillingAddress, error) {
    const q = `
INSERT INTO billing_addresses (
  id, user_id, name_on_account, billing_type, card_brand, card_last4, card_exp_month, card_exp_year, card_token,
  postal_code, state, city, street, country, is_default, created_at, updated_at
) VALUES (
  gen_random_uuid(),
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13, $14, COALESCE($15, NOW()), COALESCE($16, NOW())
)
RETURNING
  id, user_id, name_on_account, billing_type, card_brand, card_last4, card_exp_month, card_exp_year, card_token,
  postal_code, state, city, street, country, is_default, created_at, updated_at
`
    row := getQ(r, ctx).QueryRowContext(ctx, q,
        in.UserID,
        dbcommon.ToDBText(in.NameOnAccount),
        strings.TrimSpace(in.BillingType),
        dbcommon.ToDBText(in.CardBrand),
        dbcommon.ToDBText(in.CardLast4),
        dbcommon.ToDBInt(in.CardExpMonth),
        dbcommon.ToDBInt(in.CardExpYear),
        dbcommon.ToDBText(in.CardToken),
        dbcommon.ToDBInt(in.PostalCode),
        dbcommon.ToDBText(in.State),
        dbcommon.ToDBText(in.City),
        dbcommon.ToDBText(in.Street),
        dbcommon.ToDBText(in.Country),
        in.IsDefault,
        dbcommon.ToDBTime(in.CreatedAt),
        dbcommon.ToDBTime(in.UpdatedAt),
    )
    ba, err := scanBillingAddress(row)
    if err != nil {
        if dbcommon.IsUniqueViolation(err) {
            return nil, badom.ErrConflict
        }
        return nil, err
    }
    return &ba, nil
}

func (r *BillingAddressRepositoryPG) Update(ctx context.Context, id string, in badom.UpdateBillingAddressInput) (*badom.BillingAddress, error) {
    sets := []string{}
    args := []any{}
    i := 1

    if in.BillingType != nil {
        sets = append(sets, fmt.Sprintf("billing_type = $%d", i))
        args = append(args, strings.TrimSpace(*in.BillingType))
        i++
    }
    if in.NameOnAccount != nil {
        sets = append(sets, fmt.Sprintf("name_on_account = $%d", i))
        args = append(args, dbcommon.ToDBText(in.NameOnAccount))
        i++
    }
    if in.CardBrand != nil {
        sets = append(sets, fmt.Sprintf("card_brand = $%d", i))
        args = append(args, dbcommon.ToDBText(in.CardBrand))
        i++
    }
    if in.CardLast4 != nil {
        sets = append(sets, fmt.Sprintf("card_last4 = $%d", i))
        args = append(args, dbcommon.ToDBText(in.CardLast4))
        i++
    }
    if in.CardExpMonth != nil {
        sets = append(sets, fmt.Sprintf("card_exp_month = $%d", i))
        args = append(args, dbcommon.ToDBInt(in.CardExpMonth))
        i++
    }
    if in.CardExpYear != nil {
        sets = append(sets, fmt.Sprintf("card_exp_year = $%d", i))
        args = append(args, dbcommon.ToDBInt(in.CardExpYear))
        i++
    }
    if in.CardToken != nil {
        sets = append(sets, fmt.Sprintf("card_token = $%d", i))
        args = append(args, dbcommon.ToDBText(in.CardToken))
        i++
    }
    if in.PostalCode != nil {
        sets = append(sets, fmt.Sprintf("postal_code = $%d", i))
        args = append(args, dbcommon.ToDBInt(in.PostalCode))
        i++
    }
    if in.State != nil {
        sets = append(sets, fmt.Sprintf("state = $%d", i))
        args = append(args, dbcommon.ToDBText(in.State))
        i++
    }
    if in.City != nil {
        sets = append(sets, fmt.Sprintf("city = $%d", i))
        args = append(args, dbcommon.ToDBText(in.City))
        i++
    }
    if in.Street != nil {
        sets = append(sets, fmt.Sprintf("street = $%d", i))
        args = append(args, dbcommon.ToDBText(in.Street))
        i++
    }
    if in.Country != nil {
        sets = append(sets, fmt.Sprintf("country = $%d", i))
        args = append(args, dbcommon.ToDBText(in.Country))
        i++
    }
    if in.IsDefault != nil {
        sets = append(sets, fmt.Sprintf("is_default = $%d", i))
        args = append(args, *in.IsDefault)
        i++
    }

    // updated_at
    if in.UpdatedAt != nil {
        sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
        args = append(args, dbcommon.ToDBTime(in.UpdatedAt))
        i++
    } else {
        sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
        args = append(args, time.Now().UTC())
        i++
    }

    if len(sets) == 0 {
        // no-op update; return current
        return r.GetByID(ctx, id)
    }

    args = append(args, id)
    q := fmt.Sprintf(`
UPDATE billing_addresses
SET %s
WHERE id = $%d
RETURNING
  id, user_id, name_on_account, billing_type, card_brand, card_last4, card_exp_month, card_exp_year, card_token,
  postal_code, state, city, street, country, is_default, created_at, updated_at
`, strings.Join(sets, ", "), i)

    row := getQ(r, ctx).QueryRowContext(ctx, q, args...)
    ba, err := scanBillingAddress(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, badom.ErrNotFound
        }
        return nil, err
    }
    return &ba, nil
}

func (r *BillingAddressRepositoryPG) Delete(ctx context.Context, id string) error {
    res, err := getQ(r, ctx).ExecContext(ctx, `DELETE FROM billing_addresses WHERE id = $1`, id)
    if err != nil {
        return err
    }
    aff, _ := res.RowsAffected()
    if aff == 0 {
        return badom.ErrNotFound
    }
    return nil
}

// SetDefault sets the specified billing address as default for its user (and unsets others).
func (r *BillingAddressRepositoryPG) SetDefault(ctx context.Context, id string) error {
    return r.WithTx(ctx, func(ctx context.Context) error {
        // lock the target row and get user_id
        var userID string
        if err := getQ(r, ctx).QueryRowContext(ctx, `SELECT user_id FROM billing_addresses WHERE id = $1 FOR UPDATE`, id).Scan(&userID); err != nil {
            if errors.Is(err, sql.ErrNoRows) {
                return badom.ErrNotFound
            }
            return err
        }
        // unset others
        if _, err := getQ(r, ctx).ExecContext(ctx, `UPDATE billing_addresses SET is_default = FALSE WHERE user_id = $1`, userID); err != nil {
            return err
        }
        // set this one
        if _, err := getQ(r, ctx).ExecContext(ctx, `UPDATE billing_addresses SET is_default = TRUE, updated_at = NOW() WHERE id = $1`, id); err != nil {
            return err
        }
        return nil
    })
}

func (r *BillingAddressRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
    tx, err := r.DB.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    ctxWithTx := context.WithValue(ctx, ctxTxKey{}, tx)
    if err := fn(ctxWithTx); err != nil {
        _ = tx.Rollback()
        return err
    }
    return tx.Commit()
}

func (r *BillingAddressRepositoryPG) Reset(ctx context.Context) error {
    _, err := r.DB.ExecContext(ctx, `TRUNCATE TABLE billing_addresses RESTART IDENTITY CASCADE`)
    return err
}

// ========== Helpers ==========

func scanBillingAddress(s dbcommon.RowScanner) (badom.BillingAddress, error) {
    var (
        idNS, userIDNS, nameOnNS, billingTypeNS                     sql.NullString
        cardBrandNS, cardLast4NS, cardTokenNS                        sql.NullString
        stateNS, cityNS, streetNS, countryNS                         sql.NullString
        cardExpMonthNI, cardExpYearNI, postalCodeNI                  sql.NullInt64
        isDefault                                                    bool
        createdAt, updatedAt                                         time.Time
    )

    if err := s.Scan(
        &idNS,
        &userIDNS,
        &nameOnNS,
        &billingTypeNS,
        &cardBrandNS,
        &cardLast4NS,
        &cardExpMonthNI,
        &cardExpYearNI,
        &cardTokenNS,
        &postalCodeNI,
        &stateNS,
        &cityNS,
        &streetNS,
        &countryNS,
        &isDefault,
        &createdAt,
        &updatedAt,
    ); err != nil {
        return badom.BillingAddress{}, err
    }

    toPtrStr := func(ns sql.NullString) *string {
        if ns.Valid {
            v := strings.TrimSpace(ns.String)
            if v != "" {
                return &v
            }
        }
        return nil
    }
    toPtrInt := func(ni sql.NullInt64) *int {
        if ni.Valid {
            v := int(ni.Int64)
            return &v
        }
        return nil
    }

    return badom.BillingAddress{
        ID:            strings.TrimSpace(idNS.String),
        UserID:        strings.TrimSpace(userIDNS.String),
        NameOnAccount: toPtrStr(nameOnNS),
        BillingType:   strings.TrimSpace(billingTypeNS.String),
        CardBrand:     toPtrStr(cardBrandNS),
        CardLast4:     toPtrStr(cardLast4NS),
        CardExpMonth:  toPtrInt(cardExpMonthNI),
        CardExpYear:   toPtrInt(cardExpYearNI),
        CardToken:     toPtrStr(cardTokenNS),
        PostalCode:    toPtrInt(postalCodeNI),
        State:         toPtrStr(stateNS),
        City:          toPtrStr(cityNS),
        Street:        toPtrStr(streetNS),
        Country:       toPtrStr(countryNS),
        IsDefault:     isDefault,
        CreatedAt:     createdAt.UTC(),
        UpdatedAt:     updatedAt.UTC(),
    }, nil
}

func buildBillingAddressWhere(f badom.Filter) ([]string, []any) {
    where := []string{}
    args := []any{}

    // IDs IN (...)
    if len(f.IDs) > 0 {
        ph := make([]string, 0, len(f.IDs))
        for _, v := range f.IDs {
            if strings.TrimSpace(v) == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, strings.TrimSpace(v))
        }
        if len(ph) > 0 {
            where = append(where, "id IN ("+strings.Join(ph, ",")+")")
        }
    }

    // UserIDs IN (...)
    if len(f.UserIDs) > 0 {
        ph := make([]string, 0, len(f.UserIDs))
        for _, v := range f.UserIDs {
            if strings.TrimSpace(v) == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, strings.TrimSpace(v))
        }
        if len(ph) > 0 {
            where = append(where, "user_id IN ("+strings.Join(ph, ",")+")")
        }
    }

    // BillingTypes IN (...)
    if len(f.BillingTypes) > 0 {
        ph := make([]string, 0, len(f.BillingTypes))
        for _, v := range f.BillingTypes {
            if strings.TrimSpace(v) == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, strings.TrimSpace(v))
        }
        if len(ph) > 0 {
            where = append(where, "billing_type IN ("+strings.Join(ph, ",")+")")
        }
    }

    // CardBrands IN (...)
    if len(f.CardBrands) > 0 {
        ph := make([]string, 0, len(f.CardBrands))
        for _, v := range f.CardBrands {
            if strings.TrimSpace(v) == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, strings.TrimSpace(v))
        }
        if len(ph) > 0 {
            where = append(where, "card_brand IN ("+strings.Join(ph, ",")+")")
        }
    }

    // IsDefault
    if f.IsDefault != nil {
        if *f.IsDefault {
            where = append(where, "is_default = TRUE")
        } else {
            where = append(where, "is_default = FALSE")
        }
    }

    // PostalCode range
    if f.PostalCodeMin != nil {
        where = append(where, fmt.Sprintf("postal_code >= $%d", len(args)+1))
        args = append(args, *f.PostalCodeMin)
    }
    if f.PostalCodeMax != nil {
        where = append(where, fmt.Sprintf("postal_code <= $%d", len(args)+1))
        args = append(args, *f.PostalCodeMax)
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

    // NameLike, CityLike
    if f.NameLike != nil && strings.TrimSpace(*f.NameLike) != "" {
        where = append(where, fmt.Sprintf("name_on_account ILIKE $%d", len(args)+1))
        args = append(args, "%"+strings.TrimSpace(*f.NameLike)+"%")
    }
    if f.CityLike != nil && strings.TrimSpace(*f.CityLike) != "" {
        where = append(where, fmt.Sprintf("city ILIKE $%d", len(args)+1))
        args = append(args, "%"+strings.TrimSpace(*f.CityLike)+"%")
    }

    return where, args
}

func buildBillingAddressOrderBy(sort badom.Sort) string {
    col := strings.ToLower(string(sort.Column))
    switch col {
    case "createdat", "created_at":
        col = "created_at"
    case "updatedat", "updated_at":
        col = "updated_at"
    case "billingtype", "billing_type":
        col = "billing_type"
    case "isdefault", "is_default":
        col = "is_default"
    case "postalcode", "postal_code":
        col = "postal_code"
    default:
        return ""
    }

    dir := strings.ToUpper(string(sort.Order))
    if dir != "ASC" && dir != "DESC" {
        dir = "ASC"
    }
    return fmt.Sprintf("ORDER BY %s %s", col, dir)
}

// ========== tx-aware query helpers ==========

type ctxTxKey struct{}

type querier interface {
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func getQ(r *BillingAddressRepositoryPG, ctx context.Context) querier {
    if tx, ok := ctx.Value(ctxTxKey{}).(*sql.Tx); ok && tx != nil {
        return tx
    }
    return r.DB
}