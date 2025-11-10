package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	ddom "narratives/internal/domain/discount"
)

// Repository implementation for Discount (PostgreSQL)
type DiscountRepositoryPG struct {
	DB *sql.DB
}

func NewDiscountRepositoryPG(db *sql.DB) *DiscountRepositoryPG {
	return &DiscountRepositoryPG{DB: db}
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// =======================
// Queries
// =======================

func (r *DiscountRepositoryPG) List(ctx context.Context, filter ddom.Filter, sort ddom.Sort, page ddom.Page) (ddom.PageResult[ddom.Discount], error) {
	where, args := buildDiscountWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildDiscountOrderBy(sort)
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM discounts d %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return ddom.PageResult[ddom.Discount]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  d.id, d.list_id, d.description, d.discounted_by, d.discounted_at, d.updated_by, d.updated_at
FROM discounts d
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return ddom.PageResult[ddom.Discount]{}, err
	}
	defer rows.Close()

	var items []ddom.Discount
	var ids []string
	for rows.Next() {
		d, err := scanDiscountRow(rows)
		if err != nil {
			return ddom.PageResult[ddom.Discount]{}, err
		}
		items = append(items, d)
		ids = append(ids, d.ID)
	}
	if err := rows.Err(); err != nil {
		return ddom.PageResult[ddom.Discount]{}, err
	}

	// Load items for all discounts
	if len(ids) > 0 {
		im, err := r.loadDiscountItems(ctx, r.DB, ids)
		if err != nil {
			return ddom.PageResult[ddom.Discount]{}, err
		}
		for i := range items {
			if v, ok := im[items[i].ID]; ok {
				items[i].Discounts = v
			}
		}
	}

	totalPages := (total + perPage - 1) / perPage
	return ddom.PageResult[ddom.Discount]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *DiscountRepositoryPG) ListByCursor(ctx context.Context, filter ddom.Filter, _ ddom.Sort, cpage ddom.CursorPage) (ddom.CursorPageResult[ddom.Discount], error) {
	where, args := buildDiscountWhere(filter)
	if after := strings.TrimSpace(cpage.After); after != "" {
		where = append(where, fmt.Sprintf("d.id > $%d", len(args)+1))
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
  d.id, d.list_id, d.description, d.discounted_by, d.discounted_at, d.updated_by, d.updated_at
FROM discounts d
%s
ORDER BY d.id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return ddom.CursorPageResult[ddom.Discount]{}, err
	}
	defer rows.Close()

	var items []ddom.Discount
	var ids []string
	var lastID string
	for rows.Next() {
		d, err := scanDiscountRow(rows)
		if err != nil {
			return ddom.CursorPageResult[ddom.Discount]{}, err
		}
		items = append(items, d)
		ids = append(ids, d.ID)
		lastID = d.ID
	}
	if err := rows.Err(); err != nil {
		return ddom.CursorPageResult[ddom.Discount]{}, err
	}

	if len(ids) > 0 {
		im, err := r.loadDiscountItems(ctx, r.DB, ids)
		if err != nil {
			return ddom.CursorPageResult[ddom.Discount]{}, err
		}
		for i := range items {
			if v, ok := im[items[i].ID]; ok {
				items[i].Discounts = v
			}
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return ddom.CursorPageResult[ddom.Discount]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *DiscountRepositoryPG) GetByID(ctx context.Context, id string) (ddom.Discount, error) {
	const q = `
SELECT
  d.id, d.list_id, d.description, d.discounted_by, d.discounted_at, d.updated_by, d.updated_at
FROM discounts d
WHERE d.id = $1
`
	row := r.DB.QueryRowContext(ctx, q, id)
	out, err := scanDiscountRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ddom.Discount{}, ddom.ErrNotFound
		}
		return ddom.Discount{}, err
	}

	im, err := r.loadDiscountItems(ctx, r.DB, []string{out.ID})
	if err != nil {
		return ddom.Discount{}, err
	}
	out.Discounts = im[out.ID]
	return out, nil
}

func (r *DiscountRepositoryPG) GetByListID(ctx context.Context, listID string, sort ddom.Sort, page ddom.Page) (ddom.PageResult[ddom.Discount], error) {
	f := ddom.Filter{ListID: &listID}
	return r.List(ctx, f, sort, page)
}

