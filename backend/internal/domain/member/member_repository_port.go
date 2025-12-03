// backend/internal/domain/member/repository_port.go
package member

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Filter defines list conditions for members.
// NOTE: Field names/semantics align with entity.Member (camelCase in JSON/Firestore).
type Filter struct {
	// Free-text query for name/kana/email, etc.
	SearchQuery string

	// Brand filters (alias supported for backward compatibility).
	BrandIDs []string // preferred
	Brands   []string // legacy alias of BrandIDs

	// Company scope & status
	CompanyID string // owning company to scope results
	Status    string // "", "active", "inactive"

	// Ranges
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	// Permission names (AND)
	Permissions []string
}

// Common aliases
type Page = common.Page
type PageResult = common.PageResult[Member]
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Member]
type SaveOptions = common.SaveOptions

// Repository is the persistence port for the Member aggregate.
type Repository interface {
	// Common CRUD/List
	common.RepositoryCRUD[Member, MemberPatch]
	common.RepositoryList[Member, Filter]

	// Additional requirements
	ListByCursor(ctx context.Context, filter Filter, cpage CursorPage) (CursorPageResult, error)
	GetByID(ctx context.Context, id string) (Member, error)
	GetByEmail(ctx context.Context, email string) (Member, error)

	// ★ Firebase UID から Member を取得するメソッド
	//   今回の実装では「members の DocumentID = Firebase UID」という前提で
	//   GetByID を呼び出すラッパーにしています。
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (Member, error)

	Exists(ctx context.Context, id string) (bool, error)
	Save(ctx context.Context, m Member, opts *SaveOptions) (Member, error)
	Reset(ctx context.Context) error
}

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
