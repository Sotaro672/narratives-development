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

	return m, nil
}

func (r *MemberRepositoryFS) GetByDocID(ctx context.Context, docID string) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}

	docID = strings.TrimSpace(docID)
	if docID == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}

	doc, err := r.col().Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return memdom.Record{}, memdom.ErrNotFound
		}
		return memdom.Record{}, err
	}

	m, err := readMemberSnapshot(doc)
	if err != nil {
		return memdom.Record{}, err
	}

	return memdom.Record{
		DocID:  doc.Ref.ID,
		Member: m,
	}, nil
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

	return m, nil
}

// GetByFirebaseUID returns a member whose uid field matches the Firebase Auth UID.
// Firestore document ID and Firebase Auth UID are intentionally separated.
func (r *MemberRepositoryFS) GetByFirebaseUID(ctx context.Context, firebaseUID string) (memdom.Member, error) {
	rec, err := r.GetRecordByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		return memdom.Member{}, err
	}

	return rec.Member, nil
}

// GetRecordByFirebaseUID returns a member record whose uid field matches the Firebase Auth UID.
// Use this when the caller also needs the Firestore document ID.
func (r *MemberRepositoryFS) GetRecordByFirebaseUID(ctx context.Context, firebaseUID string) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}

	firebaseUID = strings.TrimSpace(firebaseUID)
	if firebaseUID == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}

	q := r.col().Where("uid", "==", firebaseUID).Limit(1)
	it := q.Documents(ctx)
	defer it.Stop()

	doc, err := it.Next()
	if err == iterator.Done {
		return memdom.Record{}, memdom.ErrNotFound
	}
	if err != nil {
		return memdom.Record{}, err
	}

	m, err := readMemberSnapshot(doc)
	if err != nil {
		return memdom.Record{}, err
	}

	return memdom.Record{
		DocID:  doc.Ref.ID,
		Member: m,
	}, nil
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

// Count は companyId / uid / status を Firestore クエリに反映し、
// その他の filter 条件は無視します。
func (r *MemberRepositoryFS) Count(ctx context.Context, f memdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	q := r.col().Query

	if f.CompanyID != "" {
		q = q.Where("companyId", "==", strings.TrimSpace(f.CompanyID))
	}
	if f.UID != "" {
		q = q.Where("uid", "==", strings.TrimSpace(f.UID))
	}
	if f.Status != "" {
		q = q.Where("status", "==", strings.TrimSpace(f.Status))
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

// List は companyId / uid / status で絞り込み、updatedAt desc / docID desc の固定ソートでページングします。
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

	if f.CompanyID != "" {
		q = q.Where("companyId", "==", strings.TrimSpace(f.CompanyID))
	}
	if f.UID != "" {
		q = q.Where("uid", "==", strings.TrimSpace(f.UID))
	}
	if f.Status != "" {
		q = q.Where("status", "==", strings.TrimSpace(f.Status))
	}

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

func (r *MemberRepositoryFS) ListWithDocID(
	ctx context.Context,
	f memdom.Filter,
	_ common.Sort,
	p memdom.Page,
) (memdom.RecordPageResult, error) {
	if r.Client == nil {
		return memdom.RecordPageResult{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(p.Number, p.PerPage, 50, 200)

	q := r.col().Query

	if f.CompanyID != "" {
		q = q.Where("companyId", "==", strings.TrimSpace(f.CompanyID))
	}
	if f.UID != "" {
		q = q.Where("uid", "==", strings.TrimSpace(f.UID))
	}
	if f.Status != "" {
		q = q.Where("status", "==", strings.TrimSpace(f.Status))
	}

	q = q.OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []memdom.Record
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return memdom.RecordPageResult{}, err
		}

		m, err := readMemberSnapshot(doc)
		if err != nil {
			return memdom.RecordPageResult{}, err
		}

		all = append(all, memdom.Record{
			DocID:  doc.Ref.ID,
			Member: m,
		})
	}

	total := len(all)
	if total == 0 {
		return memdom.RecordPageResult{
			Items:      []memdom.Record{},
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

	return memdom.RecordPageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListByCursor は companyId / uid / status で絞り込み、updatedAt desc / docID desc の固定ソートで
// DocumentID ベースのカーソルページングを行います。memdom.Sort は使用しません。
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

	if f.CompanyID != "" {
		q = q.Where("companyId", "==", strings.TrimSpace(f.CompanyID))
	}
	if f.UID != "" {
		q = q.Where("uid", "==", strings.TrimSpace(f.UID))
	}
	if f.Status != "" {
		q = q.Where("status", "==", strings.TrimSpace(f.Status))
	}

	q = q.OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := cpage.After
	skipping := after != ""

	var (
		items          []memdom.Member
		lastDocID      string
		lastIncludedID string
	)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return memdom.CursorPageResult{}, err
		}

		docID := doc.Ref.ID

		if skipping {
			if docID == after {
				skipping = false
			}
			continue
		}

		m, err := readMemberSnapshot(doc)
		if err != nil {
			return memdom.CursorPageResult{}, err
		}

		items = append(items, m)
		lastDocID = docID
		if len(items) <= limit {
			lastIncludedID = docID
		}

		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		if lastIncludedID != "" {
			next = &lastIncludedID
		} else if lastDocID != "" {
			next = &lastDocID
		}
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

	ref := r.col().NewDoc()

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = &now

	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)

	if _, err := ref.Create(ctx, m); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return memdom.Member{}, memdom.ErrConflict
		}
		return memdom.Member{}, err
	}

	return m, nil
}

