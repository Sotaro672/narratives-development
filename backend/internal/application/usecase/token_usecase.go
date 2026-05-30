// backend/internal/application/usecase/token_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	tokendom "narratives/internal/domain/token"
)

// ============================================================
// Token mint input / result DTO
// ============================================================

// MintProductsInput は、TokenUsecase が on-chain mint を実行するための入力です。
// TokenUsecase は mint request / Firestore / inventory を知らず、
// 渡された mint 条件を使って productId ごとに MintToken を実行するだけにします。
type MintProductsInput struct {
	ToAddress string

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

// ============================================================
// TokenUsecase
// ============================================================

type TokenUsecase struct {
	mintWallet tokendom.MintAuthorityWalletPort
}

// NewTokenUsecase は TokenUsecase のコンストラクタです。
// TokenUsecase は on-chain token mint 実行だけを担当します。
func NewTokenUsecase(
	mintWallet tokendom.MintAuthorityWalletPort,
) *TokenUsecase {
	return &TokenUsecase{
		mintWallet: mintWallet,
	}
}

// MintProducts は、指定された productId ごとに 1 token/NFT を mint します。
//
// 動作方針:
// - productId ごとに Amount=1 で MintToken を呼び出す
// - mint request のロード、metadata URI 確保、Firestore 更新、inventory 更新は MintUsecase 側で行う
func (u *TokenUsecase) MintProducts(
	ctx context.Context,
	input MintProductsInput,
) ([]MintedTokenForUsecase, error) {
	if u == nil || u.mintWallet == nil {
		return nil, fmt.Errorf("token usecase is not properly initialized")
	}

	to := strings.TrimSpace(input.ToAddress)
	if to == "" {
		return nil, fmt.Errorf("toAddress is empty")
	}

	metadataURI := strings.TrimSpace(input.MetadataURI)
	if metadataURI == "" {
		return nil, fmt.Errorf("metadataURI is empty")
	}

	name := strings.TrimSpace(input.BlueprintName)
	symbol := strings.TrimSpace(input.BlueprintSymbol)
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("blueprint name or symbol is empty")
	}

	productIDs := make([]string, 0, len(input.ProductIDs))
	for _, pid := range input.ProductIDs {
		p := strings.TrimSpace(pid)
		if p == "" {
			continue
		}
		productIDs = append(productIDs, p)
	}

	if len(productIDs) == 0 {
		return nil, fmt.Errorf("no valid productIDs")
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
		if res == nil {
			return nil, fmt.Errorf("mint token on chain for product %s returned nil result", pid)
		}

		minted = append(minted, MintedTokenForUsecase{
			ProductID: pid,
			Result:    res,
		})
	}

	return minted, nil
}
