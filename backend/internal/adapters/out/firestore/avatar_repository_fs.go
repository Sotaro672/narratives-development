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
	common "narratives/internal/domain/common"
)

// Firestore implementation of avatar.Repository.
type AvatarRepositoryFS struct {
	Client *firestore.Client
}

func NewAvatarRepositoryFS(client *firestore.Client) *AvatarRepositoryFS {
	return &AvatarRepositoryFS{Client: client}
}

func (r *AvatarRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("avatars")
}

// Compile-time check: ensure AvatarRepositoryFS satisfies avatar.Repository.
var _ avdom.Repository = (*AvatarRepositoryFS)(nil)

var (
	errNotFound           = errors.New("avatar: not found")
	errConflict           = errors.New("avatar: conflict")
	errBadClient          = errors.New("firestore client is nil")
	errInvalidWalletAddr  = errors.New("avatar: invalid walletAddress")
	errWalletAlreadyBound = errors.New("avatar: walletAddress already set")
)

// ==============================
// List (filter + sort + pagination)
// ==============================

func (r *AvatarRepositoryFS) List(
	ctx context.Context,
	filter avdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[avdom.Avatar], error) {
	q := r.col().Query
	q = applyAvatarFilterToQuery(q, filter)

	field, dir := mapAvatarSort(sort)
	q = q.OrderBy(field, dir).OrderBy("id", firestore.Asc)

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
			return common.PageResult[avdom.Avatar]{}, err
		}
		a, err := r.docToDomain(doc)
		if err != nil {
			return common.PageResult[avdom.Avatar]{}, err
		}
		// SearchQuery / Deleted / CreatedFrom/To / UpdatedFrom/To は
		// Firestore クエリで表現しきれないのでアプリ側で best-effort 絞り込み
		if !matchFilterPostLoad(a, filter) {
			continue
		}
		items = append(items, a)
	}

	// NOTE: Firestore で厳密な TotalCount を取るには別クエリ/集計が必要。
	totalCount := len(items)

	return common.PageResult[avdom.Avatar]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: number, // best-effort
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
	sort common.Sort,
	cpage common.CursorPage,
) (common.CursorPageResult[avdom.Avatar], error) {
	q := r.col().Query
	q = applyAvatarFilterToQuery(q, filter)

	field, dir := mapAvatarSort(sort)
	q = q.OrderBy(field, dir).OrderBy("id", firestore.Asc)

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	if after := strings.TrimSpace(cpage.After); after != "" {
		// 単純に id をカーソルとする実装（厳密にやる場合は Snapshot ベースにする）
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
			return common.CursorPageResult[avdom.Avatar]{}, err
		}
		a, err := r.docToDomain(doc)
		if err != nil {
			return common.CursorPageResult[avdom.Avatar]{}, err
		}
		if !matchFilterPostLoad(a, filter) {
			continue
		}
		items = append(items, a)
		lastID = a.ID
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return common.CursorPageResult[avdom.Avatar]{
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
		return avdom.Avatar{}, errNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return avdom.Avatar{}, errNotFound
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
		return avdom.Avatar{}, errNotFound
	}

	q := r.col().Where("walletAddress", "==", wallet).Limit(1)
	iter := q.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return avdom.Avatar{}, errNotFound
	}
	if err != nil {
		return avdom.Avatar{}, err
	}
	return r.docToDomain(doc)
}

