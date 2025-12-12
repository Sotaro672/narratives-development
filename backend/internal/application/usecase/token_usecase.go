// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"

	tokendom "narratives/internal/domain/token"
)

// ============================================================
// MintRequestForUsecase / MintRequestPort
// ============================================================

// MintRequestForUsecase は、TokenUsecase がチェーンミントを行うために
// 必要となる MintRequest 情報だけを集約した DTO です。
type MintRequestForUsecase struct {
	ID string

	ToAddress string
	// 従来の「まとめてミント」用の数量
	Amount uint64

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
type MintRequestPort interface {
	// ミント実行に必要な情報をロード
	LoadForMinting(ctx context.Context, id string) (*MintRequestForUsecase, error)

	// 従来のまとめミント用: 1回の MintToken 実行結果で MintRequest を minted 状態に更新
	MarkAsMinted(ctx context.Context, id string, result *tokendom.MintResult) error

	// ★ 新規: productId ごとに 1 ミントした場合、その結果一覧で MintRequest / Token 情報を更新
	// 実装側で:
	//   - productId, mintAddress の 1:1 マッピングを tokens コレクション等に保存
	//   - MintRequest 自体も minted=true にする
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
	log.Printf("[token] NewTokenUsecase init (mintWallet=%v, mintRequestPort=%v)", mintWallet != nil, mintRequestPort != nil)

	return &TokenUsecase{
		mintWallet:    mintWallet,
		mintRequestPt: mintRequestPort,
	}
}

// MintFromMintRequest は、指定された MintRequest を起点にトークン/NFT をミントします。
//
// 動作方針:
//   - req.ProductIDs が 1件以上ある場合:
//     → productId ごとに Amount=1 で MintToken を複数回呼び出し
//     → MarkProductsAsMinted に結果一覧を渡す（1商品=1Mint の世界）
//   - req.ProductIDs が空の場合:
//     → 従来通り Amount を使って 1回の MintToken でまとめてミント
//     → MarkAsMinted を呼び出す（1Mint に Supply=n の世界）
func (u *TokenUsecase) MintFromMintRequest(
	ctx context.Context,
	mintRequestID string,
) (*tokendom.MintResult, error) {
	id := strings.TrimSpace(mintRequestID)
	log.Printf("[token] MintFromMintRequest start id=%s", id)

	if id == "" {
		log.Printf("[token] MintFromMintRequest FAILED: mintRequestID is empty")
		return nil, fmt.Errorf("mintRequestID is empty")
	}

	if u == nil || u.mintWallet == nil || u.mintRequestPt == nil {
		log.Printf("[token] MintFromMintRequest FAILED: usecase not initialized (u=%v, mintWallet=%v, mintRequestPt=%v)",
			u != nil, u != nil && u.mintWallet != nil, u != nil && u.mintRequestPt != nil)
		return nil, fmt.Errorf("token usecase is not properly initialized")
	}

	// 1. ミント用に整形された MintRequest 情報を取得
	req, err := u.mintRequestPt.LoadForMinting(ctx, id)
	if err != nil {
		log.Printf("[token] MintFromMintRequest LoadForMinting FAILED id=%s err=%v", id, err)
		return nil, fmt.Errorf("load mint request for minting: %w", err)
	}

	// 取得した MintRequest の概要をログに出す
	log.Printf("[token] MintFromMintRequest loaded MintRequest: id=%s to=%s amount=%d name=%s symbol=%s uri=%s productIDs=%d",
		req.ID,
		strings.TrimSpace(req.ToAddress),
		req.Amount,
		strings.TrimSpace(req.BlueprintName),
		strings.TrimSpace(req.BlueprintSymbol),
		strings.TrimSpace(req.MetadataURI),
		len(req.ProductIDs),
	)

	to := strings.TrimSpace(req.ToAddress)
	if to == "" {
		log.Printf("[token] MintFromMintRequest FAILED: ToAddress is empty (mintRequestID=%s)", req.ID)
		return nil, fmt.Errorf("mint request %s has empty ToAddress", req.ID)
	}

	metadataURI := strings.TrimSpace(req.MetadataURI)
	if metadataURI == "" {
		log.Printf("[token] MintFromMintRequest FAILED: MetadataURI is empty (mintRequestID=%s)", req.ID)
		return nil, fmt.Errorf("mint request %s has empty MetadataURI", req.ID)
	}

	name := strings.TrimSpace(req.BlueprintName)
	symbol := strings.TrimSpace(req.BlueprintSymbol)
	if name == "" || symbol == "" {
		log.Printf("[token] MintFromMintRequest FAILED: name or symbol is empty (mintRequestID=%s)", req.ID)
		return nil, fmt.Errorf("mint request %s has empty name or symbol", req.ID)
	}

	// ------------------------------------------------------------
	// 2. productId の有無で挙動を分岐
	// ------------------------------------------------------------

	// productId のホワイトリスト整形（空文字を除去）
	productIDs := make([]string, 0, len(req.ProductIDs))
	for _, pid := range req.ProductIDs {
		p := strings.TrimSpace(pid)
		if p == "" {
			continue
		}
		productIDs = append(productIDs, p)
	}

	// ★ 新モード: productId が 1件以上ある場合は「1商品=1Mint」でミントする
	if len(productIDs) > 0 {
		log.Printf("[token] MintFromMintRequest using per-product mint mode: products=%v", productIDs)

		minted := make([]MintedTokenForUsecase, 0, len(productIDs))

		for _, pid := range productIDs {
			params := tokendom.MintParams{
				ToAddress:   to,
				Amount:      1, // ★ 各 productId ごとに Supply=1 の Mint を発行
				MetadataURI: metadataURI,
				Name:        name,
				Symbol:      symbol,
			}

			log.Printf("[token] MintFromMintRequest mint on-chain START (per-product) mintRequestID=%s productId=%s to=%s amount=1 name=%s symbol=%s uri=%s",
				req.ID, pid, to, name, symbol, metadataURI)

			res, err := u.mintWallet.MintToken(ctx, params)
			if err != nil {
				log.Printf("[token] MintFromMintRequest mint on-chain FAILED (per-product) mintRequestID=%s productId=%s err=%v", req.ID, pid, err)
				return nil, fmt.Errorf("mint token on chain for product %s: %w", pid, err)
			}

			log.Printf("[token] MintFromMintRequest mint on-chain OK (per-product) mintRequestID=%s productId=%s result=%+v",
				req.ID, pid, res)

			minted = append(minted, MintedTokenForUsecase{
				ProductID: pid,
				Result:    res,
			})
		}

		// 3. MintRequest / Token 情報を更新（1商品=1Mint の結果一覧を渡す）
		if err := u.mintRequestPt.MarkProductsAsMinted(ctx, req.ID, minted); err != nil {
			log.Printf("[token] MintFromMintRequest MarkProductsAsMinted FAILED mintRequestID=%s err=%v", req.ID, err)

			// すでにチェーン上ではミント済みなので、エラー内容はアプリ側で扱えるように result は返しておく
			var lastResult *tokendom.MintResult
			if len(minted) > 0 {
				lastResult = minted[len(minted)-1].Result
			}
			return lastResult, fmt.Errorf("mark mint request as minted (per-product): %w", err)
		}

		log.Printf("[token] MintFromMintRequest DONE (per-product) mintRequestID=%s mintedCount=%d", req.ID, len(minted))

		// API 互換のため、最後にミントしたトークンの結果を代表して返す
		if len(minted) > 0 {
			return minted[len(minted)-1].Result, nil
		}
		return nil, nil
	}

	// ★ 旧モード: productId が空の場合は従来通り「まとめてミント」
	amount := req.Amount
	if amount == 0 {
		amount = 1
	}

	params := tokendom.MintParams{
		ToAddress:   to,
		Amount:      amount,
		MetadataURI: metadataURI,
		Name:        name,
		Symbol:      symbol,
	}

	log.Printf("[token] MintFromMintRequest mint on-chain START (legacy) mintRequestID=%s to=%s amount=%d name=%s symbol=%s uri=%s",
		req.ID, to, amount, name, symbol, metadataURI)

	result, err := u.mintWallet.MintToken(ctx, params)
	if err != nil {
		log.Printf("[token] MintFromMintRequest mint on-chain FAILED (legacy) mintRequestID=%s err=%v", req.ID, err)
		return nil, fmt.Errorf("mint token on chain: %w", err)
	}

	// MintResult の中身をダンプ
	log.Printf("[token] MintFromMintRequest mint on-chain OK (legacy) mintRequestID=%s result=%+v", req.ID, result)

	// 3. MintRequest を minted/completed 状態に更新
	if err := u.mintRequestPt.MarkAsMinted(ctx, req.ID, result); err != nil {
		log.Printf("[token] MintFromMintRequest MarkAsMinted FAILED mintRequestID=%s err=%v", req.ID, err)
		return result, fmt.Errorf("mark mint request as minted: %w", err)
	}

	log.Printf("[token] MintFromMintRequest DONE (legacy) mintRequestID=%s", req.ID)
	return result, nil
}

// MintDirect は、MintRequest を介さずに直接ミントを行うためのヘルパーです。
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
	log.Printf("[token] MintDirect start to=%s amount=%d name=%s symbol=%s uri=%s",
		strings.TrimSpace(in.ToAddress),
		in.Amount,
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.Symbol),
		strings.TrimSpace(in.MetadataURI),
	)

	if u == nil || u.mintWallet == nil {
		log.Printf("[token] MintDirect FAILED: usecase not initialized (u=%v, mintWallet=%v)", u != nil, u != nil && u.mintWallet != nil)
		return nil, fmt.Errorf("token usecase is not properly initialized")
	}

	to := strings.TrimSpace(in.ToAddress)
	if to == "" {
		log.Printf("[token] MintDirect FAILED: ToAddress is empty")
		return nil, fmt.Errorf("ToAddress is empty")
	}
	amount := in.Amount
	if amount == 0 {
		amount = 1
	}
	metadataURI := strings.TrimSpace(in.MetadataURI)
	if metadataURI == "" {
		log.Printf("[token] MintDirect FAILED: MetadataURI is empty")
		return nil, fmt.Errorf("MetadataURI is empty")
	}
	name := strings.TrimSpace(in.Name)
	symbol := strings.TrimSpace(in.Symbol)
	if name == "" || symbol == "" {
		log.Printf("[token] MintDirect FAILED: name or symbol is empty")
		return nil, fmt.Errorf("name or symbol is empty")
	}

	params := tokendom.MintParams{
		ToAddress:   to,
		Amount:      amount,
		MetadataURI: metadataURI,
		Name:        name,
		Symbol:      symbol,
	}

	log.Printf("[token] MintDirect mint on-chain START to=%s amount=%d name=%s symbol=%s uri=%s",
		to, amount, name, symbol, metadataURI)

	result, err := u.mintWallet.MintToken(ctx, params)
	if err != nil {
		log.Printf("[token] MintDirect mint on-chain FAILED to=%s err=%v", to, err)
		return nil, fmt.Errorf("mint token on chain: %w", err)
	}

	log.Printf("[token] MintDirect mint on-chain OK to=%s result=%+v", to, result)
	return result, nil
}
