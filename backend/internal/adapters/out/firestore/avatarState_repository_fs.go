// backend/internal/adapters/out/firestore/avatarState_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	avatarstate "narratives/internal/domain/avatarState"
)

// Firestore implementation of avatarState.Repository
type AvatarStateRepositoryFS struct {
	Client *firestore.Client
}

func NewAvatarStateRepositoryFS(client *firestore.Client) *AvatarStateRepositoryFS {
	return &AvatarStateRepositoryFS{Client: client}
}

func (r *AvatarStateRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatar_states")
}

// ==============================
// Upsert (required by usecase.AvatarStateRepo)
// ==============================

// Upsert ensures avatar_state exists for the given avatarId.
// - If not found: create with zero counts
// - If exists: touch lastActiveAt / updatedAt
func (r *AvatarStateRepositoryFS) Upsert(ctx context.Context, avatarID string) error {
	if r == nil || r.Client == nil {
		return errors.New("avatarState_repository_fs: client is nil")
	}
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return errors.New("avatarState_repository_fs: avatarID is empty")
	}

	// Exists -> touch
	if _, err := r.GetByAvatarID(ctx, avatarID); err == nil {
		now := time.Now().UTC()
		patch := avatarstate.AvatarStatePatch{
			LastActiveAt: &now,
			UpdatedAt:    &now,
		}
		_, uerr := r.UpdateByAvatarID(ctx, avatarID, patch)
		return uerr
	} else {
		// NotFound -> create
		if errors.Is(err, avatarstate.ErrNotFound) {
			now := time.Now().UTC()
			updatedAt := now

			zero := int64(0)
			st, nerr := avatarstate.New(
				"",       // id (auto)
				avatarID, // avatarId
				&zero,    // followerCount
				&zero,    // followingCount
				&zero,    // postCount
				now,      // lastActiveAt
				&updatedAt,
			)
			if nerr != nil {
				return nerr
			}
			_, serr := r.Save(ctx, st, nil)
			return serr
		}
		return err
	}
}

// ==============================
// List (Filter + Sort + Paging)
// ==============================

func (r *AvatarStateRepositoryFS) List(
	ctx context.Context,
	filter avatarstate.Filter,
	sort avatarstate.Sort,
	page avatarstate.Page,
) (avatarstate.PageResult[avatarstate.AvatarState], error) {

	q := r.col().Query

	if filter.AvatarID != nil && strings.TrimSpace(*filter.AvatarID) != "" {
		q = q.Where("avatarId", "==", strings.TrimSpace(*filter.AvatarID))
	}
	if len(filter.AvatarIDs) > 0 {
		ids := make([]string, 0, len(filter.AvatarIDs))
		for _, v := range filter.AvatarIDs {
			if s := strings.TrimSpace(v); s != "" {
				ids = append(ids, s)
			}
		}
		if len(ids) > 0 && len(ids) <= 10 {
			q = q.Where("avatarId", "in", ids)
		}
	}

	// Sort
	orderField := "lastActiveAt"
	switch strings.ToLower(string(sort.Column)) {
	case "avatarid":
		orderField = "avatarId"
	case "followercount":
		orderField = "followerCount"
	case "followingcount":
		orderField = "followingCount"
	case "postcount":
		orderField = "postCount"
	case "updatedat":
		orderField = "updatedAt"
	case "lastactiveat":
		orderField = "lastActiveAt"
	default:
		orderField = "lastActiveAt"
	}
	dir := strings.ToUpper(string(sort.Order))
	desc := dir != "" && dir != "ASC"
	if desc {
		q = q.OrderBy(orderField, firestore.Desc)
	} else {
		q = q.OrderBy(orderField, firestore.Asc)
	}

	// Paging
	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}
	number := page.Number
	if number <= 0 {
		number = 1
	}
	offset := (number - 1) * perPage
	if offset > 0 {
		q = q.Offset(offset)
	}
	q = q.Limit(perPage)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var items []avatarstate.AvatarState
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return avatarstate.PageResult[avatarstate.AvatarState]{}, err
		}
		st, err := r.docToDomain(doc)
		if err != nil {
			return avatarstate.PageResult[avatarstate.AvatarState]{}, err
		}
		items = append(items, st)
	}

	return avatarstate.PageResult[avatarstate.AvatarState]{
		Items:      items,
		TotalCount: len(items),
		TotalPages: number,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

// ==============================
// ListByCursor
// ==============================

func (r *AvatarStateRepositoryFS) ListByCursor(
	ctx context.Context,
	filter avatarstate.Filter,
	sort avatarstate.Sort,
	cpage avatarstate.CursorPage,
) (avatarstate.CursorPageResult[avatarstate.AvatarState], error) {

	q := r.col().Query

	if filter.AvatarID != nil && strings.TrimSpace(*filter.AvatarID) != "" {
		q = q.Where("avatarId", "==", strings.TrimSpace(*filter.AvatarID))
	}

	orderField := "lastActiveAt"
	dir := strings.ToUpper(string(sort.Order))
	desc := dir != "" && dir != "ASC"
	if desc {
		q = q.OrderBy(orderField, firestore.Desc)
	} else {
		q = q.OrderBy(orderField, firestore.Asc)
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if after := strings.TrimSpace(cpage.After); after != "" {
		snap, err := r.col().Doc(after).Get(ctx)
		if err == nil {
			q = q.StartAfter(snap.Data()[orderField])
		}
	}
	q = q.Limit(limit + 1)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var (
		items  []avatarstate.AvatarState
		lastID string
	)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return avatarstate.CursorPageResult[avatarstate.AvatarState]{}, err
		}
		st, err := r.docToDomain(doc)
		if err != nil {
			return avatarstate.CursorPageResult[avatarstate.AvatarState]{}, err
		}
		items = append(items, st)
		lastID = st.ID
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

// ==============================
// GetByID / GetByAvatarID
// ==============================

func (r *AvatarStateRepositoryFS) GetByID(ctx context.Context, id string) (avatarstate.AvatarState, error) {
	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}
	return r.docToDomain(doc)
}

func (r *AvatarStateRepositoryFS) GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	iter := r.col().Where("avatarId", "==", avatarID).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return avatarstate.AvatarState{}, avatarstate.ErrNotFound
	}
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return r.docToDomain(doc)
}

