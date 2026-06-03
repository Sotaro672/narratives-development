// backend/internal/adapters/out/firestore/member_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
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

// Compile-time check.
var _ memdom.Repository = (*MemberRepositoryFS)(nil)

// ========================
// Queries
// ========================

func (r *MemberRepositoryFS) GetByID(ctx context.Context, id string) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
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

func (r *MemberRepositoryFS) GetByUID(ctx context.Context, uid string) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}

	if uid == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}

	q := r.col().Where("uid", "==", uid).Limit(1)
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

// GetCompanyIDByFirebaseUID is an adapter extension used by auth middleware/usecase.
//
// NOTE:
// This method is intentionally not part of member.Repository.
// The domain repository port is kept to GetByID / GetByUID / ListByCompanyID.
func (r *MemberRepositoryFS) GetCompanyIDByFirebaseUID(ctx context.Context, uid string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	rec, err := r.GetByUID(ctx, uid)
	if err != nil {
		return "", err
	}

	companyID := rec.Member.CompanyID
	if companyID == "" {
		return "", memdom.ErrNotFound
	}

	return companyID, nil
}

// ========================
// List
// ========================

func (r *MemberRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
	_ memdom.Filter,
	p memdom.Page,
) (memdom.RecordPageResult, error) {
	if r.Client == nil {
		return memdom.RecordPageResult{}, errors.New("firestore client is nil")
	}

	if companyID == "" {
		return memdom.RecordPageResult{}, errors.New("member: companyID is empty")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(p.Number, p.PerPage, 50, 200)

	q := r.col().Query.
		Where("companyId", "==", companyID).
		OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	all := make([]memdom.Record, 0)

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

// ========================
// Mutations
// ========================

func (r *MemberRepositoryFS) Create(ctx context.Context, m memdom.Member) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}

	ref := r.col().NewDoc()

	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = &now

	m.Permissions = dedupStrings(m.Permissions)
	m.AssignedBrands = dedupStrings(m.AssignedBrands)

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

func (r *MemberRepositoryFS) Update(ctx context.Context, id string, patch memdom.MemberPatch) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}

	rec, err := r.GetByID(ctx, id)
	if err != nil {
		return memdom.Record{}, err
	}

	m := rec.Member

	if patch.UID != nil {
		m.UID = *patch.UID
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
		m.Email = *patch.Email
	}
	if patch.Permissions != nil {
		m.Permissions = dedupStrings(*patch.Permissions)
	}
	if patch.AssignedBrands != nil {
		m.AssignedBrands = dedupStrings(*patch.AssignedBrands)
	}
	if patch.CompanyID != nil {
		m.CompanyID = *patch.CompanyID
	}
	if patch.Status != nil {
		m.Status = *patch.Status
	}
	if patch.CreatedAt != nil {
		m.CreatedAt = *patch.CreatedAt
	}
	if patch.UpdatedBy != nil {
		updatedBy := *patch.UpdatedBy
		if updatedBy == "" {
			return memdom.Record{}, memdom.ErrInvalidUpdatedBy
		}
		m.UpdatedBy = &updatedBy
	}

	now := time.Now().UTC()
	if patch.UpdatedAt != nil {
		now = patch.UpdatedAt.UTC()
	}
	m.UpdatedAt = &now

	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}

	if _, err := r.col().Doc(rec.DocID).Set(ctx, m); err != nil {
		return memdom.Record{}, err
	}

	return memdom.Record{
		DocID:  rec.DocID,
		Member: m,
	}, nil
}

func (r *MemberRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if id == "" {
		return memdom.ErrNotFound
	}

	ref := r.col().Doc(id)

	if _, err := ref.Delete(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return memdom.ErrNotFound
		}
		return err
	}

	return nil
}

// ========================
// Helper: decode
// ========================

func readMemberSnapshot(doc *firestore.DocumentSnapshot) (memdom.Member, error) {
	var m memdom.Member
	if err := doc.DataTo(&m); err != nil {
		return memdom.Member{}, err
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
