// backend\internal\adapters\out\firestore\message_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	msgdom "narratives/internal/domain/message"
)

type MessageRepositoryPG struct {
	DB *sql.DB
}

func NewMessageRepositoryPG(db *sql.DB) *MessageRepositoryPG {
	return &MessageRepositoryPG{DB: db}
}

// WithTx opens a transaction and executes fn within it.
func (r *MessageRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	ctxTx := dbcommon.CtxWithTx(ctx, tx)
	if err := fn(ctxTx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// =======================
// Scan helpers
// =======================

func scanMessage(s dbcommon.RowScanner) (msgdom.Message, error) {
	var (
		id, senderID, receiverID, content, status string
		createdAt                                 time.Time
		updatedAtNS, deletedAtNS, readAtNS        sql.NullTime
	)
	if err := s.Scan(
		&id, &senderID, &receiverID, &content, &status,
		&createdAt, &updatedAtNS, &deletedAtNS, &readAtNS,
	); err != nil {
		return msgdom.Message{}, err
	}
	toPtr := func(nt sql.NullTime) *time.Time {
		if nt.Valid {
			t := nt.Time.UTC()
			return &t
		}
		return nil
	}
	return msgdom.Message{
		ID:         strings.TrimSpace(id),
		SenderID:   strings.TrimSpace(senderID),
		ReceiverID: strings.TrimSpace(receiverID),
		Content:    content,
		Status:     msgdom.MessageStatus(status),
		Images:     nil, // images are not stored in messages table in this adapter
		CreatedAt:  createdAt.UTC(),
		UpdatedAt:  toPtr(updatedAtNS),
		DeletedAt:  toPtr(deletedAtNS),
		ReadAt:     toPtr(readAtNS),
	}, nil
}

// UPDATED: map subject -> LastMessageText, and fill domain fields
func scanThread(s dbcommon.RowScanner) (msgdom.MessageThread, error) {
	var (
		id, subject string
		lastMsgAt   time.Time
	)
	if err := s.Scan(&id, &subject, &lastMsgAt); err != nil {
		return msgdom.MessageThread{}, err
	}
	return msgdom.MessageThread{
		ID:              strings.TrimSpace(id),
		ParticipantIDs:  nil, // filled separately
		LastMessageID:   "",  // not available in this schema
		LastMessageAt:   lastMsgAt.UTC(),
		LastMessageText: subject,          // subject column mapped to summary
		UnreadCounts:    map[string]int{}, // not available here
		CreatedAt:       lastMsgAt.UTC(),  // fallback
		UpdatedAt:       nil,              // unknown
	}, nil
}

// Load participants for multiple thread IDs
func (r *MessageRepositoryPG) loadParticipants(ctx context.Context, ids []string) (map[string][]string, error) {
	if len(ids) == 0 {
		return map[string][]string{}, nil
	}
	run := dbcommon.GetRunner(ctx, r.DB)
	ph := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
		args = append(args, id)
	}
	q := fmt.Sprintf(`
SELECT thread_id, member_id
FROM message_thread_participants
WHERE thread_id IN (%s)
ORDER BY thread_id ASC, member_id ASC
`, strings.Join(ph, ","))
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string][]string, len(ids))
	for rows.Next() {
		var tid, mid string
		if err := rows.Scan(&tid, &mid); err != nil {
			return nil, err
		}
		res[tid] = append(res[tid], mid)
	}
	return res, rows.Err()
}

// =======================
// Message RepositoryPort impl (契約準拠)
// =======================

func (r *MessageRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	var one int
	err := run.QueryRowContext(ctx, `SELECT 1 FROM messages WHERE id = $1`, strings.TrimSpace(id)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *MessageRepositoryPG) Count(ctx context.Context, filter msgdom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	where, args := buildMessageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := run.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *MessageRepositoryPG) GetByID(ctx context.Context, id string) (msgdom.Message, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT id, sender_id, receiver_id, content, status,
       created_at, updated_at, deleted_at, read_at
FROM messages
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	m, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgdom.Message{}, msgdom.ErrNotFound
		}
		return msgdom.Message{}, err
	}
	m.Images = nil
	return m, nil
}

