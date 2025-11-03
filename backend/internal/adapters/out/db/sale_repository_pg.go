package db

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "strings"

    dbcommon "narratives/internal/adapters/out/db/common"
    saledom "narratives/internal/domain/sale"
)

type SaleRepositoryPG struct {
    DB *sql.DB
}

func NewSaleRepositoryPG(db *sql.DB) *SaleRepositoryPG {
    return &SaleRepositoryPG{DB: db}
}

// ========================
// RepositoryPort impl
// ========================

func (r *SaleRepositoryPG) GetByID(ctx context.Context, id string) (*saledom.Sale, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    const q = `
SELECT id, list_id, discount_id, prices
FROM sales
WHERE id = $1`
    row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
    s, err := scanSale(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, saledom.ErrNotFound
        }
        return nil, err
    }
    return &s, nil
}

func (r *SaleRepositoryPG) List(ctx context.Context, filter saledom.Filter, sort saledom.Sort, page saledom.Page) (saledom.PageResult, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    where, args := buildSaleWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    orderBy := buildSaleOrderBy(sort)
    if orderBy == "" {
        orderBy = "ORDER BY id ASC"
    }

    pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

    // Count
    var total int
    if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM sales "+whereSQL, args...).Scan(&total); err != nil {
        return saledom.PageResult{}, err
    }

    // Data
    q := fmt.Sprintf(`
SELECT id, list_id, discount_id, prices
FROM sales
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

    args = append(args, perPage, offset)
    rows, err := run.QueryContext(ctx, q, args...)
    if err != nil {
        return saledom.PageResult{}, err
    }
    defer rows.Close()

    items := make([]saledom.Sale, 0, perPage)
    for rows.Next() {
        s, err := scanSale(rows)
        if err != nil {
            return saledom.PageResult{}, err
        }
        items = append(items, s)
    }
    if err := rows.Err(); err != nil {
        return saledom.PageResult{}, err
    }

    return saledom.PageResult{
        Items:      items,
        TotalCount: total,
        TotalPages: dbcommon.ComputeTotalPages(total, perPage),
        Page:       pageNum,
        PerPage:    perPage,
    }, nil
}

func (r *SaleRepositoryPG) Count(ctx context.Context, filter saledom.Filter) (int, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    where, args := buildSaleWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    var total int
    if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM sales "+whereSQL, args...).Scan(&total); err != nil {
        return 0, err
    }
    return total, nil
}

func (r *SaleRepositoryPG) Create(ctx context.Context, in saledom.CreateSaleInput) (*saledom.Sale, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    pricesJSON, err := json.Marshal(in.Prices)
    if err != nil {
        return nil, err
    }

    const q = `
INSERT INTO sales (
  id, list_id, discount_id, prices
) VALUES (
  gen_random_uuid()::text, $1, $2, $3::jsonb
)
RETURNING id, list_id, discount_id, prices
`
    row := run.QueryRowContext(ctx, q,
        strings.TrimSpace(in.ListID),
        dbcommon.ToDBText(in.DiscountID),
        string(pricesJSON),
    )
    s, err := scanSale(row)
    if err != nil {
        if dbcommon.IsUniqueViolation(err) {
            return nil, saledom.ErrConflict
        }
        return nil, err
    }
    return &s, nil
}

func (r *SaleRepositoryPG) Update(ctx context.Context, id string, in saledom.UpdateSaleInput) (*saledom.Sale, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    sets := []string{}
    args := []any{}
    i := 1

    if in.ListID != nil {
        sets = append(sets, fmt.Sprintf("list_id = $%d", i))
        args = append(args, strings.TrimSpace(*in.ListID))
        i++
    }
    if in.DiscountID != nil {
        v := strings.TrimSpace(*in.DiscountID)
        if v == "" {
            sets = append(sets, "discount_id = NULL")
        } else {
            sets = append(sets, fmt.Sprintf("discount_id = $%d", i))
            args = append(args, v)
            i++
        }
    }
    if in.Prices != nil {
        jb, err := json.Marshal(*in.Prices)
        if err != nil {
            return nil, err
        }
        sets = append(sets, fmt.Sprintf("prices = $%d::jsonb", i))
        args = append(args, string(jb))
        i++
    }

    if len(sets) == 0 {
        return r.GetByID(ctx, id)
    }

    args = append(args, strings.TrimSpace(id))
    q := fmt.Sprintf(`
UPDATE sales
SET %s
WHERE id = $%d
RETURNING id, list_id, discount_id, prices
`, strings.Join(sets, ", "), i)

    row := run.QueryRowContext(ctx, q, args...)
    s, err := scanSale(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, saledom.ErrNotFound
        }
        if dbcommon.IsUniqueViolation(err) {
            return nil, saledom.ErrConflict
        }
        return nil, err
    }
    return &s, nil
}

func (r *SaleRepositoryPG) Delete(ctx context.Context, id string) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    res, err := run.ExecContext(ctx, `DELETE FROM sales WHERE id = $1`, strings.TrimSpace(id))
    if err != nil {
        return err
    }
    aff, _ := res.RowsAffected()
    if aff == 0 {
        return saledom.ErrNotFound
    }
    return nil
}

func (r *SaleRepositoryPG) Reset(ctx context.Context) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    _, err := run.ExecContext(ctx, `DELETE FROM sales`)
    return err
}

// ========================
// Helpers
// ========================

func scanSale(s dbcommon.RowScanner) (saledom.Sale, error) {
    var (
        id, listID string
        discountNS sql.NullString
        pricesRaw  []byte
    )
    if err := s.Scan(&id, &listID, &discountNS, &pricesRaw); err != nil {
        return saledom.Sale{}, err
    }

    var prices []saledom.SalePrice
    if len(pricesRaw) > 0 {
        _ = json.Unmarshal(pricesRaw, &prices) // tolerant
    }

    var discountID *string
    if discountNS.Valid {
        v := strings.TrimSpace(discountNS.String)
        if v != "" {
            discountID = &v
        }
    }

    return saledom.Sale{
        ID:         strings.TrimSpace(id),
        ListID:     strings.TrimSpace(listID),
        DiscountID: discountID,
        Prices:     prices,
    }, nil
}

func buildSaleWhere(f saledom.Filter) ([]string, []any) {
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
    addEq("list_id", f.ListID)

    if f.HasDiscount != nil {
        if *f.HasDiscount {
            where = append(where, "discount_id IS NOT NULL")
        } else {
            where = append(where, "discount_id IS NULL")
        }
    }

    if v := strings.TrimSpace(f.ModelNumber); v != "" {
        // Support both "ModelNumber" (Go default JSON key) and "modelNumber"
        where = append(where, fmt.Sprintf(`
EXISTS (
  SELECT 1
  FROM jsonb_array_elements(prices) AS v(elem)
  WHERE (elem->>'ModelNumber' = $%d OR elem->>'modelNumber' = $%d)
)`, len(args)+1, len(args)+1))
        args = append(args, v)
    }

    // Any price within range across items of the sale
    if f.MinAnyPrice != nil {
        where = append(where, fmt.Sprintf(`
EXISTS (
  SELECT 1
  FROM jsonb_array_elements(prices) AS v(elem)
  WHERE (CASE
           WHEN elem ? 'Price' THEN (elem->>'Price')::int
           WHEN elem ? 'price' THEN (elem->>'price')::int
           ELSE NULL::int
         END) >= $%d
)`, len(args)+1))
        args = append(args, *f.MinAnyPrice)
    }
    if f.MaxAnyPrice != nil {
        where = append(where, fmt.Sprintf(`
EXISTS (
  SELECT 1
  FROM jsonb_array_elements(prices) AS v(elem)
  WHERE (CASE
           WHEN elem ? 'Price' THEN (elem->>'Price')::int
           WHEN elem ? 'price' THEN (elem->>'price')::int
           ELSE NULL::int
         END) <= $%d
)`, len(args)+1))
        args = append(args, *f.MaxAnyPrice)
    }

    return where, args
}

func buildSaleOrderBy(s saledom.Sort) string {
    col := strings.ToLower(strings.TrimSpace(string(s.Column)))
    switch col {
    case "id":
        col = "id"
    case "listid", "list_id":
        col = "list_id"
    default:
        return ""
    }
    dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
    if dir != "ASC" && dir != "DESC" {
        dir = "ASC"
    }
    return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}