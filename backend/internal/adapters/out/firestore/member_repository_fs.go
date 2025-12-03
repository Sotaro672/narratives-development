// backend/internal/adapters/out/firestore/member_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	common "narratives/internal/domain/common"
	memdom "narratives/internal/domain/member"
)

// MemberRepositoryFS is a Firestore-based implementation of member.Repository.
// Uses the "members" collection.
type MemberRepositoryFS struct {
	Client *firestore.Client
}

func NewMemberRepositoryFS(client *firestore.Client) *MemberRepositoryFS {
	return &MemberRepositoryFS{Client: client}
}

func (r *MemberRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("members")
}

// Compile-time check
var _ memdom.Repository = (*MemberRepositoryFS)(nil)

// ========================
// Queries
// ========================

func (r *MemberRepositoryFS) GetByID(ctx context.Context, id string) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return memdom.Member{}, memdom.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return memdom.Member{}, memdom.ErrNotFound
		}
		return memdom.Member{}, err
	}

	m, err := readMemberSnapshot(doc)
	if err != nil {
		return memdom.Member{}, err
	}
	if m.ID == "" {
		m.ID = doc.Ref.ID
	}
	return m, nil
}

func (r *MemberRepositoryFS) GetByEmail(ctx context.Context, email string) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return memdom.Member{}, memdom.ErrNotFound
	}

	q := r.col().Where("email", "==", email).Limit(1)
	it := q.Documents(ctx)
	defer it.Stop()

	doc, err := it.Next()
	if err == iterator.Done {
		return memdom.Member{}, memdom.ErrNotFound
	}
	if err != nil {
		return memdom.Member{}, err
	}

	m, err := readMemberSnapshot(doc)
	if err != nil {
		return memdom.Member{}, err
	}
	if m.ID == "" {
		m.ID = doc.Ref.ID
	}
	return m, nil
}

// Firebase UID から取得するメソッド（現在は ID = FirebaseUID 前提でラップ）
func (r *MemberRepositoryFS) GetByFirebaseUID(ctx context.Context, firebaseUID string) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}
	uid := strings.TrimSpace(firebaseUID)
	if uid == "" {
		return memdom.Member{}, memdom.ErrNotFound
	}

	// 将来 firebaseUid フィールドを設けるなら:
	// q := r.col().Where("firebaseUid", "==", uid).Limit(1) ...
	// といった実装に差し替えればOK。
	return r.GetByID(ctx, uid)
}

func (r *MemberRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

// ========================
// List / Count
// ========================

// Count は companyId だけを Firestore クエリに反映し、その他の filter 条件は無視します。
func (r *MemberRepositoryFS) Count(ctx context.Context, f memdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	q := r.col().Query
	if cid := strings.TrimSpace(f.CompanyID); cid != "" {
		q = q.Where("companyId", "==", cid)
	}

	it := q.Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		_, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		total++
	}
	return total, nil
}

// List は companyId で絞り込み、updatedAt desc / docID desc の固定ソートでページングします。
// s / f の詳細な sort / filter 条件は使用しません。
func (r *MemberRepositoryFS) List(
	ctx context.Context,
	f memdom.Filter,
	_ common.Sort,
	p common.Page,
) (common.PageResult[memdom.Member], error) {
	if r.Client == nil {
		return common.PageResult[memdom.Member]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(p.Number, p.PerPage, 50, 200)

	q := r.col().Query
	if cid := strings.TrimSpace(f.CompanyID); cid != "" {
		q = q.Where("companyId", "==", cid)
	}
	// 固定ソート: updatedAt desc → docID desc
	q = q.OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []memdom.Member
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[memdom.Member]{}, err
		}

		m, err := readMemberSnapshot(doc)
		if err != nil {
			return common.PageResult[memdom.Member]{}, err
		}
		if m.ID == "" {
			m.ID = doc.Ref.ID
		}
		all = append(all, m)
	}

	total := len(all)
	if total == 0 {
		return common.PageResult[memdom.Member]{
			Items:      []memdom.Member{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	return common.PageResult[memdom.Member]{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListByCursor は companyId で絞り込み、updatedAt desc / docID desc の固定ソートで
// ID ベースのカーソルページングを行います。memdom.Sort は使用しません。
func (r *MemberRepositoryFS) ListByCursor(
	ctx context.Context,
	f memdom.Filter,
	cpage memdom.CursorPage,
) (memdom.CursorPageResult, error) {
	if r.Client == nil {
		return memdom.CursorPageResult{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := r.col().Query
	if cid := strings.TrimSpace(f.CompanyID); cid != "" {
		q = q.Where("companyId", "==", cid)
	}
	// 固定ソート
	q = q.OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items []memdom.Member
		last  string
	)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return memdom.CursorPageResult{}, err
		}

		m, err := readMemberSnapshot(doc)
		if err != nil {
			return memdom.CursorPageResult{}, err
		}
		if m.ID == "" {
			m.ID = doc.Ref.ID
		}

		if skipping {
			if m.ID <= after {
				continue
			}
			skipping = false
		}

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

	return memdom.CursorPageResult{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// ========================
// Mutations
// ========================

func (r *MemberRepositoryFS) Create(ctx context.Context, m memdom.Member) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}

	ref := r.col().Doc(strings.TrimSpace(m.ID))
	if m.ID == "" {
		ref = r.col().NewDoc()
		m.ID = ref.ID
	}

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = &now

	if _, err := ref.Create(ctx, m); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return memdom.Member{}, memdom.ErrConflict
		}
		return memdom.Member{}, err
	}
	return m, nil
}

// Update: required by common.RepositoryCRUD[Member, MemberPatch].
// TODO: apply fields from MemberPatch when needed.
func (r *MemberRepositoryFS) Update(ctx context.Context, id string, _ memdom.MemberPatch) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return memdom.Member{}, memdom.ErrNotFound
	}

	// For now, just return the current entity.
	return r.GetByID(ctx, id)
}

