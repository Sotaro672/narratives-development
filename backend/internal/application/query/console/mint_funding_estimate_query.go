// backend/internal/application/query/console/mint_funding_estimate_query.go
package query

import (
	"context"
	"errors"
	"fmt"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	mintdom "narratives/internal/domain/mint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

var (
	ErrMintFundingEstimateQueryNotConfigured = errors.New(
		"mint funding estimate query is not configured",
	)

	ErrMintFundingEstimateInvalidInput = errors.New(
		"mint funding estimate input is invalid",
	)

	ErrMintFundingEstimateForbidden = errors.New(
		"mint funding estimate access is forbidden",
	)

	ErrMintFundingEstimateNoPassedProducts = errors.New(
		"no passed products for mint funding estimate",
	)

	ErrMintFundingEstimateBrandWalletMissing = errors.New(
		"brand wallet address is empty",
	)
)

// GetMintFundingEstimateInput は Console から見積を取得するための入力です。
//
// Frontend から Solana 固有値を直接渡させず、
// productionId と tokenBlueprintId のみを受け取ります。
//
// 以下は Backend 側で解決します。
// - mintQuantity: passed product 数
// - toAddress: Brand.WalletAddress
// - name / symbol: TokenBlueprint
//
// metadataUri は見積条件に含めません。
// 初回 Mint 前は metadataUri が未生成でも正常なため、
// solana-bubblegum 側の estimate 処理で見積専用 URI を使用します。
type GetMintFundingEstimateInput struct {
	ProductionID     string `json:"productionId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

// MintFundingEstimateParams は solana-bubblegum の見積処理へ渡す
// application/query 層の入力です。
//
// metadataUri は実 Mint 用の値であり、SOL 見積には使用しません。
// infra/solana の型には依存させません。
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

type MintFundingEstimateFeePayer struct {
	Address         string  `json:"address"`
	BalanceLamports string  `json:"balanceLamports"`
	BalanceSOL      float64 `json:"balanceSol"`
	TargetLamports  string  `json:"targetLamports"`
	TargetSOL       float64 `json:"targetSol"`
}

type MintFundingEstimateResources struct {
	SharedMerkleTreeExists  bool    `json:"sharedMerkleTreeExists"`
	SharedMerkleTreeAddress *string `json:"sharedMerkleTreeAddress"`
	CoreCollectionExists    bool    `json:"coreCollectionExists"`
	CoreCollectionAddress   *string `json:"coreCollectionAddress"`
}

type MintFundingEstimateCosts struct {
	MintTransactionFeePerItemLamports string  `json:"mintTransactionFeePerItemLamports"`
	MintTransactionFeePerItemSOL      float64 `json:"mintTransactionFeePerItemSol"`

	MintTransactionFeeTotalLamports string  `json:"mintTransactionFeeTotalLamports"`
	MintTransactionFeeTotalSOL      float64 `json:"mintTransactionFeeTotalSol"`

	MerkleTreeCreationTransactionFeeLamports string  `json:"merkleTreeCreationTransactionFeeLamports"`
	MerkleTreeCreationTransactionFeeSOL      float64 `json:"merkleTreeCreationTransactionFeeSol"`

	MerkleTreeCreationRentLamports string  `json:"merkleTreeCreationRentLamports"`
	MerkleTreeCreationRentSOL      float64 `json:"merkleTreeCreationRentSol"`

	MerkleTreeCreationCostLamports string  `json:"merkleTreeCreationCostLamports"`
	MerkleTreeCreationCostSOL      float64 `json:"merkleTreeCreationCostSol"`

	CoreCollectionCreationTransactionFeeLamports string  `json:"coreCollectionCreationTransactionFeeLamports"`
	CoreCollectionCreationTransactionFeeSOL      float64 `json:"coreCollectionCreationTransactionFeeSol"`

	CoreCollectionCreationRentLamports string  `json:"coreCollectionCreationRentLamports"`
	CoreCollectionCreationRentSOL      float64 `json:"coreCollectionCreationRentSol"`

	CoreCollectionCreationCostLamports string  `json:"coreCollectionCreationCostLamports"`
	CoreCollectionCreationCostSOL      float64 `json:"coreCollectionCreationCostSol"`

	ProvisioningCostLamports string  `json:"provisioningCostLamports"`
	ProvisioningCostSOL      float64 `json:"provisioningCostSol"`

	EstimatedNetworkCostLamports string  `json:"estimatedNetworkCostLamports"`
	EstimatedNetworkCostSOL      float64 `json:"estimatedNetworkCostSol"`

	RequiredFeePayerBalanceLamports string  `json:"requiredFeePayerBalanceLamports"`
	RequiredFeePayerBalanceSOL      float64 `json:"requiredFeePayerBalanceSol"`

	EstimatedReserveTopUpLamports string  `json:"estimatedReserveTopUpLamports"`
	EstimatedReserveTopUpSOL      float64 `json:"estimatedReserveTopUpSol"`

	ReserveTransferFeeBufferLamports string  `json:"reserveTransferFeeBufferLamports"`
	ReserveTransferFeeBufferSOL      float64 `json:"reserveTransferFeeBufferSol"`

	RequiredReserveForTopUpLamports string  `json:"requiredReserveForTopUpLamports"`
	RequiredReserveForTopUpSOL      float64 `json:"requiredReserveForTopUpSol"`

	Sufficient bool `json:"sufficient"`
}

// MintFundingEstimateResult は Console API が返す見積結果です。
//
// solana-bubblegum service のレスポンスを Backend application 層の
// read model として表現します。
type MintFundingEstimateResult struct {
	Cluster      string `json:"cluster"`
	MintQuantity int    `json:"mintQuantity"`

	Reserve   MintFundingEstimateReserve   `json:"reserve"`
	FeePayer  MintFundingEstimateFeePayer  `json:"feePayer"`
	Resources MintFundingEstimateResources `json:"resources"`
	Estimate  MintFundingEstimateCosts     `json:"estimate"`
}

// MintFundingEstimateExecutor は SOL 見積の実処理を呼び出す関数です。
//
// application/query -> infra/solana の直接依存を避けるため、
// DI 層で MintClient.EstimateMintFunding をラップして注入します。
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
	passedProductLister mintdom.PassedProductLister
	tokenBlueprintRepo  tbdom.RepositoryPort
	brandRepo           branddom.Repository
	estimate            MintFundingEstimateExecutor
}

func NewMintFundingEstimateQuery(
	passedProductLister mintdom.PassedProductLister,
	tokenBlueprintRepo tbdom.RepositoryPort,
	brandRepo branddom.Repository,
	estimate MintFundingEstimateExecutor,
) *MintFundingEstimateQuery {
	return &MintFundingEstimateQuery{
		passedProductLister: passedProductLister,
		tokenBlueprintRepo:  tokenBlueprintRepo,
		brandRepo:           brandRepo,
		estimate:            estimate,
	}
}

// GetMintFundingEstimate は productionId と tokenBlueprintId から
// Bubblegum V2 Mint に必要な SOL 見積を構築します。
//
// 処理:
//  1. current company を取得
//  2. TokenBlueprint を取得し company 所有を確認
//  3. Brand を取得し walletAddress を解決
//  4. passed productId 数を mintQuantity として取得
//  5. read-only の SOL estimate executor を実行
//
// metadataUri は見積条件に含めません。
// metadataUri 未生成の初回 Mint でも見積可能とします。
func (q *MintFundingEstimateQuery) GetMintFundingEstimate(
	ctx context.Context,
	input GetMintFundingEstimateInput,
) (*MintFundingEstimateResult, error) {
	if q == nil ||
		q.passedProductLister == nil ||
		q.tokenBlueprintRepo == nil ||
		q.brandRepo == nil ||
		q.estimate == nil {
		return nil, ErrMintFundingEstimateQueryNotConfigured
	}

	if input.ProductionID == "" {
		return nil, fmt.Errorf(
			"%w: productionId is empty",
			ErrMintFundingEstimateInvalidInput,
		)
	}

	if input.TokenBlueprintID == "" {
		return nil, fmt.Errorf(
			"%w: tokenBlueprintId is empty",
			ErrMintFundingEstimateInvalidInput,
		)
	}

	companyID := usecase.CompanyIDFromContext(ctx)
	if companyID == "" {
		return nil, usecase.ErrCompanyIDMissing
	}

	tokenBlueprint, err := q.tokenBlueprintRepo.GetByID(
		ctx,
		input.TokenBlueprintID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get tokenBlueprint for mint funding estimate: %w",
			err,
		)
	}

	if tokenBlueprint == nil {
		return nil, fmt.Errorf(
			"%w: tokenBlueprint not found",
			tbdom.ErrNotFound,
		)
	}

	if tokenBlueprint.CompanyID != companyID {
		return nil, fmt.Errorf(
			"%w: tokenBlueprint company mismatch",
			ErrMintFundingEstimateForbidden,
		)
	}

	if tokenBlueprint.BrandID == "" {
		return nil, fmt.Errorf(
			"%w: tokenBlueprint brandId is empty",
			ErrMintFundingEstimateInvalidInput,
		)
	}

	if tokenBlueprint.Name == "" {
		return nil, fmt.Errorf(
			"%w: tokenBlueprint name is empty",
			ErrMintFundingEstimateInvalidInput,
		)
	}

	if tokenBlueprint.Symbol == "" {
		return nil, fmt.Errorf(
			"%w: tokenBlueprint symbol is empty",
			ErrMintFundingEstimateInvalidInput,
		)
	}

	brand, err := q.brandRepo.GetByID(
		ctx,
		tokenBlueprint.BrandID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get brand for mint funding estimate: %w",
			err,
		)
	}

	if brand.ID == "" {
		return nil, fmt.Errorf(
			"%w: brand not found",
			branddom.ErrNotFound,
		)
	}

	if brand.CompanyID != companyID {
		return nil, fmt.Errorf(
			"%w: brand company mismatch",
			ErrMintFundingEstimateForbidden,
		)
	}

	if brand.ID != tokenBlueprint.BrandID {
		return nil, fmt.Errorf(
			"%w: tokenBlueprint brand mismatch",
			ErrMintFundingEstimateInvalidInput,
		)
	}

	if brand.WalletAddress == "" {
		return nil, ErrMintFundingEstimateBrandWalletMissing
	}

	passedProductIDs, err := q.passedProductLister.ListPassedProductIDsByProductionID(
		ctx,
		input.ProductionID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list passed products for mint funding estimate: %w",
			err,
		)
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
		return nil, fmt.Errorf(
			"estimate Bubblegum mint funding: %w",
			err,
		)
	}

	if result == nil {
		return nil, errors.New(
			"mint funding estimate returned nil result",
		)
	}

	if result.Cluster == "" {
		return nil, errors.New(
			"mint funding estimate cluster is empty",
		)
	}

	if result.MintQuantity != mintQuantity {
		return nil, fmt.Errorf(
			"mint funding estimate quantity mismatch: expected=%d actual=%d",
			mintQuantity,
			result.MintQuantity,
		)
	}

	if result.Reserve.Address == "" {
		return nil, errors.New(
			"mint funding estimate reserve address is empty",
		)
	}

	if result.FeePayer.Address == "" {
		return nil, errors.New(
			"mint funding estimate fee payer address is empty",
		)
	}

	if result.Resources.SharedMerkleTreeExists &&
		(result.Resources.SharedMerkleTreeAddress == nil ||
			*result.Resources.SharedMerkleTreeAddress == "") {
		return nil, errors.New(
			"mint funding estimate shared merkle tree address is empty",
		)
	}

	if result.Resources.CoreCollectionExists &&
		(result.Resources.CoreCollectionAddress == nil ||
			*result.Resources.CoreCollectionAddress == "") {
		return nil, errors.New(
			"mint funding estimate core collection address is empty",
		)
	}

	return result, nil
}
