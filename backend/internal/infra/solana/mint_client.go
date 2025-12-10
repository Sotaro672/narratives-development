// backend/internal/infra/solana/mint_client.go
package solana

import (
	"context"
	"encoding/hex"
	"fmt"

	tokendom "narratives/internal/domain/token"
)

// MintClient は「Narratives が唯一保持するミント権限ウォレット」を使って
// 実際にチェーン上でミント処理を行うクライアントです。
// usecase.TokenUsecase からは tokendom.MintAuthorityWalletPort として利用されます。
type MintClient struct {
	key *MintAuthorityKey
}

// インターフェース実装チェック:
// MintClient が tokendom.MintAuthorityWalletPort を満たしていなければコンパイルエラーになります。
var _ tokendom.MintAuthorityWalletPort = (*MintClient)(nil)

// NewMintClient はミント権限キーを受け取って MintClient を初期化します。
func NewMintClient(key *MintAuthorityKey) *MintClient {
	return &MintClient{key: key}
}

// PublicKey は tokendom.MintAuthorityWalletPort の実装です。
// ミント権限ウォレットの公開鍵を string として返します。
//
// TODO: 将来的には Solana の base58 アドレス形式に揃える。
// ひとまずは ed25519.PublicKey ([]byte) を hex 文字列にして返しています。
func (c *MintClient) PublicKey(ctx context.Context) (string, error) {
	_ = ctx // 現時点では ctx を使用していないため unused 回避

	if c == nil || c.key == nil {
		return "", fmt.Errorf("mint client is not initialized (missing mint authority key)")
	}
	if len(c.key.PublicKey) == 0 {
		return "", fmt.Errorf("mint authority public key is empty")
	}

	// ed25519.PublicKey ([]byte) → string
	// ※ 実運用では base58 エンコードに変更する想定
	return hex.EncodeToString(c.key.PublicKey), nil
}

// MintToken は tokendom.MintAuthorityWalletPort インターフェースの実装です。
// TODO: Solana / Metaplex SDK を使った実装に差し替える。
func (c *MintClient) MintToken(
	ctx context.Context,
	params tokendom.MintParams,
) (*tokendom.MintResult, error) {
	_ = ctx // 現時点では ctx を使用していないため unused 回避

	if c == nil || c.key == nil {
		return nil, fmt.Errorf("mint client is not initialized (missing mint authority key)")
	}

	// ===========================================
	// TODO: ここに実際の Solana ミント処理を実装する
	//  - c.key (MintAuthorityKey) から秘密鍵/Keypair を生成
	//  - rpcClient / connection を初期化
	//  - Metaplex / Token-2022 などを用いて
	//    params.Name, params.Symbol, params.MetadataURI, params.Amount, params.ToAddress
	//    を元にミント
	//  - 署名や mintAddress を tokendom.MintResult に詰めて返す
	// ===========================================

	// ひとまずコンパイルを通すためのスタブ実装
	// （オンチェーン処理をまだ書いていないため、空の結果を返す）
	return &tokendom.MintResult{}, nil
}
