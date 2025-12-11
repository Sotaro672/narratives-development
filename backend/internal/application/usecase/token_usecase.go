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
	log.Printf("[token] NewTokenUsecase init (mintWallet=%v, mintRequestPort=%v)", mintWallet != nil, mintRequestPort != nil)

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

	// 取得した MintRequest の概要をログに出す（wallet / URI / name / symbol）
	log.Printf("[token] MintFromMintRequest loaded MintRequest: id=%s to=%s amount=%d name=%s symbol=%s uri=%s",
		req.ID,
		strings.TrimSpace(req.ToAddress),
		req.Amount,
		strings.TrimSpace(req.BlueprintName),
		strings.TrimSpace(req.BlueprintSymbol),
		strings.TrimSpace(req.MetadataURI),
	)

	to := strings.TrimSpace(req.ToAddress)
	if to == "" {
		log.Printf("[token] MintFromMintRequest FAILED: ToAddress is empty (mintRequestID=%s)", req.ID)
		return nil, fmt.Errorf("mint request %s has empty ToAddress", req.ID)
	}

	amount := req.Amount
	if amount == 0 {
		amount = 1
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

	// 2. ミント権限ウォレットを用いて実際にミント
	params := tokendom.MintParams{
		ToAddress:   to,
		Amount:      amount,
		MetadataURI: metadataURI,
		Name:        name,
		Symbol:      symbol,
	}

	log.Printf("[token] MintFromMintRequest mint on-chain START mintRequestID=%s to=%s amount=%d name=%s symbol=%s uri=%s",
		req.ID, to, amount, name, symbol, metadataURI)

	result, err := u.mintWallet.MintToken(ctx, params)
	if err != nil {
		log.Printf("[token] MintFromMintRequest mint on-chain FAILED mintRequestID=%s err=%v", req.ID, err)
		return nil, fmt.Errorf("mint token on chain: %w", err)
	}

	// MintResult の中身をダンプ（mintAddress / txSignature / authority 等を確認する用）
	log.Printf("[token] MintFromMintRequest mint on-chain OK mintRequestID=%s result=%+v", req.ID, result)

	// 3. MintRequest を minted/completed 状態に更新
	if err := u.mintRequestPt.MarkAsMinted(ctx, req.ID, result); err != nil {
		log.Printf("[token] MintFromMintRequest MarkAsMinted FAILED mintRequestID=%s err=%v", req.ID, err)
		return result, fmt.Errorf("mark mint request as minted: %w", err)
	}

	log.Printf("[token] MintFromMintRequest DONE mintRequestID=%s", req.ID)
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
