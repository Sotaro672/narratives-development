// backend/internal/adapters/out/firestore/user_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	udom "narratives/internal/domain/user"
)

// =====================================================
// Firestore User Repository
// (PostgreSQL 実装相当のインターフェースを Firestore で提供)
// =====================================================
//
// IMPORTANT:
// - users コレクションの DocID は "user.ID(=Firebase Auth UID)" に統一する。
// - これにより「userId と UID が一致しない」問題を根本的に解消する。
// =====================================================

type UserRepositoryFS struct {
	Client *firestore.Client
}

func NewUserRepositoryFS(client *firestore.Client) *UserRepositoryFS {
	return &UserRepositoryFS{Client: client}
}

func (r *UserRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("users")
}

// =====================================================
// usecase.UserRepo 準拠メソッド
// =====================================================

// GetByID(ctx, id) (udom.User, error)
func (r *UserRepositoryFS) GetByID(ctx context.Context, id string) (udom.User, error) {
	if r.Client == nil {
		return udom.User{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		// 「存在しない」とは別なので invalid とする（ハンドラが 400 にできる）
		return udom.User{}, udom.ErrInvalidID
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return udom.User{}, udom.ErrNotFound
	}
	if err != nil {
		return udom.User{}, err
	}

	return docToUser(snap)
}

// Exists(ctx, id) (bool, error)
func (r *UserRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

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

// Create(ctx, v udom.User) (udom.User, error)
//
// IMPORTANT:
// - Firestore の自動採番(NewDoc/Add)は使わない。
// - DocID は v.ID (Firebase Auth UID) を使う。
// - 既に存在する場合は conflict。
func (r *UserRepositoryFS) Create(ctx context.Context, v udom.User) (udom.User, error) {
	if r.Client == nil {
		return udom.User{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return udom.User{}, udom.ErrInvalidID
	}

	now := time.Now().UTC()

	createdAt := now
	if !v.CreatedAt.IsZero() {
		createdAt = v.CreatedAt.UTC()
	}

	updatedAt := now
	if !v.UpdatedAt.IsZero() {
		updatedAt = v.UpdatedAt.UTC()
	}

	// Postgres 実装同様 deleted_at NOT NULL 相当の扱い:
	// デフォルトは createdAt、指定されていれば max(createdAt, DeletedAt)
	deletedAt := createdAt
	if !v.DeletedAt.IsZero() {
		del := v.DeletedAt.UTC()
		if del.After(createdAt) {
			deletedAt = del
		}
	}

	ref := r.col().Doc(id)

	data := map[string]any{
		"createdAt": createdAt,
		"updatedAt": updatedAt,
		"deletedAt": deletedAt,
	}

	if v.FirstName != nil {
		if s := strings.TrimSpace(*v.FirstName); s != "" {
			data["firstName"] = s
		}
	}
	if v.FirstNameKana != nil {
		if s := strings.TrimSpace(*v.FirstNameKana); s != "" {
			data["firstNameKana"] = s
		}
	}
	if v.LastNameKana != nil {
		if s := strings.TrimSpace(*v.LastNameKana); s != "" {
			data["lastNameKana"] = s
		}
	}
	if v.LastName != nil {
		if s := strings.TrimSpace(*v.LastName); s != "" {
			data["lastName"] = s
		}
	}

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return udom.User{}, udom.ErrConflict
		}
		return udom.User{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return udom.User{}, err
	}
	return docToUser(snap)
}

// Save(ctx, v udom.User) (udom.User, error)
//
// Upsert 的挙動:
// - ID が空 → invalid
// - 存在する → Update
// - 存在しない → Create (DocID は指定 ID のまま)
func (r *UserRepositoryFS) Save(ctx context.Context, v udom.User) (udom.User, error) {
	if r.Client == nil {
		return udom.User{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return udom.User{}, udom.ErrInvalidID
	}

	exists, err := r.Exists(ctx, id)
	if err != nil {
		return udom.User{}, err
	}
	if !exists {
		// 指定 ID (UID) で新規作成
		// createdAt/updatedAt/deletedAt は Create 側で整形される
		return r.Create(ctx, v)
	}

	patch := udom.UpdateUserInput{
		FirstName:     copyIfTrimmedNonEmpty(v.FirstName),
		FirstNameKana: copyIfTrimmedNonEmpty(v.FirstNameKana),
		LastNameKana:  copyIfTrimmedNonEmpty(v.LastNameKana),
		LastName:      copyIfTrimmedNonEmpty(v.LastName),
		UpdatedAt: func(t time.Time) *time.Time {
			if t.IsZero() {
				return nil
			}
			tt := t.UTC()
			return &tt
		}(v.UpdatedAt),
		DeletedAt: func(t time.Time) *time.Time {
			if t.IsZero() {
				return nil
			}
			tt := t.UTC()
			return &tt
		}(v.DeletedAt),
	}

	updated, err := r.Update(ctx, id, patch)
	if err != nil {
		return udom.User{}, err
	}
	if updated == nil {
		return udom.User{}, udom.ErrNotFound
	}
	return *updated, nil
}

// Delete(ctx, id) error は下の Delete 実装を参照。

// ======================================================================
// Lower-level / richer query methods (PG版互換)
// ======================================================================

// List: Firestore 制約により全件取得してメモリ上で Filter/Sort/Paging を適用。
func (r *UserRepositoryFS) List(ctx context.Context, filter udom.Filter, sortOpt udom.Sort, page udom.Page) (udom.PageResult, error) {
	if r.Client == nil {
		return udom.PageResult{}, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var all []udom.User
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return udom.PageResult{}, err
		}
		u, err := docToUser(snap)
		if err != nil {
			return udom.PageResult{}, err
		}
		if matchUserFilter(u, filter) {
			all = append(all, u)
		}
	}

	sortUsers(all, sortOpt)

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	paged := all[offset:end]

	return udom.PageResult{
		Items:      paged,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Count: List と同じ条件でメモリ上でカウント
func (r *UserRepositoryFS) Count(ctx context.Context, filter udom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		u, err := docToUser(snap)
		if err != nil {
			return 0, err
		}
		if matchUserFilter(u, filter) {
			total++
		}
	}
	return total, nil
}

// Update(ctx, id, in) (*udom.User, error)
// Postgres 実装に近い PATCH 振る舞い。
func (r *UserRepositoryFS) Update(ctx context.Context, id string, in udom.UpdateUserInput) (*udom.User, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, udom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	// 存在確認
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, udom.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	setStr := func(path string, p *string) {
		if p != nil {
			v := strings.TrimSpace(*p)
			if v == "" {
				updates = append(updates, firestore.Update{
					Path:  path,
					Value: firestore.Delete,
				})
				return
			}
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: v,
			})
		}
	}

	// first/last names
	setStr("firstName", in.FirstName)
	setStr("firstNameKana", in.FirstNameKana)
	setStr("lastNameKana", in.LastNameKana)
	setStr("lastName", in.LastName)

	// updatedAt: 指定なければ NOW()
	if in.UpdatedAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: in.UpdatedAt.UTC(),
		})
	} else {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		})
	}

	// deletedAt: nil なら変更なし（PG版と同じく「0値は触らない」扱い）
	if in.DeletedAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "deletedAt",
			Value: in.DeletedAt.UTC(),
		})
	}

	if len(updates) == 0 {
		got, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return &got, nil
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, udom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, udom.ErrConflict
		}
		return nil, err
	}

	got, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &got, nil
}

