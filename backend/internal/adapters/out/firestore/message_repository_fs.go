// backend/internal/adapters/out/firestore/message_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	msgdom "narratives/internal/domain/message"
)

// MessageRepositoryFS implements message.Repository using Firestore.
type MessageRepositoryFS struct {
	Client *gfs.Client
}

func NewMessageRepositoryFS(client *gfs.Client) *MessageRepositoryFS {
	return &MessageRepositoryFS{Client: client}
}

func (r *MessageRepositoryFS) col() *gfs.CollectionRef {
	return r.Client.Collection("messages")
}

func (r *MessageRepositoryFS) threadCol() *gfs.CollectionRef {
	return r.Client.Collection("message_threads")
}

func (r *MessageRepositoryFS) threadParticipantsCol(threadID string) *gfs.CollectionRef {
	return r.threadCol().Doc(threadID).Collection("participants")
}

// ========= Transaction context helpers =========

type txKey struct{}

func txFromCtx(ctx context.Context) *gfs.Transaction {
	if v := ctx.Value(txKey{}); v != nil {
		if tx, ok := v.(*gfs.Transaction); ok {
			return tx
		}
	}
	return nil
}

// WithTx executes fn within a Firestore transaction context.
func (r *MessageRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		ctxTx := context.WithValue(ctx, txKey{}, tx)
		return fn(ctxTx)
	})
}

// =======================
// Scan / encode / decode helpers
// =======================

func decodeMessageDoc(doc *gfs.DocumentSnapshot) (msgdom.Message, error) {
	data := doc.Data()
	if data == nil {
		return msgdom.Message{}, fmt.Errorf("empty message document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	getTimePtr := func(key string) *time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			t := v.UTC()
			return &t
		}
		return nil
	}
	getStatus := func(key string) msgdom.MessageStatus {
		if v, ok := data[key].(string); ok {
			return msgdom.MessageStatus(strings.TrimSpace(v))
		}
		return ""
	}

	m := msgdom.Message{
		ID:         strings.TrimSpace(doc.Ref.ID),
		SenderID:   getStr("senderId"),
		ReceiverID: getStr("receiverId"),
		Content:    getStr("content"),
		Status:     getStatus("status"),
		Images:     nil, // images are handled separately (e.g., via storage)
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  nil,
		DeletedAt:  nil,
		ReadAt:     nil,
	}

	if v, ok := data["createdAt"].(time.Time); ok && !v.IsZero() {
		m.CreatedAt = v.UTC()
	}
	m.UpdatedAt = getTimePtr("updatedAt")
	m.DeletedAt = getTimePtr("deletedAt")
	m.ReadAt = getTimePtr("readAt")

	return m, nil
}

func encodeMessageDoc(m msgdom.Message) map[string]any {
	out := map[string]any{
		"senderId":   strings.TrimSpace(m.SenderID),
		"receiverId": strings.TrimSpace(m.ReceiverID),
		"content":    m.Content,
		"status":     string(m.Status),
		"createdAt":  m.CreatedAt.UTC(),
	}

	if m.UpdatedAt != nil && !m.UpdatedAt.IsZero() {
		out["updatedAt"] = m.UpdatedAt.UTC()
	}
	if m.DeletedAt != nil && !m.DeletedAt.IsZero() {
		out["deletedAt"] = m.DeletedAt.UTC()
	}
	if m.ReadAt != nil && !m.ReadAt.IsZero() {
		out["readAt"] = m.ReadAt.UTC()
	}

	return out
}

func decodeThreadDoc(doc *gfs.DocumentSnapshot) (msgdom.MessageThread, error) {
	data := doc.Data()
	if data == nil {
		return msgdom.MessageThread{}, fmt.Errorf("empty message_thread document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getTime := func(keys ...string) (time.Time, bool) {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				return v.UTC(), true
			}
		}
		return time.Time{}, false
	}

	id := strings.TrimSpace(doc.Ref.ID)
	lastText := getStr("lastMessageText", "subject")
	lastAt, ok := getTime("lastMessageAt", "last_message_at")
	if !ok {
		lastAt = time.Now().UTC()
	}

	t := msgdom.MessageThread{
		ID:              id,
		ParticipantIDs:  nil, // filled separately
		LastMessageID:   "",
		LastMessageAt:   lastAt,
		LastMessageText: lastText,
		UnreadCounts:    map[string]int{},
		CreatedAt:       lastAt,
		UpdatedAt:       nil,
	}
	return t, nil
}

