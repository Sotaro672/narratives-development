// backend/internal/adapters/out/firestore/avatar_repository_fs.go
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

	avdom "narratives/internal/domain/avatar"
)

// Firestore implementation of avatar.Repository (avatar domain).
type AvatarRepositoryFS struct {
	Client *firestore.Client
}

func NewAvatarRepositoryFS(client *firestore.Client) *AvatarRepositoryFS {
	return &AvatarRepositoryFS{Client: client}
}

func (r *AvatarRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatars")
}

// Ensure interface compatibility at compile time (if defined).
var _ interface {
	List(ctx context.Context, filter avdom.Filter, sort avdom.Sort, page avdom.Page) (avdom.PageResult, error)
	ListByCursor(ctx context.Context, filter avdom.Filter, sort avdom.Sort, cpage avdom.CursorPage) (avdom.CursorPageResult, error)
	GetByID(ctx context.Context, id string) (avdom.Avatar, error)
	GetByWalletAddress(ctx context.Context, wallet string) (avdom.Avatar, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter avdom.Filter) (int, error)
	Create(ctx context.Context, a avdom.Avatar) (avdom.Avatar, error)
	Update(ctx context.Context, id string, patch avdom.AvatarPatch) (avdom.Avatar, error)
	Delete(ctx context.Context, id string) error
	Save(ctx context.Context, a avdom.Avatar, opts *avdom.SaveOptions) (avdom.Avatar, error)
	Search(ctx context.Context, query string) ([]avdom.Avatar, error)
	ListTopByFollowers(ctx context.Context, limit int) ([]avdom.Avatar, error)
	Reset(ctx context.Context) error
} = (*AvatarRepositoryFS)(nil)

// ==============================
// List (filter + sort + pagination)
// ==============================

func (r *AvatarRepositoryFS) List(
	ctx context.Context,
	filter avdom.Filter,
	sort avdom.Sort,
	page avdom.Page,
) (avdom.PageResult, error) {

	q := r.col().Query
	q = applyAvatarFilterToQuery(q, filter)

	field, dir := mapAvatarSort(sort)
	q = q.OrderBy(field, dir).OrderBy("id", firestore.Asc)

	// offset/limit based pagination (simple)
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

	items := make([]avdom.Avatar, 0, perPage)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return avdom.PageResult{}, err
		}
		a, err := r.docToDomain(doc)
		if err != nil {
			return avdom.PageResult{}, err
		}
		items = append(items, a)
	}

	// Firestore には簡単な COUNT がないので、ここでは「取得件数のみ」を TotalCount に設定
	totalCount := len(items)

	return avdom.PageResult{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: number, // 厳密ではないがインターフェース互換用の値
		Page:       number,
		PerPage:    perPage,
	}, nil
}

// ==============================
// ListByCursor (simple id-based cursor)
// ==============================

func (r *AvatarRepositoryFS) ListByCursor(
	ctx context.Context,
	filter avdom.Filter,
	sort avdom.Sort,
	cpage avdom.CursorPage,
) (avdom.CursorPageResult, error) {

	q := r.col().Query
	q = applyAvatarFilterToQuery(q, filter)

	field, dir := mapAvatarSort(sort)
	q = q.OrderBy(field, dir).OrderBy("id", firestore.Asc)

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Use id as cursor if provided
	if after := strings.TrimSpace(cpage.After); after != "" {
		// カーソルを id ベースにする設計の場合、ここでは id で StartAfter。
		// 厳密にやるなら、直前ドキュメントのスナップショットを使う。
		q = q.StartAfter(after)
	}

	q = q.Limit(limit + 1)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var (
		items  []avdom.Avatar
		lastID string
	)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return avdom.CursorPageResult{}, err
		}
		a, err := r.docToDomain(doc)
		if err != nil {
			return avdom.CursorPageResult{}, err
		}
		items = append(items, a)
		lastID = a.ID
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

// ==============================
// GetByID
// ==============================

func (r *AvatarRepositoryFS) GetByID(ctx context.Context, id string) (avdom.Avatar, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return avdom.Avatar{}, errors.New("avatar: empty id")
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return avdom.Avatar{}, errors.New("avatar: not found")
	}
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(snap)
}

// ==============================
// GetByWalletAddress
// ==============================

func (r *AvatarRepositoryFS) GetByWalletAddress(ctx context.Context, wallet string) (avdom.Avatar, error) {
	wallet = strings.TrimSpace(wallet)
	if wallet == "" {
		return avdom.Avatar{}, errors.New("avatar: wallet address is empty")
	}

	q := r.col().Where("walletAddress", "==", wallet).Limit(1)
	iter := q.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return avdom.Avatar{}, errors.New("avatar: not found")
	}
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(doc)
}

// ==============================
// Exists
// ==============================

