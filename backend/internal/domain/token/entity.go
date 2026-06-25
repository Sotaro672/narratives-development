// backend/internal/domain/token/entity.go
package token

import (
	"errors"
	"time"
)

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

// ============================================================
// GetTokenByProductIDResult
// ============================================================
//
// Firestore の tokens/{docId} を productId で取得した結果。
// productId は "docId" を正とする（= 1 token doc が 1 product に紐づく想定）。
//
// Firestore 実データ前提:
// - tokens/{docId}
// - docId = productId
// - fields: brandId, tokenBlueprintId, mintAddress, metadataUri, ...
type GetTokenByProductIDResult struct {
	ProductID        string
	BrandID          string
	TokenBlueprintID string
	MetadataURI      string
	MintAddress      string
}

// ============================================================
// ResolveTokenByMintAddressResult
// ============================================================
//
// Firestore の tokens コレクションを mintAddress で逆引きした結果。
// productId は "docId" を正とする（= 1 token doc が 1 product に紐づく想定）。
type ResolveTokenByMintAddressResult struct {
	ProductID   string
	BrandID     string
	MetadataURI string
	MintAddress string
}

// ============================================================
// ListMintAddressesByTokenBlueprintIDResult
// ============================================================
//
// Firestore の tokens コレクションを tokenBlueprintId で検索し、
// 同一 blueprint に紐づく mintAddress 一覧を返す結果です。
type ListMintAddressesByTokenBlueprintIDResult struct {
	TokenBlueprintID string
	MintAddresses    []string
}

// ResolveTransferredAtByMintAddressResult represents a lookup result for order identification.
//
// Transfer entity には transferredAt を持たせない方針のため、
// mintAddress から transfer 実行日時を引きたい query では、この read result として返す。
type ResolveTransferredAtByMintAddressResult struct {
	ProductID     string    `json:"productId"`
	Attempt       int       `json:"attempt"`
	AvatarID      string    `json:"avatarId"`
	MintAddress   string    `json:"mintAddress"`
	TransferredAt time.Time `json:"transferredAt"`
}

var (
	// TokenQuery が「token document が見つからない」時に返す
	ErrNotFound = errors.New("token: not found")

	// TokenQuery が「productId が不正」時に返す
	ErrInvalidProductID = errors.New("token: invalid productId")

	// TokenQuery が「mintAddress が不正」時に返す
	ErrInvalidMintAddress = errors.New("token: invalid mintAddress")

	// TokenQuery が「tokenBlueprintId が不正」時に返す
	ErrInvalidTokenBlueprintID = errors.New("token: invalid tokenBlueprintId")
	ErrInvalidTransferredAt    = errors.New("transfer: invalid transferredAt")
)
