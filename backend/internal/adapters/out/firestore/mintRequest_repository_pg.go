// backend\internal\adapters\out\firestore\mintRequest_repository_pg.go
package firestore

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

// PG implementation of usecase.MintRequestRepo
type MintRequestRepositoryPG struct {
	DB *sql.DB
}

func NewMintRequestRepositoryPG(db *sql.DB) *MintRequestRepositoryPG {
	return &MintRequestRepositoryPG{DB: db}
}

// ============================================================
// MintRequestRepo (ports required by usecase)
// ============================================================

// GetByID(ctx, id) (MintRequest, error)
func (r *MintRequestRepositoryPG) GetByID(ctx context.Context, id string) (mrdom.MintRequest, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  id,
  token_blueprint_id,
  production_id,
  mint_quantity,
  burn_date,
  status,
  requested_by,
  requested_at,
  minted_at,
  created_at,
  created_by,
  updated_at,
  updated_by,
  deleted_at,
  deleted_by
FROM mint_requests
WHERE id = $1
`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))

	mr, err := scanMintRequest(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mrdom.MintRequest{}, mrdom.ErrNotFound
		}
		return mrdom.MintRequest{}, err
	}
	return mr, nil
}

// Exists(ctx, id) (bool, error)
func (r *MintRequestRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `SELECT 1 FROM mint_requests WHERE id = $1`
	var one int
	err := run.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create(ctx, v MintRequest) (MintRequest, error)
//
// 期待仕様:
// - v.ID が空なら DB 側で gen_random_uuid()::text で生成
// - created_at/created_by/updated_at/updated_by はここで決める
// - その他のフィールドは v の値を使う
func (r *MintRequestRepositoryPG) Create(ctx context.Context, v mrdom.MintRequest) (mrdom.MintRequest, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO mint_requests (
  id,
  token_blueprint_id,
  production_id,
  mint_quantity,
  burn_date,
  status,
  requested_by,
  requested_at,
  minted_at,
  created_at,
  created_by,
  updated_at,
  updated_by,
  deleted_at,
  deleted_by
) VALUES (
  COALESCE(NULLIF($1,''), gen_random_uuid()::text),
  $2,
  $3,
  $4,
  $5::timestamptz,
  $6,
  $7,
  $8,
  $9,
  NOW(),
  $10,
  NOW(),
  $11,
  $12,
  $13
)
RETURNING
  id,
  token_blueprint_id,
  production_id,
  mint_quantity,
  burn_date,
  status,
  requested_by,
  requested_at,
  minted_at,
  created_at,
  created_by,
  updated_at,
  updated_by,
  deleted_at,
  deleted_by
`
	row := run.QueryRowContext(
		ctx,
		q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.TokenBlueprintID),
		strings.TrimSpace(v.ProductionID),
		v.MintQuantity,
		dbcommon.ToDBTime(v.BurnDate), // NULL or time
		strings.TrimSpace(string(v.Status)),
		strPtrOrNil(v.RequestedBy),
		dbcommon.ToDBTime(v.RequestedAt),
		dbcommon.ToDBTime(v.MintedAt),
		strings.TrimSpace(v.CreatedBy),
		strings.TrimSpace(v.UpdatedBy),
		dbcommon.ToDBTime(v.DeletedAt),
		strPtrOrNil(v.DeletedBy),
	)

	mr, err := scanMintRequest(row)
	if err != nil {
		return mrdom.MintRequest{}, err
	}
	return mr, nil
}