func (r *MessageRepositoryPG) List(ctx context.Context, filter msgdom.Filter, sort msgdom.Sort, page msgdom.Page) (msgdom.PageResult[msgdom.Message], error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildMessageWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildMessageOrderBy(sort)
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM messages %s", whereSQL)
	if err := run.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return msgdom.PageResult[msgdom.Message]{}, err
	}

	q := fmt.Sprintf(`
SELECT id, sender_id, receiver_id, content, status,
       created_at, updated_at, deleted_at, read_at
FROM messages
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return msgdom.PageResult[msgdom.Message]{}, err
	}
	defer rows.Close()

	items := make([]msgdom.Message, 0, perPage)
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return msgdom.PageResult[msgdom.Message]{}, err
		}
		m.Images = nil
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return msgdom.PageResult[msgdom.Message]{}, err
	}

	totalPages := (total + perPage - 1) / perPage
	return msgdom.PageResult[msgdom.Message]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *MessageRepositoryPG) ListByCursor(ctx context.Context, filter msgdom.Filter, _ msgdom.Sort, cpage msgdom.CursorPage) (msgdom.CursorPageResult[msgdom.Message], error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildMessageWhere(filter)
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
SELECT id, sender_id, receiver_id, content, status,
       created_at, updated_at, deleted_at, read_at
FROM messages
%s
ORDER BY id ASC
LIMIT $%d
`, whereSQL, len(args)+1)

	args = append(args, limit+1)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return msgdom.CursorPageResult[msgdom.Message]{}, err
	}
	defer rows.Close()

	var items []msgdom.Message
	var lastID string
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return msgdom.CursorPageResult[msgdom.Message]{}, err
		}
		m.Images = nil
		items = append(items, m)
		lastID = m.ID
	}
	if err := rows.Err(); err != nil {
		return msgdom.CursorPageResult[msgdom.Message]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return msgdom.CursorPageResult[msgdom.Message]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *MessageRepositoryPG) CreateDraft(ctx context.Context, m msgdom.Message) (msgdom.Message, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	if m.Status == "" {
		m.Status = msgdom.StatusDraft
	}
	const q = `
INSERT INTO messages (
  id, sender_id, receiver_id, content, status,
  created_at, updated_at, deleted_at, read_at
) VALUES (
  $1,$2,$3,$4,$5,
  $6,$7,$8,$9
)
RETURNING id, sender_id, receiver_id, content, status,
          created_at, updated_at, deleted_at, read_at`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(m.ID),
		strings.TrimSpace(m.SenderID),
		strings.TrimSpace(m.ReceiverID),
		m.Content,
		string(m.Status),
		m.CreatedAt.UTC(),
		dbcommon.ToDBTime(m.UpdatedAt),
		dbcommon.ToDBTime(m.DeletedAt),
		dbcommon.ToDBTime(m.ReadAt),
	)
	saved, err := scanMessage(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return msgdom.Message{}, msgdom.ErrConflict
		}
		return msgdom.Message{}, err
	}
	saved.Images = nil
	return saved, nil
}

func (r *MessageRepositoryPG) Patch(ctx context.Context, id string, patch msgdom.MessagePatch) (msgdom.Message, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	setStr := func(col string, p *string) {
		if p != nil {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, *p)
			i++
		}
	}

	setStr("content", patch.Content)

	// UpdatedAt explicit or NOW() if any changes
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
		// No changes, return current
		return r.GetByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE messages
SET %s
WHERE id = $%d
RETURNING id, sender_id, receiver_id, content, status,
          created_at, updated_at, deleted_at, read_at
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	out, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgdom.Message{}, msgdom.ErrNotFound
		}
		return msgdom.Message{}, err
	}
	out.Images = nil
	return out, nil
}