func (r *UserRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return udom.ErrInvalidID
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return udom.ErrNotFound
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// OPTIONAL extra helpers (PG版互換)

func (r *UserRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var snaps []*firestore.DocumentSnapshot
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		snaps = append(snaps, snap)
	}

	// transaction, chunked
	const chunkSize = 400
	for i := 0; i < len(snaps); i += chunkSize {
		end := i + chunkSize
		if end > len(snaps) {
			end = len(snaps)
		}
		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, s := range snaps[i:end] {
				if err := tx.Delete(s.Ref); err != nil {
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

func (r *UserRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	// Firestore のマルチドキュメントTxが不要な前提で単純ラップ。
	// 必要になれば Client.RunTransaction で拡張可能。
	return fn(ctx)
}

// =====================================================
// Helpers: Firestore -> Domain
// =====================================================

func docToUser(doc *firestore.DocumentSnapshot) (udom.User, error) {
	data := doc.Data()
	if data == nil {
		return udom.User{}, udom.ErrNotFound
	}

	getStrPtr := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					return &s
				}
				return nil
			}
		}
		return nil
	}

	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	return udom.User{
		ID:            strings.TrimSpace(doc.Ref.ID),
		FirstName:     getStrPtr("firstName", "first_name"),
		FirstNameKana: getStrPtr("firstNameKana", "first_name_kana"),
		LastNameKana:  getStrPtr("lastNameKana", "last_name_kana"),
		LastName:      getStrPtr("lastName", "last_name"),
		CreatedAt:     getTime("createdAt", "created_at"),
		UpdatedAt:     getTime("updatedAt", "updated_at"),
		DeletedAt:     getTime("deletedAt", "deleted_at"),
	}, nil
}

