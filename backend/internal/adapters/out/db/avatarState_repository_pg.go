package db

import (
	"context"
	"database/sql" // 追加
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	avatarstate "narratives/internal/domain/avatarState"
)

// AvatarStateRepositoryPG is the PostgreSQL implementation of avatarState.Repository.
type AvatarStateRepositoryPG struct {
	DB *sql.DB // 追加：r.DB 参照に対応
}

func NewAvatarStateRepositoryPG(db *sql.DB) *AvatarStateRepositoryPG {
	return &AvatarStateRepositoryPG{DB: db} // フィールド名に合わせて修正
}

// ========== Queries ==========

func (r *AvatarStateRepositoryPG) List(ctx context.Context, filter avatarstate.Filter, sort avatarstate.Sort, page avatarstate.Page) (avatarstate.PageResult[avatarstate.AvatarState], error) {
	where, args := buildAvatarStateWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := normalizeAvatarStateOrder(sort)
	if orderBy == "" {
		orderBy = "ORDER BY last_active_at DESC, id DESC"
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM avatar_states %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return avatarstate.PageResult[avatarstate.AvatarState]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id,
  avatar_id,
  follower_count,
  following_count,
  post_count,
  last_active_at,
  updated_at
FROM avatar_states
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return avatarstate.PageResult[avatarstate.AvatarState]{}, err
	}
	defer rows.Close()

	var items []avatarstate.AvatarState
	for rows.Next() {
		item, err := scanAvatarState(rows)
		if err != nil {
			return avatarstate.PageResult[avatarstate.AvatarState]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return avatarstate.PageResult[avatarstate.AvatarState]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return avatarstate.PageResult[avatarstate.AvatarState]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *AvatarStateRepositoryPG) ListByCursor(ctx context.Context, filter avatarstate.Filter, _ avatarstate.Sort, cpage avatarstate.CursorPage) (avatarstate.CursorPageResult[avatarstate.AvatarState], error) {
	where, args := buildAvatarStateWhere(filter)

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
  id,
  avatar_id,
  follower_count,
  following_count,
  post_count,
  last_active_at,
  updated_at
FROM avatar_states
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return avatarstate.CursorPageResult[avatarstate.AvatarState]{}, err
	}
	defer rows.Close()

	var items []avatarstate.AvatarState
	var lastID string
	for rows.Next() {
		item, err := scanAvatarState(rows)
		if err != nil {
			return avatarstate.CursorPageResult[avatarstate.AvatarState]{}, err
		}
		items = append(items, item)
		lastID = item.ID
	}
	if err := rows.Err(); err != nil {
		return avatarstate.CursorPageResult[avatarstate.AvatarState]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return avatarstate.CursorPageResult[avatarstate.AvatarState]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *AvatarStateRepositoryPG) GetByID(ctx context.Context, id string) (avatarstate.AvatarState, error) {
	const q = `
SELECT
  id, avatar_id, follower_count, following_count, post_count, last_active_at, updated_at
FROM avatar_states
WHERE id = $1
`
	row := r.DB.QueryRowContext(ctx, q, id)
	item, err := scanAvatarState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}
	return item, nil
}

func (r *AvatarStateRepositoryPG) GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	const q = `
SELECT
  id, avatar_id, follower_count, following_count, post_count, last_active_at, updated_at
FROM avatar_states
WHERE avatar_id = $1
LIMIT 1
`
	row := r.DB.QueryRowContext(ctx, q, avatarID)
	item, err := scanAvatarState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}
	return item, nil
}

func (r *AvatarStateRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM avatar_states WHERE id = $1`
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

func (r *AvatarStateRepositoryPG) Count(ctx context.Context, filter avatarstate.Filter) (int, error) {
	where, args := buildAvatarStateWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM avatar_states `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ========== Mutations ==========

func (r *AvatarStateRepositoryPG) Create(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	const q = `
INSERT INTO avatar_states (
  id, avatar_id, follower_count, following_count, post_count, last_active_at, updated_at
) VALUES (
  COALESCE(NULLIF($1,''), gen_random_uuid()::text),
  $2, COALESCE($3,0), COALESCE($4,0), COALESCE($5,0), $6, $7
)
RETURNING
  id, avatar_id, follower_count, following_count, post_count, last_active_at, updated_at
`
	row := r.DB.QueryRowContext(
		ctx, q,
		s.ID,
		s.AvatarID,
		dbcommon.ToDBInt64(s.FollowerCount),
		dbcommon.ToDBInt64(s.FollowingCount),
		dbcommon.ToDBInt64(s.PostCount),
		s.LastActiveAt.UTC(),
		dbcommon.ToDBTime(s.UpdatedAt),
	)
	out, err := scanAvatarState(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return avatarstate.AvatarState{}, avatarstate.ErrConflict
		}
		return avatarstate.AvatarState{}, err
	}
	return out, nil
}

