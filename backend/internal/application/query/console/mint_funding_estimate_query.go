// backend/internal/application/query/console/mint_funding_estimate_query.go
package query

import (
	"context"
	"errors"
	"fmt"

	applicationport "narratives/internal/application/port"
	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	mintdom "narratives/internal/domain/mint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

var (
	ErrMintFundingEstimateQueryNotConfigured = errors.New("mint funding estimate query is not configured")
	ErrMintFundingEstimateInvalidInput       = errors.New("mint funding estimate input is invalid")
	ErrMintFundingEstimateForbidden          = errors.New("mint funding estimate access is forbidden")
	ErrMintFundingEstimateNoPassedProducts   = errors.New("no passed products for mint funding estimate")
	ErrMintFundingEstimateBrandWalletMissing = errors.New("brand wallet address is empty")
)

// GetMintFundingEstimateInput は Console から見積を取得するための入力です。
//
// Frontend から Solana 固有値を直接渡させず、productionId と tokenBlueprintId のみを受け取ります。
// Backend 側では mintQuantity、Brand Wallet、TokenBlueprint情報を解決します。
// metadataUri は見積条件に含めず、solana-bubblegum 側で見積専用 URI を使用します。
type GetMintFundingEstimateInput struct {
	ProductionID     string `json:"productionId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

// MintFundingEstimateParams は solana-bubblegum の見積処理へ渡す application/query 層の入力です。
// mintQuantity は見積計算に必要な内部値として使用し、Console API responseには公開しません。
type MintFundingEstimateParams struct {
	TokenBlueprintID string
	MintQuantity     int
	ToAddress        string
	Name             string
	Symbol           string
}

type MintFundingEstimateReserve struct {
	Address         string  `json:"address"`
	BalanceLamports string  `json:"balanceLamports"`
	BalanceSOL      float64 `json:"balanceSol"`
	MinimumLamports string  `json:"minimumLamports"`
	MinimumSOL      float64 `json:"minimumSol"`
}

type MintFundingEstimateResources struct {
	SharedMerkleTreeExists  bool    `json:"sharedMerkleTreeExists"`
	SharedMerkleTreeAddress *string `json:"sharedMerkleTreeAddress"`
	CoreCollectionExists    bool    `json:"coreCollectionExists"`
	CoreCollectionAddress   *string `json:"coreCollectionAddress"`
}

// MintFundingEstimateCosts は Console API に公開する SOL 見積です.
//
// InitialCreationCost:
// Shared Merkle Tree と Core Collection の初回作成費合計。
//
// TotalRequired:
// Mint手数料合計 + InitialCreationCost。
//
// Fee Payer残高・目標残高、Reserve補充量などのfunding policy内部値は公開しません。
type MintFundingEstimateCosts struct {
	MintTransactionFeePerItemLamports string  `json:"mintTransactionFeePerItemLamports"`
	MintTransactionFeePerItemSOL      float64 `json:"mintTransactionFeePerItemSol"`
	MintTransactionFeeTotalLamports   string  `json:"mintTransactionFeeTotalLamports"`
	MintTransactionFeeTotalSOL        float64 `json:"mintTransactionFeeTotalSol"`
	InitialCreationCostLamports       string  `json:"initialCreationCostLamports"`
	InitialCreationCostSOL            float64 `json:"initialCreationCostSol"`
	TotalRequiredLamports             string  `json:"totalRequiredLamports"`
	TotalRequiredSOL                  float64 `json:"totalRequiredSol"`
	Sufficient                        bool    `json:"sufficient"`
}

// MintFundingEstimateResult は Console API が返す見積結果です。
// solana-bubblegum service の公開レスポンスを Backend application 層の read model として表現します。
type MintFundingEstimateResult struct {
	Cluster   string                       `json:"cluster"`
	Reserve   MintFundingEstimateReserve   `json:"reserve"`
	Resources MintFundingEstimateResources `json:"resources"`
	Estimate  MintFundingEstimateCosts     `json:"estimate"`
}

// MintFundingEstimateExecutor は SOL 見積の実処理を呼び出す関数です。
// application/query -> infra/solana の直接依存を避けるため、DI 層で MintClient.EstimateMintFunding をラップして注入します.
//
// IMPORTANT:
// - Mint を実行しない
// - Reserve から SOL を送金しない
// - Merkle Tree を作成しない
// - Core Collection を作成しない
// - metadataUri の生成や upload を行わない
type MintFundingEstimateExecutor func(
	ctx context.Context,
	params MintFundingEstimateParams,
) (*MintFundingEstimateResult, error)

type MintFundingEstimateQuery struct {
	passedProductLister  mintdom.PassedProductLister
	tokenBlueprintRepo   applicationport.TokenBlueprintGetter
	brandRepo            branddom.Repository
	estimate             MintFundingEstimateExecutor
	companyIDFromContext applicationport.CompanyIDResolver
}

func NewMintFundingEstimateQuery(
	passedProductLister mintdom.PassedProductLister,
	tokenBlueprintRepo applicationport.TokenBlueprintGetter,
	brandRepo branddom.Repository,
	estimate MintFundingEstimateExecutor,
	companyIDFromContext applicationport.CompanyIDResolver,
) *MintFundingEstimateQuery {
	return &MintFundingEstimateQuery{
		passedProductLister:  passedProductLister,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		brandRepo:            brandRepo,
		estimate:             estimate,
		companyIDFromContext: companyIDFromContext,
	}
}

// GetMintFundingEstimate は productionId と tokenBlueprintId から Bubblegum V2 Mint に必要な SOL 見積を構築します.
//
// 処理:
//  1. current company を取得
//  2. TokenBlueprint を取得し company 所有を確認
//  3. Brand を取得し walletAddress を解決
//  4. passed productId 数を内部用 mintQuantity として取得
//  5. read-only の SOL estimate executor を実行
//
// metadataUri は見積条件に含めません。
// metadataUri 未生成の初回 Mint でも見積可能とします。
func (q *MintFundingEstimateQuery) GetMintFundingEstimate(
	ctx context.Context,
	input GetMintFundingEstimateInput,
) (*MintFundingEstimateResult, error) {
	if q == nil || q.passedProductLister == nil || q.tokenBlueprintRepo == nil || q.brandRepo == nil || q.estimate == nil || q.companyIDFromContext == nil {
		return nil, ErrMintFundingEstimateQueryNotConfigured
	}

	if input.ProductionID == "" {
		return nil, fmt.Errorf("%w: productionId is empty", ErrMintFundingEstimateInvalidInput)
	}

	if input.TokenBlueprintID == "" {
		return nil, fmt.Errorf("%w: tokenBlueprintId is empty", ErrMintFundingEstimateInvalidInput)
	}

	companyID := q.companyIDFromContext(ctx)
	if companyID == "" {
		return nil, usecase.ErrCompanyIDMissing
	}

	tokenBlueprint, err := q.tokenBlueprintRepo.GetByID(ctx, input.TokenBlueprintID)
	if err != nil {
		return nil, fmt.Errorf("get tokenBlueprint for mint funding estimate: %w", err)
	}

	if tokenBlueprint == nil {
		return nil, fmt.Errorf("%w: tokenBlueprint not found", tbdom.ErrNotFound)
	}

	if tokenBlueprint.CompanyID != companyID {
		return nil, fmt.Errorf("%w: tokenBlueprint company mismatch", ErrMintFundingEstimateForbidden)
	}

	if tokenBlueprint.BrandID == "" {
		return nil, fmt.Errorf("%w: tokenBlueprint brandId is empty", ErrMintFundingEstimateInvalidInput)
	}

	if tokenBlueprint.Name == "" {
		return nil, fmt.Errorf("%w: tokenBlueprint name is empty", ErrMintFundingEstimateInvalidInput)
	}

	if tokenBlueprint.Symbol == "" {
		return nil, fmt.Errorf("%w: tokenBlueprint symbol is empty", ErrMintFundingEstimateInvalidInput)
	}

	brand, err := q.brandRepo.GetByID(ctx, tokenBlueprint.BrandID)
	if err != nil {
		return nil, fmt.Errorf("get brand for mint funding estimate: %w", err)
	}

	if brand.ID == "" {
		return nil, fmt.Errorf("%w: brand not found", branddom.ErrNotFound)
	}

	if brand.CompanyID != companyID {
		return nil, fmt.Errorf("%w: brand company mismatch", ErrMintFundingEstimateForbidden)
	}

	if brand.ID != tokenBlueprint.BrandID {
		return nil, fmt.Errorf("%w: tokenBlueprint brand mismatch", ErrMintFundingEstimateInvalidInput)
	}

	if brand.WalletAddress == "" {
		return nil, ErrMintFundingEstimateBrandWalletMissing
	}

	passedProductIDs, err := q.passedProductLister.ListPassedProductIDsByProductionID(ctx, input.ProductionID)
	if err != nil {
		return nil, fmt.Errorf("list passed products for mint funding estimate: %w", err)
	}

	mintQuantity := len(passedProductIDs)
	if mintQuantity == 0 {
		return nil, ErrMintFundingEstimateNoPassedProducts
	}

	result, err := q.estimate(
		ctx,
		MintFundingEstimateParams{
			TokenBlueprintID: tokenBlueprint.ID,
			MintQuantity:     mintQuantity,
			ToAddress:        brand.WalletAddress,
			Name:             tokenBlueprint.Name,
			Symbol:           tokenBlueprint.Symbol,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("estimate Bubblegum mint funding: %w", err)
	}

	if result == nil {
		return nil, errors.New("mint funding estimate returned nil result")
	}

	if result.Cluster == "" {
		return nil, errors.New("mint funding estimate cluster is empty")
	}

	if result.Reserve.Address == "" {
		return nil, errors.New("mint funding estimate reserve address is empty")
	}

	if result.Reserve.BalanceLamports == "" {
		return nil, errors.New("mint funding estimate reserve balanceLamports is empty")
	}

	if result.Reserve.MinimumLamports == "" {
		return nil, errors.New("mint funding estimate reserve minimumLamports is empty")
	}

	if result.Estimate.MintTransactionFeePerItemLamports == "" {
		return nil, errors.New("mint funding estimate mintTransactionFeePerItemLamports is empty")
	}

	if result.Estimate.MintTransactionFeeTotalLamports == "" {
		return nil, errors.New("mint funding estimate mintTransactionFeeTotalLamports is empty")
	}

	if result.Estimate.InitialCreationCostLamports == "" {
		return nil, errors.New("mint funding estimate initialCreationCostLamports is empty")
	}

	if result.Estimate.TotalRequiredLamports == "" {
		return nil, errors.New("mint funding estimate totalRequiredLamports is empty")
	}

	if result.Resources.SharedMerkleTreeExists &&
		(result.Resources.SharedMerkleTreeAddress == nil || *result.Resources.SharedMerkleTreeAddress == "") {
		return nil, errors.New("mint funding estimate shared merkle tree address is empty")
	}

	if result.Resources.CoreCollectionExists &&
		(result.Resources.CoreCollectionAddress == nil || *result.Resources.CoreCollectionAddress == "") {
		return nil, errors.New("mint funding estimate core collection address is empty")
	}

	return result, nil
}
