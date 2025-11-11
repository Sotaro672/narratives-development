// backend\internal\adapters\out\firestore\list_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	ldom "narratives/internal/domain/list"
)

// Inventory adapter-style runner (query/exec abstraction)
type listRunner interface {
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
	ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
}

// ListRepositoryPG implements list.Repository using PostgreSQL.
type ListRepositoryPG struct {
	DB *sql.DB
}

func NewListRepositoryPG(db *sql.DB) *ListRepositoryPG {
	return &ListRepositoryPG{DB: db}
}

// =======================
// Queries
// =======================

func (r *ListRepositoryPG) GetByID(ctx context.Context, id string) (ldom.List, error) {
	const q = `
SELECT
  id, inventory_id, status, assignee_id, image_id, description,
  created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
FROM lists
WHERE id = $1
`
	row := r.DB.QueryRowContext(ctx, q, strings.TrimSpace(id))
	out, err := scanList(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ldom.List{}, ldom.ErrNotFound
		}
		return ldom.List{}, err
	}

	prices, err := r.fetchPrices(ctx, []string{out.ID})
	if err != nil {
		return ldom.List{}, err
	}
	out.Prices = prices[out.ID]
	return out, nil
}

func (r *ListRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM lists WHERE id = $1`
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

func (r *ListRepositoryPG) Count(ctx context.Context, filter ldom.Filter) (int, error) {
	where, args := buildListWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM lists l `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ListRepositoryPG) List(ctx context.Context, filter ldom.Filter, sort ldom.Sort, page ldom.Page) (ldom.PageResult[ldom.List], error) {
	where, args := buildListWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	orderBy := buildListOrderBy(sort)
	if orderBy == "" {
		// updated_at is nullable; fallback to created_at when null
		orderBy = "ORDER BY COALESCE(updated_at, created_at) DESC, id DESC"
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM lists l %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return ldom.PageResult[ldom.List]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id, inventory_id, status, assignee_id, image_id, description,
  created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
FROM lists l
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return ldom.PageResult[ldom.List]{}, err
	}
	defer rows.Close()

	var items []ldom.List
	var ids []string
	for rows.Next() {
		it, err := scanList(rows)
		if err != nil {
			return ldom.PageResult[ldom.List]{}, err
		}
		items = append(items, it)
		ids = append(ids, it.ID)
	}
	if err := rows.Err(); err != nil {
		return ldom.PageResult[ldom.List]{}, err
	}

	// fetch prices in bulk
	pmap, err := r.fetchPrices(ctx, ids)
	if err != nil {
		return ldom.PageResult[ldom.List]{}, err
	}
	for i := range items {
		items[i].Prices = pmap[items[i].ID]
	}

	totalPages := (total + perPage - 1) / perPage
	return ldom.PageResult[ldom.List]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *ListRepositoryPG) ListByCursor(ctx context.Context, filter ldom.Filter, _ ldom.Sort, cpage ldom.CursorPage) (ldom.CursorPageResult[ldom.List], error) {
	where, args := buildListWhere(filter)

	// Simple cursor by id ASC
	if after := strings.TrimSpace(cpage.After); after != "" {
		where = append(where, fmt.Sprintf("l.id > $%d", len(args)+1))
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
  id, inventory_id, status, assignee_id, image_id, description,
  created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
FROM lists l
%s
ORDER BY l.id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return ldom.CursorPageResult[ldom.List]{}, err
	}
	defer rows.Close()

	var items []ldom.List
	var lastID string
	var ids []string
	for rows.Next() {
		it, err := scanList(rows)
		if err != nil {
			return ldom.CursorPageResult[ldom.List]{}, err
		}
		items = append(items, it)
		ids = append(ids, it.ID)
		lastID = it.ID
	}
	if err := rows.Err(); err != nil {
		return ldom.CursorPageResult[ldom.List]{}, err
	}

	// prices
	pmap, err := r.fetchPrices(ctx, ids)
	if err != nil {
		return ldom.CursorPageResult[ldom.List]{}, err
	}
	for i := range items {
		items[i].Prices = pmap[items[i].ID]
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return ldom.CursorPageResult[ldom.List]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// =======================
// Mutations
// =======================

func (r *ListRepositoryPG) Create(ctx context.Context, l ldom.List) (ldom.List, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return ldom.List{}, err
	}
	defer tx.Rollback()

	const q = `
INSERT INTO lists (
  id, inventory_id, status, assignee_id, image_id, description,
  created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,$11,$12
)
`
	_, err = tx.ExecContext(ctx, q,
		strings.TrimSpace(l.ID),
		strings.TrimSpace(l.InventoryID),
		strings.TrimSpace(string(l.Status)),
		strings.TrimSpace(l.AssigneeID),
		strings.TrimSpace(l.ImageID),
		strings.TrimSpace(l.Description),
		strings.TrimSpace(l.CreatedBy),
		l.CreatedAt.UTC(),
		dbcommon.ToDBText(l.UpdatedBy),
		dbcommon.ToDBTime(l.UpdatedAt),
		dbcommon.ToDBTime(l.DeletedAt),
		dbcommon.ToDBText(l.DeletedBy),
	)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return ldom.List{}, ldom.ErrConflict
		}
		return ldom.List{}, err
	}

	if len(l.Prices) > 0 {
		if err := bulkInsertListPrices(ctx, tx, l.ID, l.Prices); err != nil {
			return ldom.List{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return ldom.List{}, err
	}
	return r.GetByID(ctx, l.ID)
}

func (r *ListRepositoryPG) Update(ctx context.Context, id string, patch ldom.ListPatch) (ldom.List, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return ldom.List{}, err
	}
	defer tx.Rollback()

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
	if patch.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", i))
		args = append(args, strings.TrimSpace(string(*patch.Status)))
		i++
	}
	setStr("assignee_id", patch.AssigneeID)
	setStr("image_id", patch.ImageID) // repository_port.go に合わせて image_id を更新
	setStr("description", patch.Description)

	// Optional by/updater/deleted
	if patch.UpdatedBy != nil {
		sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.UpdatedBy))
		i++
	}
	if patch.DeletedAt != nil {
		sets = append(sets, fmt.Sprintf("deleted_at = $%d", i))
		args = append(args, dbcommon.ToDBTime(patch.DeletedAt))
		i++
	}
	if patch.DeletedBy != nil {
		sets = append(sets, fmt.Sprintf("deleted_by = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.DeletedBy))
		i++
	}

	// updated_at explicit or NOW() if any sets (including prices)
	pricesWillChange := patch.Prices != nil
	if patch.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, patch.UpdatedAt.UTC())
		i++
	} else if len(sets) > 0 || pricesWillChange {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().UTC())
		i++
	}

	// Apply table updates if any
	if len(sets) > 0 {
		args = append(args, strings.TrimSpace(id))
		q := fmt.Sprintf(`
UPDATE lists
SET %s
WHERE id = $%d
`, strings.Join(sets, ", "), i)
		res, err := tx.ExecContext(ctx, q, args...)
		if err != nil {
			return ldom.List{}, err
		}
		aff, _ := res.RowsAffected()
		if aff == 0 {
			return ldom.List{}, ldom.ErrNotFound
		}
	}

	// Replace prices if provided
	if patch.Prices != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM list_prices WHERE list_id = $1`, strings.TrimSpace(id)); err != nil {
			return ldom.List{}, err
		}
		if len(*patch.Prices) > 0 {
			if err := bulkInsertListPrices(ctx, tx, strings.TrimSpace(id), *patch.Prices); err != nil {
				return ldom.List{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return ldom.List{}, err
	}
	return r.GetByID(ctx, strings.TrimSpace(id))
}

func (r *ListRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM lists WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ldom.ErrNotFound
	}
	return nil
}

func (r *ListRepositoryPG) Save(ctx context.Context, l ldom.List, _ *ldom.SaveOptions) (ldom.List, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return ldom.List{}, err
	}
	defer tx.Rollback()

	const q = `
