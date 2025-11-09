// backend\internal\adapters\out\db\announcement_repository_pg.go
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	announcement "narratives/internal/domain/announcement" // ← 修正: annoucement -> announcement
	common "narratives/internal/domain/common"

	"github.com/lib/pq"
)

// PostgreSQL implementation of announcement.Repository.
type AnnouncementRepositoryPG struct {
	db *sql.DB
}

func NewAnnouncementRepositoryPG(db *sql.DB) *AnnouncementRepositoryPG {
	return &AnnouncementRepositoryPG{db: db}
}

// ========== Repository methods ==========

func (r *AnnouncementRepositoryPG) List(ctx context.Context, filter announcement.Filter, sort common.Sort, page common.Page) (common.PageResult[announcement.Announcement], error) {
	where, args := buildAnnouncementWhere(filter)
	orderBy := buildAnnouncementOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY created_at DESC, id DESC"
	}
	limit := page.PerPage
	if limit <= 0 {
		limit = 20
	}
	number := page.Number
	if number <= 0 {
		number = 1
	}
	offset := (number - 1) * limit

	q := fmt.Sprintf(`
SELECT
  id, title, content, category, target_audience, target_token,
  target_products, target_avatars, is_published, published_at,
  attachments, status, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM announcements
%s
%s
LIMIT %d OFFSET %d
`, where, orderBy, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return common.PageResult[announcement.Announcement]{}, err
	}
	defer rows.Close()

	var items []announcement.Announcement
	for rows.Next() {
		a, err := scanAnnouncementRow(rows)
		if err != nil {
			return common.PageResult[announcement.Announcement]{}, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return common.PageResult[announcement.Announcement]{}, err
	}

	total, err := r.Count(ctx, filter)
	if err != nil {
		return common.PageResult[announcement.Announcement]{}, err
	}
	totalPages := (total + limit - 1) / limit

	return common.PageResult[announcement.Announcement]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    limit,
	}, nil
}

func (r *AnnouncementRepositoryPG) ListByCursor(ctx context.Context, filter announcement.Filter, sort common.Sort, cpage common.CursorPage) (common.CursorPageResult[announcement.Announcement], error) {
	return common.CursorPageResult[announcement.Announcement]{}, errors.New("ListByCursor: not implemented")
}

func (r *AnnouncementRepositoryPG) GetByID(ctx context.Context, id string) (announcement.Announcement, error) {
	q := `
SELECT
  id, title, content, category, target_audience, target_token,
  target_products, target_avatars, is_published, published_at,
  attachments, status, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM announcements
WHERE id = $1
`
	row := r.db.QueryRowContext(ctx, q, id)
	a, err := scanAnnouncementRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return announcement.Announcement{}, announcement.ErrNotFound // ← 修正
		}
		return announcement.Announcement{}, err
	}
	return a, nil
}

func (r *AnnouncementRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	var v int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM announcements WHERE id = $1`, id).Scan(&v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *AnnouncementRepositoryPG) Count(ctx context.Context, f announcement.Filter) (int, error) {
	where, args := buildAnnouncementWhere(f)
	q := fmt.Sprintf(`SELECT COUNT(*) FROM announcements %s`, where)
	var cnt int
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}

func (r *AnnouncementRepositoryPG) Search(ctx context.Context, query string) ([]announcement.Announcement, error) {
	qs := "%" + strings.TrimSpace(query) + "%"
	q := `
SELECT
  id, title, content, category, target_audience, target_token,
  target_products, target_avatars, is_published, published_at,
  attachments, status, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
FROM announcements
WHERE title ILIKE $1 OR content ILIKE $1
ORDER BY created_at DESC, id DESC
LIMIT 100
`
	rows, err := r.db.QueryContext(ctx, q, qs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []announcement.Announcement
	for rows.Next() {
		a, err := scanAnnouncementRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AnnouncementRepositoryPG) Create(ctx context.Context, a announcement.Announcement) (announcement.Announcement, error) {
	q := `
INSERT INTO announcements (
  id, title, content, category, target_audience, target_token,
  target_products, target_avatars, is_published, published_at,
  attachments, status, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,
  $11,$12,$13,$14,$15,$16,$17,$18
)
RETURNING
  id, title, content, category, target_audience, target_token,
  target_products, target_avatars, is_published, published_at,
  attachments, status, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	var (
		targetToken any
		updatedAt   any
		updatedBy   any
		deletedAt   any
		deletedBy   any
		publishedAt any
	)
	if a.TargetToken != nil {
		targetToken = *a.TargetToken
	}
	if a.UpdatedAt != nil {
		updatedAt = a.UpdatedAt.UTC()
	}
	if a.UpdatedBy != nil {
		updatedBy = *a.UpdatedBy
	}
	if a.DeletedAt != nil {
		deletedAt = a.DeletedAt.UTC()
	}
	if a.DeletedBy != nil {
		deletedBy = *a.DeletedBy
	}
	if a.PublishedAt != nil {
		publishedAt = a.PublishedAt.UTC()
	}

	row := r.db.QueryRowContext(ctx, q,
		a.ID, a.Title, a.Content, string(a.Category), string(a.TargetAudience), targetToken,
		pq.Array(a.TargetProducts), pq.Array(a.TargetAvatars), a.IsPublished, publishedAt,
		pq.Array(a.Attachments), string(a.Status), a.CreatedAt.UTC(), a.CreatedBy, updatedAt, updatedBy, deletedAt, deletedBy,
	)
	out, err := scanAnnouncementRow(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return announcement.Announcement{}, announcement.ErrConflict
		}
		return announcement.Announcement{}, err
	}
	return out, nil
}