func encodeThreadDoc(t msgdom.MessageThread) map[string]any {
	lastAt := t.LastMessageAt
	if lastAt.IsZero() {
		lastAt = time.Now().UTC()
	}
	return map[string]any{
		"lastMessageText": strings.TrimSpace(t.LastMessageText),
		"lastMessageAt":   lastAt.UTC(),
	}
}

func (r *MessageRepositoryFS) loadThreadParticipants(
	ctx context.Context,
	threadIDs []string,
) (map[string][]string, error) {
	res := make(map[string][]string, len(threadIDs))
	for _, tid := range threadIDs {
		tid = strings.TrimSpace(tid)
		if tid == "" {
			continue
		}
		col := r.threadParticipantsCol(tid)
		it := col.Documents(ctx)
		var members []string
		for {
			doc, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				it.Stop()
				return nil, err
			}
			// memberId is either doc ID or field
			mid := doc.Ref.ID
			if v, ok := doc.Data()["memberId"].(string); ok && strings.TrimSpace(v) != "" {
				mid = strings.TrimSpace(v)
			}
			if strings.TrimSpace(mid) != "" {
				members = append(members, mid)
			}
		}
		it.Stop()
		res[tid] = members
	}
	return res, nil
}

// =======================
// Message RepositoryPort impl
// =======================

func (r *MessageRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *MessageRepositoryFS) Count(ctx context.Context, filter msgdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		m, err := decodeMessageDoc(doc)
		if err != nil {
			return 0, err
		}
		if matchMessageFilter(m, filter) {
			total++
		}
	}
	return total, nil
}

func (r *MessageRepositoryFS) GetByID(ctx context.Context, id string) (msgdom.Message, error) {
	if r.Client == nil {
		return msgdom.Message{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return msgdom.Message{}, msgdom.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return msgdom.Message{}, msgdom.ErrNotFound
		}
		return msgdom.Message{}, err
	}

	m, err := decodeMessageDoc(doc)
	if err != nil {
		return msgdom.Message{}, err
	}
	m.Images = nil
	return m, nil
}

