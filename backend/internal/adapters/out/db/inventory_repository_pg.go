package db

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "time"

    dbcommon "narratives/internal/adapters/out/db/common"
    invdom "narratives/internal/domain/inventory"
)

// InventoryRepositoryPG implements inventory.Repository with PostgreSQL.
type InventoryRepositoryPG struct {
    DB *sql.DB
}

func NewInventoryRepositoryPG(db *sql.DB) *InventoryRepositoryPG {
    return &InventoryRepositoryPG{DB: db}
}

// =======================
// Queries
// =======================

func (r *InventoryRepositoryPG) GetByID(ctx context.Context, id string) (invdom.Inventory, error) {
    const q = `
SELECT
  id, connected_token, models, location, status,
  created_by, created_at, updated_by, updated_at
FROM inventories
WHERE id = $1
`
    row := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(id))
    inv, err := scanInventory(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return invdom.Inventory{}, invdom.ErrNotFound
        }
        return invdom.Inventory{}, err
    }
    return inv, nil
}

func (r *InventoryRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
    const q = `SELECT 1 FROM inventories WHERE id = $1`
    var one int
    err := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&one)
    if errors.Is(err, sql.ErrNoRows) {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return true, nil
}

func (r *InventoryRepositoryPG) Count(ctx context.Context, filter invdom.Filter) (int, error) {
    where, args := buildInventoryWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }
    var total int
    if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM inventories `+whereSQL, args...).Scan(&total); err != nil {
        return 0, err
    }
    return total, nil
}

func (r *InventoryRepositoryPG) List(ctx context.Context, filter invdom.Filter, sort invdom.Sort, page invdom.Page) (invdom.PageResult[invdom.Inventory], error) {
    where, args := buildInventoryWhere(filter)
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    orderBy := buildInventoryOrderBy(sort)
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
    countSQL := fmt.Sprintf("SELECT COUNT(*) FROM inventories %s", whereSQL)
    if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
        return invdom.PageResult[invdom.Inventory]{}, err
    }

    q := fmt.Sprintf(`
SELECT
  id, connected_token, models, location, status,
  created_by, created_at, updated_by, updated_at
FROM inventories
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

    args = append(args, perPage, offset)

    rows, err := r.DB.QueryContext(ctx, q, args...)
    if err != nil {
        return invdom.PageResult[invdom.Inventory]{}, err
    }
    defer rows.Close()

    var items []invdom.Inventory
    for rows.Next() {
        inv, err := scanInventory(rows)
        if err != nil {
            return invdom.PageResult[invdom.Inventory]{}, err
        }
        items = append(items, inv)
    }
    if err := rows.Err(); err != nil {
        return invdom.PageResult[invdom.Inventory]{}, err
    }

    totalPages := (total + perPage - 1) / perPage
    return invdom.PageResult[invdom.Inventory]{
        Items:      items,
        TotalCount: total,
        TotalPages: totalPages,
        Page:       number,
        PerPage:    perPage,
    }, nil
}