// ==============================
// ✅ GetByFirebaseUID (compat helper)
// ==============================
// Avatar エンティティに FirebaseUID フィールドが無い前提のため、
// Firestore の "userId" を Firebase UID として扱う。
func (r *AvatarRepositoryFS) GetByFirebaseUID(ctx context.Context, uid string) (avdom.Avatar, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return avdom.Avatar{}, errNotFound
	}

	q := r.col().Where("userId", "==", uid).Limit(1)
	iter := q.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return avdom.Avatar{}, errNotFound
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
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		a, err := r.docToDomain(doc)
		if err != nil {
			return 0, err
		}
		if matchFilterPostLoad(a, filter) {
			count++
		}
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

	// userId は Firebase UID を格納している前提（= /mall/me/avatar 解決キー）
	data := r.domainToDocData(a)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return avdom.Avatar{}, errConflict
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
//
// ✅ 重要: walletAddress は「avatar につき 1回だけ」設定可能。
// - すでに walletAddress が入っている場合は上書きしない（Conflict）。
// - 空文字/nil で walletAddress を消すことも許可しない。
// - 競合を避けるため walletAddress を含む更新は Transaction で行う。
func (r *AvatarRepositoryFS) Update(ctx context.Context, id string, patch avdom.AvatarPatch) (avdom.Avatar, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return avdom.Avatar{}, errNotFound
	}
	ref := r.col().Doc(id)

	// walletAddress を含む場合は transaction で「未設定ならセット」を保証
	if patch.WalletAddress != nil {
		want := strings.TrimSpace(*patch.WalletAddress)
		if want == "" {
			return avdom.Avatar{}, errInvalidWalletAddr
		}

		if r.Client == nil {
			return avdom.Avatar{}, errBadClient
		}

		// sanitize optional strings (empty -> nil)
		sAvatarIcon := trimPtr(patch.AvatarIcon)
		sProfile := trimPtr(patch.Profile)
		sExternalLink := trimPtr(patch.ExternalLink)

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			snap, err := tx.Get(ref)
			if status.Code(err) == codes.NotFound {
				return errNotFound
			}
			if err != nil {
				return err
			}

			// 既に walletAddress があるなら上書き禁止
			existing := getStringFieldTrimmed(snap, "walletAddress")
			if existing != "" {
				// 既に同じ値が入っている場合も「もう開設済み」として Conflict 扱い
				return errWalletAlreadyBound
			}

			var updates []firestore.Update

			// walletAddress はこの transaction で一度だけセット可能
			updates = append(updates, firestore.Update{
				Path:  "walletAddress",
				Value: want,
			})

			// ✅ firebaseUid は Avatar エンティティに存在しないため保存しない
			// patch.FirebaseUID は互換のため受け取っても無視（userId を更新したいなら別途仕様化推奨）

			if patch.AvatarName != nil {
				updates = append(updates, firestore.Update{
					Path:  "avatarName",
					Value: strings.TrimSpace(*patch.AvatarName),
				})
			}

			// ✅ entity.go 正: AvatarIconURL/Path -> AvatarIcon
			if patch.AvatarIcon != nil {
				var v any
				if sAvatarIcon == nil {
					v = nil
				} else {
					v = *sAvatarIcon
				}
				updates = append(updates, firestore.Update{
					Path:  "avatarIcon",
					Value: v,
				})
			}

			if patch.Profile != nil {
				var v any
				if sProfile == nil {
					v = nil
				} else {
					v = *sProfile
				}
				updates = append(updates, firestore.Update{
					Path:  "profile",
					Value: v,
				})
			}

			if patch.ExternalLink != nil {
				var v any
				if sExternalLink == nil {
					v = nil
				} else {
					v = *sExternalLink
				}
				updates = append(updates, firestore.Update{
					Path:  "externalLink",
					Value: v,
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

			// Always bump updatedAt
			updates = append(updates, firestore.Update{
				Path:  "updatedAt",
				Value: time.Now().UTC(),
			})

			if err := tx.Update(ref, updates); err != nil {
				if status.Code(err) == codes.NotFound {
					return errNotFound
				}
				return err
			}
			return nil
		})
		if err != nil {
			// wallet already set は conflict として返す
			if errors.Is(err, errWalletAlreadyBound) {
				return avdom.Avatar{}, errConflict
			}
			if errors.Is(err, errNotFound) {
				return avdom.Avatar{}, errNotFound
			}
			return avdom.Avatar{}, err
		}

		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return avdom.Avatar{}, errNotFound
			}
			return avdom.Avatar{}, err
		}
		return r.docToDomain(snap)
	}

	// ------------------------------
	// walletAddress を含まない通常更新
	// ------------------------------

	// Ensure exists
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return avdom.Avatar{}, errNotFound
	} else if err != nil {
		return avdom.Avatar{}, err
	}

	var updates []firestore.Update

	// ✅ firebaseUid は Avatar エンティティに存在しないため保存しない
	// patch.FirebaseUID は互換のため受け取っても無視

	if patch.AvatarName != nil {
		updates = append(updates, firestore.Update{
			Path:  "avatarName",
			Value: strings.TrimSpace(*patch.AvatarName),
		})
	}

	// ✅ entity.go 正: AvatarIconURL/Path -> AvatarIcon
	if patch.AvatarIcon != nil {
		updates = append(updates, firestore.Update{
			Path:  "avatarIcon",
			Value: optionalString(*patch.AvatarIcon),
		})
	}

	// ❌ walletAddress は通常 Update では扱わない（上書き防止のため）
	// if patch.WalletAddress != nil { ... } // <- intentionally ignored here

	if patch.Profile != nil {
		updates = append(updates, firestore.Update{
			Path:  "profile",
			Value: optionalString(*patch.Profile),
		})
	}

	if patch.ExternalLink != nil {
		updates = append(updates, firestore.Update{
			Path:  "externalLink",
			Value: optionalString(*patch.ExternalLink),
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

	if len(updates) == 0 {
		// no-op: return current
		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return avdom.Avatar{}, errNotFound
			}
			return avdom.Avatar{}, err
		}
		return r.docToDomain(snap)
	}

	// Always bump updatedAt
	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return avdom.Avatar{}, errNotFound
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
		return errNotFound
	}
	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return errNotFound
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

	// userId は Firebase UID を格納している前提
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

	fsQuery := r.col().Limit(500)
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

		// userId (Firebase UID) も検索対象に含める
		uid := strings.ToLower(strings.TrimSpace(a.UserID))

		name := strings.ToLower(strings.TrimSpace(a.AvatarName))

		wallet := ""
		if a.WalletAddress != nil {
			wallet = strings.ToLower(strings.TrimSpace(*a.WalletAddress))
		}

		profile := ""
		if a.Profile != nil {
			profile = strings.ToLower(strings.TrimSpace(*a.Profile))
		}

		link := ""
		if a.ExternalLink != nil {
			link = strings.ToLower(strings.TrimSpace(*a.ExternalLink))
		}

		icon := ""
		if a.AvatarIcon != nil {
			icon = strings.ToLower(strings.TrimSpace(*a.AvatarIcon))
		}

		if strings.Contains(uid, lowerQ) ||
			strings.Contains(name, lowerQ) ||
			strings.Contains(wallet, lowerQ) ||
			strings.Contains(profile, lowerQ) ||
			strings.Contains(link, lowerQ) ||
			strings.Contains(icon, lowerQ) {
			list = append(list, a)
		}
	}
	return list, nil
}