func (r *DiscountRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM discounts WHERE id = $1`
	var one int
	err := r.DB.QueryRowContext(ctx, q, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *DiscountRepositoryPG) Count(ctx context.Context, filter ddom.Filter) (int, error) {
	where, args := buildDiscountWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM discounts d `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// =======================
// Mutations
// =======================

func (r *DiscountRepositoryPG) Create(ctx context.Context, d ddom.Discount) (ddom.Discount, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return ddom.Discount{}, err
	}
	defer tx.Rollback()

	const q = `
INSERT INTO discounts (
  id, list_id, description, discounted_by, discounted_at, updated_by, updated_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7
)
RETURNING id, list_id, description, discounted_by, discounted_at, updated_by, updated_at
`
	row := tx.QueryRowContext(ctx, q,
		strings.TrimSpace(d.ID),
		strings.TrimSpace(d.ListID),
		dbcommon.ToDBText(d.Description),
		strings.TrimSpace(d.DiscountedBy),
		d.DiscountedAt.UTC(),
		strings.TrimSpace(d.UpdatedBy),
		d.UpdatedAt.UTC(),
	)
	out, err := scanDiscountRow(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return ddom.Discount{}, ddom.ErrConflict
		}
		return ddom.Discount{}, err
	}

	if err := r.replaceDiscountItems(ctx, tx, out.ID, d.Discounts); err != nil {
		return ddom.Discount{}, err
	}

	if err := tx.Commit(); err != nil {
		return ddom.Discount{}, err
	}

	// Load items to return consistent persisted state
	im, err := r.loadDiscountItems(ctx, r.DB, []string{out.ID})
	if err != nil {
		return ddom.Discount{}, err
	}
	out.Discounts = im[out.ID]
	return out, nil
}

func (r *DiscountRepositoryPG) Update(ctx context.Context, id string, patch ddom.DiscountPatch) (ddom.Discount, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return ddom.Discount{}, err
	}
	defer tx.Rollback()

	sets := []string{}
	args := []any{}
	i := 1

	if patch.ListID != nil {
		sets = append(sets, fmt.Sprintf("list_id = $%d", i))
		args = append(args, strings.TrimSpace(*patch.ListID))
		i++
	}
	if patch.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.Description))
		i++
	}
	if patch.DiscountedBy != nil {
		sets = append(sets, fmt.Sprintf("discounted_by = $%d", i))
		args = append(args, strings.TrimSpace(*patch.DiscountedBy))
		i++
	}
	if patch.DiscountedAt != nil {
		sets = append(sets, fmt.Sprintf("discounted_at = $%d", i))
		args = append(args, patch.DiscountedAt.UTC())
		i++
	}
	if patch.UpdatedBy != nil {
		sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, strings.TrimSpace(*patch.UpdatedBy))
		i++
	}
	// updated_at: explicit or NOW() if any other field changes
	if patch.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, patch.UpdatedAt.UTC())
		i++
	} else if len(sets) > 0 {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().UTC())
		i++
	}

	var out ddom.Discount
	if len(sets) > 0 {
		args = append(args, id)
		q := fmt.Sprintf(`
UPDATE discounts
SET %s
WHERE id = $%d
RETURNING id, list_id, description, discounted_by, discounted_at, updated_by, updated_at
`, strings.Join(sets, ", "), i)
		row := tx.QueryRowContext(ctx, q, args...)
		var err error
		out, err = scanDiscountRow(row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ddom.Discount{}, ddom.ErrNotFound
			}
			return ddom.Discount{}, err
		}
	} else {
		// No attribute updates; ensure the discount exists and fetch it
		const sel = `
SELECT id, list_id, description, discounted_by, discounted_at, updated_by, updated_at
FROM discounts
WHERE id = $1
FOR UPDATE
`
		row := tx.QueryRowContext(ctx, sel, id)
		var err error
		out, err = scanDiscountRow(row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ddom.Discount{}, ddom.ErrNotFound
			}
			return ddom.Discount{}, err
		}
	}

	// If Discounts provided, replace all items
	if patch.Discounts != nil {
		if err := r.replaceDiscountItems(ctx, tx, id, *patch.Discounts); err != nil {
			return ddom.Discount{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return ddom.Discount{}, err
	}

	// Load items after commit
	im, err := r.loadDiscountItems(ctx, r.DB, []string{out.ID})
	if err != nil {
		return ddom.Discount{}, err
	}
	out.Discounts = im[out.ID]
	return out, nil
}

func (r *DiscountRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM discounts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ddom.ErrNotFound
	}
	return nil
}