func (r *MessageRepositoryFS) List(
	ctx context.Context,
	filter msgdom.Filter,
	sort msgdom.Sort,
	page msgdom.Page,
) (msgdom.PageResult[msgdom.Message], error) {
	if r.Client == nil {
		return msgdom.PageResult[msgdom.Message]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage := normalizePage(page.Number, page.PerPage, 50)

	q := r.col().Query
	q = applyMessageSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []msgdom.Message
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return msgdom.PageResult[msgdom.Message]{}, err
		}
		m, err := decodeMessageDoc(doc)
		if err != nil {
			return msgdom.PageResult[msgdom.Message]{}, err
		}
		if matchMessageFilter(m, filter) {
			m.Images = nil
			all = append(all, m)
		}
	}

	total := len(all)
	if total == 0 {
		return msgdom.PageResult[msgdom.Message]{
			Items:      []msgdom.Message{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	totalPages := computeTotalPages(total, perPage)

	return msgdom.PageResult[msgdom.Message]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *MessageRepositoryFS) ListByCursor(
	ctx context.Context,
	filter msgdom.Filter,
	_ msgdom.Sort,
	cpage msgdom.CursorPage,
) (msgdom.CursorPageResult[msgdom.Message], error) {
	if r.Client == nil {
		return msgdom.CursorPageResult[msgdom.Message]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Cursor by doc ID (string) ASC
	q := r.col().OrderBy(gfs.DocumentID, gfs.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items []msgdom.Message
		last  string
	)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return msgdom.CursorPageResult[msgdom.Message]{}, err
		}
		m, err := decodeMessageDoc(doc)
		if err != nil {
			return msgdom.CursorPageResult[msgdom.Message]{}, err
		}

		if skipping {
			if m.ID <= after {
				continue
			}
			skipping = false
		}

		if !matchMessageFilter(m, filter) {
			continue
		}

		m.Images = nil
		items = append(items, m)
		last = m.ID

		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &last
	}

	return msgdom.CursorPageResult[msgdom.Message]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *MessageRepositoryFS) CreateDraft(ctx context.Context, m msgdom.Message) (msgdom.Message, error) {
	if r.Client == nil {
		return msgdom.Message{}, errors.New("firestore client is nil")
	}

	if m.Status == "" {
		m.Status = msgdom.StatusDraft
	}

	id := strings.TrimSpace(m.ID)
	if id == "" {
		return msgdom.Message{}, errors.New("missing id")
	}

	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}

	docRef := r.col().Doc(id)
	data := encodeMessageDoc(m)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return msgdom.Message{}, msgdom.ErrConflict
		}
		return msgdom.Message{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *MessageRepositoryFS) Patch(
	ctx context.Context,
	id string,
	patch msgdom.MessagePatch,
) (msgdom.Message, error) {
	if r.Client == nil {
		return msgdom.Message{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return msgdom.Message{}, msgdom.ErrNotFound
	}

	var updates []gfs.Update

	if patch.Content != nil {
		updates = append(updates, gfs.Update{
			Path:  "content",
			Value: *patch.Content,
		})
	}

	// updatedAt explicit or NOW() if any changes
	if patch.UpdatedAt != nil {
		if patch.UpdatedAt.IsZero() {
			updates = append(updates, gfs.Update{
				Path:  "updatedAt",
				Value: gfs.Delete,
			})
		} else {
			updates = append(updates, gfs.Update{
				Path:  "updatedAt",
				Value: patch.UpdatedAt.UTC(),
			})
		}
	} else if len(updates) > 0 {
		updates = append(updates, gfs.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		})
	}

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	_, err := r.col().Doc(id).Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return msgdom.Message{}, msgdom.ErrNotFound
		}
		return msgdom.Message{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *MessageRepositoryFS) Send(ctx context.Context, id string, at time.Time) (msgdom.Message, error) {
	return r.updateStatus(ctx, id, msgdom.StatusDraft, msgdom.StatusSent, at, false, false)
}

func (r *MessageRepositoryFS) Cancel(ctx context.Context, id string, at time.Time) (msgdom.Message, error) {
	return r.updateStatus(ctx, id, msgdom.StatusSent, msgdom.StatusCanceled, at, false, false)
}

func (r *MessageRepositoryFS) MarkDelivered(ctx context.Context, id string, at time.Time) (msgdom.Message, error) {
	return r.updateStatus(ctx, id, msgdom.StatusSent, msgdom.StatusDelivered, at, false, false)
}

func (r *MessageRepositoryFS) MarkRead(ctx context.Context, id string, at time.Time) (msgdom.Message, error) {
	return r.updateStatus(ctx, id, msgdom.StatusDelivered, msgdom.StatusRead, at, true, false)
}

func (r *MessageRepositoryFS) updateStatus(
	ctx context.Context,
	id string,
	expect msgdom.MessageStatus,
	next msgdom.MessageStatus,
	at time.Time,
	setReadAt bool,
	_ bool, // setCanceledAt not used (no canceledAt field)
) (msgdom.Message, error) {
	if r.Client == nil {
		return msgdom.Message{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return msgdom.Message{}, msgdom.ErrNotFound
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}

	docRef := r.col().Doc(id)

	var out msgdom.Message
	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return msgdom.ErrNotFound
			}
			return err
		}
		m, err := decodeMessageDoc(doc)
		if err != nil {
			return err
		}
		if m.Status != expect {
			// align with PG behavior: treat mismatched status as not found
			return msgdom.ErrNotFound
		}

		m.Status = next
		mu := at
		m.UpdatedAt = &mu
		if setReadAt && m.ReadAt == nil {
			rr := at
			m.ReadAt = &rr
		}

		if err := tx.Set(docRef, encodeMessageDoc(m), gfs.MergeAll); err != nil {
			return err
		}
		out = m
		return nil
	})
	if err != nil {
		if errors.Is(err, msgdom.ErrNotFound) {
			return msgdom.Message{}, msgdom.ErrNotFound
		}
		return msgdom.Message{}, err
	}

	out.Images = nil
	return out, nil
}