INSERT INTO lists (
  id, inventory_id, status, assignee_id, image_id, description,
  created_by, created_at, updated_by, updated_at, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,$11,$12
)
ON CONFLICT (id) DO UPDATE SET
  inventory_id = EXCLUDED.inventory_id,
  status       = EXCLUDED.status,
  assignee_id  = EXCLUDED.assignee_id,
  image_id     = EXCLUDED.image_id,
  description  = EXCLUDED.description,
  created_by   = LEAST(lists.created_by, EXCLUDED.created_by),
  created_at   = LEAST(lists.created_at, EXCLUDED.created_at),
  updated_by   = EXCLUDED.updated_by,
  updated_at   = COALESCE(EXCLUDED.updated_at, lists.updated_at),
  deleted_at   = EXCLUDED.deleted_at,
  deleted_by   = EXCLUDED.deleted_by
`
	_, err = tx.ExecContext(ctx, q,
		strings.TrimSpace(l.ID),
		strings.TrimSpace(l.InventoryID),
		strings.TrimSpace(string(l.Status)),
		strings.TrimSpace(l.AssigneeID),
		strings.TrimSpace(l.ImageID),
		strings.TrimSpace(l.Description),
		strings.TrimSpace(l.CreatedBy),
		l.CreatedAt.UTC(),
		dbcommon.ToDBText(l.UpdatedBy),
		dbcommon.ToDBTime(l.UpdatedAt),
		dbcommon.ToDBTime(l.DeletedAt),
		dbcommon.ToDBText(l.DeletedBy),
	)
	if err != nil {
		return ldom.List{}, err
	}

	// Replace prices with provided set
	if _, err := tx.ExecContext(ctx, `DELETE FROM list_prices WHERE list_id = $1`, strings.TrimSpace(l.ID)); err != nil {
		return ldom.List{}, err
	}
	if len(l.Prices) > 0 {
		if err := bulkInsertListPrices(ctx, tx, l.ID, l.Prices); err != nil {
			return ldom.List{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return ldom.List{}, err
	}
	return r.GetByID(ctx, l.ID)
}

// =======================
// Helpers
// =======================

func (r *ListRepositoryPG) fetchPrices(ctx context.Context, ids []string) (map[string][]ldom.ListPrice, error) {
	out := make(map[string][]ldom.ListPrice, len(ids))
	clean := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			clean = append(clean, id)
		}
	}
	if len(clean) == 0 {
		return out, nil
	}
	ph := make([]string, len(clean))
	args := make([]any, len(clean))
	for i := range clean {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = clean[i]
	}

	q := fmt.Sprintf(`
