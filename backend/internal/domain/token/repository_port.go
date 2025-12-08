// backend\internal\domain\token\repository_port.go
package token

import (
	"context"
	"errors"
)

// =====================================================
// MintAuthorityWalletPort
// -----------------------------------------------------
// 「システムが唯一保持するミント権限ウォレット」を表すドメインポート。
// 実装は infra 側（例: GCP Secret Manager + Solana RPC）に置き、
// Usecase はこのポートを通じてのみミントを実行する。
// =====================================================

// MintAuthorityWalletPort は、ミント権限ウォレットに対する操作を抽象化するポートです。
type MintAuthorityWalletPort interface {
	// PublicKey は、ミント権限ウォレットの公開鍵 (base58 文字列) を返します。
	// - 公開鍵はキャッシュしてよい（実装側判断）
	// - ネットワークエラーや Secret 読み込み失敗時はエラーを返す
	PublicKey(ctx context.Context) (string, error)

	// MintToken は、与えられたパラメータに基づいてトークン/NFT をミントします。
	// - Solana / Metaplex への実際の送信はアダプタ側の責務
	// - 成功時はトランザクションシグネチャや Mint アドレスなどを返す
	MintToken(ctx context.Context, params MintParams) (*MintResult, error)
}

// MintParams はミント時に必要となる情報を表します。
// Solana 特有の構造体（Transaction など）はここでは扱わず、
// ドメインにとって意味のある最小限の情報だけを持たせます。
type MintParams struct {
	// ToAddress は、トークンを受け取るウォレットアドレス (base58) です。
	ToAddress string

	// Amount はミントする数量です。
	// - NFT の場合は通常 1 固定
	// - 将来的に FT（Fungible Token）対応を見据えて uint64 としておく
	Amount uint64

	// MetadataURI は、Metaplex 形式の JSON メタデータを格納した URI（例: GCS, Arweave）です。
	MetadataURI string

	// Name / Symbol は、オンチェーンメタデータに格納されるトークン名・シンボルです。
	// - TokenBlueprint から引き継ぐ想定
	Name   string
	Symbol string
}

// MintResult はミント処理の結果を表します。
type MintResult struct {
	// Signature はブロックチェーン上のトランザクションシグネチャです。
	Signature string

	// MintAddress は作成されたトークンの Mint アドレス (base58) です。
	// - NFT の場合は 1 トークン = 1 Mint アドレス
	MintAddress string
}

// 共通エラー定義（必要に応じてアダプタ側で wrap して使う）
var (
	// ErrMintAuthorityNotConfigured は、ミント権限ウォレットの設定が存在しない場合のエラーです。
	ErrMintAuthorityNotConfigured = errors.New("token: mint authority wallet is not configured")

	// ErrMintFailed は、チェーン側でミントに失敗したことを表す汎用エラーです。
	ErrMintFailed = errors.New("token: mint failed")
)