func (r *MessageRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return msgdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return msgdom.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *MessageRepositoryFS) Save(ctx context.Context, m msgdom.Message) (msgdom.Message, error) {
	if r.Client == nil {
		return msgdom.Message{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(m.ID)
	if id == "" {
		return msgdom.Message{}, errors.New("missing id")
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	m.ID = id

	docRef := r.col().Doc(id)
	data := encodeMessageDoc(m)

	_, err := docRef.Set(ctx, data, gfs.MergeAll)
	if err != nil {
		return msgdom.Message{}, err
	}

	return r.GetByID(ctx, id)
}

// =======================
// Threads (ThreadRepository impl)
// =======================

func buildThreadFilterMatch(t msgdom.MessageThread, f msgdom.ThreadFilter, participants []string) bool {
	// Search by lastMessageText (subject equivalent)
	if q := strings.TrimSpace(f.SearchQuery); q != "" {
		lq := strings.ToLower(q)
		if !strings.Contains(strings.ToLower(t.LastMessageText), lq) {
			return false
		}
	}

	// ParticipantID / ParticipantIDs
	if f.ParticipantID != nil && strings.TrimSpace(*f.ParticipantID) != "" {
		want := strings.TrimSpace(*f.ParticipantID)
		found := false
		for _, p := range participants {
			if p == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(f.ParticipantIDs) > 0 {
		ok := false
		for _, wantID := range f.ParticipantIDs {
			wantID = strings.TrimSpace(wantID)
			if wantID == "" {
				continue
			}
			for _, p := range participants {
				if p == wantID {
					ok = true
					break
				}
			}
			if ok {
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Use LastMessageAt for created/updated/last-message filters
	lm := t.LastMessageAt

	if f.CreatedFrom != nil && lm.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !lm.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && lm.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !lm.Before(f.UpdatedTo.UTC()) {
		return false
	}
	if f.LastMessageFrom != nil && lm.Before(f.LastMessageFrom.UTC()) {
		return false
	}
	if f.LastMessageTo != nil && !lm.Before(f.LastMessageTo.UTC()) {
		return false
	}

	return true
}

func (r *MessageRepositoryFS) CountThreads(ctx context.Context, filter msgdom.ThreadFilter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.threadCol().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		t, err := decodeThreadDoc(doc)
		if err != nil {
			return 0, err
		}
		partsMap, err := r.loadThreadParticipants(ctx, []string{t.ID})
		if err != nil {
			return 0, err
		}
		parts := partsMap[t.ID]
		if buildThreadFilterMatch(t, filter, parts) {
			total++
		}
	}
	return total, nil
}

func (r *MessageRepositoryFS) ListThreads(
	ctx context.Context,
	filter msgdom.ThreadFilter,
	sort msgdom.ThreadSort,
	page msgdom.Page,
) (msgdom.PageResult[msgdom.MessageThread], error) {
	if r.Client == nil {
		return msgdom.PageResult[msgdom.MessageThread]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage := normalizePage(page.Number, page.PerPage, 50)

	q := r.threadCol().Query
	q = applyThreadSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []msgdom.MessageThread
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return msgdom.PageResult[msgdom.MessageThread]{}, err
		}
		t, err := decodeThreadDoc(doc)
		if err != nil {
			return msgdom.PageResult[msgdom.MessageThread]{}, err
		}
		partsMap, err := r.loadThreadParticipants(ctx, []string{t.ID})
		if err != nil {
			return msgdom.PageResult[msgdom.MessageThread]{}, err
		}
		parts := partsMap[t.ID]
		if buildThreadFilterMatch(t, filter, parts) {
			t.ParticipantIDs = append([]string(nil), parts...)
			all = append(all, t)
		}
	}

	total := len(all)
	if total == 0 {
		return msgdom.PageResult[msgdom.MessageThread]{
			Items:      []msgdom.MessageThread{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	totalPages := computeTotalPages(total, perPage)

	return msgdom.PageResult[msgdom.MessageThread]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *MessageRepositoryFS) ListThreadsByCursor(
	ctx context.Context,
	filter msgdom.ThreadFilter,
	_ msgdom.ThreadSort,
	cpage msgdom.CursorPage,
) (msgdom.CursorPageResult[msgdom.MessageThread], error) {
	if r.Client == nil {
		return msgdom.CursorPageResult[msgdom.MessageThread]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := r.threadCol().OrderBy(gfs.DocumentID, gfs.Asc)
	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items []msgdom.MessageThread
		last  string
	)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return msgdom.CursorPageResult[msgdom.MessageThread]{}, err
		}
		t, err := decodeThreadDoc(doc)
		if err != nil {
			return msgdom.CursorPageResult[msgdom.MessageThread]{}, err
		}

		if skipping {
			if t.ID <= after {
				continue
			}
			skipping = false
		}

		partsMap, err := r.loadThreadParticipants(ctx, []string{t.ID})
		if err != nil {
			return msgdom.CursorPageResult[msgdom.MessageThread]{}, err
		}
		parts := partsMap[t.ID]
		if !buildThreadFilterMatch(t, filter, parts) {
			continue
		}

		t.ParticipantIDs = append([]string(nil), parts...)
		items = append(items, t)
		last = t.ID

		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &last
	}

	return msgdom.CursorPageResult[msgdom.MessageThread]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *MessageRepositoryFS) GetThreadByID(ctx context.Context, id string) (msgdom.MessageThread, error) {
	if r.Client == nil {
		return msgdom.MessageThread{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return msgdom.MessageThread{}, msgdom.ErrNotFound
	}

	doc, err := r.threadCol().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return msgdom.MessageThread{}, msgdom.ErrNotFound
		}
		return msgdom.MessageThread{}, err
	}

	t, err := decodeThreadDoc(doc)
	if err != nil {
		return msgdom.MessageThread{}, err
	}
	partsMap, err := r.loadThreadParticipants(ctx, []string{t.ID})
	if err != nil {
		return msgdom.MessageThread{}, err
	}
	t.ParticipantIDs = append([]string(nil), partsMap[t.ID]...)
	return t, nil
}

func (r *MessageRepositoryFS) SaveThread(ctx context.Context, t msgdom.MessageThread) (msgdom.MessageThread, error) {
	if r.Client == nil {
		return msgdom.MessageThread{}, errors.New("firestore client is nil")
	}

	if tx := txFromCtx(ctx); tx != nil {
		return r.saveThreadWithTx(ctx, tx, t)
	}

	var out msgdom.MessageThread
	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		var err error
		out, err = r.saveThreadWithTx(ctx, tx, t)
		return err
	})
	if err != nil {
		return msgdom.MessageThread{}, err
	}
	return out, nil
}

func (r *MessageRepositoryFS) saveThreadWithTx(
	ctx context.Context,
	tx *gfs.Transaction,
	t msgdom.MessageThread,
) (msgdom.MessageThread, error) {
	id := strings.TrimSpace(t.ID)
	if id == "" {
		return msgdom.MessageThread{}, msgdom.ErrInvalid
	}

	docRef := r.threadCol().Doc(id)
	data := encodeThreadDoc(t)

	if err := tx.Set(docRef, data, gfs.MergeAll); err != nil {
		return msgdom.MessageThread{}, err
	}

	// Replace participants
	partsCol := r.threadParticipantsCol(id)
	it := partsCol.Documents(ctx)
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			it.Stop()
			return msgdom.MessageThread{}, err
		}
		if err := tx.Delete(doc.Ref); err != nil {
			it.Stop()
			return msgdom.MessageThread{}, err
		}
	}
	it.Stop()

	for _, pid := range t.ParticipantIDs {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		pref := partsCol.Doc(pid)
		if err := tx.Set(pref, map[string]any{
			"memberId": pid,
		}, gfs.MergeAll); err != nil {
			return msgdom.MessageThread{}, err
		}
	}

	// Rebuild thread object
	saved := t
	saved.ID = id
	return saved, nil
}

func (r *MessageRepositoryFS) DeleteThread(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if tx := txFromCtx(ctx); tx != nil {
		return r.deleteThreadWithTx(ctx, tx, id)
	}

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		return r.deleteThreadWithTx(ctx, tx, id)
	})
}

