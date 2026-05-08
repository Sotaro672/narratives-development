package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	itdom "narratives/internal/domain/member"
)

// InvitationTokenRepositoryFS is a Firestore-based implementation of
// InvitationTokenRepository.
// Uses the "invitationTokens" collection.
//
// ドキュメント構造の想定：
// - コレクション: "invitationTokens"
// - ドキュメントID: token (INV_xxx など)
// - フィールド:
//   - token          : string （任意。docID と重複してもよい）
//   - memberId       : string
//   - email          : string
//   - companyId      : string (任意)
//   - assignedBrands : []string (任意)
//   - permissions    : []string (任意)
//   - expiresAt      : timestamp/string (任意)
//   - usedAt         : timestamp/string (任意)
//   - createdAt      : timestamp/string
//   - updatedAt      : timestamp/string (任意)
type InvitationTokenRepositoryFS struct {
	Client *firestore.Client
}

func NewInvitationTokenRepositoryFS(client *firestore.Client) *InvitationTokenRepositoryFS {
	return &InvitationTokenRepositoryFS{Client: client}
}

func (r *InvitationTokenRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("invitationTokens")
}

// ============================================================
// Domain ポート itdom.InvitationTokenRepository 相当の実装
// ============================================================

// FindByToken retrieves an invitation token document by token string.
// token は基本的にドキュメントIDとして扱う想定です。
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

	// usedAt が入っている token は消費済みとして扱う
	if it.UsedAt != nil && !it.UsedAt.IsZero() {
		return itdom.InvitationToken{}, itdom.ErrInvitationTokenNotFound
	}

	return it, nil
}

// Save は InvitationToken を作成/更新します。
// - Token が空なら新規 docID を発行
// - Token が指定されていればそのIDのドキュメントに Set
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

	if t.UpdatedAt == nil {
		t.UpdatedAt = &now
	} else {
		*t.UpdatedAt = now
	}

	if _, err := ref.Set(ctx, t); err != nil {
		return itdom.InvitationToken{}, err
	}
	return t, nil
}

// Delete はトークン文字列を指定してドキュメントを削除します。
func (r *InvitationTokenRepositoryFS) Delete(ctx context.Context, token string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if token == "" {
		return itdom.ErrInvitationTokenNotFound
	}

	ref := r.col().Doc(token)
	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return itdom.ErrInvitationTokenNotFound
		}
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// ConsumeInvitationToken marks the invitation token as used.
// usecase.InvitationTokenRepository インターフェースに対応。
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

// ============================================================
// Application ポート usecase.InvitationTokenRepository 用の実装
// （ResolveXXX / CreateInvitationToken / ConsumeInvitationToken）
// ============================================================

// ResolveMemberIDByToken は token → memberID の解決を行います。
// ※ 既存コード互換用に残しています（不要であれば後で削除可）。
func (r *InvitationTokenRepositoryFS) ResolveMemberIDByToken(
	ctx context.Context,
	token string,
) (string, error) {
	it, err := r.FindByToken(ctx, token)
	if err != nil {
		return "", err
	}
	memberID := it.MemberID
	if memberID == "" {
		return "", itdom.ErrInvitationTokenNotFound
	}
	return memberID, nil
}

// ResolveInvitationInfoByToken は token → InvitationInfo
// （MemberID / Email / CompanyID / AssignedBrandIDs / Permissions）
// を解決して返します。
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
		Email:            it.Email,
		CompanyID:        it.CompanyID,
		AssignedBrandIDs: it.AssignedBrandIDs,
		Permissions:      it.Permissions,
	}

	if info.MemberID == "" {
		return itdom.InvitationInfo{}, itdom.ErrInvitationTokenNotFound
	}

	return info, nil
}

// CreateInvitationToken は InvitationInfo に紐づく新しい招待トークンを作成し、
// その token 文字列を返します。
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
		Email:            info.Email,
		CompanyID:        info.CompanyID,
		AssignedBrandIDs: info.AssignedBrandIDs,
		Permissions:      info.Permissions,
		CreatedAt:        now,
		UpdatedAt:        &now,
		UsedAt:           nil,
	}

	saved, err := r.Save(ctx, t)
	if err != nil {
		return "", err
	}
	return saved.Token, nil
}

// ========================
// Helper: decode InvitationToken snapshot
// ========================

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
		Email:            asString(data["email"]),
		CompanyID:        asString(data["companyId"]),
		AssignedBrandIDs: asStringSlice(data["assignedBrands"]),
		Permissions:      asStringSlice(data["permissions"]),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		UsedAt:           usedAt,
	}

	return t, nil
}
