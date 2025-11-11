// backend\internal\adapters\out\firestore\inquiry_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	idom "narratives/internal/domain/inquiry"
)

// InquiryRepositoryPG implements inquiry.Repository using PostgreSQL.
type InquiryRepositoryPG struct {
	DB *sql.DB
}

func NewInquiryRepositoryPG(db *sql.DB) *InquiryRepositoryPG {
	return &InquiryRepositoryPG{DB: db}
}

// =======================
// Queries
// =======================

func (r *InquiryRepositoryPG) List(ctx context.Context, filter idom.Filter, sort idom.Sort, page idom.Page) (idom.PageResult[idom.Inquiry], error) {
	where, args := buildInquiryWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildInquiryOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY created_at DESC, id DESC"
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM inquiries %s", whereSQL)
	if err := r.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return idom.PageResult[idom.Inquiry]{}, err
	}

	q := fmt.Sprintf(`
SELECT
  id, avatar_id, subject, content, status, inquiry_type,
  product_blueprint_id, token_blueprint_id, assignee_id, image,
  created_at, updated_at, updated_by, deleted_at, deleted_by
FROM inquiries
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return idom.PageResult[idom.Inquiry]{}, err
	}
	defer rows.Close()

	var items []idom.Inquiry
	for rows.Next() {
		in, err := scanInquiry(rows)
		if err != nil {
			return idom.PageResult[idom.Inquiry]{}, err
		}
		items = append(items, in)
	}
	if err := rows.Err(); err != nil {
		return idom.PageResult[idom.Inquiry]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return idom.PageResult[idom.Inquiry]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *InquiryRepositoryPG) ListByCursor(ctx context.Context, filter idom.Filter, _ idom.Sort, cpage idom.CursorPage) (idom.CursorPageResult[idom.Inquiry], error) {
	where, args := buildInquiryWhere(filter)
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
  id, avatar_id, subject, content, status, inquiry_type,
  product_blueprint_id, token_blueprint_id, assignee_id, image,
  created_at, updated_at, updated_by, deleted_at, deleted_by
FROM inquiries
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return idom.CursorPageResult[idom.Inquiry]{}, err
	}
	defer rows.Close()

	var items []idom.Inquiry
	var lastID string
	for rows.Next() {
		in, err := scanInquiry(rows)
		if err != nil {
			return idom.CursorPageResult[idom.Inquiry]{}, err
		}
		items = append(items, in)
		lastID = in.ID
	}
	if err := rows.Err(); err != nil {
		return idom.CursorPageResult[idom.Inquiry]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return idom.CursorPageResult[idom.Inquiry]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *InquiryRepositoryPG) GetByID(ctx context.Context, id string) (idom.Inquiry, error) {
	const q = `
SELECT
  id, avatar_id, subject, content, status, inquiry_type,
  product_blueprint_id, token_blueprint_id, assignee_id, image,
  created_at, updated_at, updated_by, deleted_at, deleted_by
FROM inquiries
WHERE id = $1
`
	row := r.DB.QueryRowContext(ctx, q, id)
	in, err := scanInquiry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return idom.Inquiry{}, idom.ErrNotFound
		}
		return idom.Inquiry{}, err
	}
	return in, nil
}

func (r *InquiryRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	const q = `SELECT 1 FROM inquiries WHERE id = $1`
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

func (r *InquiryRepositoryPG) Count(ctx context.Context, filter idom.Filter) (int, error) {
	where, args := buildInquiryWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM inquiries `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// =======================
// Mutations
// =======================

func (r *InquiryRepositoryPG) Create(ctx context.Context, inq idom.Inquiry) (idom.Inquiry, error) {
	const q = `
INSERT INTO inquiries (
  id, avatar_id, subject, content, status, inquiry_type,
  product_blueprint_id, token_blueprint_id, assignee_id, image,
  created_at, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,
  $11,$12,$13,$14,$15
)
RETURNING
  id, avatar_id, subject, content, status, inquiry_type,
  product_blueprint_id, token_blueprint_id, assignee_id, image,
  created_at, updated_at, updated_by, deleted_at, deleted_by
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(inq.ID),
		strings.TrimSpace(inq.AvatarID),
		strings.TrimSpace(inq.Subject),
		strings.TrimSpace(inq.Content),
		strings.TrimSpace(string(inq.Status)),
		strings.TrimSpace(string(inq.InquiryType)),
		dbcommon.ToDBText(inq.ProductBlueprintID),
		dbcommon.ToDBText(inq.TokenBlueprintID),
		dbcommon.ToDBText(inq.AssigneeID),
		dbcommon.ToDBText(inq.ImageID), // column: image
		inq.CreatedAt.UTC(),
		inq.UpdatedAt.UTC(),
		dbcommon.ToDBText(inq.UpdatedBy),
		dbcommon.ToDBTime(inq.DeletedAt),
		dbcommon.ToDBText(inq.DeletedBy),
	)
	out, err := scanInquiry(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return idom.Inquiry{}, idom.ErrConflict
		}
		return idom.Inquiry{}, err
	}
	return out, nil
}

func (r *InquiryRepositoryPG) Update(ctx context.Context, id string, patch idom.InquiryPatch) (idom.Inquiry, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if patch.Subject != nil {
		sets = append(sets, fmt.Sprintf("subject = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Subject))
		i++
	}
	if patch.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", i))
		args = append(args, strings.TrimSpace(*patch.Content))
		i++
	}
	if patch.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", i))
		args = append(args, strings.TrimSpace(string(*patch.Status)))
		i++
	}
	if patch.InquiryType != nil {
		sets = append(sets, fmt.Sprintf("inquiry_type = $%d", i))
		args = append(args, strings.TrimSpace(string(*patch.InquiryType)))
		i++
	}
	if patch.ProductBlueprintID != nil {
		sets = append(sets, fmt.Sprintf("product_blueprint_id = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.ProductBlueprintID))
		i++
	}
	if patch.TokenBlueprintID != nil {
		sets = append(sets, fmt.Sprintf("token_blueprint_id = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.TokenBlueprintID))
		i++
	}
	if patch.AssigneeID != nil {
		sets = append(sets, fmt.Sprintf("assignee_id = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.AssigneeID))
		i++
	}
	// Image(ID) の更新（domain の InquiryPatch は Image を想定）
	if patch.Image != nil {
		sets = append(sets, fmt.Sprintf("image = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.Image))
		i++
	}
	if patch.UpdatedBy != nil {
		sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, dbcommon.ToDBText(patch.UpdatedBy))
		i++
	}
	// updated_at: explicit or NOW()
	if patch.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, patch.UpdatedAt.UTC())
		i++
	} else if len(sets) > 0 {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, time.Now().UTC())
		i++
	}
	// deleted_at/by optional
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

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(`
UPDATE inquiries
SET %s
WHERE id = $%d
RETURNING
  id, avatar_id, subject, content, status, inquiry_type,
  product_blueprint_id, token_blueprint_id, assignee_id, image,
  created_at, updated_at, updated_by, deleted_at, deleted_by
`, strings.Join(sets, ", "), i)

	row := r.DB.QueryRowContext(ctx, q, args...)
	out, err := scanInquiry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return idom.Inquiry{}, idom.ErrNotFound
		}
		return idom.Inquiry{}, err
	}
	return out, nil
}

func (r *InquiryRepositoryPG) Delete(ctx context.Context, id string) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM inquiries WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return idom.ErrNotFound
	}
	return nil
}

func (r *InquiryRepositoryPG) Save(ctx context.Context, inq idom.Inquiry, _ *idom.SaveOptions) (idom.Inquiry, error) {
	const q = `
INSERT INTO inquiries (
  id, avatar_id, subject, content, status, inquiry_type,
  product_blueprint_id, token_blueprint_id, assignee_id, image,
  created_at, updated_at, updated_by, deleted_at, deleted_by
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,
  $11,$12,$13,$14,$15
)
ON CONFLICT (id) DO UPDATE SET
  avatar_id            = EXCLUDED.avatar_id,
  subject              = EXCLUDED.subject,
  content              = EXCLUDED.content,
  status               = EXCLUDED.status,
  inquiry_type         = EXCLUDED.inquiry_type,
  product_blueprint_id = EXCLUDED.product_blueprint_id,
  token_blueprint_id   = EXCLUDED.token_blueprint_id,
  assignee_id          = EXCLUDED.assignee_id,
  image                = EXCLUDED.image,
  created_at           = LEAST(inquiries.created_at, EXCLUDED.created_at),
  updated_by           = EXCLUDED.updated_by,
  updated_at           = COALESCE(EXCLUDED.updated_at, NOW()),
  deleted_at           = EXCLUDED.deleted_at,
  deleted_by           = EXCLUDED.deleted_by
RETURNING
  id, avatar_id, subject, content, status, inquiry_type,
  product_blueprint_id, token_blueprint_id, assignee_id, image,
  created_at, updated_at, updated_by, deleted_at, deleted_by
`
	row := r.DB.QueryRowContext(ctx, q,
		strings.TrimSpace(inq.ID),
		strings.TrimSpace(inq.AvatarID),
		strings.TrimSpace(inq.Subject),
		strings.TrimSpace(inq.Content),
		strings.TrimSpace(string(inq.Status)),
		strings.TrimSpace(string(inq.InquiryType)),
		dbcommon.ToDBText(inq.ProductBlueprintID),
		dbcommon.ToDBText(inq.TokenBlueprintID),
		dbcommon.ToDBText(inq.AssigneeID),
		dbcommon.ToDBText(inq.ImageID), // column: image
		inq.CreatedAt.UTC(),
		inq.UpdatedAt.UTC(),
		dbcommon.ToDBText(inq.UpdatedBy),
		dbcommon.ToDBTime(inq.DeletedAt),
		dbcommon.ToDBText(inq.DeletedBy),
	)
	out, err := scanInquiry(row)
	if err != nil {
		return idom.Inquiry{}, err
	}
	return out, nil
}

// =======================
// Helpers
// =======================

func scanInquiry(s dbcommon.RowScanner) (idom.Inquiry, error) {
	var (
		idNS, avatarIDNS, subjectNS, contentNS   sql.NullString
		statusNS, inquiryTypeNS                  sql.NullString
		productBlueprintIDNS, tokenBlueprintIDNS sql.NullString
		assigneeIDNS, imageNS                    sql.NullString
		updatedByNS, deletedByNS                 sql.NullString
		createdAt                                time.Time
		updatedAt                                time.Time
		deletedAtNT                              sql.NullTime
	)

	if err := s.Scan(
		&idNS, &avatarIDNS, &subjectNS, &contentNS, &statusNS, &inquiryTypeNS,
		&productBlueprintIDNS, &tokenBlueprintIDNS, &assigneeIDNS, &imageNS,
		&createdAt, &updatedAt, &updatedByNS, &deletedAtNT, &deletedByNS,
	); err != nil {
		return idom.Inquiry{}, err
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

	return idom.Inquiry{
		ID:                 strings.TrimSpace(idNS.String),
		AvatarID:           strings.TrimSpace(avatarIDNS.String),
		Subject:            strings.TrimSpace(subjectNS.String),
		Content:            strings.TrimSpace(contentNS.String),
		Status:             idom.InquiryStatus(strings.TrimSpace(statusNS.String)),
		InquiryType:        idom.InquiryType(strings.TrimSpace(inquiryTypeNS.String)),
		ProductBlueprintID: toPtrStr(productBlueprintIDNS),
		TokenBlueprintID:   toPtrStr(tokenBlueprintIDNS),
		AssigneeID:         toPtrStr(assigneeIDNS),
		ImageID:            toPtrStr(imageNS), // column: image → field: ImageID
		CreatedAt:          createdAt.UTC(),
		UpdatedAt:          updatedAt.UTC(),
		UpdatedBy:          toPtrStr(updatedByNS),
		DeletedAt:          toPtrTime(deletedAtNT),
		DeletedBy:          toPtrStr(deletedByNS),
	}, nil
}

func buildInquiryWhere(f idom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	// Free text search
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		where = append(where, fmt.Sprintf(
			"(subject ILIKE $%d OR content ILIKE $%d OR COALESCE(updated_by,'') ILIKE $%d OR COALESCE(assignee_id,'') ILIKE $%d)",
			len(args)+1, len(args)+1, len(args)+1, len(args)+1,
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
			where = append(where, "id IN ("+strings.Join(ph, ",")+")")
		}
	}

	// Avatar
	if f.AvatarID != nil && strings.TrimSpace(*f.AvatarID) != "" {
		where = append(where, fmt.Sprintf("avatar_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.AvatarID))
	}

	// Assignee
	if f.AssigneeID != nil && strings.TrimSpace(*f.AssigneeID) != "" {
		where = append(where, fmt.Sprintf("assignee_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.AssigneeID))
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

	// InquiryType single and list
	if f.InquiryType != nil && strings.TrimSpace(string(*f.InquiryType)) != "" {
		where = append(where, fmt.Sprintf("inquiry_type = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(string(*f.InquiryType)))
	}
	if len(f.InquiryTypes) > 0 {
		ph := []string{}
		for _, it := range f.InquiryTypes {
			v := strings.TrimSpace(string(it))
			if v == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, v)
		}
		if len(ph) > 0 {
			where = append(where, "inquiry_type IN ("+strings.Join(ph, ",")+")")
		}
	}

	// Blueprint IDs
	if f.ProductBlueprintID != nil && strings.TrimSpace(*f.ProductBlueprintID) != "" {
		where = append(where, fmt.Sprintf("product_blueprint_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.ProductBlueprintID))
	}
	if f.TokenBlueprintID != nil && strings.TrimSpace(*f.TokenBlueprintID) != "" {
		where = append(where, fmt.Sprintf("token_blueprint_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.TokenBlueprintID))
	}

	// HasImage tri-state
	if f.HasImage != nil {
		if *f.HasImage {
			where = append(where, "image IS NOT NULL")
		} else {
			where = append(where, "image IS NULL")
		}
	}

	// UpdatedBy / DeletedBy
	if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" {
		where = append(where, fmt.Sprintf("updated_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.UpdatedBy))
	}
	if f.DeletedBy != nil && strings.TrimSpace(*f.DeletedBy) != "" {
		where = append(where, fmt.Sprintf("deleted_by = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.DeletedBy))
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
	if f.DeletedFrom != nil {
		where = append(where, fmt.Sprintf("(deleted_at IS NOT NULL AND deleted_at >= $%d)", len(args)+1))
		args = append(args, f.DeletedFrom.UTC())
	}
	if f.DeletedTo != nil {
		where = append(where, fmt.Sprintf("(deleted_at IS NOT NULL AND deleted_at < $%d)", len(args)+1))
		args = append(args, f.DeletedTo.UTC())
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			where = append(where, "deleted_at IS NOT NULL")
		} else {
			where = append(where, "deleted_at IS NULL")
		}
	}

	return where, args
}

func buildInquiryOrderBy(sort idom.Sort) string {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "id":
		col = "id"
	case "avatarid", "avatar_id":
		col = "avatar_id"
	case "subject":
		col = "subject"
	case "status":
		col = "status"
	case "inquirytype", "inquiry_type":
		col = "inquiry_type"
	case "assigneeid", "assignee_id":
		col = "assignee_id"
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	case "deletedat", "deleted_at":
		col = "deleted_at"
	default:
		return ""
	}

	dir := strings.ToUpper(string(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}
