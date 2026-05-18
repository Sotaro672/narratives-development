// backend/internal/application/usecase/token_usecase.go
package usecase

import (
	"context"
	"fmt"

	tokendom "narratives/internal/domain/token"
)

// ============================================================
// MintRequestForUsecase / MintRequestPort
// ============================================================

// MintRequestForUsecase は、TokenUsecase がチェーンミントを行うために
// 必要となる MintRequest 情報だけを集約した DTO です。
type MintRequestForUsecase struct {
	ID string

	// ✅ 受取先アドレス（ブランドウォレット等）
	// NOTE:
	// - これは「NFT/トークンを受け取るアドレス」であり、FeePayer（ガス支払い）ではありません。
	// - FeePayer はインフラ側（mint/transfer 実装）で master wallet に統一しています。
	ToAddress string

	// ★ 新規: productId ごとに 1 ミントしたい場合の productId 一覧
	// これが 1件以上入っている場合、
	//   - 各 productId ごとに Amount=1 で MintToken を呼び出す
	//   - 1商品 = 1Mint (Supply=1) の NFT 的な運用が可能になる
	ProductIDs []string

	BlueprintName   string
	BlueprintSymbol string

	MetadataURI string
}

// MintedTokenForUsecase は、「どの productId に対して、どの MintResult が紐づくか」を表す DTO です。
type MintedTokenForUsecase struct {
	ProductID string
	Result    *tokendom.MintResult
}

// MintRequestPort は、TokenUsecase から見た「ミント対象 MintRequest」の
// 取得および更新を行うためのポートです。
// 現在のフローでは「1商品=1Mint」モードのみを想定しています。
type MintRequestPort interface {
	// ミント実行に必要な情報をロード
	LoadForMinting(ctx context.Context, id string) (*MintRequestForUsecase, error)

	// ★ productId ごとに 1 ミントした結果一覧で MintRequest / Token 情報を更新
	// 実装側で:
	//   - productId, mintAddress の 1:1 マッピングを tokens コレクション等に保存
	//   - MintRequest (mints テーブル) 自体も minted=true にする
	MarkProductsAsMinted(ctx context.Context, id string, minted []MintedTokenForUsecase) error
}

// ============================================================
// TokenUsecase
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

// MintFromMintRequest は、指定された MintRequest を起点に
// 「1商品=1Mint」でトークン/NFT をミントします。
//
// 動作方針:
//   - req.ProductIDs が 1件以上あることを前提とし、
//     productId ごとに Amount=1 で MintToken を複数回呼び出す
//   - その結果一覧を MarkProductsAsMinted に渡す
func (u *TokenUsecase) MintFromMintRequest(
	ctx context.Context,
	mintRequestID string,
) (*tokendom.MintResult, error) {
	id := mintRequestID

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

	to := req.ToAddress
	if to == "" {
		return nil, fmt.Errorf("mint request %s has empty ToAddress", req.ID)
	}

	metadataURI := req.MetadataURI
	if metadataURI == "" {
		return nil, fmt.Errorf("mint request %s has empty MetadataURI", req.ID)
	}

	name := req.BlueprintName
	symbol := req.BlueprintSymbol
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("mint request %s has empty name or symbol", req.ID)
	}

	// ------------------------------------------------------------
	// 2. productId のホワイトリスト整形（空文字を除去）
	// ------------------------------------------------------------

	productIDs := make([]string, 0, len(req.ProductIDs))
	for _, pid := range req.ProductIDs {
		p := pid
		if p == "" {
			continue
		}
		productIDs = append(productIDs, p)
	}

	if len(productIDs) == 0 {
		return nil, fmt.Errorf("mint request %s has no valid productIDs", req.ID)
	}

	// ------------------------------------------------------------
	// 3. 「1商品=1Mint」で順次ミント
	// ------------------------------------------------------------

	minted := make([]MintedTokenForUsecase, 0, len(productIDs))

	for _, pid := range productIDs {
		params := tokendom.MintParams{
			ToAddress:   to,
			Amount:      1, // ★ 各 productId ごとに Supply=1 の Mint を発行
			MetadataURI: metadataURI,
			Name:        name,
			Symbol:      symbol,
		}

		res, err := u.mintWallet.MintToken(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("mint token on chain for product %s: %w", pid, err)
		}

		minted = append(minted, MintedTokenForUsecase{
			ProductID: pid,
			Result:    res,
		})
	}

	// 4. MintRequest / Token 情報を更新（1商品=1Mint の結果一覧を渡す）
	if err := u.mintRequestPt.MarkProductsAsMinted(ctx, req.ID, minted); err != nil {
		// すでにチェーン上ではミント済みなので、エラー内容はアプリ側で扱えるように result は返しておく
		var lastResult *tokendom.MintResult
		if len(minted) > 0 {
			lastResult = minted[len(minted)-1].Result
		}
		return lastResult, fmt.Errorf("mark mint request as minted (per-product): %w", err)
	}

	// API 互換のため、最後にミントしたトークンの結果を代表して返す
	if len(minted) > 0 {
		return minted[len(minted)-1].Result, nil
	}
	return nil, nil
}
