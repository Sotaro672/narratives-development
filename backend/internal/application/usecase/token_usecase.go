// backend/internal/application/usecase/token_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	tokendom "narratives/internal/domain/token"
)

// ============================================================
// MintRequestForUsecase / MintRequestPort
// ------------------------------------------------------------
// 前提:
//   - mint_usecase.go が「どのアイテムを何枚ミントするか」「どの TokenBlueprint を元に
//     どの MetadataURI を使うか」といった *ミントリクエストを組み立てる* 責務を持つ。
//   - その結果として永続化された MintRequest（ドメインエンティティ）は、
//     実際のチェーンミントを行う TokenUsecase から利用される。
//   - TokenUsecase は、MintRequestPort を通じて「ミントに必要な情報だけ」を取得し、
//     MintAuthorityWalletPort.MintToken(...) を呼び出してチェーンにミントする。
// ============================================================

// MintRequestForUsecase は、TokenUsecase がチェーンミントを行うために
// 必要となる MintRequest 情報だけを集約した DTO です。
//
// 実際の MintRequest ドメイン（テーブル定義・状態遷移など）は infra / domain 層で定義し、
// そこから「ミント用ビュー」としてこの DTO に詰め替えるイメージです。
type MintRequestForUsecase struct {
	// ID は MintRequest の識別子です。
	ID string

	// ToAddress はトークンを受け取るウォレットアドレス (base58) です。
	// - 通常は購入者のウォレットアドレスやブランド指定のウォレットなど。
	ToAddress string

	// Amount はミント数量です。NFT の場合は通常 1。
	// - mint_usecase.go 側で検品結果や数量ロジックに基づいて決定しておく。
	Amount uint64

	// BlueprintName / BlueprintSymbol は、オンチェーンに載せるトークン名・シンボルです。
	// - TokenBlueprint から引き継いだ値を mint_usecase.go 側で詰めておく想定。
	BlueprintName   string
	BlueprintSymbol string

	// MetadataURI は、Metaplex 形式の JSON メタデータを格納した URI です。
	// - 例: GCS にアップロードされた metadata.json の公開URL
	// - mint_usecase.go 側で ProductBlueprint / TokenBlueprint / Inspection 情報から
	//   組み立て、GCS へアップロードした結果の URI をここにセットしておく。
	MetadataURI string
}

// MintRequestPort は、TokenUsecase から見た「ミント対象 MintRequest」の
// 取得および更新を行うためのポートです。
//
// 実装例:
//   - backend/internal/adapters/out/firestore/mint_request_repository_fs.go
//   - そこから domain/mintRequest エンティティを読み、必要なフィールドを
//     MintRequestForUsecase に詰めて返す。
type MintRequestPort interface {
	// LoadForMinting は、指定された MintRequest を「ミントに必要な形」で読み込みます。
	//
	// 推奨する責務:
	//   - ステータスチェック（例: status == "approved" or "readyToMint" か）
	//   - すでに minted 済みでないかのチェック
	//   - ToAddress / Amount / MetadataURI / BlueprintName / BlueprintSymbol が
	//     すべて揃っているかの検証
	//
	// いずれかに問題がある場合はエラーを返し、TokenUsecase 側では
	// 「この MintRequest はミントできない」と判断できるようにする。
	LoadForMinting(ctx context.Context, id string) (*MintRequestForUsecase, error)

	// MarkAsMinted は、チェーン上でのミント成功結果をもとに MintRequest を更新します。
	//
	// 典型的な更新内容:
	//   - status: "minted" / "completed" などへの遷移
	//   - onChainTxSignature: result.Signature
	//   - mintAddress:        result.MintAddress
	//   - mintedAt:           現在時刻
	//
	// ※ ここでは時刻や更新者は実装側で決めてよい。
	MarkAsMinted(ctx context.Context, id string, result *tokendom.MintResult) error
}

// ============================================================
// TokenUsecase
// ------------------------------------------------------------
// - 「システムが唯一保持するミント権限ウォレット」(MintAuthorityWalletPort)
//   を利用して実際のミント処理を行うアプリケーションサービス。
// - MintRequest や TokenBlueprint などのドメインを、DTO/ポート経由で利用する。
// ============================================================

