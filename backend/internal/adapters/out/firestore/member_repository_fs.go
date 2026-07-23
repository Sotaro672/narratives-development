// backend/internal/adapters/out/firestore/member_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	fscommon "narratives/internal/adapters/out/firestore/common"
	memdom "narratives/internal/domain/member"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	membersCollectionName    = "members"
	memberUIDsCollectionName = "memberUIDs"
)

// MemberRepositoryFS is a Firestore-based implementation of member.Repository.
// Uses the "members" collection.
type MemberRepositoryFS struct {
	Client *firestore.Client
}
type memberUIDDocument struct {
	MemberID  string    `firestore:"memberId"`
	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

func NewMemberRepositoryFS(client *firestore.Client) *MemberRepositoryFS {
	return &MemberRepositoryFS{
		Client: client,
	}
}
func (r *MemberRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(membersCollectionName)
}
func (r *MemberRepositoryFS) memberUIDsCol() *firestore.CollectionRef {
	return r.Client.Collection(memberUIDsCollectionName)
}

// Compile-time check.
var _ memdom.Repository = (*MemberRepositoryFS)(nil)

// ========================
// Queries
// ========================
func (r *MemberRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (memdom.Record, error) {
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
		return memdom.Record{}, fmt.Errorf(
			"get member %q: %w",
			id,
			err,
		)
	}
	m, err := readMemberSnapshot(doc)
	if err != nil {
		return memdom.Record{}, fmt.Errorf(
			"decode member %q: %w",
			id,
			err,
		)
	}
	return memdom.Record{
		DocID:  doc.Ref.ID,
		Member: m,
	}, nil
}
func (r *MemberRepositoryFS) GetByUID(
	ctx context.Context,
	uid string,
) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}
	if uid == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}
	uidRef := r.memberUIDsCol().Doc(uid)
	uidDoc, err := uidRef.Get(ctx)
	if err == nil {
		mapping, decodeErr := readMemberUIDSnapshot(uidDoc)
		if decodeErr != nil {
			return memdom.Record{}, decodeErr
		}
		rec, getErr := r.GetByID(ctx, mapping.MemberID)
		if getErr == nil {
			return rec, nil
		}
		if !errors.Is(getErr, memdom.ErrNotFound) {
			return memdom.Record{}, getErr
		}
	} else if status.Code(err) != codes.NotFound {
		return memdom.Record{}, fmt.Errorf(
			"get member UID mapping %q: %w",
			uid,
			err,
		)
	}
	q := r.col().
		Where("uid", "==", uid).
		Limit(2)
	it := q.Documents(ctx)
	defer it.Stop()
	records := make([]memdom.Record, 0, 2)
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return memdom.Record{}, fmt.Errorf(
				"query member by UID %q: %w",
				uid,
				err,
			)
		}
		m, err := readMemberSnapshot(doc)
		if err != nil {
			return memdom.Record{}, fmt.Errorf(
				"decode member %q found by UID: %w",
				doc.Ref.ID,
				err,
			)
		}
		records = append(records, memdom.Record{
			DocID:  doc.Ref.ID,
			Member: m,
		})
	}
	switch len(records) {
	case 0:
		return memdom.Record{}, memdom.ErrNotFound
	case 1:
		return records[0], nil
	default:
		return memdom.Record{}, memdom.ErrConflict
	}
}