func (r *MessageRepositoryFS) deleteThreadWithTx(
	ctx context.Context,
	tx *gfs.Transaction,
	id string,
) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return msgdom.ErrInvalid
	}

	docRef := r.threadCol().Doc(id)

	// Ensure exists
	if _, err := tx.Get(docRef); err != nil {
		if status.Code(err) == codes.NotFound {
			return msgdom.ErrNotFound
		}
		return err
	}

	// Delete participants
	partsCol := r.threadParticipantsCol(id)
	it := partsCol.Documents(ctx)
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			it.Stop()
			return err
		}
		if err := tx.Delete(doc.Ref); err != nil {
			it.Stop()
			return err
		}
	}
	it.Stop()

	// Delete thread
	if err := tx.Delete(docRef); err != nil {
		return err
	}
	return nil
}

// =======================
// WHERE/ORDER equivalents & helpers
// =======================

func matchMessageFilter(m msgdom.Message, f msgdom.Filter) bool {
	// PG版互換: 常に削除済みは除外
	if m.DeletedAt != nil {
		return false
	}

	if q := strings.TrimSpace(f.SearchQuery); q != "" {
		lq := strings.ToLower(q)
		if !strings.Contains(strings.ToLower(m.Content), lq) {
			return false
		}
	}
	if f.SenderID != nil && strings.TrimSpace(*f.SenderID) != "" {
		if m.SenderID != strings.TrimSpace(*f.SenderID) {
			return false
		}
	}
	if f.ReceiverID != nil && strings.TrimSpace(*f.ReceiverID) != "" {
		if m.ReceiverID != strings.TrimSpace(*f.ReceiverID) {
			return false
		}
	}
	if len(f.Statuses) > 0 {
		ok := false
		for _, st := range f.Statuses {
			if m.Status == st {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if f.UnreadOnly {
		if m.Status == msgdom.StatusRead {
			return false
		}
	}
	if f.CreatedFrom != nil && m.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !m.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil {
		if m.UpdatedAt == nil || m.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
			return false
		}
	}
	if f.UpdatedTo != nil {
		if m.UpdatedAt == nil || !m.UpdatedAt.Before(f.UpdatedTo.UTC()) {
			return false
		}
	}

	return true
}

func applyMessageSort(q gfs.Query, sort msgdom.Sort) gfs.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	var field string

	switch col {
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "status":
		field = "status"
	default:
		// default: createdAt DESC, id DESC
		return q.OrderBy("createdAt", gfs.Desc).OrderBy(gfs.DocumentID, gfs.Desc)
	}

	dir := gfs.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = gfs.Desc
	}
	return q.OrderBy(field, dir).OrderBy(gfs.DocumentID, gfs.Asc)
}

