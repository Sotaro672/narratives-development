// backend/internal/domain/token/entity.go
package token

import (
	"errors"
	"time"
)

// MintParams は、Solana Bubblegum V2 で cNFT をミントする際に
// MintAuthorityWalletPort に渡す最小限のパラメータです.
//
// - 1 productId = 1 cNFT とします。
// - ProductID は Bubblegum mint service 側の idempotency key として使用します。
// - TokenBlueprintID は mint 対象の MPL Core Collection 解決に使用します。
// - Amount は常に 1 を指定します。
type MintParams struct {
	// cNFT に紐づく productId
	ProductID string

	// cNFT が属する TokenBlueprint の ID
	TokenBlueprintID string

	// cNFT を受け取るウォレットアドレス (base58)
	ToAddress string

	// ミント数量
	Amount uint64

	// Metaplex 形式 JSON メタデータの URI
	MetadataURI string

	// トークン名 / シンボル（TokenBlueprint 由来）
	Name   string
	Symbol string
}

// AssetStandard は、on-chain asset の方式を表します。
type AssetStandard string

const (
	AssetStandardBubblegumV2 AssetStandard = "BUBBLEGUM_V2"
)

// MintResult は、チェーン上のミント結果です。
// 1 回の MintToken 実行に対して 1 件生成されます。
type MintResult struct {
	// ミントトランザクションのシグネチャ (base58)
	Signature string

	// on-chain asset の方式
	AssetStandard AssetStandard

	// mint が実行された Solana cluster
	Cluster string

	// Bubblegum V2 cNFT の一意な asset ID (base58)
	AssetID string

	// asset が格納されている Merkle Tree のアドレス (base58)
	TreeAddress string

	// Merkle Tree 内の leaf index
	// 0 は正当な値です。
	LeafIndex uint64

	// cNFT が属する MPL Core Collection のアドレス (base58)
	CoreCollectionAddress string

	// mint transaction が確定した slot
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
//   - tokens/{docId}
//   - docId = productId
//   - fields: assetStandard, cluster, assetId, treeAddress, leafIndex,
//     coreCollectionAddress, brandId, tokenBlueprintId, metadataUri,
//     mintedAt, txSignature, toAddress
type GetTokenByProductIDResult struct {
	ProductID        string
	BrandID          string
	TokenBlueprintID string
	MetadataURI      string

	AssetStandard         AssetStandard
	Cluster               string
	AssetID               string
	TreeAddress           string
	LeafIndex             uint64
	CoreCollectionAddress string
}

// ============================================================
// ResolveTokenByAssetIDResult
// ============================================================
//
// Firestore の tokens コレクションを assetId で逆引きした結果。
// productId は "docId" を正とする（= 1 token doc が 1 product に紐づく想定）。
type ResolveTokenByAssetIDResult struct {
	ProductID   string
	BrandID     string
	MetadataURI string
	AssetID     string
}

// ============================================================
// ListAssetIDsByTokenBlueprintIDResult
// ============================================================
//
// Firestore の tokens コレクションを tokenBlueprintId で検索し、
// 同一 blueprint に紐づく assetId 一覧を返す結果です。
type ListAssetIDsByTokenBlueprintIDResult struct {
	TokenBlueprintID string
	AssetIDs         []string
}

// ResolveTransferredAtByAssetIDResult represents a lookup result for order identification.
//
// Transfer entity には transferredAt を持たせない方針のため、
// assetId から transfer 実行日時を引きたい query では、この read result として返す。
type ResolveTransferredAtByAssetIDResult struct {
	ProductID     string    `json:"productId"`
	Attempt       int       `json:"attempt"`
	AvatarID      string    `json:"avatarId"`
	AssetID       string    `json:"assetId"`
	TransferredAt time.Time `json:"transferredAt"`
}

var (
	// TokenQuery が「token document が見つからない」時に返す
	ErrNotFound = errors.New("token: not found")

	// TokenQuery が「productId が不正」時に返す
	ErrInvalidProductID = errors.New("token: invalid productId")

	// TokenQuery が「assetId が不正」時に返す
	ErrInvalidAssetID = errors.New("token: invalid assetId")

	// TokenQuery が「tokenBlueprintId が不正」時に返す
	ErrInvalidTokenBlueprintID = errors.New("token: invalid tokenBlueprintId")

	ErrInvalidTransferredAt = errors.New("transfer: invalid transferredAt")
)
