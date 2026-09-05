// backend/internal/infra/solana/mint_client.go
package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"google.golang.org/api/idtoken"

	tokendom "narratives/internal/domain/token"
)

const (
	envBubblegumServiceURL             = "SOLANA_BUBBLEGUM_SERVICE_URL"
	envBubblegumServiceAudience        = "SOLANA_BUBBLEGUM_SERVICE_AUDIENCE"
	envBubblegumMintAuthorityPublicKey = "SOLANA_BUBBLEGUM_MINT_AUTHORITY_PUBLIC_KEY"

	bubblegumMintPath           = "/mint"
	bubblegumEstimatePath       = "/estimate"
	bubblegumReserveBalancePath = "/reserve-balance"

	bubblegumRequestTimeout             = 45 * time.Second
	maxBubblegumResponseBodyBytes int64 = 256 * 1024
)

// MintClient は Bubblegum V2 internal service を呼び出す
// tokendom.MintAuthorityWalletPort の実装です。
//
// IMPORTANT:
//   - Go backend では Solana private key を保持しません。
//   - Go backend から private key を送信しません。
//   - Bubblegum V2 mint signer / fee payer / tree authority は solana-bubblegum service 側で解決します。
//   - 1 productId = 1 cNFT mint とします。
//   - productId を idempotency key として internal service に渡します。
type MintClient struct {
	httpClient *http.Client
	serviceURL string
}

var _ tokendom.MintAuthorityWalletPort = (*MintClient)(nil)

// NewMintClient は環境変数から Bubblegum V2 internal service client を生成します.
//
// Required:
// - SOLANA_BUBBLEGUM_SERVICE_URL
//
// Optional:
// - SOLANA_BUBBLEGUM_SERVICE_AUDIENCE
//
// SOLANA_BUBBLEGUM_SERVICE_AUDIENCE が設定されている場合は Google Cloud ID Token を付与します。
func NewMintClient(ctx context.Context) (*MintClient, error) {
	serviceURL := os.Getenv(envBubblegumServiceURL)
	if serviceURL == "" {
		return nil, fmt.Errorf("%s is empty", envBubblegumServiceURL)
	}

	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", envBubblegumServiceURL, err)
	}
	if parsedURL == nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("%s must be an absolute http or https URL", envBubblegumServiceURL)
	}

	serviceURL = strings.TrimRight(serviceURL, "/")
	audience := os.Getenv(envBubblegumServiceAudience)

	var httpClient *http.Client
	if audience != "" {
		authenticatedClient, err := idtoken.NewClient(ctx, audience)
		if err != nil {
			return nil, fmt.Errorf("create bubblegum authenticated http client: %w", err)
		}
		authenticatedClient.Timeout = bubblegumRequestTimeout
		httpClient = authenticatedClient
	} else {
		httpClient = &http.Client{Timeout: bubblegumRequestTimeout}
	}

	return &MintClient{httpClient: httpClient, serviceURL: serviceURL}, nil
}

// PublicKey は Bubblegum V2 mint authority の公開鍵を返します。
// private key は solana-bubblegum service / Secret Manager 側で管理します。
func (c *MintClient) PublicKey(ctx context.Context) (string, error) {
	_ = ctx
	if c == nil {
		return "", errors.New("bubblegum mint client is nil")
	}

	publicKey := os.Getenv(envBubblegumMintAuthorityPublicKey)
	if publicKey == "" {
		return "", fmt.Errorf("%s is empty", envBubblegumMintAuthorityPublicKey)
	}
	return publicKey, nil
}