func (r *AvatarRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	_, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ==============================
// Count
// ==============================

func (r *AvatarRepositoryFS) Count(ctx context.Context, filter avdom.Filter) (int, error) {
	q := r.col().Query
	q = applyAvatarFilterToQuery(q, filter)

	iter := q.Documents(ctx)
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
// Create
// ==============================

func (r *AvatarRepositoryFS) Create(ctx context.Context, a avdom.Avatar) (avdom.Avatar, error) {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}

	var ref *firestore.DocumentRef
	if strings.TrimSpace(a.ID) == "" {
		ref = r.col().NewDoc()
		a.ID = ref.ID
	} else {
		ref = r.col().Doc(a.ID)
	}

	data := r.domainToDocData(a)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avdom.Avatar{}, errors.New("avatar: conflict")
		}
		return avdom.Avatar{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(snap)
}

// ==============================
// Update (patch)
// ==============================

func (r *AvatarRepositoryFS) Update(ctx context.Context, id string, patch avdom.AvatarPatch) (avdom.Avatar, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return avdom.Avatar{}, errors.New("avatar: empty id")
	}
	ref := r.col().Doc(id)

	// Ensure exists
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return avdom.Avatar{}, errors.New("avatar: not found")
	} else if err != nil {
		return avdom.Avatar{}, err
	}

	var updates []firestore.Update

	if patch.AvatarName != nil {
		updates = append(updates, firestore.Update{
			Path:  "avatarName",
			Value: strings.TrimSpace(*patch.AvatarName),
		})
	}
	if patch.AvatarIconID != nil {
		updates = append(updates, firestore.Update{
			Path:  "avatarIconId",
			Value: optionalString(*patch.AvatarIconID),
		})
	}
	if patch.WalletAddress != nil {
		updates = append(updates, firestore.Update{
			Path:  "walletAddress",
			Value: optionalString(*patch.WalletAddress),
		})
	}
	if patch.Bio != nil {
		updates = append(updates, firestore.Update{
			Path:  "bio",
			Value: optionalString(*patch.Bio),
		})
	}
	if patch.Website != nil {
		updates = append(updates, firestore.Update{
			Path:  "website",
			Value: optionalString(*patch.Website),
		})
	}
	if patch.UserID != nil {
		updates = append(updates, firestore.Update{
			Path:  "userId",
			Value: strings.TrimSpace(*patch.UserID),
		})
	}
	if patch.DeletedAt != nil {
		if patch.DeletedAt.IsZero() {
			updates = append(updates, firestore.Update{
				Path:  "deletedAt",
				Value: nil,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "deletedAt",
				Value: patch.DeletedAt.UTC(),
			})
		}
	}

	// Always bump updatedAt if something to update
	if len(updates) == 0 {
		// no-op: return current
		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return avdom.Avatar{}, errors.New("avatar: not found")
			}
			return avdom.Avatar{}, err
		}
		return r.docToDomain(snap)
	}

	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return avdom.Avatar{}, errors.New("avatar: not found")
		}
		return avdom.Avatar{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(snap)
}

// ==============================
// Delete
// ==============================

func (r *AvatarRepositoryFS) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("avatar: empty id")
	}
	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return errors.New("avatar: not found")
	} else if err != nil {
		return err
	}
	_, err := ref.Delete(ctx)
	return err
}

// ==============================
// Save (upsert)
// ==============================

func (r *AvatarRepositoryFS) Save(ctx context.Context, a avdom.Avatar, _ *avdom.SaveOptions) (avdom.Avatar, error) {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}

	var ref *firestore.DocumentRef
	if strings.TrimSpace(a.ID) == "" {
		ref = r.col().NewDoc()
		a.ID = ref.ID
	} else {
		ref = r.col().Doc(a.ID)
	}

	data := r.domainToDocData(a)

	if _, err := ref.Set(ctx, data, firestore.MergeAll); err != nil {
		return avdom.Avatar{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(snap)
}

// ==============================
// Search (simple client-side filter)
// ==============================

func (r *AvatarRepositoryFS) Search(ctx context.Context, query string) ([]avdom.Avatar, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []avdom.Avatar{}, nil
	}

	// Firestore で部分一致は難しいため、全件/限定件取得→アプリ側フィルタ
	fsQuery := r.col().Limit(200)

	iter := fsQuery.Documents(ctx)
	defer iter.Stop()

	lowerQ := strings.ToLower(q)
	var list []avdom.Avatar

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		a, err := r.docToDomain(doc)
		if err != nil {
			return nil, err
		}
		name := strings.ToLower(strings.TrimSpace(a.AvatarName))
		wallet := ""
		if a.WalletAddress != nil {
			wallet = strings.ToLower(strings.TrimSpace(*a.WalletAddress))
		}
		if strings.Contains(name, lowerQ) || strings.Contains(wallet, lowerQ) {
			list = append(list, a)
		}
	}
	return list, nil
}

// ==============================
// ListTopByFollowers
// ==============================