// ★★ 修正ポイント: ユースケースのインターフェースに揃える
//
//	Save(ctx, d ddom.Discount) (ddom.Discount, error)
func (r *DiscountRepositoryPG) Save(ctx context.Context, d ddom.Discount) (ddom.Discount, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return ddom.Discount{}, err
	}
	defer tx.Rollback()

	const q = `
INSERT INTO discounts (
  id, list_id, description, discounted_by, discounted_at, updated_by, updated_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7
)
ON CONFLICT (id) DO UPDATE SET
  list_id       = EXCLUDED.list_id,
  description   = EXCLUDED.description,
  discounted_by = EXCLUDED.discounted_by,
  discounted_at = EXCLUDED.discounted_at,
  updated_by    = EXCLUDED.updated_by,
  updated_at    = EXCLUDED.updated_at
RETURNING id, list_id, description, discounted_by, discounted_at, updated_by, updated_at
`
	row := tx.QueryRowContext(ctx, q,
		strings.TrimSpace(d.ID),
		strings.TrimSpace(d.ListID),
		dbcommon.ToDBText(d.Description),
		strings.TrimSpace(d.DiscountedBy),
		d.DiscountedAt.UTC(),
		strings.TrimSpace(d.UpdatedBy),
		d.UpdatedAt.UTC(),
	)
	out, err := scanDiscountRow(row)
	if err != nil {
		return ddom.Discount{}, err
	}

	// Replace items
	if err := r.replaceDiscountItems(ctx, tx, out.ID, d.Discounts); err != nil {
		return ddom.Discount{}, err
	}

	if err := tx.Commit(); err != nil {
		return ddom.Discount{}, err
	}

	im, err := r.loadDiscountItems(ctx, r.DB, []string{out.ID})
	if err != nil {
		return ddom.Discount{}, err
	}
	out.Discounts = im[out.ID]
	return out, nil
}

// =======================
// Helpers
// =======================

func scanDiscountRow(s dbcommon.RowScanner) (ddom.Discount, error) {
	var (
		idNS, listIDNS, discountedByNS, updatedByNS sql.NullString
		descNS                                      sql.NullString
		discountedAt, updatedAt                     time.Time
	)
	if err := s.Scan(
		&idNS, &listIDNS, &descNS, &discountedByNS, &discountedAt, &updatedByNS, &updatedAt,
	); err != nil {
		return ddom.Discount{}, err
	}

	var descPtr *string
	if descNS.Valid {
		v := strings.TrimSpace(descNS.String)
		if v != "" {
			descPtr = &v
		}
	}

	return ddom.Discount{
		ID:           strings.TrimSpace(idNS.String),
		ListID:       strings.TrimSpace(listIDNS.String),
		Description:  descPtr,
		DiscountedBy: strings.TrimSpace(discountedByNS.String),
		DiscountedAt: discountedAt.UTC(),
		UpdatedBy:    strings.TrimSpace(updatedByNS.String),
		UpdatedAt:    updatedAt.UTC(),
		Discounts:    nil, // filled by caller
	}, nil
}

func (r *DiscountRepositoryPG) loadDiscountItems(ctx context.Context, q queryer, ids []string) (map[string][]ddom.DiscountItem, error) {
	if len(ids) == 0 {
		return map[string][]ddom.DiscountItem{}, nil
	}
	ph := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
		args = append(args, id)
	}
	sqlq := fmt.Sprintf(`
SELECT discount_id, model_number, percent
FROM discount_items
WHERE discount_id IN (%s)
ORDER BY discount_id ASC, model_number ASC
`, strings.Join(ph, ","))

	rows, err := q.QueryContext(ctx, sqlq, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]ddom.DiscountItem, len(ids))
	for rows.Next() {
		var did, mn string
		var percent int
		if err := rows.Scan(&did, &mn, &percent); err != nil {
			return nil, err
		}
		out[did] = append(out[did], ddom.DiscountItem{
			ModelNumber: strings.TrimSpace(mn),
			Discount:    percent,
		})
	}
	return out, rows.Err()
}