// ==============================
// Exists / Count
// ==============================

func (r *AvatarStateRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *AvatarStateRepositoryFS) Count(ctx context.Context, _ avatarstate.Filter) (int, error) {
	iter := r.col().Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		_, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

// ==============================
// Create / Update / Delete / Save
// ==============================

func (r *AvatarStateRepositoryFS) Create(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	now := time.Now().UTC()
	if s.UpdatedAt == nil {
		s.UpdatedAt = &now
	}
	if s.LastActiveAt.IsZero() {
		s.LastActiveAt = now
	}

	ref := r.col().Doc(s.ID)
	if s.ID == "" {
		ref = r.col().NewDoc()
		s.ID = ref.ID
	}

	_, err := ref.Create(ctx, r.domainToDocData(s))
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avatarstate.AvatarState{}, avatarstate.ErrConflict
		}
		return avatarstate.AvatarState{}, err
	}
	return s, nil
}

func (r *AvatarStateRepositoryFS) Update(ctx context.Context, id string, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	return r.updateBy(ctx, r.col().Doc(id), patch)
}

func (r *AvatarStateRepositoryFS) UpdateByAvatarID(ctx context.Context, avatarID string, patch avatarstate.AvatarStatePatch) (avatarstate.AvatarState, error) {
	iter := r.col().Where("avatarId", "==", avatarID).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return avatarstate.AvatarState{}, avatarstate.ErrNotFound
	}
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return r.updateBy(ctx, doc.Ref, patch)
}