func (r *AvatarRepositoryFS) ListTopByFollowers(ctx context.Context, limit int) ([]avdom.Avatar, error) {
	// AvatarState と連携しない限り follower_count は持っていない想定なので、
	// ここでは createdAt DESC のシンプルな代替実装とする。
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := r.col().OrderBy("createdAt", firestore.Desc).Limit(limit)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var list []avdom.Avatar
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		a, err := r.docToDomain(doc)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

// ==============================
// Reset (development/testing)
// ==============================

func (r *AvatarRepositoryFS) Reset(ctx context.Context) error {
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
	log.Printf("[firestore] Reset avatars: deleted %d docs\n", count)
	return nil
}

// ==============================
// Mapping helpers
// ==============================

func (r *AvatarRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (avdom.Avatar, error) {
	var raw struct {
		UserID        string     `firestore:"userId"`
		AvatarName    string     `firestore:"avatarName"`
		AvatarIconID  *string    `firestore:"avatarIconId"`
		WalletAddress *string    `firestore:"walletAddress"`
		Bio           *string    `firestore:"bio"`
		Website       *string    `firestore:"website"`
		CreatedAt     time.Time  `firestore:"createdAt"`
		UpdatedAt     time.Time  `firestore:"updatedAt"`
		DeletedAt     *time.Time `firestore:"deletedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return avdom.Avatar{}, err
	}

	a := avdom.Avatar{
		ID:         doc.Ref.ID,
		UserID:     strings.TrimSpace(raw.UserID),
		AvatarName: strings.TrimSpace(raw.AvatarName),
		CreatedAt:  raw.CreatedAt.UTC(),
		UpdatedAt:  raw.UpdatedAt.UTC(),
	}

	if raw.AvatarIconID != nil && strings.TrimSpace(*raw.AvatarIconID) != "" {
		v := strings.TrimSpace(*raw.AvatarIconID)
		a.AvatarIconID = &v
	}
	if raw.WalletAddress != nil && strings.TrimSpace(*raw.WalletAddress) != "" {
		v := strings.TrimSpace(*raw.WalletAddress)
		a.WalletAddress = &v
	}
	if raw.Bio != nil && strings.TrimSpace(*raw.Bio) != "" {
		v := strings.TrimSpace(*raw.Bio)
		a.Bio = &v
	}
	if raw.Website != nil && strings.TrimSpace(*raw.Website) != "" {
		v := strings.TrimSpace(*raw.Website)
		a.Website = &v
	}
	if raw.DeletedAt != nil && !raw.DeletedAt.IsZero() {
		t := raw.DeletedAt.UTC()
		a.DeletedAt = &t
	}

	return a, nil
}

func (r *AvatarRepositoryFS) domainToDocData(a avdom.Avatar) map[string]any {
	data := map[string]any{
		"userId":     strings.TrimSpace(a.UserID),
		"avatarName": strings.TrimSpace(a.AvatarName),
		"createdAt":  a.CreatedAt.UTC(),
		"updatedAt":  a.UpdatedAt.UTC(),
	}

	if a.AvatarIconID != nil && strings.TrimSpace(*a.AvatarIconID) != "" {
		data["avatarIconId"] = strings.TrimSpace(*a.AvatarIconID)
	}
	if a.WalletAddress != nil && strings.TrimSpace(*a.WalletAddress) != "" {
		data["walletAddress"] = strings.TrimSpace(*a.WalletAddress)
	}
	if a.Bio != nil && strings.TrimSpace(*a.Bio) != "" {
		data["bio"] = strings.TrimSpace(*a.Bio)
	}
	if a.Website != nil && strings.TrimSpace(*a.Website) != "" {
		data["website"] = strings.TrimSpace(*a.Website)
	}
	if a.DeletedAt != nil && !a.DeletedAt.IsZero() {
		data["deletedAt"] = a.DeletedAt.UTC()
	}

	return data
}

// ==============================
// Query helpers
// ==============================

func applyAvatarFilterToQuery(q firestore.Query, f avdom.Filter) firestore.Query {
	// NOTE: Firestore の制約により、ここでは代表的なもののみクエリに反映。
	// SearchQuery などの部分一致はアプリ側でフィルタする想定。

	if f.UserID != nil && strings.TrimSpace(*f.UserID) != "" {
		q = q.Where("userId", "==", strings.TrimSpace(*f.UserID))
	}

	// JoinedFrom/JoinedTo → createdAt range
	if f.JoinedFrom != nil {
		q = q.Where("createdAt", ">=", f.JoinedFrom.UTC())
	}
	if f.JoinedTo != nil {
		q = q.Where("createdAt", "<", f.JoinedTo.UTC())
	}

	return q
}

func mapAvatarSort(s avdom.Sort) (field string, dir firestore.Direction) {
	col := strings.ToLower(string(s.Column))
	switch col {
	case "avatarname":
		field = "avatarName"
	case "createdat":
		field = "createdAt"
	case "updatedat":
		field = "updatedAt"
	default:
		field = "createdAt"
	}

	if strings.EqualFold(string(s.Order), "asc") {
		dir = firestore.Asc
	} else {
		dir = firestore.Desc
	}
	return
}

// ==============================
// small utils
// ==============================

func optionalString(v string) any {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil
	}
	return s
}