func (r *MessageRepositoryPG) Send(ctx context.Context, id string, at time.Time) (msgdom.Message, error) {
	return r.updateStatus(ctx, id, msgdom.StatusDraft, msgdom.StatusSent, at, false, false)
}

func (r *MessageRepositoryPG) Cancel(ctx context.Context, id string, at time.Time) (msgdom.Message, error) {
	return r.updateStatus(ctx, id, msgdom.StatusSent, msgdom.StatusCanceled, at, false, false)
}

func (r *MessageRepositoryPG) MarkDelivered(ctx context.Context, id string, at time.Time) (msgdom.Message, error) {
	return r.updateStatus(ctx, id, msgdom.StatusSent, msgdom.StatusDelivered, at, false, false)
}

func (r *MessageRepositoryPG) MarkRead(ctx context.Context, id string, at time.Time) (msgdom.Message, error) {
	return r.updateStatus(ctx, id, msgdom.StatusDelivered, msgdom.StatusRead, at, true, false)
}

func (r *MessageRepositoryPG) updateStatus(ctx context.Context, id string, expect msgdom.MessageStatus, next msgdom.MessageStatus, at time.Time, setReadAt bool, setCanceledAt bool) (msgdom.Message, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	if at.IsZero() {
		at = time.Now()
	}
	clauses := []string{"status = $2", "updated_at = $3"}
	args := []any{strings.TrimSpace(id), string(next), at.UTC()}
	i := 4
	if setReadAt {
		clauses = append(clauses, fmt.Sprintf("read_at = COALESCE(read_at, $%d)", i))
		args = append(args, at.UTC())
		i++
	}
	// 現在のスキーマには canceled_at カラムが無いため、フラグは無視（将来拡張用）
	if setCanceledAt {
		// 例: 追加時は以下を有効化
		// clauses = append(clauses, fmt.Sprintf("canceled_at = COALESCE(canceled_at, $%d)", i))
		// args = append(args, at.UTC())
		// i++
	}

	q := fmt.Sprintf(`
UPDATE messages
SET %s
WHERE id = $1 AND status = $%d
RETURNING id, sender_id, receiver_id, content, status,
          created_at, updated_at, deleted_at, read_at
`, strings.Join(clauses, ", "), i)
	args = append(args, string(expect))

	row := run.QueryRowContext(ctx, q, args...)
	m, err := scanMessage(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgdom.Message{}, msgdom.ErrNotFound
		}
		return msgdom.Message{}, err
	}
	m.Images = nil
	return m, nil
}

func (r *MessageRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM messages WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return msgdom.ErrNotFound
	}
	return nil
}

func (r *MessageRepositoryPG) Save(ctx context.Context, m msgdom.Message) (msgdom.Message, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
INSERT INTO messages (
  id, sender_id, receiver_id, content, status,
  created_at, updated_at, deleted_at, read_at
) VALUES (
  $1,$2,$3,$4,$5,
  $6,$7,$8,$9
)
ON CONFLICT (id) DO UPDATE SET
  sender_id  = EXCLUDED.sender_id,
  receiver_id= EXCLUDED.receiver_id,
  content    = EXCLUDED.content,
  status     = EXCLUDED.status,
  created_at = LEAST(messages.created_at, EXCLUDED.created_at),
  updated_at = COALESCE(EXCLUDED.updated_at, messages.updated_at),
  deleted_at = EXCLUDED.deleted_at,
  read_at    = COALESCE(EXCLUDED.read_at, messages.read_at)
RETURNING id, sender_id, receiver_id, content, status,
          created_at, updated_at, deleted_at, read_at
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(m.ID),
		strings.TrimSpace(m.SenderID),
		strings.TrimSpace(m.ReceiverID),
		m.Content,
		string(m.Status),
		m.CreatedAt.UTC(),
		dbcommon.ToDBTime(m.UpdatedAt),
		dbcommon.ToDBTime(m.DeletedAt),
		dbcommon.ToDBTime(m.ReadAt),
	)
	out, err := scanMessage(row)
	if err != nil {
		return msgdom.Message{}, err
	}
	out.Images = nil
	return out, nil
}

