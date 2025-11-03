package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"
    "time"

    dbcommon "narratives/internal/adapters/out/db/common"
    mrdom "narratives/internal/domain/mintRequest"
)

// PG implementation of mintRequest.RepositoryPort
type MintRequestRepositoryPG struct {
    DB *sql.DB
}

func NewMintRequestRepositoryPG(db *sql.DB) *MintRequestRepositoryPG {
    return &MintRequestRepositoryPG{DB: db}
}

// =======================
// RepositoryPort impl
// =======================

func (r *MintRequestRepositoryPG) GetByID(ctx context.Context, id string) (*mrdom.MintRequest, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    const q = `
SELECT
  id, token_blueprint_id, production_id, mint_quantity, burn_date, status,
  requested_by, requested_at, minted_at,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM mint_requests
WHERE id = $1`
    row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
    mr, err := scanMintRequest(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, mrdom.ErrNotFound
        }
        return nil, err
    }
    return &mr, nil
}

func (r *MintRequestRepositoryPG) List(ctx context.Context, filter mrdom.Filter, sort mrdom.Sort, page mrdom.Page) (mrdom.PageResult, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    where, args := buildMintRequestWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    orderBy := buildMintRequestOrderBy(sort)
    if orderBy == "" {
        orderBy = "ORDER BY created_at DESC, id DESC"
    }

    pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

    var total int
    if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM mint_requests "+whereSQL, args...).Scan(&total); err != nil {
        return mrdom.PageResult{}, err
    }

    q := fmt.Sprintf(`
SELECT
  id, token_blueprint_id, production_id, mint_quantity, burn_date, status,
  requested_by, requested_at, minted_at,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM mint_requests
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

    args = append(args, perPage, offset)

    rows, err := run.QueryContext(ctx, q, args...)
    if err != nil {
        return mrdom.PageResult{}, err
    }
    defer rows.Close()

    items := make([]mrdom.MintRequest, 0, perPage)
    for rows.Next() {
        mr, err := scanMintRequest(rows)
        if err != nil {
            return mrdom.PageResult{}, err
        }
        items = append(items, mr)
    }
    if err := rows.Err(); err != nil {
        return mrdom.PageResult{}, err
    }

    return mrdom.PageResult{
        Items:      items,
        TotalCount: total,
        TotalPages: dbcommon.ComputeTotalPages(total, perPage),
        Page:       pageNum,
        PerPage:    perPage,
    }, nil
}

func (r *MintRequestRepositoryPG) Create(ctx context.Context, in mrdom.CreateMintRequest) (*mrdom.MintRequest, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    const q = `
INSERT INTO mint_requests (
  id, token_blueprint_id, production_id, mint_quantity, burn_date, status,
  requested_by, requested_at, minted_at,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  gen_random_uuid(), $1, $2, $3, $4::date, 'planning',
  NULL, NULL, NULL,
  NOW(), $5, NOW(), $5, NULL, NULL
)
RETURNING
  id, token_blueprint_id, production_id, mint_quantity, burn_date, status,
  requested_by, requested_at, minted_at,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
    row := run.QueryRowContext(ctx, q,
        strings.TrimSpace(in.TokenBlueprintID),
        strings.TrimSpace(in.ProductionID),
        in.MintQuantity,
        dbcommon.ToDBTime(in.BurnDate), // cast to ::date in SQL
        strings.TrimSpace(in.CreatedBy),
    )
    mr, err := scanMintRequest(row)
    if err != nil {
        return nil, err
    }
    return &mr, nil
}