func (r *MemberRepositoryFS) Save(ctx context.Context, m memdom.Member, _ *memdom.SaveOptions) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}

	if strings.TrimSpace(m.ID) == "" {
		ref := r.col().NewDoc()
		m.ID = ref.ID
	}

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = &now

	ref := r.col().Doc(m.ID)
	// ★ MergeAll をやめて、構造体ごと上書き
	if _, err := ref.Set(ctx, m); err != nil {
		return memdom.Member{}, err
	}
	return m, nil
}

func (r *MemberRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return memdom.ErrNotFound
	}

	ref := r.col().Doc(id)
	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return memdom.ErrNotFound
		}
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

func (r *MemberRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		return nil
	}

	const chunkSize = 400
	for i := 0; i < len(refs); i += chunkSize {
		end := i + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[i:end]

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

// ========================
// Helper: decode with legacy string timestamps support
// ========================

func readMemberSnapshot(doc *firestore.DocumentSnapshot) (memdom.Member, error) {
	// Fast path: native decode (when all timestamp fields are Firestore Timestamp)
	var m memdom.Member
	if err := doc.DataTo(&m); err == nil {
		if m.ID == "" {
			m.ID = doc.Ref.ID
		}
		return m, nil
	}

	// Fallback: decode from map and convert string/timestamp to time
	data := doc.Data()
	asString := func(v any) string {
		if s, ok := v.(string); ok {
			return s
		}
		return ""
	}
	asStringSlice := func(v any) []string {
		if v == nil {
			return nil
		}
		if ss, ok := v.([]string); ok {
			return ss
		}
		arr, ok := v.([]interface{})
		if !ok {
			return nil
		}
		out := make([]string, 0, len(arr))
		for _, x := range arr {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}
	asTimePtr := func(v any) (*time.Time, error) {
		switch t := v.(type) {
		case time.Time:
			tt := t.UTC()
			return &tt, nil
		case *time.Time:
			if t == nil {
				return nil, nil
			}
			tt := t.UTC()
			return &tt, nil
		case string:
			s := strings.TrimSpace(t)
			if s == "" {
				return nil, nil
			}
			// Try RFC3339 first
			if parsed, err := time.Parse(time.RFC3339, s); err == nil {
				tt := parsed.UTC()
				return &tt, nil
			}
			// Loose fallback (e.g., "2006-01-02 15:04:05Z07:00")
			if parsed, err := time.Parse("2006-01-02 15:04:05Z07:00", s); err == nil {
				tt := parsed.UTC()
				return &tt, nil
			}
			return nil, fmt.Errorf("invalid time string: %q", s)
		default:
			return nil, nil
		}
	}

	m = memdom.Member{
		ID:             doc.Ref.ID,
		FirstName:      asString(data["firstName"]),
		LastName:       asString(data["lastName"]),
		FirstNameKana:  asString(data["firstNameKana"]),
		LastNameKana:   asString(data["lastNameKana"]),
		Email:          asString(data["email"]),
		Permissions:    asStringSlice(data["permissions"]),
		AssignedBrands: asStringSlice(data["assignedBrands"]),
		CompanyID:      asString(data["companyId"]),
		Status:         asString(data["status"]),
	}

	// createdAt (required-ish): if missing or invalid, leave zero value
	if v, err := asTimePtr(data["createdAt"]); err == nil && v != nil {
		m.CreatedAt = *v
	}

	// optional times
	if v, _ := asTimePtr(data["updatedAt"]); v != nil {
		m.UpdatedAt = v
	}
	if v, _ := asTimePtr(data["deletedAt"]); v != nil {
		m.DeletedAt = v
	}

	return m, nil
}
