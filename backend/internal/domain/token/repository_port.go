// backend/internal/domain/token/port.go
package token

import (
	"context"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Mint 権限ウォレット用ポート
// ============================================================

// MintAuthorityWalletPort は、システムが保持する「ミント権限ウォレット」を
// ドメイン層から抽象化するポートです。
type MintAuthorityWalletPort interface {
	// ミント権限ウォレットの公開鍵（アドレス表示用など）
	PublicKey(ctx context.Context) (string, error)

	// 実際にミントを実行する
	MintToken(ctx context.Context, params MintParams) (*MintResult, error)
}

// ============================================================
// TokenBlueprint リポジトリ用ポート
// ============================================================
//
// TokenUsecase から見た「TokenBlueprint の minted 状態を更新するための最小インターフェース」。
// 具体実装（Firestore など）は adapters/out 側で tbdom.RepositoryPort を満たす形で実装し、
// そのサブセットとしてこのポートも満たす想定です。
type TokenBlueprintRepositoryPort interface {
	// 指定 ID の TokenBlueprint を取得
	GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)

	// minted 状態などを更新
	Update(ctx context.Context, id string, input tbdom.UpdateTokenBlueprintInput) (*tbdom.TokenBlueprint, error)
}