func (r *MintRequestRepositoryPG) Update(ctx context.Context, id string, patch mrdom.UpdateMintRequest) (*mrdom.MintRequest, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    id = strings.TrimSpace(id)
    if id == "" {
        return nil, mrdom.ErrNotFound
    }

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
    setInt := func(col string, p *int) {
        if p != nil {
            sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
            args = append(args, *p)
            i++
        }
    }
    setStatus := func(p *mrdom.MintRequestStatus) {
        if p != nil {
            sets = append(sets, fmt.Sprintf("status = $%d", i))
            args = append(args, strings.TrimSpace(string(*p)))
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
    setDate := func(col string, p *time.Time) {
        if p != nil {
            sets = append(sets, fmt.Sprintf("%s = $%d::date", col, i))
            args = append(args, p.UTC())
            i++
        }
    }

    // Updatable fields
    setStatus(patch.Status)
    setText("token_blueprint_id", patch.TokenBlueprintID)
    setInt("mint_quantity", patch.MintQuantity)
    setDate("burn_date", patch.BurnDate)

    setText("requested_by", patch.RequestedBy)
    setTime("requested_at", patch.RequestedAt)
    setTime("minted_at", patch.MintedAt)

    // Soft delete fields (must be paired per DDL constraint)
    setTime("deleted_at", patch.DeletedAt)
    setText("deleted_by", patch.DeletedBy)

    // Always update audit
    now := time.Now().UTC()
    sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
    args = append(args, now)
    i++
    sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
    args = append(args, strings.TrimSpace(patch.UpdatedBy))
    i++

    if len(sets) == 0 {
        // nothing to update, return current
        return r.GetByID(ctx, id)
    }

    args = append(args, id)
    q := fmt.Sprintf(`
UPDATE mint_requests
SET %s
WHERE id = $%d
RETURNING
  id, token_blueprint_id, production_id, mint_quantity, burn_date, status,
  requested_by, requested_at, minted_at,
  created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(sets, ", "), i)

    row := run.QueryRowContext(ctx, q, args...)
    mr, err := scanMintRequest(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, mrdom.ErrNotFound
        }
        return nil, err
    }
    return &mr, nil
}

func (r *MintRequestRepositoryPG) Delete(ctx context.Context, id string) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    res, err := run.ExecContext(ctx, `DELETE FROM mint_requests WHERE id = $1`, strings.TrimSpace(id))
    if err != nil {
        return err
    }
    aff, _ := res.RowsAffected()
    if aff == 0 {
        return mrdom.ErrNotFound
    }
    return nil
}

func (r *MintRequestRepositoryPG) Count(ctx context.Context, filter mrdom.Filter) (int, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    where, args := buildMintRequestWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }
    var total int
    if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM mint_requests "+whereSQL, args...).Scan(&total); err != nil {
        return 0, err
    }
    return total, nil
}

func (r *MintRequestRepositoryPG) Reset(ctx context.Context) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    _, err := run.ExecContext(ctx, `DELETE FROM mint_requests`)
    return err
}

// =======================
// Helpers
// =======================

func scanMintRequest(s dbcommon.RowScanner) (mrdom.MintRequest, error) {
    var (
        id, tokenBlueprintID, productionID, status string
        mintQuantity                               int
        burnDateNS                                 sql.NullTime
        requestedByNS                              sql.NullString
        requestedAtNS, mintedAtNS                  sql.NullTime
        createdAt, updatedAt                       time.Time
        createdBy, updatedBy                       string
        deletedAtNS                                sql.NullTime
        deletedByNS                                sql.NullString
    )
    if err := s.Scan(
        &id, &tokenBlueprintID, &productionID, &mintQuantity, &burnDateNS, &status,
        &requestedByNS, &requestedAtNS, &mintedAtNS,
        &createdAt, &createdBy, &updatedAt, &updatedBy, &deletedAtNS, &deletedByNS,
    ); err != nil {
        return mrdom.MintRequest{}, err
    }

    toTimePtr := func(nt sql.NullTime) *time.Time {
        if nt.Valid {
            t := nt.Time.UTC()
            return &t
        }
        return nil
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

    return mrdom.MintRequest{
        ID:               strings.TrimSpace(id),
        TokenBlueprintID: strings.TrimSpace(tokenBlueprintID),
        ProductionID:     strings.TrimSpace(productionID),
        MintQuantity:     mintQuantity,
        BurnDate:         toTimePtr(burnDateNS),
        Status:           mrdom.MintRequestStatus(strings.TrimSpace(status)),
        RequestedBy:      toStrPtr(requestedByNS),
        RequestedAt:      toTimePtr(requestedAtNS),
        MintedAt:         toTimePtr(mintedAtNS),

        CreatedAt: createdAt.UTC(),
        CreatedBy: strings.TrimSpace(createdBy),
        UpdatedAt: updatedAt.UTC(),
        UpdatedBy: strings.TrimSpace(updatedBy),
        DeletedAt: toTimePtr(deletedAtNS),
        DeletedBy: toStrPtr(deletedByNS),
    }, nil
}

func buildMintRequestWhere(f mrdom.Filter) ([]string, []any) {
    where := []string{}
    args := []any{}

    if v := strings.TrimSpace(f.ProductionID); v != "" {
        where = append(where, fmt.Sprintf("production_id = $%d", len(args)+1))
        args = append(args, v)
    }
    if v := strings.TrimSpace(f.TokenBlueprintID); v != "" {
        where = append(where, fmt.Sprintf("token_blueprint_id = $%d", len(args)+1))
        args = append(args, v)
    }
    if len(f.Statuses) > 0 {
        ph := make([]string, 0, len(f.Statuses))
        for range f.Statuses {
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            // append value below in same loop
        }
        // Actually append args in second loop to keep order
        argsCap := len(args)
        for _, st := range f.Statuses {
            args = append(args, strings.TrimSpace(string(st)))
        }
        // build placeholders based on final arg positions
        ph = ph[:0]
        for i := range f.Statuses {
            ph = append(ph, fmt.Sprintf("$%d", argsCap+i+1))
        }
        where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(ph, ",")))
    }
    if v := strings.TrimSpace(f.RequestedBy); v != "" {
        where = append(where, fmt.Sprintf("requested_by = $%d", len(args)+1))
        args = append(args, v)
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
    if f.RequestFrom != nil {
        where = append(where, fmt.Sprintf("(requested_at IS NOT NULL AND requested_at >= $%d)", len(args)+1))
        args = append(args, f.RequestFrom.UTC())
    }
    if f.RequestTo != nil {
        where = append(where, fmt.Sprintf("(requested_at IS NOT NULL AND requested_at < $%d)", len(args)+1))
        args = append(args, f.RequestTo.UTC())
    }
    if f.MintedFrom != nil {
        where = append(where, fmt.Sprintf("(minted_at IS NOT NULL AND minted_at >= $%d)", len(args)+1))
        args = append(args, f.MintedFrom.UTC())
    }
    if f.MintedTo != nil {
        where = append(where, fmt.Sprintf("(minted_at IS NOT NULL AND minted_at < $%d)", len(args)+1))
        args = append(args, f.MintedTo.UTC())
    }
    if f.BurnFrom != nil {
        where = append(where, fmt.Sprintf("(burn_date IS NOT NULL AND burn_date >= $%d)", len(args)+1))
        args = append(args, f.BurnFrom.UTC())
    }
    if f.BurnTo != nil {
        where = append(where, fmt.Sprintf("(burn_date IS NOT NULL AND burn_date < $%d)", len(args)+1))
        args = append(args, f.BurnTo.UTC())
    }

    if f.Deleted != nil {
        if *f.Deleted {
            where = append(where, "deleted_at IS NOT NULL")
        } else {
            where = append(where, "deleted_at IS NULL")
        }
    }

    return where, args
}

func buildMintRequestOrderBy(sort mrdom.Sort) string {
    col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
    switch col {
    case "createdat", "created_at":
        col = "created_at"
    case "updatedat", "updated_at":
        col = "updated_at"
    case "burndate", "burn_date":
        col = "burn_date"
    case "mintedat", "minted_at":
        col = "minted_at"
    case "requestedat", "requested_at":
        col = "requested_at"
    default:
        return ""
    }
    dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
    if dir != "ASC" && dir != "DESC" {
        dir = "DESC"
    }
    // Secondary sort by id for stability
    return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}