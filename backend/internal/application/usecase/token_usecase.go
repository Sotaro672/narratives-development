package usecase

import (
	"context"
	"fmt"
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
	Amount    uint64

	BlueprintName   string
	BlueprintSymbol string

	MetadataURI string
}

// MintRequestPort は、TokenUsecase から見た「ミント対象 MintRequest」の
// 取得および更新を行うためのポートです。
type MintRequestPort interface {
	LoadForMinting(ctx context.Context, id string) (*MintRequestForUsecase, error)
	MarkAsMinted(ctx context.Context, id string, result *tokendom.MintResult) error
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

// MintFromMintRequest は、指定された MintRequest を起点にトークン/NFT をミントします。
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
		return result, fmt.Errorf("mark mint request as minted: %w", err)
	}

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
