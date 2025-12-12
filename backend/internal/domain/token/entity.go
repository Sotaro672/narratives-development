// backend/internal/domain/token/entity.go
package token

// MintParams は、Solana 上でトークン/NFT をミントする際に
// MintAuthorityWalletPort に渡す最小限のパラメータです。
//
// - 「1商品=1Mint」モードでは、Amount は常に 1 を指定します。
// - 旧来の「まとめてミント」モードでは、Amount に発行枚数を指定します。
type MintParams struct {
	// トークンを受け取るウォレットアドレス (base58)
	ToAddress string

	// ミント数量（NFTなら通常 1）
	Amount uint64

	// Metaplex 形式 JSON メタデータの URI
	MetadataURI string

	// トークン名 / シンボル（TokenBlueprint 由来）
	Name   string
	Symbol string
}

// MintResult は、チェーン上のミント結果です。
// 1 回の MintToken 実行に対して 1 件生成されます。
type MintResult struct {
	// ミントトランザクションのシグネチャ (base58)
	Signature string

	// 作成された mint アカウントのアドレス (base58)
	MintAddress string

	// オプション: どのスロットで確定したかなど
	Slot uint64
}