func (r *AnnouncementRepositoryPG) Update(ctx context.Context, id string, p announcement.AnnouncementPatch) (announcement.Announcement, error) {
	set := make([]string, 0, 12)
	args := make([]any, 0, 16)
	i := 1

	if p.Title != nil {
		set = append(set, fmt.Sprintf("title = $%d", i))
		args = append(args, strings.TrimSpace(*p.Title))
		i++
	}
	if p.Content != nil {
		set = append(set, fmt.Sprintf("content = $%d", i))
		args = append(args, strings.TrimSpace(*p.Content))
		i++
	}
	if p.Category != nil {
		set = append(set, fmt.Sprintf("category = $%d", i))
		args = append(args, string(*p.Category))
		i++
	}
	if p.TargetAudience != nil {
		set = append(set, fmt.Sprintf("target_audience = $%d", i))
		args = append(args, string(*p.TargetAudience))
		i++
	}
	if p.TargetToken != nil {
		set = append(set, fmt.Sprintf("target_token = $%d", i))
		if strings.TrimSpace(*p.TargetToken) == "" {
			args = append(args, nil) // clear
		} else {
			args = append(args, strings.TrimSpace(*p.TargetToken))
		}
		i++
	}
	if p.TargetProducts != nil {
		set = append(set, fmt.Sprintf("target_products = $%d", i))
		args = append(args, pq.Array(*p.TargetProducts))
		i++
	}
	if p.TargetAvatars != nil {
		set = append(set, fmt.Sprintf("target_avatars = $%d", i))
		args = append(args, pq.Array(*p.TargetAvatars))
		i++
	}
	if p.IsPublished != nil {
		set = append(set, fmt.Sprintf("is_published = $%d", i))
		args = append(args, *p.IsPublished)
		i++
	}
	if p.PublishedAt != nil {
		set = append(set, fmt.Sprintf("published_at = $%d", i))
		if p.PublishedAt.IsZero() {
			args = append(args, nil)
		} else {
			args = append(args, p.PublishedAt.UTC())
		}
		i++
	}
	if p.Attachments != nil {
		set = append(set, fmt.Sprintf("attachments = $%d", i))
		args = append(args, pq.Array(*p.Attachments))
		i++
	}
	if p.Status != nil {
		set = append(set, fmt.Sprintf("status = $%d", i))
		args = append(args, string(*p.Status))
		i++
	}
	if p.UpdatedBy != nil {
		set = append(set, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, strings.TrimSpace(*p.UpdatedBy))
		i++
	}
	if p.DeletedAt != nil {
		set = append(set, fmt.Sprintf("deleted_at = $%d", i))
		if p.DeletedAt.IsZero() {
			args = append(args, nil)
		} else {
			args = append(args, p.DeletedAt.UTC())
		}
		i++
	}
	if p.DeletedBy != nil {
		set = append(set, fmt.Sprintf("deleted_by = $%d", i))
		if strings.TrimSpace(*p.DeletedBy) == "" {
			args = append(args, nil)
		} else {
			args = append(args, strings.TrimSpace(*p.DeletedBy))
		}
		i++
	}

	// Always bump updated_at
	set = append(set, fmt.Sprintf("updated_at = $%d", i))
	now := time.Now().UTC()
	args = append(args, now)
	i++

	if len(set) == 0 {
		return r.GetByID(ctx, id)
	}

	q := fmt.Sprintf(`
UPDATE announcements
SET %s
WHERE id = $%d
RETURNING
  id, title, content, category, target_audience, target_token,
  target_products, target_avatars, is_published, published_at,
  attachments, status, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(set, ", "), i)
	args = append(args, id)

	row := r.db.QueryRowContext(ctx, q, args...)
	a, err := scanAnnouncementRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return announcement.Announcement{}, announcement.ErrNotFound
		}
		return announcement.Announcement{}, err
	}
	return a, nil
}

func (r *AnnouncementRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return announcement.ErrNotFound
	}
	return nil
}

func (r *AnnouncementRepositoryPG) Save(ctx context.Context, a announcement.Announcement, _ *common.SaveOptions) (announcement.Announcement, error) {
	q := `
INSERT INTO announcements (
  id, title, content, category, target_audience, target_token,
  target_products, target_avatars, is_published, published_at,
  attachments, status, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,
  $11,$12,$13,$14,$15,$16,$17,$18
)
ON CONFLICT (id) DO UPDATE SET
  title = EXCLUDED.title,
  content = EXCLUDED.content,
  category = EXCLUDED.category,
  target_audience = EXCLUDED.target_audience,
  target_token = EXCLUDED.target_token,
  target_products = EXCLUDED.target_products,
  target_avatars = EXCLUDED.target_avatars,
  is_published = EXCLUDED.is_published,
  published_at = EXCLUDED.published_at,
  attachments = EXCLUDED.attachments,
  status = EXCLUDED.status,
  created_at = LEAST(announcements.created_at, EXCLUDED.created_at),
  created_by = COALESCE(EXCLUDED.created_by, announcements.created_by),
  updated_at = EXCLUDED.updated_at,
  updated_by = COALESCE(EXCLUDED.updated_by, announcements.updated_by),
  deleted_at = EXCLUDED.deleted_at,
  deleted_by = EXCLUDED.deleted_by
RETURNING
  id, title, content, category, target_audience, target_token,
  target_products, target_avatars, is_published, published_at,
  attachments, status, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by
`
	var (
		targetToken any
		updatedAt   any
		updatedBy   any
		deletedAt   any
		deletedBy   any
		publishedAt any
	)
	if a.TargetToken != nil {
		targetToken = *a.TargetToken
	}
	if a.UpdatedAt != nil {
		updatedAt = a.UpdatedAt.UTC()
	}
	if a.UpdatedBy != nil {
		updatedBy = *a.UpdatedBy
	}
	if a.DeletedAt != nil {
		deletedAt = a.DeletedAt.UTC()
	}
	if a.DeletedBy != nil {
		deletedBy = *a.DeletedBy
	}
	if a.PublishedAt != nil {
		publishedAt = a.PublishedAt.UTC()
	}

	row := r.db.QueryRowContext(ctx, q,
		a.ID, a.Title, a.Content, string(a.Category), string(a.TargetAudience), targetToken,
		pq.Array(a.TargetProducts), pq.Array(a.TargetAvatars), a.IsPublished, publishedAt,
		pq.Array(a.Attachments), string(a.Status), a.CreatedAt.UTC(), a.CreatedBy, updatedAt, updatedBy, deletedAt, deletedBy,
	)
	out, err := scanAnnouncementRow(row)
	if err != nil {
		return announcement.Announcement{}, err
	}
	return out, nil
}

// ========== Helpers ==========

func scanAnnouncementRow(s dbcommon.RowScanner) (announcement.Announcement, error) {
	var (
		id, title, content                  string
		categoryStr, audienceStr, statusStr string

		targetTokenNS, updatedByNS, deletedByNS sql.NullString

		targetProducts, targetAvatars, attachments []string

		isPublished bool

		createdAt                               time.Time
		createdBy                               string
		publishedAtNT, updatedAtNT, deletedAtNT sql.NullTime
	)

	err := s.Scan(
		&id, &title, &content, &categoryStr, &audienceStr, &targetTokenNS,
		pq.Array(&targetProducts), pq.Array(&targetAvatars), &isPublished, &publishedAtNT,
		pq.Array(&attachments), &statusStr, &createdAt, &createdBy, &updatedAtNT, &updatedByNS, &deletedAtNT, &deletedByNS,
	)
	if err != nil {
		return announcement.Announcement{}, err
	}

	// Optional pointers
	var (
		publishedAtPtr, updatedAtPtr, deletedAtPtr *time.Time
		targetTokenPtr, updatedByPtr, deletedByPtr *string
	)
	if publishedAtNT.Valid {
		t := publishedAtNT.Time.UTC()
		publishedAtPtr = &t
	}
	if updatedAtNT.Valid {
		t := updatedAtNT.Time.UTC()
		updatedAtPtr = &t
	}
	if deletedAtNT.Valid {
		t := deletedAtNT.Time.UTC()
		deletedAtPtr = &t
	}
	if targetTokenNS.Valid {
		v := targetTokenNS.String
		targetTokenPtr = &v
	}
	if updatedByNS.Valid {
		v := updatedByNS.String
		updatedByPtr = &v
	}
	if deletedByNS.Valid {
		v := deletedByNS.String
		deletedByPtr = &v
	}

	return announcement.New(
		id, title, content,
		announcement.AnnouncementCategory(categoryStr),
		announcement.TargetAudience(audienceStr),
		targetTokenPtr,
		targetProducts, targetAvatars, attachments,
		isPublished,
		announcement.AnnouncementStatus(statusStr),
		createdAt.UTC(),
		createdBy,
		publishedAtPtr, updatedAtPtr, deletedAtPtr,
		updatedByPtr, deletedByPtr,
	)
}

func buildAnnouncementWhere(f announcement.Filter) (string, []any) {
	var conds []string
	var args []any
	i := 1

	if qs := strings.TrimSpace(f.SearchQuery); qs != "" {
		like := "%" + qs + "%"
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR content ILIKE $%d)", i, i))
		args = append(args, like)
		i++
	}

	if len(f.Categories) > 0 {
		ph := make([]string, len(f.Categories))
		for idx, c := range f.Categories {
			ph[idx] = fmt.Sprintf("$%d", i)
			args = append(args, string(c))
			i++
		}
		conds = append(conds, fmt.Sprintf("category IN (%s)", strings.Join(ph, ",")))
	}

	if len(f.Audiences) > 0 {
		ph := make([]string, len(f.Audiences))
		for idx, a := range f.Audiences {
			ph[idx] = fmt.Sprintf("$%d", i)
			args = append(args, string(a))
			i++
		}
		conds = append(conds, fmt.Sprintf("target_audience IN (%s)", strings.Join(ph, ",")))
	}

	if len(f.Statuses) > 0 {
		ph := make([]string, len(f.Statuses))
		for idx, s := range f.Statuses {
			ph[idx] = fmt.Sprintf("$%d", i)
			args = append(args, string(s))
			i++
		}
		conds = append(conds, fmt.Sprintf("status IN (%s)", strings.Join(ph, ",")))
	}

	if f.TargetToken != nil && strings.TrimSpace(*f.TargetToken) != "" {
		conds = append(conds, fmt.Sprintf("target_token = $%d", i))
		args = append(args, strings.TrimSpace(*f.TargetToken))
		i++
	}

	if len(f.TargetProducts) > 0 {
		conds = append(conds, fmt.Sprintf("target_products && $%d", i))
		args = append(args, pq.Array(f.TargetProducts))
		i++
	}

	if len(f.TargetAvatars) > 0 {
		conds = append(conds, fmt.Sprintf("target_avatars && $%d", i))
		args = append(args, pq.Array(f.TargetAvatars))
		i++
	}

	if f.IsPublished != nil {
		conds = append(conds, fmt.Sprintf("is_published = $%d", i))
		args = append(args, *f.IsPublished)
		i++
	}

	if f.CreatedFrom != nil {
		conds = append(conds, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, (*f.CreatedFrom).UTC())
		i++
	}
	if f.CreatedTo != nil {
		conds = append(conds, fmt.Sprintf("created_at <= $%d", i))
		args = append(args, (*f.CreatedTo).UTC())
		i++
	}
	if f.UpdatedFrom != nil {
		conds = append(conds, fmt.Sprintf("(updated_at IS NOT NULL AND updated_at >= $%d)", i))
		args = append(args, (*f.UpdatedFrom).UTC())
		i++
	}
	if f.UpdatedTo != nil {
		conds = append(conds, fmt.Sprintf("(updated_at IS NOT NULL AND updated_at <= $%d)", i))
		args = append(args, (*f.UpdatedTo).UTC())
		i++
	}
	if f.PublishedFrom != nil {
		conds = append(conds, fmt.Sprintf("(published_at IS NOT NULL AND published_at >= $%d)", i))
		args = append(args, (*f.PublishedFrom).UTC())
		i++
	}
	if f.PublishedTo != nil {
		conds = append(conds, fmt.Sprintf("(published_at IS NOT NULL AND published_at <= $%d)", i))
		args = append(args, (*f.PublishedTo).UTC())
		i++
	}

	if f.Deleted != nil {
		if *f.Deleted {
			conds = append(conds, "deleted_at IS NOT NULL")
		} else {
			conds = append(conds, "deleted_at IS NULL")
		}
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	return where, args
}

func buildAnnouncementOrderBy(_ common.Sort) string {
	// Unknown common.Sort shape; default for stability.
	return "ORDER BY created_at DESC, id DESC"
}
