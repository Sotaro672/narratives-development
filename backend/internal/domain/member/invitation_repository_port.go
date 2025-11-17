// backend/internal/domain/member/invitation_repository_port.go
package member

import (
	"context"
	"errors"
	"time"
)

// ============================================================
// Domain errors for InvitationToken
// ============================================================

var (
	// 招待トークンが見つからない場合
	ErrInvitationTokenNotFound = errors.New("invitation: token not found")
	// 必要なら衝突エラーなども今後追加可能
	// ErrInvitationTokenConflict = errors.New("invitation: token conflict")
)

// ============================================================
// InvitationToken エンティティ
// - Firestore "invitationTokens" コレクションに対応
// ============================================================

type InvitationToken struct {
	// Firestore ドキュメントID としても使われる招待トークン文字列（例: "INV_xxx"）
	Token string

	// 対象メンバー
	MemberID string

	// 会社・ブランド・権限など、招待時点の情報スナップショット
	CompanyID        string
	AssignedBrandIDs []string
	Permissions      []string

	// ライフサイクル
	CreatedAt time.Time  // 作成日時（必須）
	ExpiresAt *time.Time // 有効期限（任意）
	UsedAt    *time.Time // 使用日時（任意）
	UpdatedAt *time.Time // 更新日時（任意）
}

// ============================================================
// InvitationToken 用 Repository ポート（ヘキサゴナルの out ポート）
// ============================================================

type InvitationTokenRepository interface {
	// トークン文字列から InvitationToken を取得
	// 見つからない場合は ErrInvitationTokenNotFound を返す
	FindByToken(ctx context.Context, token string) (InvitationToken, error)

	// InvitationToken を保存（作成/更新）
	// Token が空の場合は実装側で新規IDを採番してもよい
	Save(ctx context.Context, t InvitationToken) (InvitationToken, error)

	// トークン文字列を指定して削除
	// 見つからない場合は ErrInvitationTokenNotFound を返すことを推奨
	Delete(ctx context.Context, token string) error
}