// =======================
// Threads (ThreadRepository 契約準拠)
// =======================

func buildThreadWhere(f msgdom.ThreadFilter) ([]string, []any) {
	where := []string{}
	args := []any{}

	if q := strings.TrimSpace(f.SearchQuery); q != "" {
		// subject カラムをスレッド要約として検索に利用
		where = append(where, fmt.Sprintf("(mt.subject ILIKE $%d)", len(args)+1))
		args = append(args, "%"+q+"%")
	}
	if f.ParticipantID != nil && strings.TrimSpace(*f.ParticipantID) != "" {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM message_thread_participants p WHERE p.thread_id = mt.id AND p.member_id = $%d)", len(args)+1))
		args = append(args, strings.TrimSpace(*f.ParticipantID))
	}
	if len(f.ParticipantIDs) > 0 {
		ors := []string{}
		for _, id := range f.ParticipantIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			ors = append(ors, fmt.Sprintf("EXISTS (SELECT 1 FROM message_thread_participants p WHERE p.thread_id = mt.id AND p.member_id = $%d)", len(args)+1))
			args = append(args, id)
		}
		if len(ors) > 0 {
			where = append(where, "("+strings.Join(ors, " OR ")+")")
		}
	}
	// created/updated が無いスキーマ想定のため last_message_at を流用
	if f.CreatedFrom != nil {
		where = append(where, fmt.Sprintf("mt.last_message_at >= $%d", len(args)+1))
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedTo != nil {
		where = append(where, fmt.Sprintf("mt.last_message_at < $%d", len(args)+1))
		args = append(args, f.CreatedTo.UTC())
	}
	if f.UpdatedFrom != nil {
		where = append(where, fmt.Sprintf("mt.last_message_at >= $%d", len(args)+1))
		args = append(args, f.UpdatedFrom.UTC())
	}
	if f.UpdatedTo != nil {
		where = append(where, fmt.Sprintf("mt.last_message_at < $%d", len(args)+1))
		args = append(args, f.UpdatedTo.UTC())
	}
	if f.LastMessageFrom != nil {
		where = append(where, fmt.Sprintf("mt.last_message_at >= $%d", len(args)+1))
		args = append(args, f.LastMessageFrom.UTC())
	}
	if f.LastMessageTo != nil {
		where = append(where, fmt.Sprintf("mt.last_message_at < $%d", len(args)+1))
		args = append(args, f.LastMessageTo.UTC())
	}

	return where, args
}

func buildThreadOrderBy(sort msgdom.ThreadSort) string {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	switch col {
	case "lastmessageat", "last_message_at":
		col = "mt.last_message_at"
	case "createdat", "created_at":
		// fallback: use last_message_at if created_at not present
		col = "mt.last_message_at"
	case "updatedat", "updated_at":
		col = "mt.last_message_at"
	default:
		col = "mt.last_message_at"
	}
	dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, mt.id %s", col, dir, dir)
}

