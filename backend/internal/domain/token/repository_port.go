// backend\internal\domain\token\repository_port.go
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
// Token query port
// ============================================================
//
// Firestore 実データ（tokens/{docId}）の前提:
// - docId = productId
// - fields: brandId, tokenBlueprintId, mintAddress, metadataUri, ...
//
// 本ポートは以下の read-model query を提供する:
// 1. mintAddress をキーに tokens を逆引きする
// 2. tokenBlueprintId をキーに同一 blueprint 配下の mintAddress 一覧を取得する
//
// ※ metadata の JSON 本体（URI の先の内容）まで取得したい場合は、
//    別途 HTTP fetch 用ポートを設ける（このポートには含めない）のが推奨です。

type TokenQueryPort interface {
	ResolveTokenByMintAddress(
		ctx context.Context,
		mintAddress string,
	) (ResolveTokenByMintAddressResult, error)

	ListMintAddressesByTokenBlueprintID(
		ctx context.Context,
		tokenBlueprintID string,
	) (ListMintAddressesByTokenBlueprintIDResult, error)
}