func (r *InventoryRepositoryPG) ListByCursor(ctx context.Context, filter invdom.Filter, _ invdom.Sort, cpage invdom.CursorPage) (invdom.CursorPageResult[invdom.Inventory], error) {
    where, args := buildInventoryWhere(filter)

    // Simple cursor by id ASC
    if after := strings.TrimSpace(cpage.After); after != "" {
        where = append(where, fmt.Sprintf("id > $%d", len(args)+1))
        args = append(args, after)
    }
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = "WHERE " + strings.Join(where, " AND ")
    }

    limit := cpage.Limit
    if limit <= 0 || limit > 200 {
        limit = 50
    }

    q := fmt.Sprintf(`
SELECT
  id, connected_token, models, location, status,
  created_by, created_at, updated_by, updated_at
FROM inventories
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

    args = append(args, limit+1)

    rows, err := r.DB.QueryContext(ctx, q, args...)
    if err != nil {
        return invdom.CursorPageResult[invdom.Inventory]{}, err
    }
    defer rows.Close()

    var items []invdom.Inventory
    var lastID string
    for rows.Next() {
        inv, err := scanInventory(rows)
        if err != nil {
            return invdom.CursorPageResult[invdom.Inventory]{}, err
        }
        items = append(items, inv)
        lastID = inv.ID
    }
    if err := rows.Err(); err != nil {
        return invdom.CursorPageResult[invdom.Inventory]{}, err
    }

    var next *string
    if len(items) > limit {
        items = items[:limit]
        next = &lastID
    }

    return invdom.CursorPageResult[invdom.Inventory]{
        Items:      items,
        NextCursor: next,
        Limit:      limit,
    }, nil
}

// =======================
// Mutations
// =======================

func (r *InventoryRepositoryPG) Create(ctx context.Context, inv invdom.Inventory) (invdom.Inventory, error) {
    modelsJSON, err := json.Marshal(inv.Models)
    if err != nil {
        return invdom.Inventory{}, err
    }
    const q = `
INSERT INTO inventories (
  id, connected_token, models, location, status,
  created_by, created_at, updated_by, updated_at
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9
)
RETURNING
  id, connected_token, models, location, status,
  created_by, created_at, updated_by, updated_at
`
    row := r.DB.QueryRowContext(ctx, q,
        strings.TrimSpace(inv.ID),
        dbcommon.ToDBText(inv.ConnectedToken),
        modelsJSON,
        strings.TrimSpace(inv.Location),
        strings.TrimSpace(string(inv.Status)),
        strings.TrimSpace(inv.CreatedBy),
        inv.CreatedAt.UTC(),
        strings.TrimSpace(inv.UpdatedBy),
        inv.UpdatedAt.UTC(),
    )
    out, err := scanInventory(row)
    if err != nil {
        if dbcommon.IsUniqueViolation(err) {
            return invdom.Inventory{}, invdom.ErrConflict
        }
        return invdom.Inventory{}, err
    }
    return out, nil
}

func (r *InventoryRepositoryPG) Update(ctx context.Context, id string, patch invdom.InventoryPatch) (invdom.Inventory, error) {
    sets := []string{}
    args := []any{}
    i := 1

    if patch.Models != nil {
        b, err := json.Marshal(*patch.Models)
        if err != nil {
            return invdom.Inventory{}, err
        }
        sets = append(sets, fmt.Sprintf("models = $%d", i))
        args = append(args, b)
        i++
    }
    if patch.Location != nil {
        sets = append(sets, fmt.Sprintf("location = $%d", i))
        args = append(args, strings.TrimSpace(*patch.Location))
        i++
    }
    if patch.Status != nil {
        sets = append(sets, fmt.Sprintf("status = $%d", i))
        args = append(args, strings.TrimSpace(string(*patch.Status)))
        i++
    }
    if patch.ConnectedToken != nil {
        v := strings.TrimSpace(*patch.ConnectedToken)
        if v == "" {
            sets = append(sets, "connected_token = NULL")
        } else {
            sets = append(sets, fmt.Sprintf("connected_token = $%d", i))
            args = append(args, v)
            i++
        }
    }
    if patch.UpdatedBy != nil {
        sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
        args = append(args, strings.TrimSpace(*patch.UpdatedBy))
        i++
    }
    // updated_at explicit or NOW()
    if patch.UpdatedAt != nil {
        sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
        args = append(args, patch.UpdatedAt.UTC())
        i++
    } else if len(sets) > 0 {
        sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
        args = append(args, time.Now().UTC())
        i++
    }

    if len(sets) == 0 {
        return r.GetByID(ctx, id)
    }

    args = append(args, strings.TrimSpace(id))
    q := fmt.Sprintf(`
UPDATE inventories
SET %s
WHERE id = $%d
RETURNING
  id, connected_token, models, location, status,
  created_by, created_at, updated_by, updated_at
`, strings.Join(sets, ", "), i)

    row := r.DB.QueryRowContext(ctx, q, args...)
    out, err := scanInventory(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return invdom.Inventory{}, invdom.ErrNotFound
        }
        return invdom.Inventory{}, err
    }
    return out, nil
}

func (r *InventoryRepositoryPG) Delete(ctx context.Context, id string) error {
    res, err := r.DB.ExecContext(ctx, `DELETE FROM inventories WHERE id = $1`, strings.TrimSpace(id))
    if err != nil {
        return err
    }
    aff, _ := res.RowsAffected()
    if aff == 0 {
        return invdom.ErrNotFound
    }
    return nil
}

func (r *InventoryRepositoryPG) Save(ctx context.Context, inv invdom.Inventory, _ *invdom.SaveOptions) (invdom.Inventory, error) {
    modelsJSON, err := json.Marshal(inv.Models)
    if err != nil {
        return invdom.Inventory{}, err
    }

    const q = `
INSERT INTO inventories (
  id, connected_token, models, location, status,
  created_by, created_at, updated_by, updated_at
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9
)
ON CONFLICT (id) DO UPDATE SET
  connected_token = EXCLUDED.connected_token,
  models          = EXCLUDED.models,
  location        = EXCLUDED.location,
  status          = EXCLUDED.status,
  created_by      = LEAST(inventories.created_by, EXCLUDED.created_by),
  created_at      = LEAST(inventories.created_at, EXCLUDED.created_at),
  updated_by      = EXCLUDED.updated_by,
  updated_at      = COALESCE(EXCLUDED.updated_at, NOW())
RETURNING
  id, connected_token, models, location, status,
  created_by, created_at, updated_by, updated_at
`
    row := r.DB.QueryRowContext(ctx, q,
        strings.TrimSpace(inv.ID),
        dbcommon.ToDBText(inv.ConnectedToken),
        modelsJSON,
        strings.TrimSpace(inv.Location),
        strings.TrimSpace(string(inv.Status)),
        strings.TrimSpace(inv.CreatedBy),
        inv.CreatedAt.UTC(),
        strings.TrimSpace(inv.UpdatedBy),
        inv.UpdatedAt.UTC(),
    )
    out, err := scanInventory(row)
    if err != nil {
        return invdom.Inventory{}, err
    }
    return out, nil
}

// =======================
// Helpers
// =======================

func scanInventory(s dbcommon.RowScanner) (invdom.Inventory, error) {
    var (
        idNS, connTokNS, locationNS, statusNS        sql.NullString
        createdByNS, updatedByNS                     sql.NullString
        modelsBytes                                  []byte
        createdAt, updatedAt                         time.Time
    )

    if err := s.Scan(
        &idNS, &connTokNS, &modelsBytes, &locationNS, &statusNS,
        &createdByNS, &createdAt, &updatedByNS, &updatedAt,
    ); err != nil {
        return invdom.Inventory{}, err
    }

    var models []invdom.InventoryModel
    if len(modelsBytes) > 0 {
        if err := json.Unmarshal(modelsBytes, &models); err != nil {
            return invdom.Inventory{}, err
        }
    } else {
        models = []invdom.InventoryModel{}
    }

    var connectedToken *string
    if connTokNS.Valid {
        v := strings.TrimSpace(connTokNS.String)
        if v != "" {
            connectedToken = &v
        }
    }

    return invdom.Inventory{
        ID:             strings.TrimSpace(idNS.String),
        ConnectedToken: connectedToken,
        Models:         models,
        Location:       strings.TrimSpace(locationNS.String),
        Status:         invdom.InventoryStatus(strings.TrimSpace(statusNS.String)),
        CreatedBy:      strings.TrimSpace(createdByNS.String),
        CreatedAt:      createdAt.UTC(),
        UpdatedBy:      strings.TrimSpace(updatedByNS.String),
        UpdatedAt:      updatedAt.UTC(),
    }, nil
}

func buildInventoryWhere(f invdom.Filter) ([]string, []any) {
    where := []string{}
    args := []any{}

    // Free text search (id, location, created_by, updated_by)
    if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
        where = append(where, fmt.Sprintf("(id ILIKE $%d OR location ILIKE $%d OR created_by ILIKE $%d OR updated_by ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1, len(args)+1))
        args = append(args, "%"+sq+"%")
    }

    // IDs IN (...)
    if len(f.IDs) > 0 {
        ph := make([]string, 0, len(f.IDs))
        for _, v := range f.IDs {
            v = strings.TrimSpace(v)
            if v == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, v)
        }
        if len(ph) > 0 {
            where = append(where, "id IN ("+strings.Join(ph, ",")+")")
        }
    }

    // ConnectedToken equals
    if f.ConnectedToken != nil {
        v := strings.TrimSpace(*f.ConnectedToken)
        if v == "" {
            where = append(where, "connected_token IS NULL")
        } else {
            where = append(where, fmt.Sprintf("connected_token = $%d", len(args)+1))
            args = append(args, v)
        }
    }

    // Location equals
    if f.Location != nil && strings.TrimSpace(*f.Location) != "" {
        where = append(where, fmt.Sprintf("location = $%d", len(args)+1))
        args = append(args, strings.TrimSpace(*f.Location))
    }

    // Status single and list
    if f.Status != nil && strings.TrimSpace(string(*f.Status)) != "" {
        where = append(where, fmt.Sprintf("status = $%d", len(args)+1))
        args = append(args, strings.TrimSpace(string(*f.Status)))
    }
    if len(f.Statuses) > 0 {
        ph := []string{}
        for _, st := range f.Statuses {
            v := strings.TrimSpace(string(st))
            if v == "" {
                continue
            }
            ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
            args = append(args, v)
        }
        if len(ph) > 0 {
            where = append(where, "status IN ("+strings.Join(ph, ",")+")")
        }
    }

    // By users
    if f.CreatedBy != nil && strings.TrimSpace(*f.CreatedBy) != "" {
        where = append(where, fmt.Sprintf("created_by = $%d", len(args)+1))
        args = append(args, strings.TrimSpace(*f.CreatedBy))
    }
    if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" {
        where = append(where, fmt.Sprintf("updated_by = $%d", len(args)+1))
        args = append(args, strings.TrimSpace(*f.UpdatedBy))
    }

    // Date ranges
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

func buildInventoryOrderBy(sort invdom.Sort) string {
    col := strings.ToLower(string(sort.Column))
    switch col {
    case "id":
        col = "id"
    case "location":
        col = "location"
    case "status":
        col = "status"
    case "connectedtoken", "connected_token":
        col = "connected_token"
    case "createdat", "created_at":
        col = "created_at"
    case "updatedat", "updated_at":
        col = "updated_at"
    default:
        return ""
    }
    dir := strings.ToUpper(string(sort.Order))
    if dir != "ASC" && dir != "DESC" {
        dir = "ASC"
    }
    return fmt.Sprintf("ORDER BY %s %s", col, dir)
}