func (r *MessageRepositoryPG) CountThreads(ctx context.Context, filter msgdom.ThreadFilter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	where, args := buildThreadWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := run.QueryRowContext(ctx, `SELECT COUNT(*) FROM message_threads mt `+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *MessageRepositoryPG) ListThreads(ctx context.Context, filter msgdom.ThreadFilter, sort msgdom.ThreadSort, page msgdom.Page) (msgdom.PageResult[msgdom.MessageThread], error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildThreadWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildThreadOrderBy(sort)
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
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM message_threads mt %s", whereSQL)
	if err := run.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return msgdom.PageResult[msgdom.MessageThread]{}, err
	}

	q := fmt.Sprintf(`
SELECT mt.id, mt.subject, mt.last_message_at
FROM message_threads mt
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)
	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return msgdom.PageResult[msgdom.MessageThread]{}, err
	}
	defer rows.Close()

	var items []msgdom.MessageThread
	var ids []string
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return msgdom.PageResult[msgdom.MessageThread]{}, err
		}
		items = append(items, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return msgdom.PageResult[msgdom.MessageThread]{}, err
	}

	parts, err := r.loadParticipants(ctx, ids)
	if err != nil {
		return msgdom.PageResult[msgdom.MessageThread]{}, err
	}
	for i := range items {
		items[i].ParticipantIDs = append([]string(nil), parts[items[i].ID]...)
	}

	totalPages := (total + perPage - 1) / perPage
	return msgdom.PageResult[msgdom.MessageThread]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *MessageRepositoryPG) ListThreadsByCursor(ctx context.Context, filter msgdom.ThreadFilter, _ msgdom.ThreadSort, cpage msgdom.CursorPage) (msgdom.CursorPageResult[msgdom.MessageThread], error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildThreadWhere(filter)
	if after := strings.TrimSpace(cpage.After); after != "" {
		where = append(where, fmt.Sprintf("mt.id > $%d", len(args)+1))
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
SELECT mt.id, mt.subject, mt.last_message_at
FROM message_threads mt
%s
ORDER BY mt.id ASC
LIMIT $%d
`, whereSQL, len(args)+1)
	args = append(args, limit+1)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return msgdom.CursorPageResult[msgdom.MessageThread]{}, err
	}
	defer rows.Close()

	var items []msgdom.MessageThread
	var lastID string
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return msgdom.CursorPageResult[msgdom.MessageThread]{}, err
		}
		items = append(items, t)
		lastID = t.ID
	}
	if err := rows.Err(); err != nil {
		return msgdom.CursorPageResult[msgdom.MessageThread]{}, err
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	// participants
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	parts, err := r.loadParticipants(ctx, ids)
	if err != nil {
		return msgdom.CursorPageResult[msgdom.MessageThread]{}, err
	}
	for i := range items {
		items[i].ParticipantIDs = append([]string(nil), parts[items[i].ID]...)
	}

	return msgdom.CursorPageResult[msgdom.MessageThread]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *MessageRepositoryPG) GetThreadByID(ctx context.Context, id string) (msgdom.MessageThread, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT mt.id, mt.subject, mt.last_message_at
FROM message_threads mt
WHERE mt.id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	t, err := scanThread(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgdom.MessageThread{}, msgdom.ErrNotFound
		}
		return msgdom.MessageThread{}, err
	}
	parts, err := r.loadParticipants(ctx, []string{t.ID})
	if err != nil {
		return msgdom.MessageThread{}, err
	}
	t.ParticipantIDs = append([]string(nil), parts[t.ID]...)
	return t, nil
}

func (r *MessageRepositoryPG) SaveThread(ctx context.Context, t msgdom.MessageThread) (msgdom.MessageThread, error) {
	// Use existing tx if provided, otherwise start a new one for atomicity
	if tx := dbcommon.TxFromCtx(ctx); tx != nil {
		return r.saveThreadWithRunner(ctx, tx, t)
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return msgdom.MessageThread{}, err
	}
	out, err := r.saveThreadWithRunner(ctx, tx, t)
	if err != nil {
		_ = tx.Rollback()
		return msgdom.MessageThread{}, err
	}
	return out, tx.Commit()
}

