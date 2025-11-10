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

// AvatarStateRepositoryFS is the Firestore implementation of avatarstate.Repository.
type AvatarStateRepositoryFS struct {
	Client *firestore.Client
}

func NewAvatarStateRepositoryFS(client *firestore.Client) *AvatarStateRepositoryFS {
	return &AvatarStateRepositoryFS{Client: client}
}

// collection name
func (r *AvatarStateRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatar_states")
}

// ========== Queries ==========

// List:
// Firestore では複雑な WHERE / LIKE / 複数範囲指定が制限されるため、
// ここでは代表的な条件のみクエリに反映し、それ以外は簡易運用または将来対応とします。
func (r *AvatarStateRepositoryFS) List(
	ctx context.Context,
	filter avatarstate.Filter,
	sort avatarstate.Sort,
	page avatarstate.Page,
) (avatarstate.PageResult[avatarstate.AvatarState], error) {

	q := r.col().Query

	// --- Filter: AvatarID (完全一致) ---
	if filter.AvatarID != nil && strings.TrimSpace(*filter.AvatarID) != "" {
		q = q.Where("avatarId", "==", strings.TrimSpace(*filter.AvatarID))
	}

	// --- Filter: AvatarIDs (IN) ---
	if len(filter.AvatarIDs) > 0 {
		ids := make([]string, 0, len(filter.AvatarIDs))
		for _, v := range filter.AvatarIDs {
			if s := strings.TrimSpace(v); s != "" {
				ids = append(ids, s)
			}
		}
		// Firestore IN は最大 10 要素制限あり。超える場合は呼び出し側で分割すべき。
		if len(ids) > 0 && len(ids) <= 10 {
			q = q.Where("avatarId", "in", ids)
		}
	}

	// NOTE:
	// SearchQuery や follower/post の範囲などは、
	// 要件次第で「全件取得後にアプリ側で絞り込み」を検討。
	// ここではシンプルに未対応（将来拡張）とする。

	// --- Sort ---
	orderField := "lastActiveAt"
	switch strings.ToLower(string(sort.Column)) {
	case "id":
		orderField = "id"
	case "avatarid", "avatar_id":
		orderField = "avatarId"
	case "followercount", "follower_count":
		orderField = "followerCount"
	case "followingcount", "following_count":
		orderField = "followingCount"
	case "postcount", "post_count":
		orderField = "postCount"
	case "updatedat", "updated_at":
		orderField = "updatedAt"
	case "lastactiveat", "last_active_at":
		orderField = "lastActiveAt"
	default:
		// デフォルト: lastActiveAt DESC
		orderField = "lastActiveAt"
	}

	dir := strings.ToUpper(string(sort.Order))
	desc := dir != "" && dir != "ASC"

	if desc {
		q = q.OrderBy(orderField, firestore.Desc)
	} else {
		q = q.OrderBy(orderField, firestore.Asc)
	}

	// --- Paging ---
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

	// Firestoreは offset が重いので、ここでは簡易に offset を使用
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

	// TotalCount は正確に数えると全スキャンになるため、
	// 運用/要件に応じて別途集計 or 近似にすることも検討。
	// ここでは簡易に「取得件数のみ / TotalPages=number」で返す。
	return avatarstate.PageResult[avatarstate.AvatarState]{
		Items:      items,
		TotalCount: len(items),
		TotalPages: number, // 厳密ではないがインターフェース維持用
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *AvatarStateRepositoryFS) ListByCursor(
	ctx context.Context,
	filter avatarstate.Filter,
	sort avatarstate.Sort,
	cpage avatarstate.CursorPage,
) (avatarstate.CursorPageResult[avatarstate.AvatarState], error) {

	q := r.col().Query

	// 簡易フィルタ: AvatarID
	if filter.AvatarID != nil && strings.TrimSpace(*filter.AvatarID) != "" {
		q = q.Where("avatarId", "==", strings.TrimSpace(*filter.AvatarID))
	}

	// ソートは lastActiveAt DESC を基本とする
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

	// Cursor: After を docID として扱う
	if after := strings.TrimSpace(cpage.After); after != "" {
		snap, err := r.col().Doc(after).Get(ctx)
		if err == nil {
			q = q.StartAfter(snap.Data()[orderField])
		}
	}

	q = q.Limit(limit + 1)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var items []avatarstate.AvatarState
	var lastID string

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
	iter := r.col().
		Where("avatarId", "==", avatarID).
		Limit(1).
		Documents(ctx)
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
	// NOTE: 正確な件数取得は全スキャンになるため、必要に応じて別集計を推奨。
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

// ========== Mutations ==========

func (r *AvatarStateRepositoryFS) Create(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	now := time.Now().UTC()
	if s.UpdatedAt == nil {
		s.UpdatedAt = &now
	}
	if s.LastActiveAt.IsZero() {
		s.LastActiveAt = now
	}

	var ref *firestore.DocumentRef
	if s.ID != "" {
		ref = r.col().Doc(s.ID)
	} else {
		ref = r.col().NewDoc()
		s.ID = ref.ID
	}

	_, err := ref.Create(ctx, r.domainToDocData(s))
	if err != nil {
		// 既存IDとの衝突など
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
	// avatarId で 1件取得して Update
	iter := r.col().
		Where("avatarId", "==", avatarID).
		Limit(1).
		Documents(ctx)
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
		updates = append(updates, firestore.Update{Path: "followerCount", Value: *patch.FollowerCount})
	}
	if patch.FollowingCount != nil {
		updates = append(updates, firestore.Update{Path: "followingCount", Value: *patch.FollowingCount})
	}
	if patch.PostCount != nil {
		updates = append(updates, firestore.Update{Path: "postCount", Value: *patch.PostCount})
	}
	if patch.LastActiveAt != nil {
		updates = append(updates, firestore.Update{Path: "lastActiveAt", Value: patch.LastActiveAt.UTC()})
	}

	// updatedAt: 明示指定があればそれを優先、なければ現在時刻
	if patch.UpdatedAt != nil {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: patch.UpdatedAt.UTC()})
	} else {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: time.Now().UTC()})
	}

	if len(updates) == 0 {
		// 変更指定がなければ現状値を返す
		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return avatarstate.AvatarState{}, avatarstate.ErrNotFound
			}
			return avatarstate.AvatarState{}, err
		}
		return r.docToDomain(snap)
	}

	_, err := ref.Update(ctx, updates)
	if err != nil {
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
	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return avatarstate.ErrNotFound
	}
	_, err = ref.Delete(ctx)
	return err
}