func (r *DiscountRepositoryPG) replaceDiscountItems(ctx context.Context, ex execer, discountID string, items []ddom.DiscountItem) error {
	// Delete existing
	if _, err := ex.ExecContext(ctx, `DELETE FROM discount_items WHERE discount_id = $1`, discountID); err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	// Bulk insert
	sb := strings.Builder{}
	sb.WriteString(`INSERT INTO discount_items (discount_id, model_number, percent) VALUES `)
	args := make([]any, 0, len(items)*3)
	for i, it := range items {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d)", len(args)+1, len(args)+2, len(args)+3))
		args = append(args, discountID, strings.TrimSpace(it.ModelNumber), it.Discount)
	}
	_, err := ex.ExecContext(ctx, sb.String(), args...)
	return err
}

func buildDiscountWhere(f ddom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// Free text search
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(where, fmt.Sprintf(
			"(d.id ILIKE $%d OR d.list_id ILIKE $%d OR COALESCE(d.description,'') ILIKE $%d OR d.discounted_by ILIKE $%d OR d.updated_by ILIKE $%d)",
			len(args)+1, len(args)+1, len(args)+1, len(args)+1, len(args)+1,
		))
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
			where = append(where, "d.id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// ListID
	if f.ListID != nil && strings.TrimSpace(*f.ListID) != "" {
		where = append(where, fmt.Sprintf("d.list_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.ListID))
	}
	// ListIDs IN (...)
	if len(f.ListIDs) > 0 {
		ph := []string{}
		for _, v := range f.ListIDs {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "d.list_id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// By users
	if f.DiscountedBy != nil && strings.TrimSpace(*f.DiscountedBy) != "" {
		where = append(where, fmt.Sprintf("d.discounted_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.DiscountedBy))
	}
	if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" {
		where = append(where, fmt.Sprintf("d.updated_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.UpdatedBy))
	}

	// Item conditions via EXISTS
	itemConds := []string{}
	if len(f.ModelNumbers) > 0 {
		ph := []string{}
		for _, v := range f.ModelNumbers {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			itemConds = append(itemConds, "di.model_number IN ("+strings.Join(ph, ",")+")")
		}
	}
	if f.PercentMin != nil {
		itemConds = append(itemConds, fmt.Sprintf("di.percent >= $%d", len(args)+1))
		args = append(args, *f.PercentMin)
	}
	if f.PercentMax != nil {
		itemConds = append(itemConds, fmt.Sprintf("di.percent <= $%d", len(args)+1))
		args = append(args, *f.PercentMax)
	}
	if len(itemConds) > 0 {
		where = append(where, "(EXISTS (SELECT 1 FROM discount_items di WHERE di.discount_id = d.id AND "+strings.Join(itemConds, " AND ")+"))")
	}

	// Date ranges
	if f.DiscountedFrom != nil {
		where = append(where, fmt.Sprintf("d.discounted_at >= $%d", len(args)+1))
		args = append(args, f.DiscountedFrom.UTC())
	}
	if f.DiscountedTo != nil {
		where = append(where, fmt.Sprintf("d.discounted_at < $%d", len(args)+1))
		args = append(args, f.DiscountedTo.UTC())
	}
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("d.updated_at >= $%d", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("d.updated_at < $%d", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}

	return where, args
}

func buildDiscountOrderBy(sort ddom.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "id":
		col = "d.id"
	case "listid", "list_id":
		col = "d.list_id"
	case "discountedby", "discounted_by":
		col = "d.discounted_by"
	case "discountedat", "discounted_at":
		col = "d.discounted_at"
	case "updatedby", "updated_by":
		col = "d.updated_by"
	case "updatedat", "updated_at":
		col = "d.updated_at"
	case "description":
		col = "d.description"
	default:
		return ""
	}

	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}