func (r *MessageRepositoryPG) saveThreadWithRunner(ctx context.Context, run dbcommon.Runner, t msgdom.MessageThread) (msgdom.MessageThread, error) {
	const insUpsert = `
INSERT INTO message_threads (id, subject, last_message_at)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
  subject = EXCLUDED.subject,
  last_message_at = EXCLUDED.last_message_at
RETURNING id, subject, last_message_at`
	row := run.QueryRowContext(ctx, insUpsert,
		strings.TrimSpace(t.ID),
		strings.TrimSpace(t.LastMessageText), // map
		t.LastMessageAt.UTC(),
	)
	saved, err := scanThread(row)
	if err != nil {
		return msgdom.MessageThread{}, err
	}

	// Replace participants
	if _, err := run.ExecContext(ctx, `DELETE FROM message_thread_participants WHERE thread_id = $1`, saved.ID); err != nil {
		return msgdom.MessageThread{}, err
	}
	if len(t.ParticipantIDs) > 0 {
		vals := make([]string, 0, len(t.ParticipantIDs))
		args := make([]any, 0, len(t.ParticipantIDs)+1)
		args = append(args, saved.ID)
		for _, pid := range t.ParticipantIDs {
			vals = append(vals, fmt.Sprintf("($1, $%d)", len(args)+1))
			args = append(args, strings.TrimSpace(pid))
		}
		q := fmt.Sprintf(`INSERT INTO message_thread_participants (thread_id, member_id) VALUES %s`, strings.Join(vals, ","))
		if _, err := run.ExecContext(ctx, q, args...); err != nil {
			return msgdom.MessageThread{}, err
		}
	}

	// Reload participants
	parts, err := r.loadParticipants(ctx, []string{saved.ID})
	if err != nil {
		return msgdom.MessageThread{}, err
	}
	saved.ParticipantIDs = append([]string(nil), parts[saved.ID]...)
	return saved, nil
}

func (r *MessageRepositoryPG) DeleteThread(ctx context.Context, id string) error {
	// Use existing tx if provided, otherwise start a new one
	if tx := dbcommon.TxFromCtx(ctx); tx != nil {
		return r.deleteThreadWithRunner(ctx, tx, id)
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := r.deleteThreadWithRunner(ctx, tx, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *MessageRepositoryPG) deleteThreadWithRunner(ctx context.Context, run dbcommon.Runner, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return msgdom.ErrInvalid
	}
	if _, err := run.ExecContext(ctx, `DELETE FROM message_thread_participants WHERE thread_id = $1`, id); err != nil {
		return err
	}
	res, err := run.ExecContext(ctx, `DELETE FROM message_threads WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return msgdom.ErrNotFound
	}
	return nil
}

// =======================
// WHERE/ORDER builders (messages)
// =======================

func buildMessageWhere(f msgdom.Filter) ([]string, []any) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}

	if q := strings.TrimSpace(f.SearchQuery); q != "" {
		where = append(where, fmt.Sprintf("(content ILIKE $%d)", len(args)+1))
		args = append(args, "%"+q+"%")
	}
	if f.SenderID != nil && strings.TrimSpace(*f.SenderID) != "" {
		where = append(where, fmt.Sprintf("sender_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.SenderID))
	}
	if f.ReceiverID != nil && strings.TrimSpace(*f.ReceiverID) != "" {
		where = append(where, fmt.Sprintf("receiver_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(*f.ReceiverID))
	}
	if len(f.Statuses) > 0 {
		ph := []string{}
		for _, s := range f.Statuses {
			if strings.TrimSpace(string(s)) == "" {
				continue
			}
			ph = append(ph, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, strings.TrimSpace(string(s)))
		}
		if len(ph) > 0 {
			where = append(where, "status IN ("+strings.Join(ph, ",")+")")
		}
	}
	if f.UnreadOnly {
		where = append(where, "status <> 'read'")
	}
	if f.CreatedFrom != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, f.CreatedFrom.UTC())
	}
	if f.CreatedTo != nil {
		where = append(where, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, f.CreatedTo.UTC())
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

func buildMessageOrderBy(sort msgdom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	switch col {
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	case "status":
		col = "status"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(sort.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, id %s", col, dir, dir)
}

// =======================
// Existing helpers and non-required methods
// =======================

// UploadMessageImage is out of scope for PG adapter (kept for compatibility)
func (r *MessageRepositoryPG) UploadMessageImage(ctx context.Context, fileName string, contentType string, _ io.Reader) (string, error) {
	_ = ctx
	_ = fileName
	_ = contentType
	return "", fmt.Errorf("%w: UploadMessageImage not implemented in PG adapter", msgdom.ErrInvalid)
}