func (r *AvatarStateRepositoryFS) DeleteByAvatarID(ctx context.Context, avatarID string) error {
	iter := r.col().
		Where("avatarId", "==", avatarID).
		Documents(ctx)
	defer iter.Stop()

	batch := r.Client.Batch()
	var count int

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		batch.Delete(doc.Ref)
		count++
	}

	if count == 0 {
		return avatarstate.ErrNotFound
	}
	_, err := batch.Commit(ctx)
	return err
}

func (r *AvatarStateRepositoryFS) Save(ctx context.Context, s avatarstate.AvatarState, _ *avatarstate.SaveOptions) (avatarstate.AvatarState, error) {
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
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

// Upsert: domainインターフェース互換用ヘルパ
func (r *AvatarStateRepositoryFS) Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error) {
	return r.Save(ctx, s, nil)
}

// ========== Helpers ==========

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

	// avatarstate.New でドメインバリデーションを通す前提
	st, err := avatarstate.New(
		doc.Ref.ID,
		strings.TrimSpace(raw.AvatarID),
		raw.FollowerCount,
		raw.FollowingCount,
		raw.PostCount,
		raw.LastActiveAt.UTC(),
		raw.UpdatedAt,
	)
	if err != nil {
		return avatarstate.AvatarState{}, err
	}
	return st, nil
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

// Reset (development/testing): 全削除
func (r *AvatarStateRepositoryFS) Reset(ctx context.Context) error {
	iter := r.col().Documents(ctx)
	batch := r.Client.Batch()
	count := 0

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		batch.Delete(doc.Ref)
		count++
		if count%400 == 0 {
			if _, err := batch.Commit(ctx); err != nil {
				return err
			}
			batch = r.Client.Batch()
		}
	}

	if count > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
	}
	log.Printf("[firestore] Reset avatar_states: deleted %d docs\n", count)
	return nil
}