func (r *MemberRepositoryFS) CreateWithDocID(ctx context.Context, m memdom.Member) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}

	ref := r.col().NewDoc()

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = &now

	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)

	if _, err := ref.Create(ctx, m); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return memdom.Record{}, memdom.ErrConflict
		}
		return memdom.Record{}, err
	}

	return memdom.Record{
		DocID:  ref.ID,
		Member: m,
	}, nil
}

// Update: required by common.RepositoryCRUD[Member, MemberPatch].
func (r *MemberRepositoryFS) Update(ctx context.Context, id string, patch memdom.MemberPatch) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return memdom.Member{}, memdom.ErrNotFound
	}

	rec, err := r.GetByDocID(ctx, id)
	if err != nil {
		return memdom.Member{}, err
	}

	m := rec.Member

	if patch.UID != nil {
		m.UID = strings.TrimSpace(*patch.UID)
	}
	if patch.FirstName != nil {
		m.FirstName = *patch.FirstName
	}
	if patch.LastName != nil {
		m.LastName = *patch.LastName
	}
	if patch.FirstNameKana != nil {
		m.FirstNameKana = *patch.FirstNameKana
	}
	if patch.LastNameKana != nil {
		m.LastNameKana = *patch.LastNameKana
	}
	if patch.Email != nil {
		m.Email = strings.TrimSpace(*patch.Email)
	}
	if patch.Permissions != nil {
		m.Permissions = dedupStrings(*patch.Permissions)
	}
	if patch.AssignedBrands != nil {
		m.AssignedBrands = dedupStrings(*patch.AssignedBrands)
	}
	if patch.CompanyID != nil {
		m.CompanyID = strings.TrimSpace(*patch.CompanyID)
	}
	if patch.Status != nil {
		m.Status = strings.TrimSpace(*patch.Status)
	}
	if patch.CreatedAt != nil {
		m.CreatedAt = *patch.CreatedAt
	}
	if patch.UpdatedAt != nil {
		m.UpdatedAt = patch.UpdatedAt
	}
	if patch.UpdatedBy != nil {
		m.UpdatedBy = patch.UpdatedBy
	}
	if patch.DeletedAt != nil {
		m.DeletedAt = patch.DeletedAt
	}
	if patch.DeletedBy != nil {
		m.DeletedBy = patch.DeletedBy
	}

	return r.SaveByDocID(ctx, rec.DocID, m, nil)
}