SELECT list_id, model_number, price
FROM list_prices
WHERE list_id IN (%s)
ORDER BY list_id ASC, model_number ASC
`, strings.Join(ph, ","))

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var listID, modelNumber string
		var price int
		if err := rows.Scan(&listID, &modelNumber, &price); err != nil {
			return nil, err
		}
		out[listID] = append(out[listID], ldom.ListPrice{
			ModelNumber: strings.TrimSpace(modelNumber),
			Price:       price,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func bulkInsertListPrices(ctx context.Context, ex listRunner, listID string, prices []ldom.ListPrice) error {
	if len(prices) == 0 {
		return nil
	}
	// aggregate and clean
	agg := aggregateListPrices(prices)

	sb := strings.Builder{}
	sb.WriteString(`INSERT INTO list_prices (list_id, model_number, price) VALUES `)
	args := make([]any, 0, len(agg)*3)
	for i, p := range agg {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d)", len(args)+1, len(args)+2, len(args)+3))
		args = append(args, strings.TrimSpace(listID), strings.TrimSpace(p.ModelNumber), p.Price)
	}
	_, err := ex.ExecContext(ctx, sb.String(), args...)
	return err
}

func aggregateListPrices(prices []ldom.ListPrice) []ldom.ListPrice {
	// last write wins per modelNumber
	tmp := make(map[string]int, len(prices))
	order := make([]string, 0, len(prices))
	for _, p := range prices {
		mn := strings.TrimSpace(p.ModelNumber)
		if mn == "" {
			continue
		}
		if _, ok := tmp[mn]; !ok {
			order = append(order, mn)
		}
		tmp[mn] = p.Price
	}
	out := make([]ldom.ListPrice, 0, len(tmp))
	for _, mn := range order {
		out = append(out, ldom.ListPrice{ModelNumber: mn, Price: tmp[mn]})
	}
	return out
}

func scanList(s dbcommon.RowScanner) (ldom.List, error) {
	var (
		idNS, invIDNS, statusNS, assigneeNS, imageIDNS, descNS sql.NullString
		createdByNS, updatedByNS, deletedByNS                  sql.NullString
		createdAt                                              time.Time
		updatedAtNS, deletedAtNS                               sql.NullTime
	)
	if err := s.Scan(
		&idNS, &invIDNS, &statusNS, &assigneeNS, &imageIDNS, &descNS,
		&createdByNS, &createdAt, &updatedByNS, &updatedAtNS, &deletedAtNS, &deletedByNS,
	); err != nil {
		return ldom.List{}, err
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
	toPtrTime := func(nt sql.NullTime) *time.Time {
		if nt.Valid {
			t := nt.Time.UTC()
			return &t
		}
		return nil
	}

	return ldom.List{
		ID:          strings.TrimSpace(idNS.String),
		InventoryID: strings.TrimSpace(invIDNS.String),
		Status:      ldom.ListStatus(strings.TrimSpace(statusNS.String)),
		AssigneeID:  strings.TrimSpace(assigneeNS.String),
		ImageID:     strings.TrimSpace(imageIDNS.String),
		Description: strings.TrimSpace(descNS.String),
		Prices:      nil, // filled by caller
		CreatedBy:   strings.TrimSpace(createdByNS.String),
		CreatedAt:   createdAt.UTC(),
		UpdatedBy:   toPtrStr(updatedByNS),
		UpdatedAt:   toPtrTime(updatedAtNS),
		DeletedAt:   toPtrTime(deletedAtNS),
		DeletedBy:   toPtrStr(deletedByNS),
	}, nil
}

func buildListWhere(f ldom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// Free text search across some columns
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(where, fmt.Sprintf("(l.id ILIKE $%d OR l.description ILIKE $%d OR l.image_id ILIKE $%d OR l.assignee_id ILIKE $%d OR l.inventory_id ILIKE $%d OR l.created_by ILIKE $%d OR COALESCE(l.updated_by,'') ILIKE $%d OR COALESCE(l.deleted_by,'') ILIKE $%d)",
			len(args)+1, len(args)+1, len(args)+1, len(args)+1, len(args)+1, len(args)+1, len(args)+1, len(args)+1))
		args = append(args, "%"+sq+"%")
	}

	// IDs IN (...)
	if len(f.IDs) > 0 {
		ph := []string{}
		for _, v := range f.IDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "l.id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// Inventory ID equals / IN
	if f.InventoryID != nil && strings.TrimSpace(*f.InventoryID) != "" {
		where = append(where, fmt.Sprintf("l.inventory_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.InventoryID))
	}
	if len(f.InventoryIDs) > 0 {
		ph := []string{}
		for _, v := range f.InventoryIDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "l.inventory_id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// Assignee
	if f.AssigneeID != nil && strings.TrimSpace(*f.AssigneeID) != "" {
		where = append(where, fmt.Sprintf("l.assignee_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.AssigneeID))
	}

	// Status single and list
	if f.Status != nil && strings.TrimSpace(string(*f.Status)) != "" {
		where = append(where, fmt.Sprintf("l.status = $%d", len(args)+1))
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
			where = append(where, "l.status IN ("+strings.Join(ph, ",")+")")
		}
	}

	// Price-related filters via EXISTS on list_prices
	priceFilters := (len(f.ModelNumbers) > 0) || f.MinPrice != nil || f.MaxPrice != nil
	if priceFilters {
		conds := []string{"lp.list_id = l.id"}
		if len(f.ModelNumbers) > 0 {
			ph := []string{}
			for _, mn := range f.ModelNumbers {
				mn = strings.TrimSpace(mn)
				if mn == "" {
					continue
				}
				ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
				args = append(args, mn)
			}
			if len(ph) > 0 {
				conds = append(conds, "lp.model_number IN ("+strings.Join(ph, ",")+")")
			}
		}
		if f.MinPrice != nil {
			conds = append(conds, fmt.Sprintf("lp.price >= $%d", len(args)+1))
			args = append(args, *f.MinPrice)
		}
		if f.MaxPrice != nil {
			conds = append(conds, fmt.Sprintf("lp.price <= $%d", len(args)+1))
			args = append(args, *f.MaxPrice)
		}
		where = append(where, "EXISTS (SELECT 1 FROM list_prices lp WHERE "+strings.Join(conds, " AND ")+")")
	}

	// Date ranges
	if f.CreatedFrom != nil {
		where = append(where, fmt.Sprintf("l.created_at >= $%d", len(args)+1))
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedTo != nil {
		where = append(where, fmt.Sprintf("l.created_at < $%d", len(args)+1))
		args = append(args, f.CreatedTo.UTC())
	}
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("(l.updated_at IS NOT NULL AND l.updated_at >= $%d)", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("(l.updated_at IS NOT NULL AND l.updated_at < $%d)", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}
	if f.DeletedFrom != nil {
		where = append(where, fmt.Sprintf("(l.deleted_at IS NOT NULL AND l.deleted_at >= $%d)", len(args)+1))
		args = append(args, f.DeletedFrom.UTC())
	}
	if f.DeletedTo != nil {
		where = append(where, fmt.Sprintf("(l.deleted_at IS NOT NULL AND l.deleted_at < $%d)", len(args)+1))
		args = append(args, f.DeletedTo.UTC())
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			where = append(where, "l.deleted_at IS NOT NULL")
		} else {
			where = append(where, "l.deleted_at IS NULL")
		}
	}

	return where, args
}

func buildListOrderBy(sort ldom.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "id":
		col = "l.id"
	case "inventoryid", "inventory_id":
		col = "l.inventory_id"
	case "status":
		col = "l.status"
	case "assigneeid", "assignee_id":
		col = "l.assignee_id"
	case "imageid", "image_id", "imageurl", "image_url": // accept legacy aliases
		col = "l.image_id"
	case "createdat", "created_at":
		col = "l.created_at"
	case "updatedat", "updated_at":
		col = "l.updated_at"
	case "deletedat", "deleted_at":
		col = "l.deleted_at"
	default:
		return ""
	}
	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}
