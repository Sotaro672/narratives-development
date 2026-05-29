// backend/internal/application/usecase/token_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	tokendom "narratives/internal/domain/token"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// MintRequestForUsecase / MintRequestPort
// ============================================================

// MintRequestForUsecase は、TokenUsecase がチェーンミントを行うために
// 必要となる MintRequest 情報だけを集約した DTO です。
type MintRequestForUsecase struct {
	ID string

	// TokenBlueprintID は、metadata URI の確保や tokenBlueprint minted 化に使います。
	TokenBlueprintID string

	// ActorID は、metadata URI 確保や tokenBlueprint minted 化の実行者として使います。
	ActorID string

	// 受取先アドレス（ブランドウォレット等）
	// NOTE:
	// - これは「NFT/トークンを受け取るアドレス」であり、FeePayer（ガス支払い）ではありません。
	// - FeePayer はインフラ側（mint/transfer 実装）で master wallet に統一しています。
	ToAddress string

	// productId ごとに 1 ミントしたい場合の productId 一覧。
	// これが 1件以上入っている場合、
	// - 各 productId ごとに Amount=1 で MintToken を呼び出す
	// - 1商品 = 1Mint (Supply=1) の NFT 的な運用が可能になる
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
//
// 現在のフローでは「1商品=1Mint」モードのみを想定しています。
type MintRequestPort interface {
	// LoadForMinting:
	// - ミント実行に必要な情報をロードします。
	// - TokenBlueprintID / ActorID / ToAddress / ProductIDs / BlueprintName /
	//   BlueprintSymbol / MetadataURI を返す想定です。
	LoadForMinting(ctx context.Context, id string) (*MintRequestForUsecase, error)

	// MarkProductsAsMinted:
	// - productId ごとに 1 ミントした結果一覧で MintRequest / Token 情報を更新します。
	// - 実装側で:
	//   - productId, mintAddress の 1:1 マッピングを tokens コレクション等に保存
	//   - MintRequest (mints テーブル) 自体も minted=true にする
	MarkProductsAsMinted(ctx context.Context, id string, minted []MintedTokenForUsecase) error
}

// ============================================================
// TokenBlueprint dependencies
// ============================================================

type TokenBlueprintMetadataEnsurer interface {
	EnsureMetadataURI(ctx context.Context, tb *tbdom.TokenBlueprint, actorID string) (*tbdom.TokenBlueprint, error)
}

type TokenBlueprintMintMarker interface {
	MarkTokenBlueprintMinted(
		ctx context.Context,
		tokenBlueprintID string,
		actorID string,
	) (*tbdom.TokenBlueprint, error)
}

// ============================================================
// TokenUsecase
// ============================================================

type TokenUsecase struct {
	mintWallet    tokendom.MintAuthorityWalletPort
	mintRequestPt MintRequestPort

	tbRepo            tbdom.RepositoryPort
	tbMetadataEnsurer TokenBlueprintMetadataEnsurer
	tbMintMarker      TokenBlueprintMintMarker
}

// NewTokenUsecase は TokenUsecase のコンストラクタです。
func NewTokenUsecase(
	mintWallet tokendom.MintAuthorityWalletPort,
	mintRequestPort MintRequestPort,
) *TokenUsecase {
	return &TokenUsecase{
		mintWallet:        mintWallet,
		mintRequestPt:     mintRequestPort,
		tbRepo:            nil,
		tbMetadataEnsurer: nil,
		tbMintMarker:      nil,
	}
}

func (u *TokenUsecase) SetTokenBlueprintRepo(repo tbdom.RepositoryPort) {
	if u == nil {
		return
	}
	u.tbRepo = repo
}

func (u *TokenUsecase) SetTokenBlueprintMetadataEnsurer(e TokenBlueprintMetadataEnsurer) {
	if u == nil {
		return
	}
	u.tbMetadataEnsurer = e
}

func (u *TokenUsecase) SetTokenBlueprintMintMarker(marker TokenBlueprintMintMarker) {
	if u == nil {
		return
	}
	u.tbMintMarker = marker
}

// MintFromMintRequest は、指定された MintRequest を起点に
// 「1商品=1Mint」でトークン/NFT をミントします。
//
// 動作方針:
//   - req.ProductIDs が 1件以上あることを前提とし、
//     productId ごとに Amount=1 で MintToken を複数回呼び出す
//   - metadata URI が未確定の場合、TokenBlueprintMetadataEnsurer で確保する
//   - その結果一覧を MarkProductsAsMinted に渡す
//   - tokenBlueprint minted 化は TokenBlueprintMintMarker が設定されていれば行う
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

	req, err := u.mintRequestPt.LoadForMinting(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load mint request for minting: %w", err)
	}
	if req == nil {
		return nil, fmt.Errorf("mint request %s is nil", id)
	}

	to := strings.TrimSpace(req.ToAddress)
	if to == "" {
		return nil, fmt.Errorf("mint request %s has empty ToAddress", req.ID)
	}

	tokenBlueprintID := strings.TrimSpace(req.TokenBlueprintID)
	actorID := strings.TrimSpace(req.ActorID)

	metadataURI := strings.TrimSpace(req.MetadataURI)
	if tokenBlueprintID != "" && u.tbMetadataEnsurer != nil {
		if u.tbRepo == nil {
			return nil, fmt.Errorf("tokenBlueprint repo is nil")
		}

		tb, err := u.tbRepo.GetByID(ctx, tokenBlueprintID)
		if err != nil {
			return nil, fmt.Errorf("get tokenBlueprint for metadata ensure: %w", err)
		}
		if tb == nil {
			return nil, fmt.Errorf("tokenBlueprint not found (id=%s)", tokenBlueprintID)
		}

		updated, err := u.tbMetadataEnsurer.EnsureMetadataURI(ctx, tb, actorID)
		if err != nil {
			return nil, fmt.Errorf("ensure metadata uri: %w", err)
		}
		if updated == nil {
			updated = tb
		}

		metadataURI = strings.TrimSpace(updated.MetadataURI)
	}

	if metadataURI == "" {
		return nil, fmt.Errorf("mint request %s has empty MetadataURI", req.ID)
	}

	name := strings.TrimSpace(req.BlueprintName)
	symbol := strings.TrimSpace(req.BlueprintSymbol)
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("mint request %s has empty name or symbol", req.ID)
	}

	productIDs := make([]string, 0, len(req.ProductIDs))
	for _, pid := range req.ProductIDs {
		p := strings.TrimSpace(pid)
		if p == "" {
			continue
		}
		productIDs = append(productIDs, p)
	}

	if len(productIDs) == 0 {
		return nil, fmt.Errorf("mint request %s has no valid productIDs", req.ID)
	}

	minted := make([]MintedTokenForUsecase, 0, len(productIDs))

	for _, pid := range productIDs {
		params := tokendom.MintParams{
			ToAddress:   to,
			Amount:      1,
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

	if err := u.mintRequestPt.MarkProductsAsMinted(ctx, req.ID, minted); err != nil {
		var lastResult *tokendom.MintResult
		if len(minted) > 0 {
			lastResult = minted[len(minted)-1].Result
		}
		return lastResult, fmt.Errorf("mark mint request as minted (per-product): %w", err)
	}

	if u.tbMintMarker != nil && tokenBlueprintID != "" {
		_, _ = u.tbMintMarker.MarkTokenBlueprintMinted(ctx, tokenBlueprintID, actorID)
	}

	if len(minted) > 0 {
		return minted[len(minted)-1].Result, nil
	}

	return nil, nil
}
