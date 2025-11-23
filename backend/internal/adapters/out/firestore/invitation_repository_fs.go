// backend/internal/adapters/out/firestore/invitation_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	itdom "narratives/internal/domain/member" // InvitationToken / InvitationInfo 定義元
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

	token = strings.TrimSpace(token)
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
	// docID を優先して Token にセット
	if strings.TrimSpace(it.Token) == "" {
		it.Token = doc.Ref.ID
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

	id := strings.TrimSpace(t.Token)
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
	// UpdatedAt フィールドを持っていれば更新
	if t.UpdatedAt != nil {
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

	token = strings.TrimSpace(token)
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

// ============================================================
// Application ポート usecase.InvitationTokenRepository 用の実装
// （ResolveXXX / CreateInvitationToken）
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
	memberID := strings.TrimSpace(it.MemberID)
	if memberID == "" {
		return "", itdom.ErrInvitationTokenNotFound
	}
	return memberID, nil
}

// ResolveInvitationInfoByToken は token → InvitationInfo
// （MemberID / Email / CompanyID / AssignedBrandIDs / Permissions）
// を解決して返します。
// usecase.InvitationTokenRepository インターフェースに対応。
// 戻り値は値型（memdom.InvitationInfo, error）です。
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

	// InvitationToken → InvitationInfo へ詰め替え
	info := itdom.InvitationInfo{
		MemberID:         strings.TrimSpace(it.MemberID),
		Email:            strings.TrimSpace(it.Email),
		CompanyID:        strings.TrimSpace(it.CompanyID),
		AssignedBrandIDs: it.AssignedBrandIDs,
		Permissions:      it.Permissions,
	}

	// MemberID が空の場合は NotFound 相当扱い
	if info.MemberID == "" {
		return itdom.InvitationInfo{}, itdom.ErrInvitationTokenNotFound
	}

	return info, nil
}

// CreateInvitationToken は InvitationInfo に紐づく新しい招待トークンを作成し、
// その token 文字列を返します。
// info には memberID / email / companyId / assignedBrands / permissions が含まれます。
func (r *InvitationTokenRepositoryFS) CreateInvitationToken(
	ctx context.Context,
	info itdom.InvitationInfo,
) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	memberID := strings.TrimSpace(info.MemberID)
	if memberID == "" {
		return "", fmt.Errorf("memberID is empty")
	}

	now := time.Now().UTC()
	t := itdom.InvitationToken{
		Token:            "", // 空なら Save 側で NewDoc() により採番される
		MemberID:         memberID,
		Email:            strings.TrimSpace(info.Email),
		CompanyID:        strings.TrimSpace(info.CompanyID),
		AssignedBrandIDs: info.AssignedBrandIDs,
		Permissions:      info.Permissions,
		CreatedAt:        now,
	}

	saved, err := r.Save(ctx, t)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(saved.Token), nil
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

	t := itdom.InvitationToken{
		Token:            doc.Ref.ID,
		MemberID:         asString(data["memberId"]),
		Email:            asString(data["email"]),
		CompanyID:        asString(data["companyId"]),
		AssignedBrandIDs: asStringSlice(data["assignedBrands"]),
		Permissions:      asStringSlice(data["permissions"]),
	}

	return t, nil
}