func applyThreadSort(q gfs.Query, sort msgdom.ThreadSort) gfs.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	field := "lastMessageAt" // default

	switch col {
	case "lastmessageat", "last_message_at":
		field = "lastMessageAt"
	case "createdat", "created_at":
		field = "lastMessageAt"
	case "updatedat", "updated_at":
		field = "lastMessageAt"
	default:
		field = "lastMessageAt"
	}

	dir := gfs.Desc
	if strings.EqualFold(string(sort.Order), "asc") {
		dir = gfs.Asc
	}

	return q.OrderBy(field, dir).OrderBy(gfs.DocumentID, dir)
}

func normalizePage(number, perPage, defaultPerPage int) (int, int) {
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if number <= 0 {
		number = 1
	}
	return number, perPage
}

func computeTotalPages(total, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	if total == 0 {
		return 0
	}
	return (total + perPage - 1) / perPage
}

// =======================
// Upload (not implemented / infra-dependent)
// =======================

func (r *MessageRepositoryFS) UploadMessageImage(ctx context.Context, fileName string, contentType string, _ io.Reader) (string, error) {
	_ = ctx
	_ = fileName
	_ = contentType
	return "", fmt.Errorf("%w: UploadMessageImage not implemented in Firestore adapter", msgdom.ErrInvalid)
}