// GetCompanyIDByFirebaseUID is an adapter extension used by auth middleware/usecase.
//
// NOTE:
// This method is intentionally not part of member.Repository.
// The domain repository port is kept to GetByID / GetByUID / ListByCompanyID.
func (r *MemberRepositoryFS) GetCompanyIDByFirebaseUID(
	ctx context.Context,
	uid string,
) (string, error) {
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
		return memdom.RecordPageResult{}, errors.New(
			"member: companyID is empty",
		)
	}
	pageNum, perPage, offset := fscommon.NormalizePage(
		p.Number,
		p.PerPage,
		50,
		200,
	)
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
func (r *MemberRepositoryFS) Create(
	ctx context.Context,
	m memdom.Member,
) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}
	now := time.Now().UTC()
	m = normalizeMemberForCreate(m, now)
	memberRef := r.col().NewDoc()
	if m.UID == "" {
		if _, err := memberRef.Create(ctx, m); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return memdom.Record{}, memdom.ErrConflict
			}
			return memdom.Record{}, fmt.Errorf(
				"create member %q: %w",
				memberRef.ID,
				err,
			)
		}
		return memdom.Record{
			DocID:  memberRef.ID,
			Member: m,
		}, nil
	}
	uidRef := r.memberUIDsCol().Doc(m.UID)
	err := r.Client.RunTransaction(
		ctx,
		func(ctx context.Context, tx *firestore.Transaction) error {
			_, exists, err := getMemberUIDMappingInTransaction(
				tx,
				uidRef,
			)
			if err != nil {
				return err
			}
			if exists {
				return memdom.ErrConflict
			}
			memberIDs, err := findMemberIDsByUIDInTransaction(
				tx,
				r.col(),
				m.UID,
				1,
			)
			if err != nil {
				return err
			}
			if len(memberIDs) > 0 {
				return memdom.ErrConflict
			}
			if err := tx.Create(memberRef, m); err != nil {
				return fmt.Errorf(
					"create member %q in transaction: %w",
					memberRef.ID,
					err,
				)
			}
			if err := tx.Create(uidRef, memberUIDDocument{
				MemberID:  memberRef.ID,
				CreatedAt: now,
				UpdatedAt: now,
			}); err != nil {
				return fmt.Errorf(
					"create member UID mapping %q: %w",
					m.UID,
					err,
				)
			}
			return nil
		},
	)
	if err != nil {
		if errors.Is(err, memdom.ErrConflict) ||
			status.Code(err) == codes.AlreadyExists {
			return memdom.Record{}, memdom.ErrConflict
		}
		return memdom.Record{}, fmt.Errorf(
			"create member transaction: %w",
			err,
		)
	}
	return memdom.Record{
		DocID:  memberRef.ID,
		Member: m,
	}, nil
}
func (r *MemberRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch memdom.MemberPatch,
) (memdom.Record, error) {
	if r.Client == nil {
		return memdom.Record{}, errors.New("firestore client is nil")
	}
	if id == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}
	memberRef := r.col().Doc(id)
	var updatedRecord memdom.Record
	err := r.Client.RunTransaction(
		ctx,
		func(ctx context.Context, tx *firestore.Transaction) error {
			memberDoc, err := tx.Get(memberRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return memdom.ErrNotFound
				}
				return fmt.Errorf(
					"get member %q in transaction: %w",
					id,
					err,
				)
			}
			current, err := readMemberSnapshot(memberDoc)
			if err != nil {
				return fmt.Errorf(
					"decode member %q in transaction: %w",
					id,
					err,
				)
			}
			now := time.Now().UTC()
			updated, err := applyMemberPatch(
				current,
				patch,
				now,
			)
			if err != nil {
				return err
			}
			oldUID := current.UID
			newUID := updated.UID
			var (
				newUIDRef     *firestore.DocumentRef
				newUIDMapping memberUIDDocument
				newUIDExists  bool
				deleteOldUID  bool
				oldUIDRef     *firestore.DocumentRef
			)
			if newUID != "" {
				newUIDRef = r.memberUIDsCol().Doc(newUID)
				newUIDMapping, newUIDExists, err =
					getMemberUIDMappingInTransaction(
						tx,
						newUIDRef,
					)
				if err != nil {
					return err
				}
				if newUIDExists &&
					newUIDMapping.MemberID != id {
					return memdom.ErrConflict
				}
				memberIDs, err := findMemberIDsByUIDInTransaction(
					tx,
					r.col(),
					newUID,
					2,
				)
				if err != nil {
					return err
				}
				for _, memberID := range memberIDs {
					if memberID != id {
						return memdom.ErrConflict
					}
				}
			}
			if oldUID != "" && oldUID != newUID {
				oldUIDRef = r.memberUIDsCol().Doc(oldUID)
				oldUIDMapping, oldUIDExists, err :=
					getMemberUIDMappingInTransaction(
						tx,
						oldUIDRef,
					)
				if err != nil {
					return err
				}
				deleteOldUID = oldUIDExists &&
					oldUIDMapping.MemberID == id
			}
			if err := tx.Set(memberRef, updated); err != nil {
				return fmt.Errorf(
					"update member %q in transaction: %w",
					id,
					err,
				)
			}
			if deleteOldUID {
				if err := tx.Delete(oldUIDRef); err != nil {
					return fmt.Errorf(
						"delete old member UID mapping %q: %w",
						oldUID,
						err,
					)
				}
			}
			if newUID != "" {
				createdAt := now
				if newUIDExists &&
					!newUIDMapping.CreatedAt.IsZero() {
					createdAt = newUIDMapping.CreatedAt.UTC()
				}
				mapping := memberUIDDocument{
					MemberID:  id,
					CreatedAt: createdAt,
					UpdatedAt: now,
				}
				if newUIDExists {
					if err := tx.Set(newUIDRef, mapping); err != nil {
						return fmt.Errorf(
							"update member UID mapping %q: %w",
							newUID,
							err,
						)
					}
				} else {
					if err := tx.Create(newUIDRef, mapping); err != nil {
						return fmt.Errorf(
							"create member UID mapping %q: %w",
							newUID,
							err,
						)
					}
				}
			}
			updatedRecord = memdom.Record{
				DocID:  id,
				Member: updated,
			}
			return nil
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, memdom.ErrNotFound):
			return memdom.Record{}, memdom.ErrNotFound
		case errors.Is(err, memdom.ErrConflict),
			status.Code(err) == codes.AlreadyExists:
			return memdom.Record{}, memdom.ErrConflict
		default:
			return memdom.Record{}, fmt.Errorf(
				"update member transaction: %w",
				err,
			)
		}
	}
	return updatedRecord, nil
}
func (r *MemberRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if id == "" {
		return memdom.ErrNotFound
	}
	memberRef := r.col().Doc(id)
	err := r.Client.RunTransaction(
		ctx,
		func(ctx context.Context, tx *firestore.Transaction) error {
			memberDoc, err := tx.Get(memberRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return memdom.ErrNotFound
				}
				return fmt.Errorf(
					"get member %q before delete: %w",
					id,
					err,
				)
			}
			member, err := readMemberSnapshot(memberDoc)
			if err != nil {
				return fmt.Errorf(
					"decode member %q before delete: %w",
					id,
					err,
				)
			}
			uid := member.UID
			var (
				uidRef           *firestore.DocumentRef
				deleteUIDMapping bool
			)
			if uid != "" {
				uidRef = r.memberUIDsCol().Doc(uid)
				mapping, exists, err :=
					getMemberUIDMappingInTransaction(
						tx,
						uidRef,
					)
				if err != nil {
					return err
				}
				deleteUIDMapping = exists &&
					mapping.MemberID == id
			}
			if err := tx.Delete(memberRef); err != nil {
				return fmt.Errorf(
					"delete member %q: %w",
					id,
					err,
				)
			}
			if deleteUIDMapping {
				if err := tx.Delete(uidRef); err != nil {
					return fmt.Errorf(
						"delete member UID mapping %q: %w",
						uid,
						err,
					)
				}
			}
			return nil
		},
	)
	if err != nil {
		if errors.Is(err, memdom.ErrNotFound) {
			return memdom.ErrNotFound
		}
		return fmt.Errorf(
			"delete member transaction: %w",
			err,
		)
	}
	return nil
}

