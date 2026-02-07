// backend\internal\application\usecase\avatar\ports.go
package avatar

import (
	"context"

	avatardom "narratives/internal/domain/avatar"
	avataricon "narratives/internal/domain/avatarIcon"
	avatarstate "narratives/internal/domain/avatarState"
	cartdom "narratives/internal/domain/cart"
	walletdom "narratives/internal/domain/wallet"
)

// AvatarRepo は Avatar 本体の永続化ポートです。
// Firestore 実装（avatar_repository_fs.go）が Create/Update/Delete を提供する前提で揃えます。
type AvatarRepo interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
	Create(ctx context.Context, a avatardom.Avatar) (avatardom.Avatar, error)
	Update(ctx context.Context, id string, patch avatardom.AvatarPatch) (avatardom.Avatar, error)
	Delete(ctx context.Context, id string) error
}

type AvatarStateRepo interface {
	GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error)
	// Upsert がない実装もあるため、必要時はアダプタ側でエラー返却可
	Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error)
}

type AvatarIconRepo interface {
	GetByAvatarID(ctx context.Context, avatarID string) ([]avataricon.AvatarIcon, error)
	// Repo 実装が Save(ctx, icon) 以外（例: Save(ctx, icon, opts)）の場合は
	// アダプタ側で opts=nil などに委譲してください。
	Save(ctx context.Context, ic avataricon.AvatarIcon, opts *avataricon.SaveOptions) (avataricon.AvatarIcon, error)
}

type AvatarIconObjectStoragePort interface {
	DeleteObjects(ctx context.Context, ops []avataricon.GCSDeleteOp) error

	// ✅ 画像が空でも avatarDocId/ の “入れ物” を作る（例: <avatarId>/.keep を作成）
	EnsurePrefix(ctx context.Context, bucket, prefix string) error
}

// ✅ Wallet 永続化ポート
// - wallets コレクションは docId=avatarId を期待値とする
type WalletRepo interface {
	Save(ctx context.Context, avatarID string, w walletdom.Wallet) error
}

// ✅ Cart 永続化ポート
// - carts コレクションは docId=avatarId を期待値とする
type CartRepo interface {
	Upsert(ctx context.Context, c *cartdom.Cart) error
	DeleteByAvatarID(ctx context.Context, avatarID string) error
}

// AvatarWalletService は Avatar 作成時に Solana wallet を開設するためのポートです。
type AvatarWalletService interface {
	OpenAvatarWallet(ctx context.Context, avatarID string) (avatardom.SolanaAvatarWallet, error)
}