func (r *AvatarStateRepositoryPG) Update(ctx context.Context, id string, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	return r.updateBy(ctx, "id = $%d", id, patch)
}

func (r *AvatarStateRepositoryPG) UpdateByAvatarID(ctx context.Context, avatarID string, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	return r.updateBy(ctx, "avatar_id = $%d", avatarID, patch)
}

func (r *AvatarStateRepositoryPG) updateBy(ctx context.Context, whereFmt string, whereVal any, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.FollowerCount != nil {
		sets = append(sets, fmt.Sprintf("follower_count = $%d", i))
		args = append(args, *patch.FollowerCount)
		i++
	}
	if patch.FollowingCount != nil {
		sets = append(sets, fmt.Sprintf("following_count = $%d", i))
		args = append(args, *patch.FollowingCount)
		i++
	}
	if patch.PostCount != nil {
		sets = append(sets, fmt.Sprintf("post_count = $%d", i))
		args = append(args, *patch.PostCount)
		i++
	}
	if patch.LastActiveAt != nil {
		sets = append(sets, fmt.Sprintf("last_active_at = $%d", i))
		args = append(args, patch.LastActiveAt.UTC())
		i++
	}

	// updated_at: explicit or NOW()
	if patch.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, dbcommon.ToDBTime(patch.UpdatedAt))
		i++
	} else {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		now := time.Now().UTC()
		args = append(args, now)
		i++
	}

	if len(sets) == 0 {
		// nothing to update; determine key type to fetch current
		switch whereFmt {
		case "id = $%d":
			return r.GetByID(ctx, whereVal.(string))
		default:
			return r.GetByAvatarID(ctx, whereVal.(string))
		}
	}

	where := fmt.Sprintf(whereFmt, i)
	args = append(args, whereVal)

	q := fmt.Sprintf(`
UPDATE avatar_states
SET %s
WHERE %s
RETURNING
  id, avatar_id, follower_count, following_count, post_count, last_active_at, updated_at
`, strings.Join(sets, ", "), where)

	row := r.DB.QueryRowContext(ctx, q, args...)
	out, err := scanAvatarState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}
	return out, nil
}

func (r *AvatarStateRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM avatar_states WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return avatarstate.ErrNotFound
	}
	return nil
}