// =====================================================
// Helpers: Filter / Sort (in-memory, PG版の buildUserWhere / buildUserOrderBy 相当)
// =====================================================

func matchUserFilter(u udom.User, f udom.Filter) bool {
	// IDs
	if len(f.IDs) > 0 {
		match := false
		for _, id := range f.IDs {
			if strings.TrimSpace(id) == u.ID {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	// FirstNameLike
	if v := strings.TrimSpace(f.FirstNameLike); v != "" {
		p := strings.ToLower(v)
		name := ""
		if u.FirstName != nil {
			name = strings.ToLower(strings.TrimSpace(*u.FirstName))
		}
		if !strings.Contains(name, p) {
			return false
		}
	}

	// LastNameLike
	if v := strings.TrimSpace(f.LastNameLike); v != "" {
		p := strings.ToLower(v)
		name := ""
		if u.LastName != nil {
			name = strings.ToLower(strings.TrimSpace(*u.LastName))
		}
		if !strings.Contains(name, p) {
			return false
		}
	}

	// NameLike (first OR last)
	if v := strings.TrimSpace(f.NameLike); v != "" {
		p := strings.ToLower(v)
		fn := ""
		ln := ""
		if u.FirstName != nil {
			fn = strings.ToLower(strings.TrimSpace(*u.FirstName))
		}
		if u.LastName != nil {
			ln = strings.ToLower(strings.TrimSpace(*u.LastName))
		}
		if !strings.Contains(fn, p) && !strings.Contains(ln, p) {
			return false
		}
	}

	// Time ranges
	if f.CreatedFrom != nil && u.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !u.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && u.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !u.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}
	if f.DeletedFrom != nil && u.DeletedAt.Before(f.DeletedFrom.UTC()) {
		return false
	}
	if f.DeletedTo != nil && !u.DeletedAt.Before(f.DeletedTo.UTC()) {
		return false
	}

	return true
}

func sortUsers(items []udom.User, s udom.Sort) {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	asc := dir == "ASC"

	less := func(i, j int) bool {
		a := items[i]
		b := items[j]

		switch col {
		case "createdat", "created_at":
			if a.CreatedAt.Equal(b.CreatedAt) {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.CreatedAt.Before(b.CreatedAt)
			}
			return a.CreatedAt.After(b.CreatedAt)

		case "updatedat", "updated_at":
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.UpdatedAt.Before(b.UpdatedAt)
			}
			return a.UpdatedAt.After(b.UpdatedAt)

		case "deletedat", "deleted_at":
			if a.DeletedAt.Equal(b.DeletedAt) {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.DeletedAt.Before(b.DeletedAt)
			}
			return a.DeletedAt.After(b.DeletedAt)

		case "first_name", "firstname":
			var af, bf string
			if a.FirstName != nil {
				af = *a.FirstName
			}
			if b.FirstName != nil {
				bf = *b.FirstName
			}
			if af == bf {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return af < bf
			}
			return af > bf

		case "last_name", "lastname":
			var al, bl string
			if a.LastName != nil {
				al = *a.LastName
			}
			if b.LastName != nil {
				bl = *b.LastName
			}
			if al == bl {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return al < bl
			}
			return al > bl

		default:
			// デフォルト: createdAt DESC, id DESC
			if a.CreatedAt.Equal(b.CreatedAt) {
				return a.ID > b.ID
			}
			return a.CreatedAt.After(b.CreatedAt)
		}
	}

	sort.SliceStable(items, less)
}

// =====================================================
// Small helpers
// =====================================================

func copyIfTrimmedNonEmpty(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}
