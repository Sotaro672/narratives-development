// backend/internal/domain/token/port.go など
package token

import "context"

type MintAuthorityWalletPort interface {
	// ミント権限ウォレットの公開鍵（アドレス表示用など）
	PublicKey(ctx context.Context) (string, error)

	// 実際にミントを実行する
	MintToken(ctx context.Context, params MintParams) (*MintResult, error)
}
