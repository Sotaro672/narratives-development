// backend/internal/domain/token/entity.go
package token

// MintParams は、ミント実行に必要な最小限のパラメータです。
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
type MintResult struct {
	// ミントトランザクションのシグネチャ (base58)
	Signature string

	// 作成された mint アカウントのアドレス (base58)
	MintAddress string

	// オプション: どのスロットで確定したかなど
	Slot uint64
}
