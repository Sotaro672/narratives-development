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

	id = strings.TrimSpace(id)
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

// GetCompanyIDByFirebaseUID is an adapter extension used by auth middleware/usecase.
//
// NOTE:
// This method is intentionally not part of member.Repository.
// The domain repository port is kept to GetByID / ListByCompanyID.
func (r *MemberRepositoryFS) GetCompanyIDByFirebaseUID(ctx context.Context, uid string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", memdom.ErrNotFound
	}

	q := r.col().Where("uid", "==", uid).Limit(1)
	it := q.Documents(ctx)
	defer it.Stop()

	doc, err := it.Next()
	if err == iterator.Done {
		return "", memdom.ErrNotFound
	}
	if err != nil {
		return "", err
	}

	data := doc.Data()

	if v, ok := data["companyId"].(string); ok {
		companyID := strings.TrimSpace(v)
		if companyID != "" {
			return companyID, nil
		}
	}

	m, err := readMemberSnapshot(doc)
	if err != nil {
		return "", err
	}

	companyID := strings.TrimSpace(m.CompanyID)
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
	f memdom.Filter,
	p memdom.Page,
) (memdom.RecordPageResult, error) {
	if r.Client == nil {
		return memdom.RecordPageResult{}, errors.New("firestore client is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return memdom.RecordPageResult{}, errors.New("member: companyID is empty")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(p.Number, p.PerPage, 50, 200)

	q := r.col().Query.
		Where("companyId", "==", companyID)

	if f.UID != "" {
		q = q.Where("uid", "==", strings.TrimSpace(f.UID))
	}

	if f.Status != "" {
		q = q.Where("status", "==", strings.TrimSpace(f.Status))
	}

	// Firestore 側では companyId / uid / status の確定条件だけを使う。
	// SearchQuery / BrandIDs / Permissions / date range は読み出し後にメモリ上で絞り込む。
	q = q.OrderBy("updatedAt", firestore.Desc).
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

		rec := memdom.Record{
			DocID:  doc.Ref.ID,
			Member: m,
		}

		if !matchesMemberFilter(rec, f) {
			continue
		}

		all = append(all, rec)
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

	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)
	m.FirstName = strings.TrimSpace(m.FirstName)
	m.LastName = strings.TrimSpace(m.LastName)
	m.FirstNameKana = strings.TrimSpace(m.FirstNameKana)
	m.LastNameKana = strings.TrimSpace(m.LastNameKana)
	m.CompanyID = strings.TrimSpace(m.CompanyID)
	m.Status = strings.TrimSpace(m.Status)
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

	id = strings.TrimSpace(id)
	if id == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}

	rec, err := r.GetByID(ctx, id)
	if err != nil {
		return memdom.Record{}, err
	}

	m := rec.Member

	if patch.UID != nil {
		m.UID = strings.TrimSpace(*patch.UID)
	}
	if patch.FirstName != nil {
		m.FirstName = strings.TrimSpace(*patch.FirstName)
	}
	if patch.LastName != nil {
		m.LastName = strings.TrimSpace(*patch.LastName)
	}
	if patch.FirstNameKana != nil {
		m.FirstNameKana = strings.TrimSpace(*patch.FirstNameKana)
	}
	if patch.LastNameKana != nil {
		m.LastNameKana = strings.TrimSpace(*patch.LastNameKana)
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
	if patch.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(*patch.UpdatedBy)
		if updatedBy == "" {
			return memdom.Record{}, memdom.ErrInvalidUpdatedBy
		}
		m.UpdatedBy = &updatedBy
	}
	if patch.DeletedAt != nil {
		m.DeletedAt = patch.DeletedAt
	}
	if patch.DeletedBy != nil {
		deletedBy := strings.TrimSpace(*patch.DeletedBy)
		if deletedBy == "" {
			return memdom.Record{}, memdom.ErrInvalidDeletedBy
		}
		m.DeletedBy = &deletedBy
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
// Helper: filters
// ========================

func matchesMemberFilter(rec memdom.Record, f memdom.Filter) bool {
	m := rec.Member

	if len(f.BrandIDs) > 0 && !hasAnyString(m.AssignedBrands, f.BrandIDs) {
		return false
	}

	if len(f.Permissions) > 0 && !hasAllStrings(m.Permissions, f.Permissions) {
		return false
	}

	if f.CreatedFrom != nil && m.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}

	if f.CreatedTo != nil && m.CreatedAt.After(f.CreatedTo.UTC()) {
		return false
	}

	if f.UpdatedFrom != nil {
		if m.UpdatedAt == nil || m.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
			return false
		}
	}

	if f.UpdatedTo != nil {
		if m.UpdatedAt == nil || m.UpdatedAt.After(f.UpdatedTo.UTC()) {
			return false
		}
	}

	query := strings.ToLower(strings.TrimSpace(f.SearchQuery))
	if query != "" {
		targets := []string{
			rec.DocID,
			m.UID,
			m.FirstName,
			m.LastName,
			m.FirstNameKana,
			m.LastNameKana,
			m.Email,
			memdom.FormatLastFirst(m.LastName, m.FirstName),
		}

		found := false
		for _, target := range targets {
			if strings.Contains(strings.ToLower(target), query) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func hasAnyString(values []string, candidates []string) bool {
	if len(values) == 0 || len(candidates) == 0 {
		return false
	}

	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		set[v] = struct{}{}
	}

	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := set[c]; ok {
			return true
		}
	}

	return false
}

func hasAllStrings(values []string, required []string) bool {
	if len(required) == 0 {
		return true
	}

	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		set[v] = struct{}{}
	}

	for _, r := range required {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if _, ok := set[r]; !ok {
			return false
		}
	}

	return true
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
