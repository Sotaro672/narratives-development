// backend/internal/domain/member/invitation_repository_port.go
package member

import (
	"context"
	"errors"
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
