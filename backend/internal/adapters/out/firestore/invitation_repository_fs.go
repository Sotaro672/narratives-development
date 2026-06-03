// backend/internal/adapters/out/firestore/invitation_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	itdom "narratives/internal/domain/invitation"
)

// InvitationTokenRepositoryFS is a Firestore-based implementation of
// invitation.Repository.
//
// Uses the "invitationTokens" collection.
//
// ドキュメント構造の想定：
// - コレクション: "invitationTokens"
// - ドキュメントID: token
// - フィールド:
//   - memberId       : string
//   - email          : string
//   - companyId      : string
//   - assignedBrands : []string
//   - permissions    : []string
//   - createdAt      : timestamp
//   - usedAt         : timestamp (optional)
//   - updatedAt      : timestamp (optional)
type InvitationTokenRepositoryFS struct {
	Client *firestore.Client
}

func NewInvitationTokenRepositoryFS(client *firestore.Client) *InvitationTokenRepositoryFS {
	return &InvitationTokenRepositoryFS{Client: client}
}

func (r *InvitationTokenRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("invitationTokens")
}

// FindByToken retrieves an invitation token document by token string.
// token は Firestore document ID として扱う。
func (r *InvitationTokenRepositoryFS) FindByToken(
	ctx context.Context,
	token string,
) (itdom.InvitationToken, error) {
	if r.Client == nil {
		return itdom.InvitationToken{}, errors.New("firestore client is nil")
	}

	if token == "" {
		return itdom.InvitationToken{}, itdom.ErrInvitationTokenNotFound
	}

	doc, err := r.col().Doc(token).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return itdom.InvitationToken{}, itdom.ErrInvitationTokenNotFound
		}
		return itdom.InvitationToken{}, err
	}

	it, err := readInvitationTokenSnapshot(doc)
	if err != nil {
		return itdom.InvitationToken{}, err
	}

	if it.Token == "" {
		it.Token = doc.Ref.ID
	}

	if it.UsedAt != nil && !it.UsedAt.IsZero() {
		return itdom.InvitationToken{}, itdom.ErrInvitationTokenNotFound
	}

	return it, nil
}

// Save creates or updates an invitation token document.
// Token が空なら新規 docID を発行し、Token が指定されていればその ID に保存する。
func (r *InvitationTokenRepositoryFS) Save(
	ctx context.Context,
	t itdom.InvitationToken,
) (itdom.InvitationToken, error) {
	if r.Client == nil {
		return itdom.InvitationToken{}, errors.New("firestore client is nil")
	}

	id := t.Token
	var ref *firestore.DocumentRef
	if id == "" {
		ref = r.col().NewDoc()
		id = ref.ID
		t.Token = id
	} else {
		ref = r.col().Doc(id)
	}

	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}

	t.UpdatedAt = &now

	data := map[string]any{
		"memberId":       t.MemberID,
		"email":          t.Email,
		"companyId":      t.CompanyID,
		"assignedBrands": t.AssignedBrandIDs,
		"permissions":    t.Permissions,
		"createdAt":      t.CreatedAt,
		"updatedAt":      now,
	}

	if t.UsedAt != nil && !t.UsedAt.IsZero() {
		data["usedAt"] = *t.UsedAt
	}

	if _, err := ref.Set(ctx, data); err != nil {
		return itdom.InvitationToken{}, err
	}

	return t, nil
}

// ConsumeInvitationToken marks the invitation token as used.
func (r *InvitationTokenRepositoryFS) ConsumeInvitationToken(
	ctx context.Context,
	token string,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if token == "" {
		return itdom.ErrInvitationTokenNotFound
	}

	ref := r.col().Doc(token)
	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return itdom.ErrInvitationTokenNotFound
		}
		return err
	}

	current, err := readInvitationTokenSnapshot(doc)
	if err != nil {
		return err
	}

	if current.UsedAt != nil && !current.UsedAt.IsZero() {
		return itdom.ErrInvitationTokenNotFound
	}

	now := time.Now().UTC()
	updates := []firestore.Update{
		{Path: "usedAt", Value: now},
		{Path: "updatedAt", Value: now},
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return itdom.ErrInvitationTokenNotFound
		}
		return err
	}

	return nil
}

// ResolveInvitationInfoByToken resolves token to InvitationInfo.
func (r *InvitationTokenRepositoryFS) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (itdom.InvitationInfo, error) {
	if r.Client == nil {
		return itdom.InvitationInfo{}, errors.New("firestore client is nil")
	}

	it, err := r.FindByToken(ctx, token)
	if err != nil {
		return itdom.InvitationInfo{}, err
	}

	info := itdom.InvitationInfo{
		MemberID:         it.MemberID,
		CompanyID:        it.CompanyID,
		AssignedBrandIDs: it.AssignedBrandIDs,
		Permissions:      it.Permissions,
		Email:            it.Email,
	}

	if info.MemberID == "" {
		return itdom.InvitationInfo{}, itdom.ErrInvitationTokenNotFound
	}

	return info, nil
}

// CreateInvitationToken creates a new invitation token and returns the token string.
func (r *InvitationTokenRepositoryFS) CreateInvitationToken(
	ctx context.Context,
	info itdom.InvitationInfo,
) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	memberID := info.MemberID
	if memberID == "" {
		return "", fmt.Errorf("memberID is empty")
	}

	now := time.Now().UTC()
	t := itdom.InvitationToken{
		Token:            "",
		MemberID:         memberID,
		CompanyID:        info.CompanyID,
		AssignedBrandIDs: info.AssignedBrandIDs,
		Permissions:      info.Permissions,
		Email:            info.Email,
		CreatedAt:        now,
		UsedAt:           nil,
		UpdatedAt:        &now,
	}

	saved, err := r.Save(ctx, t)
	if err != nil {
		return "", err
	}

	return saved.Token, nil
}

func readInvitationTokenSnapshot(doc *firestore.DocumentSnapshot) (itdom.InvitationToken, error) {
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
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}

		return out
	}

	asTimePtr := func(v any) *time.Time {
		switch t := v.(type) {
		case time.Time:
			tt := t.UTC()
			return &tt
		case *time.Time:
			if t == nil {
				return nil
			}
			tt := t.UTC()
			return &tt
		default:
			return nil
		}
	}

	createdAt := time.Time{}
	if v, ok := data["createdAt"]; ok {
		if t := asTimePtr(v); t != nil {
			createdAt = *t
		}
	}

	updatedAt := func() *time.Time {
		v, ok := data["updatedAt"]
		if !ok {
			return nil
		}
		return asTimePtr(v)
	}()

	usedAt := func() *time.Time {
		v, ok := data["usedAt"]
		if !ok {
			return nil
		}
		return asTimePtr(v)
	}()

	t := itdom.InvitationToken{
		Token:            doc.Ref.ID,
		MemberID:         asString(data["memberId"]),
		CompanyID:        asString(data["companyId"]),
		AssignedBrandIDs: asStringSlice(data["assignedBrands"]),
		Permissions:      asStringSlice(data["permissions"]),
		Email:            asString(data["email"]),
		CreatedAt:        createdAt,
		UsedAt:           usedAt,
		UpdatedAt:        updatedAt,
	}

	return t, nil
}
