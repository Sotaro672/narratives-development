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
// RepositoryPort 準拠メソッド
// =====================================================

// GetByID(ctx, id) (*udom.User, error)
func (r *UserRepositoryFS) GetByID(ctx context.Context, id string) (*udom.User, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		// 「存在しない」とは別なので invalid とする（ハンドラが 400 にできる）
		return nil, udom.ErrInvalidID
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, udom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	u, err := docToUser(snap)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

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

// ✅ NEW: GetNameByID(ctx, id) (string, error)
// - "lastName firstName" の順で返す（どちらか欠けても best-effort）
// - 両方空なら空文字（ErrNotFoundにはしない：画面表示用途のため）
func (r *UserRepositoryFS) GetNameByID(ctx context.Context, id string) (string, error) {
	u, err := r.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", udom.ErrNotFound
	}

	ln := ""
	fn := ""
	if u.LastName != nil {
		ln = strings.TrimSpace(*u.LastName)
	}
	if u.FirstName != nil {
		fn = strings.TrimSpace(*u.FirstName)
	}

	// lastName -> firstName
	switch {
	case ln != "" && fn != "":
		return ln + " " + fn, nil
	case ln != "":
		return ln, nil
	case fn != "":
		return fn, nil
	default:
		return "", nil
	}
}

// Create(ctx, in) (*udom.User, error)
func (r *UserRepositoryFS) Create(ctx context.Context, in udom.CreateUserInput) (*udom.User, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	updatedAt := now
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	// deletedAt: デフォルトは createdAt、指定されていれば max(createdAt, DeletedAt)
	deletedAt := createdAt
	if in.DeletedAt != nil && !in.DeletedAt.IsZero() {
		del := in.DeletedAt.UTC()
		if del.After(createdAt) {
			deletedAt = del
		}
	}

	data := map[string]any{
		"createdAt": createdAt,
		"updatedAt": updatedAt,
		"deletedAt": deletedAt,
	}

	setIfNonEmpty := func(key string, p *string) {
		if p == nil {
			return
		}
		s := strings.TrimSpace(*p)
		if s == "" {
			return
		}
		data[key] = s
	}

	setIfNonEmpty("firstName", in.FirstName)
	setIfNonEmpty("firstNameKana", in.FirstNameKana)
	setIfNonEmpty("lastNameKana", in.LastNameKana)
	setIfNonEmpty("lastName", in.LastName)

	// NOTE:
	// RepositoryPort の Create には id が無い。
	// 既存設計（DocID=UID）に合わせるため、この実装では「作成対象IDは in ではなく別レイヤで確定」している前提。
	// もしこの repo が直接 Create で UID を受け取る設計なら、別途 Create 引数を見直してください。
	//
	// ここでは互換のため、data 内に "id" が無い状態では作成できないので ErrInvalidID とする。
	rawID, ok := data["id"].(string)
	_ = rawID
	if !ok {
		return nil, udom.ErrInvalidID
	}

	// ↑上の "id" は現状コード上でセットされないので、実際にはここに到達しません。
	// 本プロジェクトの現行 Create 呼び出しが「udom.User を渡す」形だったため、
	// RepositoryPort に合わせて呼び出し側も合わせる必要があります。
	// （このコメントはコンパイル自体は通ります）

	return nil, errors.New("Create requires user id (DocID) to be provided by caller; adjust wiring to pass UID as doc id")
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
		if p == nil {
			return
		}
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

	// deletedAt: nil なら変更なし
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
		return got, nil
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

	return r.GetByID(ctx, id)
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
		FirstName:     getStrPtr("firstName"),
		FirstNameKana: getStrPtr("firstNameKana"),
		LastNameKana:  getStrPtr("lastNameKana"),
		LastName:      getStrPtr("lastName"),
		CreatedAt:     getTime("createdAt"),
		UpdatedAt:     getTime("updatedAt"),
		DeletedAt:     getTime("deletedAt"),
	}, nil
}

// =====================================================
// Helpers: Filter / Sort (in-memory)
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
		case "createdat":
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

		case "updatedat":
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

		case "deletedat":
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
