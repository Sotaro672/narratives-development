// backend/internal/domain/token/port.go
package token

import (
	"context"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Mint 権限ウォレット用ポート
// ============================================================

type MintAuthorityWalletPort interface {
	PublicKey(ctx context.Context) (string, error)
	MintToken(ctx context.Context, params MintParams) (*MintResult, error)
}

// ============================================================
// TokenBlueprint リポジトリ用ポート
// ============================================================

type TokenBlueprintRepositoryPort interface {
	GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
	Update(ctx context.Context, id string, input tbdom.UpdateTokenBlueprintInput) (*tbdom.TokenBlueprint, error)
}

// ============================================================
// Token query port (mintAddress -> productId(docId) + brandId + metadataUri)
// ============================================================
//
// Firestore 実データ（tokens/{docId}）の前提:
// - docId = productId
// - fields: brandId, tokenBlueprintId, mintAddress, metadataUri, ...
//
// 本ポートは mintAddress をキーに tokens を検索し、以下を返す:
// - productId (= docId)
// - brandId
// - metadataUri（"中身"＝そのまま URI 文字列）
//
// ※ metadata の JSON 本体（URI の先の内容）まで取得したい場合は、
//    別途 HTTP fetch 用ポートを設ける（このポートには含めない）のが推奨です。

// TokenQueryPort は mintAddress から tokens を逆引きする read-model 用ポートです。
type TokenQueryPort interface {
	ResolveTokenByMintAddress(
		ctx context.Context,
		mintAddress string,
	) (ResolveTokenByMintAddressResult, error)
}
