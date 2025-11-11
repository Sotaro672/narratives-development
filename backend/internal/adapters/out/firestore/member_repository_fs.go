// backend/internal/adapters/out/firestore/member_repository_fs.go
package firestore

import (
	"context"
	"errors"
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

	var m memdom.Member
	if err := doc.DataTo(&m); err != nil {
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

	var m memdom.Member
	if err := doc.DataTo(&m); err != nil {
		return memdom.Member{}, err
	}
	if m.ID == "" {
		m.ID = doc.Ref.ID
	}
	return m, nil
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

func (r *MemberRepositoryFS) Count(ctx context.Context, f memdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		var m memdom.Member
		if err := doc.DataTo(&m); err != nil {
			return 0, err
		}
		if m.ID == "" {
			m.ID = doc.Ref.ID
		}

		if matchMemberFilter(m, f) {
			total++
		}
	}
	return total, nil
}

func (r *MemberRepositoryFS) List(
	ctx context.Context,
	f memdom.Filter,
	s common.Sort,
	p common.Page,
) (common.PageResult[memdom.Member], error) {
	if r.Client == nil {
		return common.PageResult[memdom.Member]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(p.Number, p.PerPage, 50, 200)

	q := r.col().Query
	q = applyMemberSort(q, s)

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
		var m memdom.Member
		if err := doc.DataTo(&m); err != nil {
			return common.PageResult[memdom.Member]{}, err
		}
		if m.ID == "" {
			m.ID = doc.Ref.ID
		}
		if matchMemberFilter(m, f) {
			all = append(all, m)
		}
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

func (r *MemberRepositoryFS) ListByCursor(
	ctx context.Context,
	f memdom.Filter,
	s memdom.Sort,
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
	q = applyMemberSortForCursor(q, s)

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

		var m memdom.Member
		if err := doc.DataTo(&m); err != nil {
			return memdom.CursorPageResult{}, err
		}
		if m.ID == "" {
			m.ID = doc.Ref.ID
		}

		if !matchMemberFilter(m, f) {
			continue
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
	if _, err := ref.Set(ctx, m, firestore.MergeAll); err != nil {
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
// Helper: Filter & Sort
// ========================

func matchMemberFilter(m memdom.Member, f memdom.Filter) bool {
	// Free text
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		hay := strings.ToLower(m.ID + " " + m.FirstName + " " + m.LastName + " " + m.Email)
		if !strings.Contains(hay, lq) {
			return false
		}
	}

	// RoleIDs / Roles
	if len(f.RoleIDs) > 0 || len(f.Roles) > 0 {
		want := append(append([]string{}, f.RoleIDs...), f.Roles...)
		if !fscommon.ContainsString(want, string(m.Role)) {
			return false
		}
	}

	// BrandIDs / Brands
	if len(f.BrandIDs) > 0 || len(f.Brands) > 0 {
		want := append(append([]string{}, f.BrandIDs...), f.Brands...)
		if !fscommon.IntersectsStrings(want, m.AssignedBrands) {
			return false
		}
	}

	// Permissions (AND)
	if len(f.Permissions) > 0 && !fscommon.HasAllStrings(m.Permissions, f.Permissions) {
		return false
	}

	// CreatedAt / UpdatedAt ranges
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

func applyMemberSort(q firestore.Query, s common.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	var field string

	switch col {
	case "name":
		field = "firstName"
	case "email":
		field = "email"
	case "joinedat":
		field = "createdAt"
	case "updatedat":
		field = "updatedAt"
	default:
		return q.OrderBy("updatedAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Asc
	if strings.EqualFold(string(s.Order), "desc") {
		dir = firestore.Desc
	}

	return q.OrderBy(field, dir).
		OrderBy(firestore.DocumentID, dir)
}

func applyMemberSortForCursor(q firestore.Query, s memdom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	var field string

	switch col {
	case "name":
		field = "firstName"
	case "email":
		field = "email"
	case "joinedat":
		field = "createdAt"
	case "updatedat":
		field = "updatedAt"
	default:
		return q.OrderBy("updatedAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Asc
	if strings.EqualFold(string(s.Order), "desc") {
		dir = firestore.Desc
	}

	return q.OrderBy(field, dir).
		OrderBy(firestore.DocumentID, dir)
}
