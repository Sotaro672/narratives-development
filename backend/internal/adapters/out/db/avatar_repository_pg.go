package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	avdom "narratives/internal/domain/avatar"
)

// AvatarRepositoryPG は Avatar リポジトリの PostgreSQL 実装です。
type AvatarRepositoryPG struct {
	DB *sql.DB
}

func NewAvatarRepositoryPG(db *sql.DB) *AvatarRepositoryPG {
	return &AvatarRepositoryPG{DB: db}
}

// ========================================
// List (filter + sort + pagination)
// ========================================
func (r *AvatarRepositoryPG) List(ctx context.Context, filter avdom.Filter, sort avdom.Sort, page avdom.Page) (avdom.PageResult, error) {
	where, args := buildAvatarWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	// ソート（スキーマに存在するカラムのみ許可）
	orderBy := "ORDER BY created_at DESC"
	if col, dir := normalizeAvatarSort(sort); col != "" {
		orderBy = fmt.Sprintf("ORDER BY %s %s", col, dir)
	}

	// ページング
	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	number := page.Number
	if number <= 0 {
		number = 1
	}
	offset := (number - 1) * perPage

	// 件数
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM avatars %s`, whereSQL)
	var total int
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return avdom.PageResult{}, err
	}

	// 本体
	q := fmt.Sprintf(`
SELECT
  id,
  user_id,
  avatar_name,
  avatar_icon_id,
  wallet_address,
  bio,
  website,
  created_at,
  updated_at,
  deleted_at
FROM avatars
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return avdom.PageResult{}, err
	}
	defer rows.Close()

	var items []avdom.Avatar
	for rows.Next() {
		var a avdom.Avatar
		if err := scanAvatar(rows, &a); err != nil {
			return avdom.PageResult{}, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return avdom.PageResult{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return avdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

// ========================================
// ListByCursor (simple id-based cursor)
// ========================================
func (r *AvatarRepositoryPG) ListByCursor(ctx context.Context, filter avdom.Filter, _ avdom.Sort, cpage avdom.CursorPage) (avdom.CursorPageResult, error) {
	where, args := buildAvatarWhere(filter)

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
  user_id,
  avatar_name,
  avatar_icon_id,
  wallet_address,
  bio,
  website,
  created_at,
  updated_at,
  deleted_at
FROM avatars
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1) // 次ページ判定用に +1

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return avdom.CursorPageResult{}, err
	}
	defer rows.Close()

	var items []avdom.Avatar
	var lastID string
	for rows.Next() {
		var a avdom.Avatar
		if err := scanAvatar(rows, &a); err != nil {
			return avdom.CursorPageResult{}, err
		}
		items = append(items, a)
		lastID = a.ID
	}
	if err := rows.Err(); err != nil {
		return avdom.CursorPageResult{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return avdom.CursorPageResult{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// ========================================
// GetByID
// ========================================
func (r *AvatarRepositoryPG) GetByID(ctx context.Context, id string) (avdom.Avatar, error) {
	const q = `
SELECT
  id, user_id, avatar_name, avatar_icon_id,
  wallet_address, bio, website, created_at, updated_at, deleted_at
FROM avatars
WHERE id = $1
`
	var a avdom.Avatar
	if err := scanAvatar(r.DB.QueryRowContext(ctx, q, id), &a); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avdom.Avatar{}, sql.ErrNoRows
		}
		return avdom.Avatar{}, err
	}
	return a, nil
}

// ========================================
// GetByWalletAddress
// ========================================
func (r *AvatarRepositoryPG) GetByWalletAddress(ctx context.Context, wallet string) (avdom.Avatar, error) {
	const q = `
SELECT
  id, user_id, avatar_name, avatar_icon_id,
  wallet_address, bio, website, created_at, updated_at, deleted_at
FROM avatars
WHERE wallet_address = $1
LIMIT 1
`
	var a avdom.Avatar
	if err := scanAvatar(r.DB.QueryRowContext(ctx, q, wallet), &a); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avdom.Avatar{}, sql.ErrNoRows
		}
		return avdom.Avatar{}, err
	}
	return a, nil
}

// ========================================
// Exists
// ========================================
func (r *AvatarRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM avatars WHERE id = $1`
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

// ========================================
// Count
// ========================================
func (r *AvatarRepositoryPG) Count(ctx context.Context, filter avdom.Filter) (int, error) {
	where, args := buildAvatarWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM avatars `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ========================================
// Create
// ========================================
func (r *AvatarRepositoryPG) Create(ctx context.Context, a avdom.Avatar) (avdom.Avatar, error) {
	const q = `
INSERT INTO avatars (
  id, user_id, avatar_name, avatar_icon_id,
  wallet_address, bio, website
) VALUES (
  COALESCE(NULLIF($1,''), gen_random_uuid()::text),
  $2,$3,$4,$5,$6,$7
)
RETURNING
  id, user_id, avatar_name, avatar_icon_id,
  wallet_address, bio, website, created_at, updated_at, deleted_at
`
	var out avdom.Avatar
	var createdAt, updatedAt sql.NullTime
	var deletedAt sql.NullTime

	err := r.DB.QueryRowContext(ctx, q,
		a.ID,
		a.UserID,
		a.AvatarName,
		dbcommon.ToDBText(a.AvatarIconID),
		dbcommon.ToDBText(a.WalletAddress),
		dbcommon.ToDBText(a.Bio),
		dbcommon.ToDBText(a.Website),
	).Scan(
		&out.ID,
		&out.UserID,
		&out.AvatarName,
		&out.AvatarIconID,
		&out.WalletAddress,
		&out.Bio,
		&out.Website,
		&createdAt,
		&updatedAt,
		&deletedAt,
	)
	if err != nil {
		// 一意制約違反などはそのまま返す（ドメイン側に専用エラーが無いため）
		return avdom.Avatar{}, err
	}

	if createdAt.Valid {
		out.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		out.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		out.DeletedAt = &t
	} else {
		out.DeletedAt = nil
	}

	return out, nil
}

// ========================================
// Update (patch)
// ========================================
func (r *AvatarRepositoryPG) Update(ctx context.Context, id string, patch avdom.AvatarPatch) (avdom.Avatar, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.AvatarName != nil {
		sets = append(sets, fmt.Sprintf("avatar_name = $%d", i))
		args = append(args, *patch.AvatarName)
		i++
	}
	if patch.AvatarIconID != nil {
		sets = append(sets, fmt.Sprintf("avatar_icon_id = $%d", i))
		args = append(args, *patch.AvatarIconID)
		i++
	}
	if patch.WalletAddress != nil {
		sets = append(sets, fmt.Sprintf("wallet_address = $%d", i))
		args = append(args, *patch.WalletAddress)
		i++
	}
	if patch.Bio != nil {
		sets = append(sets, fmt.Sprintf("bio = $%d", i))
		args = append(args, *patch.Bio)
		i++
	}
	if patch.Website != nil {
		sets = append(sets, fmt.Sprintf("website = $%d", i))
		args = append(args, *patch.Website)
		i++
	}

	if len(sets) == 0 {
		// 更新対象なし -> 現状値を返す
		return r.GetByID(ctx, id)
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
	args = append(args, time.Now().UTC())
	i++

	args = append(args, id)

	q := fmt.Sprintf(`
UPDATE avatars
SET %s
WHERE id = $%d
RETURNING
  id, user_id, avatar_name, avatar_icon_id,
  wallet_address, bio, website, created_at, updated_at, deleted_at
`, strings.Join(sets, ", "), i)

	var out avdom.Avatar
	var createdAt, updatedAt sql.NullTime
	var deletedAt sql.NullTime
	if err := r.DB.QueryRowContext(ctx, q, args...).Scan(
		&out.ID,
		&out.UserID,
		&out.AvatarName,
		&out.AvatarIconID,
		&out.WalletAddress,
		&out.Bio,
		&out.Website,
		&createdAt,
		&updatedAt,
		&deletedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return avdom.Avatar{}, sql.ErrNoRows
		}
		return avdom.Avatar{}, err
	}

	if createdAt.Valid {
		out.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		out.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		out.DeletedAt = &t
	} else {
		out.DeletedAt = nil
	}

	return out, nil
}

// ========================================
// Delete
// ========================================
func (r *AvatarRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM avatars WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ========================================
// Save (upsert)
// ========================================
func (r *AvatarRepositoryPG) Save(ctx context.Context, a avdom.Avatar, _ *avdom.SaveOptions) (avdom.Avatar, error) {
	const q = `
INSERT INTO avatars (
  id, user_id, avatar_name, avatar_icon_id,
  wallet_address, bio, website
) VALUES (
  $1,$2,$3,$4,$5,$6,$7
)
ON CONFLICT (id) DO UPDATE SET
  user_id        = EXCLUDED.user_id,
  avatar_name    = EXCLUDED.avatar_name,
  avatar_icon_id = EXCLUDED.avatar_icon_id,
  wallet_address = EXCLUDED.wallet_address,
  bio            = EXCLUDED.bio,
  website        = EXCLUDED.website,
  updated_at     = NOW()
RETURNING
  id, user_id, avatar_name, avatar_icon_id,
  wallet_address, bio, website, created_at, updated_at, deleted_at
`
	var out avdom.Avatar
	var createdAt, updatedAt sql.NullTime
	var deletedAt sql.NullTime
	if err := r.DB.QueryRowContext(ctx, q,
		a.ID,
		a.UserID,
		a.AvatarName,
		dbcommon.ToDBText(a.AvatarIconID),
		dbcommon.ToDBText(a.WalletAddress),
		dbcommon.ToDBText(a.Bio),
		dbcommon.ToDBText(a.Website),
	).Scan(
		&out.ID,
		&out.UserID,
		&out.AvatarName,
		&out.AvatarIconID,
		&out.WalletAddress,
		&out.Bio,
		&out.Website,
		&createdAt,
		&updatedAt,
		&deletedAt,
	); err != nil {
		return avdom.Avatar{}, err
	}

	if createdAt.Valid {
		out.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		out.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		out.DeletedAt = &t
	} else {
		out.DeletedAt = nil
	}

	return out, nil
}

// ========================================
// Search (simple ILIKE on names)
// ========================================
func (r *AvatarRepositoryPG) Search(ctx context.Context, query string) ([]avdom.Avatar, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []avdom.Avatar{}, nil
	}
	const sqlq = `
SELECT
  id,
  user_id,
  avatar_name,
  avatar_icon_id,
  wallet_address,
  bio,
  website,
  created_at,
  updated_at,
  deleted_at
FROM avatars
WHERE (avatar_name ILIKE $1 OR wallet_address ILIKE $1)
ORDER BY avatar_name ASC
LIMIT 50
`
	rows, err := r.DB.QueryContext(ctx, sqlq, "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []avdom.Avatar
	for rows.Next() {
		var a avdom.Avatar
		if err := scanAvatar(rows, &a); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// ========================================
// ListTopByFollowers (fallback: created_at desc)
// ========================================
func (r *AvatarRepositoryPG) ListTopByFollowers(ctx context.Context, limit int) ([]avdom.Avatar, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `
SELECT
  id,
  user_id,
  avatar_name,
  avatar_icon_id,
  wallet_address,
  bio,
  website,
  created_at,
  updated_at,
  deleted_at
FROM avatars
ORDER BY created_at DESC
LIMIT $1
`
	rows, err := r.DB.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []avdom.Avatar
	for rows.Next() {
		var a avdom.Avatar
		if err := scanAvatar(rows, &a); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// ========================================
// Reset (development/testing)
// ========================================
func (r *AvatarRepositoryPG) Reset(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, `TRUNCATE TABLE avatars RESTART IDENTITY CASCADE`)
	return err
}

// ========================================
// Scan helper
// ========================================
// scanAvatar は共通の RowScanner を使用します。
func scanAvatar(s dbcommon.RowScanner, a *avdom.Avatar) error {
	var (
		id, userID, avatarName      sql.NullString
		avatarIconID                sql.NullString
		walletAddress, bio, website sql.NullString
		createdAt, updatedAt        sql.NullTime
		deletedAt                   sql.NullTime
	)
	if err := s.Scan(
		&id, &userID, &avatarName, &avatarIconID,
		&walletAddress, &bio, &website, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return err
	}

	a.ID = id.String
	a.UserID = userID.String
	a.AvatarName = avatarName.String

	if avatarIconID.Valid {
		v := avatarIconID.String
		a.AvatarIconID = &v
	} else {
		a.AvatarIconID = nil
	}

	if walletAddress.Valid {
		v := walletAddress.String
		a.WalletAddress = &v
	} else {
		a.WalletAddress = nil
	}

	if bio.Valid {
		v := bio.String
		a.Bio = &v
	} else {
		a.Bio = nil
	}
	if website.Valid {
		v := website.String
		a.Website = &v
	} else {
		a.Website = nil
	}

	if createdAt.Valid {
		a.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		a.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		a.DeletedAt = &t
	} else {
		a.DeletedAt = nil
	}

	return nil
}

// ========================================
// WHERE/ORDER helpers
// ========================================
func buildAvatarWhere(f avdom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// 検索（名前/ウォレット）
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(where, fmt.Sprintf("(avatar_name ILIKE $%d OR wallet_address ILIKE $%d)",
			len(args)+1, len(args)+1))
		args = append(args, "%"+sq+"%")
	}

	// 期間（created_at ベース）
	if f.JoinedFrom != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, *f.JoinedFrom)
	}
	if f.JoinedTo != nil {
		where = append(where, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, *f.JoinedTo)
	}

	// Verified はスキーマに列が無いため無視（将来列追加で対応）

	return where, args
}

// ソート正規化
func normalizeAvatarSort(sort avdom.Sort) (column string, direction string) {
	switch strings.ToLower(string(sort.Column)) {
	case "avatarname":
		column = "avatar_name"
	case "createdat":
		column = "created_at"
	case "updatedat":
		column = "updated_at"
	default:
		column = "" // デフォルトを使用（created_at DESC）
	}

	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	direction = dir
	return
}