type TokenUsecase struct {
	mintWallet    tokendom.MintAuthorityWalletPort
	mintRequestPt MintRequestPort
}

// NewTokenUsecase は TokenUsecase のコンストラクタです。
func NewTokenUsecase(
	mintWallet tokendom.MintAuthorityWalletPort,
	mintRequestPort MintRequestPort,
) *TokenUsecase {
	return &TokenUsecase{
		mintWallet:    mintWallet,
		mintRequestPt: mintRequestPort,
	}
}

// MintFromMintRequest は、指定された MintRequest を起点にトークン/NFT をミントします。
//
// フロー:
//  1. MintRequestPort.LoadForMinting でミント可能なリクエスト情報を取得
//  2. MintAuthorityWalletPort.MintToken でチェーン上にミント
//  3. MintRequestPort.MarkAsMinted で MintRequest を更新
func (u *TokenUsecase) MintFromMintRequest(
	ctx context.Context,
	mintRequestID string,
) (*tokendom.MintResult, error) {
	id := strings.TrimSpace(mintRequestID)
	if id == "" {
		return nil, fmt.Errorf("mintRequestID is empty")
	}

	if u == nil || u.mintWallet == nil || u.mintRequestPt == nil {
		return nil, fmt.Errorf("token usecase is not properly initialized")
	}

	// 1. ミント用に整形された MintRequest 情報を取得
	req, err := u.mintRequestPt.LoadForMinting(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load mint request for minting: %w", err)
	}

	to := strings.TrimSpace(req.ToAddress)
	if to == "" {
		return nil, fmt.Errorf("mint request %s has empty ToAddress", req.ID)
	}

	amount := req.Amount
	if amount == 0 {
		// NFT 前提の場合は 1 をデフォルトとする
		amount = 1
	}

	metadataURI := strings.TrimSpace(req.MetadataURI)
	if metadataURI == "" {
		return nil, fmt.Errorf("mint request %s has empty MetadataURI", req.ID)
	}

	name := strings.TrimSpace(req.BlueprintName)
	symbol := strings.TrimSpace(req.BlueprintSymbol)
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("mint request %s has empty name or symbol", req.ID)
	}

	// 2. ミント権限ウォレットを用いて実際にミント
	params := tokendom.MintParams{
		ToAddress:   to,
		Amount:      amount,
		MetadataURI: metadataURI,
		Name:        name,
		Symbol:      symbol,
	}

	result, err := u.mintWallet.MintToken(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("mint token on chain: %w", err)
	}

	// 3. MintRequest を minted/completed 状態に更新
	if err := u.mintRequestPt.MarkAsMinted(ctx, req.ID, result); err != nil {
		// ミント自体は成功しているので、結果は返しつつ更新エラーをラップ
		return result, fmt.Errorf("mark mint request as minted: %w", err)
	}

	return result, nil
}

// MintDirect は、MintRequest を介さずに直接ミントを行うためのヘルパーです。
// - 管理画面からのテストミントや、将来のユースケース拡張用に用意しています。
type MintDirectInput struct {
	ToAddress   string
	Amount      uint64
	MetadataURI string
	Name        string
	Symbol      string
}

func (u *TokenUsecase) MintDirect(
	ctx context.Context,
	in MintDirectInput,
) (*tokendom.MintResult, error) {
	if u == nil || u.mintWallet == nil {
		return nil, fmt.Errorf("token usecase is not properly initialized")
	}

	to := strings.TrimSpace(in.ToAddress)
	if to == "" {
		return nil, fmt.Errorf("ToAddress is empty")
	}
	amount := in.Amount
	if amount == 0 {
		amount = 1
	}
	metadataURI := strings.TrimSpace(in.MetadataURI)
	if metadataURI == "" {
		return nil, fmt.Errorf("MetadataURI is empty")
	}
	name := strings.TrimSpace(in.Name)
	symbol := strings.TrimSpace(in.Symbol)
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("name or symbol is empty")
	}

	params := tokendom.MintParams{
		ToAddress:   to,
		Amount:      amount,
		MetadataURI: metadataURI,
		Name:        name,
		Symbol:      symbol,
	}

	return u.mintWallet.MintToken(ctx, params)
}