func (r *AvatarStateRepositoryPG) DeleteByAvatarID(ctx context.Context, avatarID string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM avatar_states WHERE avatar_id = $1`, avatarID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return avatarstate.ErrNotFound
	}
	return nil
}

func (r *AvatarStateRepositoryPG) Save(ctx context.Context, s avatarstate.AvatarState, _ *avatarstate.SaveOptions) (avatarstate.AvatarState, error) {
	const q = `
INSERT INTO avatar_states (
  id, avatar_id, follower_count, following_count, post_count, last_active_at, updated_at
) VALUES (
  $1, $2, COALESCE($3,0), COALESCE($4,0), COALESCE($5,0), $6, $7
)
ON CONFLICT (id) DO UPDATE SET
  avatar_id       = EXCLUDED.avatar_id,
  follower_count  = EXCLUDED.follower_count,
  following_count = EXCLUDED.following_count,
  post_count      = EXCLUDED.post_count,
  last_active_at  = EXCLUDED.last_active_at,
  updated_at      = COALESCE(EXCLUDED.updated_at, NOW())
RETURNING
  id, avatar_id, follower_count, following_count, post_count, last_active_at, updated_at
`
	row := r.DB.QueryRowContext(
		ctx, q,
		s.ID,
		s.AvatarID,
		dbcommon.ToDBInt64(s.FollowerCount),
		dbcommon.ToDBInt64(s.FollowingCount),
		dbcommon.ToDBInt64(s.PostCount),
		s.LastActiveAt.UTC(),
		dbcommon.ToDBTime(s.UpdatedAt),
	)
	out, err := scanAvatarState(row)
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return out, nil
}

// Upsert: usecase 側インターフェースに合わせて追加（内部は Save に委譲）
func (r *AvatarStateRepositoryPG) Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	return r.Save(ctx, s, nil)
}

// ========== Helpers ==========

func scanAvatarState(s dbcommon.RowScanner) (avatarstate.AvatarState, error) {
	var (
		idNS, avatarIDNS sql.NullString
		followerNI64     sql.NullInt64
		followingNI64    sql.NullInt64
		postNI64         sql.NullInt64
		lastActiveAt     time.Time
		updatedAtNT      sql.NullTime
	)
	if err := s.Scan(
		&idNS,
		&avatarIDNS,
		&followerNI64,
		&followingNI64,
		&postNI64,
		&lastActiveAt,
		&updatedAtNT,
	); err != nil {
		return avatarstate.AvatarState{}, err
	}

	// Build pointers for counts (table uses NOT NULL, but domain fields are pointers)
	var (
		followerPtr, followingPtr, postPtr *int64
		updatedPtr                         *time.Time
	)
	if followerNI64.Valid {
		v := followerNI64.Int64
		followerPtr = &v
	}
	if followingNI64.Valid {
		v := followingNI64.Int64
		followingPtr = &v
	}
	if postNI64.Valid {
		v := postNI64.Int64
		postPtr = &v
	}
	if updatedAtNT.Valid {
		t := updatedAtNT.Time.UTC()
		updatedPtr = &t
	}

	return avatarstate.New(
		strings.TrimSpace(idNS.String),
		strings.TrimSpace(avatarIDNS.String),
		followerPtr,
		followingPtr,
		postPtr,
		lastActiveAt.UTC(),
		updatedPtr,
	)
}

func buildAvatarStateWhere(f avatarstate.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// Search: id or avatar_id
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(where, fmt.Sprintf("(id ILIKE $%d OR avatar_id ILIKE $%d)", len(args)+1, len(args)+1))
		args = append(args, "%"+sq+"%")
	}

	// AvatarID exact
	if f.AvatarID != nil && strings.TrimSpace(*f.AvatarID) != "" {
		where = append(where, fmt.Sprintf("avatar_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.AvatarID))
	}

	// AvatarIDs IN (...)
	if len(f.AvatarIDs) > 0 {
		placeholders := make([]string, 0, len(f.AvatarIDs))
		for _, v := range f.AvatarIDs {
			if strings.TrimSpace(v) == "" {
				continue
			}
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, strings.TrimSpace(v))
		}
		if len(placeholders) > 0 {
			where = append(where, "avatar_id IN ("+strings.Join(placeholders, ",")+")")
		}
	}

	// Count ranges
	if f.FollowerMin != nil {
		where = append(where, fmt.Sprintf("follower_count >= $%d", len(args)+1))
		args = append(args, *f.FollowerMin)
	}
	if f.FollowerMax != nil {
		where = append(where, fmt.Sprintf("follower_count <= $%d", len(args)+1))
		args = append(args, *f.FollowerMax)
	}
	if f.FollowingMin != nil {
		where = append(where, fmt.Sprintf("following_count >= $%d", len(args)+1))
		args = append(args, *f.FollowingMin)
	}
	if f.FollowingMax != nil {
		where = append(where, fmt.Sprintf("following_count <= $%d", len(args)+1))
		args = append(args, *f.FollowingMax)
	}
	if f.PostMin != nil {
		where = append(where, fmt.Sprintf("post_count >= $%d", len(args)+1))
		args = append(args, *f.PostMin)
	}
	if f.PostMax != nil {
		where = append(where, fmt.Sprintf("post_count <= $%d", len(args)+1))
		args = append(args, *f.PostMax)
	}

	// Time ranges
	if f.LastActiveFrom != nil {
		where = append(where, fmt.Sprintf("last_active_at >= $%d", len(args)+1))
		args = append(args, f.LastActiveFrom.UTC())
	}
	if f.LastActiveTo != nil {
		where = append(where, fmt.Sprintf("last_active_at < $%d", len(args)+1))
		args = append(args, f.LastActiveTo.UTC())
	}
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("(updated_at IS NOT NULL AND updated_at >= $%d)", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("(updated_at IS NOT NULL AND updated_at < $%d)", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}

	return where, args
}

func normalizeAvatarStateOrder(sort avatarstate.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "id":
		col = "id"
	case "avatarid", "avatar_id":
		col = "avatar_id"
	case "followercount", "follower_count":
		col = "follower_count"
	case "followingcount", "following_count":
		col = "following_count"
	case "postcount", "post_count":
		col = "post_count"
	case "lastactiveat", "last_active_at":
		col = "last_active_at"
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