func (r *MemberRepositoryFS) Save(ctx context.Context, m memdom.Member, _ *memdom.SaveOptions) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = &now

	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)

	if m.UID != "" {
		rec, err := r.GetRecordByFirebaseUID(ctx, m.UID)
		if err == nil {
			if _, err := r.col().Doc(rec.DocID).Set(ctx, m); err != nil {
				return memdom.Member{}, err
			}
			return m, nil
		}
	}

	ref := r.col().NewDoc()
	if _, err := ref.Set(ctx, m); err != nil {
		return memdom.Member{}, err
	}

	return m, nil
}

func (r *MemberRepositoryFS) SaveByDocID(
	ctx context.Context,
	docID string,
	m memdom.Member,
	_ *memdom.SaveOptions,
) (memdom.Member, error) {
	if r.Client == nil {
		return memdom.Member{}, errors.New("firestore client is nil")
	}

	docID = strings.TrimSpace(docID)
	if docID == "" {
		return memdom.Member{}, memdom.ErrNotFound
	}

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = &now

	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)

	if _, err := r.col().Doc(docID).Set(ctx, m); err != nil {
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

// ========================
// Helper: decode with legacy string timestamps support
// ========================

func readMemberSnapshot(doc *firestore.DocumentSnapshot) (memdom.Member, error) {
	var m memdom.Member
	if err := doc.DataTo(&m); err == nil {
		return m, nil
	}

	data := doc.Data()

	asString := func(v any) string {
		if s, ok := v.(string); ok {
			return s
		}
		return ""
	}

	asStringPtr := func(v any) *string {
		if s, ok := v.(string); ok && s != "" {
			ss := s
			return &ss
		}
		return nil
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
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}

		return out
	}

	asTimeValue := func(v any) (time.Time, bool, error) {
		switch t := v.(type) {
		case time.Time:
			return t.UTC(), true, nil
		case string:
			if t == "" {
				return time.Time{}, false, nil
			}

			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				return parsed.UTC(), true, nil
			}

			if parsed, err := time.Parse("2006-01-02 15:04:05Z07:00", t); err == nil {
				return parsed.UTC(), true, nil
			}

			return time.Time{}, false, fmt.Errorf("invalid time string: %q", t)
		default:
			return time.Time{}, false, nil
		}
	}

	m = memdom.Member{
		UID:            asString(data["uid"]),
		FirstName:      asString(data["firstName"]),
		LastName:       asString(data["lastName"]),
		FirstNameKana:  asString(data["firstNameKana"]),
		LastNameKana:   asString(data["lastNameKana"]),
		Email:          asString(data["email"]),
		Permissions:    asStringSlice(data["permissions"]),
		AssignedBrands: asStringSlice(data["assignedBrands"]),
		CompanyID:      asString(data["companyId"]),
		Status:         asString(data["status"]),
		UpdatedBy:      asStringPtr(data["updatedBy"]),
		DeletedBy:      asStringPtr(data["deletedBy"]),
	}

	if createdAt, ok, err := asTimeValue(data["createdAt"]); err != nil {
		return memdom.Member{}, err
	} else if ok {
		m.CreatedAt = createdAt
	}

	if updatedAt, ok, err := asTimeValue(data["updatedAt"]); err != nil {
		return memdom.Member{}, err
	} else if ok {
		m.UpdatedAt = &updatedAt
	}

	if deletedAt, ok, err := asTimeValue(data["deletedAt"]); err != nil {
		return memdom.Member{}, err
	} else if ok {
		m.DeletedAt = &deletedAt
	}

	return m, nil
}

func dedupStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))

	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		out = append(out, v)
	}

	return out
}