type bubblegumMintRequest struct {
	ProductID        string `json:"productId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
	BrandID          string `json:"brandId"`
	ToAddress        string `json:"toAddress"`
	Name             string `json:"name"`
	Symbol           string `json:"symbol"`
	MetadataURI      string `json:"metadataUri"`
}

type bubblegumMintResponse struct {
	Signature             string `json:"signature"`
	AssetStandard         string `json:"assetStandard"`
	Cluster               string `json:"cluster"`
	AssetID               string `json:"assetId"`
	TreeAddress           string `json:"treeAddress"`
	LeafIndex             uint64 `json:"leafIndex"`
	CoreCollectionAddress string `json:"coreCollectionAddress"`
	Slot                  uint64 `json:"slot"`
}

// MintFundingEstimateParams は solana-bubblegum /estimate に渡す入力です。
// metadataUri は初回Mint前に未生成でも正常なため、見積入力には含めません。
// mintQuantity はSOL見積計算に必要な内部入力としてsolana-bubblegum serviceへ渡します。
type MintFundingEstimateParams struct {
	TokenBlueprintID string
	MintQuantity     int
	ToAddress        string
	Name             string
	Symbol           string
}

type bubblegumMintFundingEstimateRequest struct {
	TokenBlueprintID string `json:"tokenBlueprintId"`
	MintQuantity     int    `json:"mintQuantity"`
	ToAddress        string `json:"toAddress"`
	Name             string `json:"name"`
	Symbol           string `json:"symbol"`
}

type ReserveBalanceResult struct {
	Cluster         string  `json:"cluster"`
	Address         string  `json:"address"`
	BalanceLamports string  `json:"balanceLamports"`
	BalanceSOL      float64 `json:"balanceSol"`
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

// MintFundingEstimateCosts はConsoleへ返すSOL見積の公開費用です。
//
// InitialCreationCost:
// - Shared Merkle Tree初回作成費
// - Core Collection初回作成費
// の合計。
//
// TotalRequired:
// - Mint手数料合計
// - InitialCreationCost
// の合計。
//
// Fee Payer目標残高、Reserve補充量、Reserve最低残高などのfunding policy内部値は公開しません。
type MintFundingEstimateCosts struct {
	MintTransactionFeePerItemLamports string  `json:"mintTransactionFeePerItemLamports"`
	MintTransactionFeePerItemSOL      float64 `json:"mintTransactionFeePerItemSol"`

	MintTransactionFeeTotalLamports string  `json:"mintTransactionFeeTotalLamports"`
	MintTransactionFeeTotalSOL      float64 `json:"mintTransactionFeeTotalSol"`

	InitialCreationCostLamports string  `json:"initialCreationCostLamports"`
	InitialCreationCostSOL      float64 `json:"initialCreationCostSol"`

	TotalRequiredLamports string  `json:"totalRequiredLamports"`
	TotalRequiredSOL      float64 `json:"totalRequiredSol"`

	Sufficient bool `json:"sufficient"`
}

type MintFundingEstimateResult struct {
	Cluster   string                       `json:"cluster"`
	Reserve   MintFundingEstimateReserve   `json:"reserve"`
	Resources MintFundingEstimateResources `json:"resources"`
	Estimate  MintFundingEstimateCosts     `json:"estimate"`
}

type bubblegumErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetReserveBalance は Bubblegum V2 internal service の /reserve-balance を呼び、Reserve Wallet の現在残高を取得します。
//
// IMPORTANT:
//   - read-only です。
//   - Reserve から SOL を送金しません。
//   - Fee Payer を補充しません。
//   - Mint / Merkle Tree / Core Collection の作成を行いません。
func (c *MintClient) GetReserveBalance(ctx context.Context) (*ReserveBalanceResult, error) {
	if c == nil {
		return nil, errors.New("bubblegum mint client is nil")
	}
	if c.httpClient == nil {
		return nil, errors.New("bubblegum mint http client is nil")
	}
	if c.serviceURL == "" {
		return nil, errors.New("bubblegum mint service URL is empty")
	}

	endpoint := c.serviceURL + bubblegumReserveBalancePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create bubblegum reserve balance request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call bubblegum reserve balance service: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxBubblegumResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read bubblegum reserve balance response: %w", err)
	}
	if int64(len(responseBody)) > maxBubblegumResponseBodyBytes {
		return nil, errors.New("bubblegum reserve balance response body is too large")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, decodeBubblegumServiceError(response.StatusCode, responseBody)
	}

	var result ReserveBalanceResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode bubblegum reserve balance response: %w", err)
	}
	if result.Cluster == "" {
		return nil, errors.New("bubblegum reserve balance response cluster is empty")
	}
	if result.Address == "" {
		return nil, errors.New("bubblegum reserve balance response address is empty")
	}
	if result.BalanceLamports == "" {
		return nil, errors.New("bubblegum reserve balance response balanceLamports is empty")
	}

	return &result, nil
}

// EstimateMintFunding は Bubblegum V2 internal service の /estimate を呼び、
// Reserve Wallet残高とMintに必要なSOL見積を取得します。
//
// IMPORTANT:
//   - Mintを実行しません。
//   - ReserveからSOLを送金しません。
//   - Merkle Tree / Core Collection / cNFTを作成しません。
//   - metadataUriの生成・取得・uploadを行いません。
//   - mintQuantityは見積入力には使用しますがresponseには公開しません。
//   - Fee Payer残高、Fee Payer目標残高、Reserve補充量などのfunding policy内部値はresponseには公開しません。
//   - Idempotency-Keyは不要です。
func (c *MintClient) EstimateMintFunding(ctx context.Context, params MintFundingEstimateParams) (*MintFundingEstimateResult, error) {
	if c == nil {
		return nil, errors.New("bubblegum mint client is nil")
	}
	if c.httpClient == nil {
		return nil, errors.New("bubblegum mint http client is nil")
	}
	if c.serviceURL == "" {
		return nil, errors.New("bubblegum mint service URL is empty")
	}
	if params.TokenBlueprintID == "" {
		return nil, errors.New("TokenBlueprintID is empty")
	}
	if params.MintQuantity <= 0 {
		return nil, fmt.Errorf("MintQuantity must be greater than 0: mintQuantity=%d", params.MintQuantity)
	}
	if params.ToAddress == "" {
		return nil, errors.New("ToAddress is empty")
	}
	if params.Name == "" {
		return nil, errors.New("Name is empty")
	}
	if params.Symbol == "" {
		return nil, errors.New("Symbol is empty")
	}

	requestBody := bubblegumMintFundingEstimateRequest{
		TokenBlueprintID: params.TokenBlueprintID,
		MintQuantity:     params.MintQuantity,
		ToAddress:        params.ToAddress,
		Name:             params.Name,
		Symbol:           params.Symbol,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal bubblegum mint funding estimate request: %w", err)
	}

	endpoint := c.serviceURL + bubblegumEstimatePath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create bubblegum mint funding estimate request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call bubblegum mint funding estimate service: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxBubblegumResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read bubblegum mint funding estimate response: %w", err)
	}
	if int64(len(responseBody)) > maxBubblegumResponseBodyBytes {
		return nil, errors.New("bubblegum mint funding estimate response body is too large")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, decodeBubblegumServiceError(response.StatusCode, responseBody)
	}

	var result MintFundingEstimateResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode bubblegum mint funding estimate response: %w", err)
	}

	if result.Cluster == "" {
		return nil, errors.New("bubblegum mint funding estimate response cluster is empty")
	}
	if result.Reserve.Address == "" {
		return nil, errors.New("bubblegum mint funding estimate response reserve address is empty")
	}
	if result.Reserve.BalanceLamports == "" {
		return nil, errors.New("bubblegum mint funding estimate response reserve balanceLamports is empty")
	}
	if result.Reserve.MinimumLamports == "" {
		return nil, errors.New("bubblegum mint funding estimate response reserve minimumLamports is empty")
	}
	if result.Estimate.MintTransactionFeePerItemLamports == "" {
		return nil, errors.New("bubblegum mint funding estimate response mintTransactionFeePerItemLamports is empty")
	}
	if result.Estimate.MintTransactionFeeTotalLamports == "" {
		return nil, errors.New("bubblegum mint funding estimate response mintTransactionFeeTotalLamports is empty")
	}
	if result.Estimate.InitialCreationCostLamports == "" {
		return nil, errors.New("bubblegum mint funding estimate response initialCreationCostLamports is empty")
	}
	if result.Estimate.TotalRequiredLamports == "" {
		return nil, errors.New("bubblegum mint funding estimate response totalRequiredLamports is empty")
	}

	return &result, nil
}

// MintToken は Bubblegum V2 internal service を経由して productId 1件分の cNFT を mint します.
//
// POST {SOLANA_BUBBLEGUM_SERVICE_URL}/mint
//
// Request:
//
//	{
//	  "productId": "...",
//	  "tokenBlueprintId": "...",
//	  "brandId": "...",
//	  "toAddress": "...",
//	  "name": "...",
//	  "symbol": "...",
//	  "metadataUri": "..."
//	}
//
// IMPORTANT:
// - 実MintではmetadataUriを必須とします。
// - productId は internal service 側の idempotency key として使用します。
// - tokenBlueprintId は MPL Core Collection の解決に使用します。
// - toAddress は cNFT の leaf owner として使用します。
// - HTTP transport error 時に Go 側で POST retry はしません。
func (c *MintClient) MintToken(ctx context.Context, params tokendom.MintParams) (*tokendom.MintResult, error) {
	if c == nil {
		return nil, errors.New("bubblegum mint client is nil")
	}
	if c.httpClient == nil {
		return nil, errors.New("bubblegum mint http client is nil")
	}
	if c.serviceURL == "" {
		return nil, errors.New("bubblegum mint service URL is empty")
	}
	if params.ProductID == "" {
		return nil, errors.New("ProductID is empty")
	}
	if params.TokenBlueprintID == "" {
		return nil, errors.New("TokenBlueprintID is empty")
	}
	if params.BrandID == "" {
		return nil, errors.New("BrandID is empty")
	}
	if params.ToAddress == "" {
		return nil, errors.New("ToAddress is empty")
	}
	if params.Name == "" {
		return nil, errors.New("Name is empty")
	}
	if params.Symbol == "" {
		return nil, errors.New("Symbol is empty")
	}
	if params.MetadataURI == "" {
		return nil, errors.New("MetadataURI is empty")
	}
	if params.Amount != 0 && params.Amount != 1 {
		return nil, fmt.Errorf("Bubblegum V2 mint requires Amount=1: amount=%d", params.Amount)
	}

	requestBody := bubblegumMintRequest{
		ProductID:        params.ProductID,
		TokenBlueprintID: params.TokenBlueprintID,
		BrandID:          params.BrandID,
		ToAddress:        params.ToAddress,
		Name:             params.Name,
		Symbol:           params.Symbol,
		MetadataURI:      params.MetadataURI,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal bubblegum mint request: %w", err)
	}

	endpoint := c.serviceURL + bubblegumMintPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create bubblegum mint request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Idempotency-Key", params.ProductID)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call bubblegum mint service: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxBubblegumResponseBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read bubblegum mint response: %w", err)
	}
	if int64(len(responseBody)) > maxBubblegumResponseBodyBytes {
		return nil, errors.New("bubblegum mint response body is too large")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, decodeBubblegumServiceError(response.StatusCode, responseBody)
	}

	var result bubblegumMintResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode bubblegum mint response: %w", err)
	}
	if result.Signature == "" {
		return nil, errors.New("bubblegum mint response signature is empty")
	}
	if result.AssetStandard == "" {
		return nil, errors.New("bubblegum mint response assetStandard is empty")
	}

	assetStandard, err := mapBubblegumAssetStandard(result.AssetStandard)
	if err != nil {
		return nil, err
	}

	if result.Cluster == "" {
		return nil, errors.New("bubblegum mint response cluster is empty")
	}
	if result.AssetID == "" {
		return nil, errors.New("bubblegum mint response assetId is empty")
	}
	if result.TreeAddress == "" {
		return nil, errors.New("bubblegum mint response treeAddress is empty")
	}
	if result.CoreCollectionAddress == "" {
		return nil, errors.New("bubblegum mint response coreCollectionAddress is empty")
	}

	return &tokendom.MintResult{
		Signature:             result.Signature,
		AssetStandard:         assetStandard,
		Cluster:               result.Cluster,
		AssetID:               result.AssetID,
		TreeAddress:           result.TreeAddress,
		LeafIndex:             result.LeafIndex,
		CoreCollectionAddress: result.CoreCollectionAddress,
		Slot:                  result.Slot,
	}, nil
}

func mapBubblegumAssetStandard(value string) (tokendom.AssetStandard, error) {
	switch value {
	case "bubblegum-v2":
		return tokendom.AssetStandardBubblegumV2, nil
	default:
		return "", fmt.Errorf("unsupported bubblegum assetStandard: %s", value)
	}
}

func decodeBubblegumServiceError(statusCode int, body []byte) error {
	var serviceErr bubblegumErrorResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &serviceErr); err == nil {
			switch {
			case serviceErr.Error != "" && serviceErr.Message != "":
				return fmt.Errorf("bubblegum service status=%d error=%s message=%s", statusCode, serviceErr.Error, serviceErr.Message)
			case serviceErr.Error != "":
				return fmt.Errorf("bubblegum service status=%d error=%s", statusCode, serviceErr.Error)
			case serviceErr.Message != "":
				return fmt.Errorf("bubblegum service status=%d message=%s", statusCode, serviceErr.Message)
			}
		}
	}

	return fmt.Errorf("bubblegum service returned status=%d", statusCode)
}