// ==============================
// ListTopByFollowers (placeholder impl)
// ==============================

func (r *AvatarRepositoryFS) ListTopByFollowers(ctx context.Context, limit int) ([]avdom.Avatar, error) {
	// follower 情報が別にある前提なので、ここでは createdAt DESC を代用。
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
// Reset (development/testing) - use Transaction instead of deprecated Batch
// ==============================

func (r *AvatarRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errBadClient
	}

	// まず全ドキュメントIDを取得
	iter := r.col().Documents(ctx)
	defer iter.Stop()

	var ids []string
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		ids = append(ids, doc.Ref.ID)
	}

	if len(ids) == 0 {
		log.Printf("[firestore] Reset avatars: deleted 0 docs\n")
		return nil
	}

	const chunkSize = 400
	deleted := 0

	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]

		// 各チャンクをトランザクションで削除
		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, id := range chunk {
				ref := r.col().Doc(id)
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		deleted += len(chunk)
	}

	log.Printf("[firestore] Reset avatars: deleted %d docs\n", deleted)
	return nil
}

// ==============================
// Mapping helpers
// ==============================

func (r *AvatarRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (avdom.Avatar, error) {
	var raw struct {
		// ✅ userId は Firebase UID を格納している前提
		UserID string `firestore:"userId"`

		AvatarName    string     `firestore:"avatarName"`
		AvatarIcon    *string    `firestore:"avatarIcon"`
		WalletAddress *string    `firestore:"walletAddress"`
		Profile       *string    `firestore:"profile"`
		ExternalLink  *string    `firestore:"externalLink"`
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

	if raw.AvatarIcon != nil && strings.TrimSpace(*raw.AvatarIcon) != "" {
		v := strings.TrimSpace(*raw.AvatarIcon)
		a.AvatarIcon = &v
	}
	if raw.WalletAddress != nil && strings.TrimSpace(*raw.WalletAddress) != "" {
		v := strings.TrimSpace(*raw.WalletAddress)
		a.WalletAddress = &v
	}
	if raw.Profile != nil && strings.TrimSpace(*raw.Profile) != "" {
		v := strings.TrimSpace(*raw.Profile)
		a.Profile = &v
	}
	if raw.ExternalLink != nil && strings.TrimSpace(*raw.ExternalLink) != "" {
		v := strings.TrimSpace(*raw.ExternalLink)
		a.ExternalLink = &v
	}
	if raw.DeletedAt != nil && !raw.DeletedAt.IsZero() {
		t := raw.DeletedAt.UTC()
		a.DeletedAt = &t
	}

	return a, nil
}

func (r *AvatarRepositoryFS) domainToDocData(a avdom.Avatar) map[string]any {
	data := map[string]any{
		// ✅ userId は Firebase UID を格納している前提
		"userId":     strings.TrimSpace(a.UserID),
		"avatarName": strings.TrimSpace(a.AvatarName),
		"createdAt":  a.CreatedAt.UTC(),
		"updatedAt":  a.UpdatedAt.UTC(),
	}

	if a.AvatarIcon != nil && strings.TrimSpace(*a.AvatarIcon) != "" {
		data["avatarIcon"] = strings.TrimSpace(*a.AvatarIcon)
	}
	if a.WalletAddress != nil && strings.TrimSpace(*a.WalletAddress) != "" {
		data["walletAddress"] = strings.TrimSpace(*a.WalletAddress)
	}
	if a.Profile != nil && strings.TrimSpace(*a.Profile) != "" {
		data["profile"] = strings.TrimSpace(*a.Profile)
	}
	if a.ExternalLink != nil && strings.TrimSpace(*a.ExternalLink) != "" {
		data["externalLink"] = strings.TrimSpace(*a.ExternalLink)
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
	// Firestore 制約のため、代表的な条件のみをクエリに反映。
	if f.UserID != nil && strings.TrimSpace(*f.UserID) != "" {
		q = q.Where("userId", "==", strings.TrimSpace(*f.UserID))
	}

	// ✅ 互換: Filter.FirebaseUID が渡された場合も userId に寄せる（= Firebase UID を userId に格納している前提）
	if f.FirebaseUID != nil && strings.TrimSpace(*f.FirebaseUID) != "" {
		q = q.Where("userId", "==", strings.TrimSpace(*f.FirebaseUID))
	}

	if f.WalletAddress != nil && strings.TrimSpace(*f.WalletAddress) != "" {
		q = q.Where("walletAddress", "==", strings.TrimSpace(*f.WalletAddress))
	}
	if f.JoinedFrom != nil {
		q = q.Where("createdAt", ">=", f.JoinedFrom.UTC())
	}
	if f.JoinedTo != nil {
		q = q.Where("createdAt", "<", f.JoinedTo.UTC())
	}
	// Deleted / SearchQuery / CreatedFrom/To / UpdatedFrom/To は post-load で対応
	return q
}

// Firestoreで表現しなかった条件を post-load で絞り込み
func matchFilterPostLoad(a avdom.Avatar, f avdom.Filter) bool {
	// Deleted
	if f.Deleted != nil {
		if *f.Deleted {
			if a.DeletedAt == nil {
				return false
			}
		} else {
			if a.DeletedAt != nil {
				return false
			}
		}
	}

	// Created/Updated ranges
	if f.CreatedFrom != nil && a.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && a.CreatedAt.After(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && a.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && a.UpdatedAt.After(f.UpdatedTo.UTC()) {
		return false
	}

	// SearchQuery: id, (firebaseUid=UserID), avatarName, profile, externalLink, walletAddress, avatarIcon の部分一致
	sq := strings.TrimSpace(f.SearchQuery)
	if sq != "" {
		q := strings.ToLower(sq)

		id := strings.ToLower(strings.TrimSpace(a.ID))
		uid := strings.ToLower(strings.TrimSpace(a.UserID))
		name := strings.ToLower(strings.TrimSpace(a.AvatarName))

		wallet := ""
		if a.WalletAddress != nil {
			wallet = strings.ToLower(strings.TrimSpace(*a.WalletAddress))
		}

		profile := ""
		if a.Profile != nil {
			profile = strings.ToLower(strings.TrimSpace(*a.Profile))
		}

		link := ""
		if a.ExternalLink != nil {
			link = strings.ToLower(strings.TrimSpace(*a.ExternalLink))
		}

		icon := ""
		if a.AvatarIcon != nil {
			icon = strings.ToLower(strings.TrimSpace(*a.AvatarIcon))
		}

		if !strings.Contains(id, q) &&
			!strings.Contains(uid, q) &&
			!strings.Contains(name, q) &&
			!strings.Contains(wallet, q) &&
			!strings.Contains(profile, q) &&
			!strings.Contains(link, q) &&
			!strings.Contains(icon, q) {
			return false
		}
	}

	return true
}

func mapAvatarSort(s common.Sort) (field string, dir firestore.Direction) {
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

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func getStringFieldTrimmed(snap *firestore.DocumentSnapshot, field string) string {
	if snap == nil {
		return ""
	}
	m := snap.Data()
	if m == nil {
		return ""
	}
	v, ok := m[field]
	if !ok || v == nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}