func (r *AvatarStateRepositoryFS) updateBy(
	ctx context.Context,
	ref *firestore.DocumentRef,
	patch avatarstate.AvatarStatePatch,
) (avatarstate.AvatarState, error) {

	var updates []firestore.Update

	if patch.FollowerCount != nil {
		updates = append(updates, firestore.Update{
			Path:  "followerCount",
			Value: *patch.FollowerCount,
		})
	}
	if patch.FollowingCount != nil {
		updates = append(updates, firestore.Update{
			Path:  "followingCount",
			Value: *patch.FollowingCount,
		})
	}
	if patch.PostCount != nil {
		updates = append(updates, firestore.Update{
			Path:  "postCount",
			Value: *patch.PostCount,
		})
	}
	if patch.LastActiveAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "lastActiveAt",
			Value: patch.LastActiveAt.UTC(),
		})
	}

	now := time.Now().UTC()
	if patch.UpdatedAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: patch.UpdatedAt.UTC(),
		})
	} else {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: now,
		})
	}

	if len(updates) == 0 {
		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return avatarstate.AvatarState{}, avatarstate.ErrNotFound
			}
			return avatarstate.AvatarState{}, err
		}
		return r.docToDomain(snap)
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return avatarstate.AvatarState{}, avatarstate.ErrNotFound
		}
		return avatarstate.AvatarState{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return r.docToDomain(snap)
}

func (r *AvatarStateRepositoryFS) Delete(ctx context.Context, id string) error {
	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return avatarstate.ErrNotFound
	}
	_, err := ref.Delete(ctx)
	return err
}

// DeleteByAvatarID: Transaction-based bulk delete (no WriteBatch)
func (r *AvatarStateRepositoryFS) DeleteByAvatarID(ctx context.Context, avatarID string) error {
	iter := r.col().Where("avatarId", "==", avatarID).Documents(ctx)
	defer iter.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		return avatarstate.ErrNotFound
	}

	const chunkSize = 400
	for start := 0; start < len(refs); start += chunkSize {
		end := start + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[start:end]

		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range chunk {
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *AvatarStateRepositoryFS) Save(ctx context.Context, s avatarstate.AvatarState, _ *avatarstate.SaveOptions) (avatarstate.AvatarState, error) {
	now := time.Now().UTC()
	if s.UpdatedAt == nil {
		s.UpdatedAt = &now
	}
	if s.LastActiveAt.IsZero() {
		s.LastActiveAt = now
	}

	ref := r.col().Doc(s.ID)
	if s.ID == "" {
		ref = r.col().NewDoc()
		s.ID = ref.ID
	}

	_, err := ref.Set(ctx, r.domainToDocData(s))
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return s, nil
}

// ==============================
// Helpers
// ==============================

func (r *AvatarStateRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (avatarstate.AvatarState, error) {
	var raw struct {
		AvatarID       string     `firestore:"avatarId"`
		FollowerCount  *int64     `firestore:"followerCount"`
		FollowingCount *int64     `firestore:"followingCount"`
		PostCount      *int64     `firestore:"postCount"`
		LastActiveAt   time.Time  `firestore:"lastActiveAt"`
		UpdatedAt      *time.Time `firestore:"updatedAt"`
	}
	if err := doc.DataTo(&raw); err != nil {
		return avatarstate.AvatarState{}, err
	}
	return avatarstate.New(
		doc.Ref.ID,
		strings.TrimSpace(raw.AvatarID),
		raw.FollowerCount,
		raw.FollowingCount,
		raw.PostCount,
		raw.LastActiveAt.UTC(),
		raw.UpdatedAt,
	)
}

func (r *AvatarStateRepositoryFS) domainToDocData(s avatarstate.AvatarState) map[string]any {
	data := map[string]any{
		"avatarId":     s.AvatarID,
		"lastActiveAt": s.LastActiveAt,
	}
	if s.FollowerCount != nil {
		data["followerCount"] = *s.FollowerCount
	}
	if s.FollowingCount != nil {
		data["followingCount"] = *s.FollowingCount
	}
	if s.PostCount != nil {
		data["postCount"] = *s.PostCount
	}
	if s.UpdatedAt != nil {
		data["updatedAt"] = s.UpdatedAt.UTC()
	}
	return data
}

// ==============================
// Reset (for testing)
// Transaction-based bulk delete (no WriteBatch)
// ==============================

func (r *AvatarStateRepositoryFS) Reset(ctx context.Context) error {
	iter := r.col().Documents(ctx)
	defer iter.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		log.Printf("[firestore] Reset avatar_states: no docs to delete\n")
		return nil
	}

	const chunkSize = 400
	deletedCount := 0

	for start := 0; start < len(refs); start += chunkSize {
		end := start + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[start:end]

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range chunk {
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		deletedCount += len(chunk)
	}

	log.Printf("[firestore] Reset avatar_states (transactional): deleted %d docs\n", deletedCount)
	return nil
}