// ========================
// Helpers
// ========================
func applyMemberPatch(
	m memdom.Member,
	patch memdom.MemberPatch,
	now time.Time,
) (memdom.Member, error) {
	now = now.UTC()
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
		m.Email = normalizeMemberEmail(*patch.Email)
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
		m.CreatedAt = patch.CreatedAt.UTC()
	}
	if patch.UpdatedBy != nil {
		updatedBy := *patch.UpdatedBy
		if updatedBy == "" {
			return memdom.Member{}, memdom.ErrInvalidUpdatedBy
		}
		m.UpdatedBy = &updatedBy
	}
	if patch.UpdatedAt != nil {
		now = patch.UpdatedAt.UTC()
	}
	m = normalizeMemberValues(m)
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	} else {
		m.CreatedAt = m.CreatedAt.UTC()
	}
	m.UpdatedAt = &now
	return m, nil
}
func normalizeMemberForCreate(
	m memdom.Member,
	now time.Time,
) memdom.Member {
	now = now.UTC()
	m = normalizeMemberValues(m)
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	} else {
		m.CreatedAt = m.CreatedAt.UTC()
	}
	m.UpdatedAt = &now
	return m
}
func normalizeMemberValues(m memdom.Member) memdom.Member {
	m.Email = normalizeMemberEmail(m.Email)
	m.Permissions = dedupStrings(m.Permissions)
	m.AssignedBrands = dedupStrings(m.AssignedBrands)
	return m
}
func normalizeMemberEmail(email string) string {
	return strings.ToLower(email)
}
func readMemberSnapshot(
	doc *firestore.DocumentSnapshot,
) (memdom.Member, error) {
	if doc == nil {
		return memdom.Member{}, errors.New(
			"member document snapshot is nil",
		)
	}
	var m memdom.Member
	if err := doc.DataTo(&m); err != nil {
		return memdom.Member{}, err
	}
	return m, nil
}
func readMemberUIDSnapshot(
	doc *firestore.DocumentSnapshot,
) (memberUIDDocument, error) {
	if doc == nil {
		return memberUIDDocument{}, errors.New(
			"member UID document snapshot is nil",
		)
	}
	var stored memberUIDDocument
	if err := doc.DataTo(&stored); err != nil {
		return memberUIDDocument{}, fmt.Errorf(
			"decode member UID mapping %q: %w",
			doc.Ref.ID,
			err,
		)
	}
	if stored.MemberID == "" {
		return memberUIDDocument{}, fmt.Errorf(
			"member UID mapping %q has empty memberId",
			doc.Ref.ID,
		)
	}
	if !stored.CreatedAt.IsZero() {
		stored.CreatedAt = stored.CreatedAt.UTC()
	}
	if !stored.UpdatedAt.IsZero() {
		stored.UpdatedAt = stored.UpdatedAt.UTC()
	}
	return stored, nil
}
func getMemberUIDMappingInTransaction(
	tx *firestore.Transaction,
	ref *firestore.DocumentRef,
) (memberUIDDocument, bool, error) {
	doc, err := tx.Get(ref)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return memberUIDDocument{}, false, nil
		}
		return memberUIDDocument{}, false, fmt.Errorf(
			"get member UID mapping %q in transaction: %w",
			ref.ID,
			err,
		)
	}
	mapping, err := readMemberUIDSnapshot(doc)
	if err != nil {
		return memberUIDDocument{}, false, err
	}
	return mapping, true, nil
}
func findMemberIDsByUIDInTransaction(
	tx *firestore.Transaction,
	members *firestore.CollectionRef,
	uid string,
	limit int,
) ([]string, error) {
	if uid == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 2
	}
	q := members.
		Where("uid", "==", uid).
		Limit(limit)
	it := tx.Documents(q)
	defer it.Stop()
	memberIDs := make([]string, 0, limit)
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"query members by UID %q in transaction: %w",
				uid,
				err,
			)
		}
		memberIDs = append(memberIDs, doc.Ref.ID)
	}
	return memberIDs, nil
}
func dedupStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, value := range in {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
