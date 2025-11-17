// backend\internal\adapters\out\firestore\invitation_repository_fs.go
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

	itdom "narratives/internal/domain/member" // InvitationToken / InvitationTokenRepository がここにある想定
)

// InvitationTokenRepositoryFS is a Firestore-based implementation of
// itdom.InvitationTokenRepository.
// Uses the "invitationTokens" collection.
//
// ドキュメント構造の想定：
// - コレクション: "invitationTokens"
// - ドキュメントID: token (INV_xxx など)
// - フィールド:
//   - token        : string （任意。docID と重複してもよい）
//   - memberId     : string
//   - companyId    : string (任意)
//   - assignedBrands : []string (任意)
//   - permissions  : []string (任意)
//   - expiresAt    : timestamp/string (任意)
//   - usedAt       : timestamp/string (任意)
//   - createdAt    : timestamp/string
type InvitationTokenRepositoryFS struct {
	Client *firestore.Client
}

func NewInvitationTokenRepositoryFS(client *firestore.Client) *InvitationTokenRepositoryFS {
	return &InvitationTokenRepositoryFS{Client: client}
}

func (r *InvitationTokenRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("invitationTokens")
}

// Compile-time check（domain 側にインターフェースがある前提）
// var _ itdom.InvitationTokenRepository = (*InvitationTokenRepositoryFS)(nil)

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
	// UpdatedAt を InvitationToken に持たせるかは domain 側の定義次第だが、
	// あればここで更新する運用にできる。
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

// ========================
// Helper: decode InvitationToken snapshot
// ========================

func readInvitationTokenSnapshot(doc *firestore.DocumentSnapshot) (itdom.InvitationToken, error) {
	// まずはそのまま構造体にマッピングを試みる
	var t itdom.InvitationToken
	if err := doc.DataTo(&t); err == nil {
		if strings.TrimSpace(t.Token) == "" {
			t.Token = doc.Ref.ID
		}
		return t, nil
	}

	// フォールバック: map から手動変換
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
		switch tt := v.(type) {
		case time.Time:
			tu := tt.UTC()
			return &tu, nil
		case *time.Time:
			if tt == nil {
				return nil, nil
			}
			tu := tt.UTC()
			return &tu, nil
		case string:
			s := strings.TrimSpace(tt)
			if s == "" {
				return nil, nil
			}
			if parsed, err := time.Parse(time.RFC3339, s); err == nil {
				tu := parsed.UTC()
				return &tu, nil
			}
			if parsed, err := time.Parse("2006-01-02 15:04:05Z07:00", s); err == nil {
				tu := parsed.UTC()
				return &tu, nil
			}
			return nil, fmt.Errorf("invalid time string: %q", s)
		default:
			return nil, nil
		}
	}

	t = itdom.InvitationToken{
		Token:            asString(data["token"]),
		MemberID:         asString(data["memberId"]),
		CompanyID:        asString(data["companyId"]),
		AssignedBrandIDs: asStringSlice(data["assignedBrands"]),
		Permissions:      asStringSlice(data["permissions"]),
	}

	// createdAt
	if v, err := asTimePtr(data["createdAt"]); err == nil && v != nil {
		t.CreatedAt = *v
	}

	// expiresAt / usedAt はオプション
	if v, _ := asTimePtr(data["expiresAt"]); v != nil {
		t.ExpiresAt = v
	}
	if v, _ := asTimePtr(data["usedAt"]); v != nil {
		t.UsedAt = v
	}

	// UpdatedAt があればセット（domain 定義に依存）
	if v, _ := asTimePtr(data["updatedAt"]); v != nil {
		t.UpdatedAt = v
	}

	// token が空なら docID を使う
	if strings.TrimSpace(t.Token) == "" {
		t.Token = doc.Ref.ID
	}

	return t, nil
}