// Save(ctx, v MintRequest) (MintRequest, error)
//
// Upsert 動作 (id 主キー):
// - INSERT しつつ ON CONFLICT(id) DO UPDATE
// - created_at/created_by は最初の値をできるだけ保持
// - updated_at/updated_by は新しい値で更新
// - 他のカラムは新しい値で上書き
func (r *MintRequestRepositoryPG) Save(ctx context.Context, v mrdom.MintRequest) (mrdom.MintRequest, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO mint_requests (
  id,
  token_blueprint_id,
  production_id,
  mint_quantity,
  burn_date,
  status,
  requested_by,
  requested_at,
  minted_at,
  created_at,
  created_by,
  updated_at,
  updated_by,
  deleted_at,
  deleted_by
) VALUES (
  COALESCE(NULLIF($1,''), gen_random_uuid()::text),
  $2,
  $3,
  $4,
  $5::timestamptz,
  $6,
  $7,
  $8,
  $9,
  NOW(),
  $10,
  NOW(),
  $11,
  $12,
  $13
)
ON CONFLICT (id) DO UPDATE SET
  token_blueprint_id = EXCLUDED.token_blueprint_id,
  production_id      = EXCLUDED.production_id,
  mint_quantity      = EXCLUDED.mint_quantity,
  burn_date          = EXCLUDED.burn_date,
  status             = EXCLUDED.status,
  requested_by       = EXCLUDED.requested_by,
  requested_at       = EXCLUDED.requested_at,
  minted_at          = EXCLUDED.minted_at,
  -- created_at/created_by should preserve earliest known values
  created_at         = LEAST(mint_requests.created_at, EXCLUDED.created_at),
  created_by         = CASE
                          WHEN mint_requests.created_by IS NULL OR mint_requests.created_by = ''
                          THEN EXCLUDED.created_by
                          ELSE mint_requests.created_by
                       END,
  updated_at         = NOW(),
  updated_by         = EXCLUDED.updated_by,
  deleted_at         = EXCLUDED.deleted_at,
  deleted_by         = EXCLUDED.deleted_by
RETURNING
  id,
  token_blueprint_id,
  production_id,
  mint_quantity,
  burn_date,
  status,
  requested_by,
  requested_at,
  minted_at,
  created_at,
  created_by,
  updated_at,
  updated_by,
  deleted_at,
  deleted_by
`
	row := run.QueryRowContext(
		ctx,
		q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.TokenBlueprintID),
		strings.TrimSpace(v.ProductionID),
		v.MintQuantity,
		dbcommon.ToDBTime(v.BurnDate),
		strings.TrimSpace(string(v.Status)),
		strPtrOrNil(v.RequestedBy),
		dbcommon.ToDBTime(v.RequestedAt),
		dbcommon.ToDBTime(v.MintedAt),
		strings.TrimSpace(v.CreatedBy),
		strings.TrimSpace(v.UpdatedBy),
		dbcommon.ToDBTime(v.DeletedAt),
		strPtrOrNil(v.DeletedBy),
	)

	mr, err := scanMintRequest(row)
	if err != nil {
		return mrdom.MintRequest{}, err
	}
	return mr, nil
}

// Delete(ctx, id) error
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

// ============================================================
// Convenience / extra query methods
// これらは usecase.MintRequestRepo には必須じゃないけど今後も使えるので残す
// ============================================================

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

	// COUNT
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM mint_requests "+whereSQL, args...).Scan(&total); err != nil {
		return mrdom.PageResult{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id,
  token_blueprint_id,
  production_id,
  mint_quantity,
  burn_date,
  status,
  requested_by,
  requested_at,
  minted_at,
  created_at,
  created_by,
  updated_at,
  updated_by,
  deleted_at,
  deleted_by
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

// Update は便利用（ユースケースPortの必須ではない）
func (r *MintRequestRepositoryPG) Update(ctx context.Context, id string, patch mrdom.UpdateMintRequest) (mrdom.MintRequest, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	id = strings.TrimSpace(id)
	if id == "" {
		return mrdom.MintRequest{}, mrdom.ErrNotFound
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
			sets = append(sets, fmt.Sprintf("%s = $%d::timestamptz", col, i))
			args = append(args, p.UTC())
			i++
		}
	}

	// Updatable business fields
	setStatus(patch.Status)
	setText("token_blueprint_id", patch.TokenBlueprintID)
	setInt("mint_quantity", patch.MintQuantity)
	setDate("burn_date", patch.BurnDate)

	setText("requested_by", patch.RequestedBy)
	setTime("requested_at", patch.RequestedAt)
	setTime("minted_at", patch.MintedAt)

	// Soft delete-ish fields
	setTime("deleted_at", patch.DeletedAt)
	setText("deleted_by", patch.DeletedBy)

	// audit
	now := time.Now().UTC()
	sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
	args = append(args, now)
	i++
	sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
	args = append(args, strings.TrimSpace(patch.UpdatedBy))
	i++

	if len(sets) == 0 {
		// nothing changed; just return current
		return r.GetByID(ctx, id)
	}

	args = append(args, id)

	q := fmt.Sprintf(`
UPDATE mint_requests
SET %s
WHERE id = $%d
RETURNING
  id,
  token_blueprint_id,
  production_id,
  mint_quantity,
  burn_date,
  status,
  requested_by,
  requested_at,
  minted_at,
  created_at,
  created_by,
  updated_at,
  updated_by,
  deleted_at,
  deleted_by
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)

	mr, err := scanMintRequest(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mrdom.MintRequest{}, mrdom.ErrNotFound
		}
		return mrdom.MintRequest{}, err
	}
	return mr, nil
}

// Count / Reset keep existing behavior for admin/testing.

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

// ============================================================
// Helpers
// ============================================================

func scanMintRequest(s dbcommon.RowScanner) (mrdom.MintRequest, error) {
	var (
		id               string
		tokenBlueprintID string
		productionID     string
		mintQuantity     int
		burnDateNS       sql.NullTime
		status           string
		requestedByNS    sql.NullString
		requestedAtNS    sql.NullTime
		mintedAtNS       sql.NullTime
		createdAt        time.Time
		createdBy        string
		updatedAt        time.Time
		updatedBy        string
		deletedAtNS      sql.NullTime
		deletedByNS      sql.NullString
	)

	if err := s.Scan(
		&id,
		&tokenBlueprintID,
		&productionID,
		&mintQuantity,
		&burnDateNS,
		&status,
		&requestedByNS,
		&requestedAtNS,
		&mintedAtNS,
		&createdAt,
		&createdBy,
		&updatedAt,
		&updatedBy,
		&deletedAtNS,
		&deletedByNS,
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

		Status:      mrdom.MintRequestStatus(strings.TrimSpace(status)),
		RequestedBy: toStrPtr(requestedByNS),
		RequestedAt: toTimePtr(requestedAtNS),
		MintedAt:    toTimePtr(mintedAtNS),

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
		// We'll build placeholders after appending
		basePos := len(args)
		for _, st := range f.Statuses {
			args = append(args, strings.TrimSpace(string(st)))
		}
		ph := make([]string, 0, len(f.Statuses))
		for i := range f.Statuses {
			ph = append(ph, fmt.Sprintf("$%d", basePos+i+1))
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

// tiny helper
func strPtrOrNil(p *string) any {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return v
}
