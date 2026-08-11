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
	envBubblegumServiceURL = "SOLANA_BUBBLEGUM_SERVICE_URL"

	envBubblegumServiceAudience = "SOLANA_BUBBLEGUM_SERVICE_AUDIENCE"

	envBubblegumMintAuthorityPublicKey = "SOLANA_BUBBLEGUM_MINT_AUTHORITY_PUBLIC_KEY"

	bubblegumMintPath = "/mint"

	bubblegumRequestTimeout = 45 * time.Second

	maxBubblegumResponseBodyBytes int64 = 256 * 1024
)

// MintClient は Bubblegum V2 internal service を呼び出す
// tokendom.MintAuthorityWalletPort の実装です。
//
// IMPORTANT:
//   - Go backend では Solana private key を保持しません。
//   - Go backend から private key を送信しません。
//   - Bubblegum V2 mint signer / fee payer / tree authority は
//     solana-bubblegum service 側で解決します。
//   - 1 productId = 1 cNFT mint とします。
//   - productId を idempotency key として internal service に渡します。
type MintClient struct {
	httpClient *http.Client
	serviceURL string
}

// インターフェース実装チェック。
var _ tokendom.MintAuthorityWalletPort = (*MintClient)(nil)

// NewMintClient は環境変数から Bubblegum V2 internal service client を生成します.
//
// Required:
// - SOLANA_BUBBLEGUM_SERVICE_URL
//
// Optional:
// - SOLANA_BUBBLEGUM_SERVICE_AUDIENCE
//
// SOLANA_BUBBLEGUM_SERVICE_AUDIENCE が設定されている場合は、
// Google Cloud ID Token を付与する HTTP client を使用します。
//
// ローカル開発で認証なしの internal service を使う場合は、
// SOLANA_BUBBLEGUM_SERVICE_AUDIENCE を設定しません。
func NewMintClient(
	ctx context.Context,
) (*MintClient, error) {
	serviceURL := os.Getenv(envBubblegumServiceURL)
	if serviceURL == "" {
		return nil, fmt.Errorf(
			"%s is empty",
			envBubblegumServiceURL,
		)
	}

	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return nil, fmt.Errorf(
			"parse %s: %w",
			envBubblegumServiceURL,
			err,
		)
	}

	if parsedURL == nil ||
		parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" &&
			parsedURL.Scheme != "https") {
		return nil, fmt.Errorf(
			"%s must be an absolute http or https URL",
			envBubblegumServiceURL,
		)
	}

	serviceURL = strings.TrimRight(serviceURL, "/")

	audience := os.Getenv(envBubblegumServiceAudience)

	var httpClient *http.Client

	if audience != "" {
		authenticatedClient, err := idtoken.NewClient(
			ctx,
			audience,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create bubblegum authenticated http client: %w",
				err,
			)
		}

		authenticatedClient.Timeout = bubblegumRequestTimeout
		httpClient = authenticatedClient
	} else {
		httpClient = &http.Client{
			Timeout: bubblegumRequestTimeout,
		}
	}

	return &MintClient{
		httpClient: httpClient,
		serviceURL: serviceURL,
	}, nil
}

// PublicKey は Bubblegum V2 mint authority の公開鍵を返します.
//
// private key は solana-bubblegum service / Secret Manager 側で管理し、
// Go backend には保持しません。
func (c *MintClient) PublicKey(
	ctx context.Context,
) (string, error) {
	_ = ctx

	if c == nil {
		return "", errors.New(
			"bubblegum mint client is nil",
		)
	}

	publicKey := os.Getenv(
		envBubblegumMintAuthorityPublicKey,
	)
	if publicKey == "" {
		return "", fmt.Errorf(
			"%s is empty",
			envBubblegumMintAuthorityPublicKey,
		)
	}

	return publicKey, nil
}

type bubblegumMintRequest struct {
	ProductID string `json:"productId"`

	ToAddress string `json:"toAddress"`

	Name string `json:"name"`

	Symbol string `json:"symbol"`

	MetadataURI string `json:"metadataUri"`

	AssetStandard string `json:"assetStandard"`
}

type bubblegumMintResponse struct {
	Signature string `json:"signature"`

	AssetID string `json:"assetId"`

	TreeAddress string `json:"treeAddress"`

	LeafIndex uint64 `json:"leafIndex"`

	Slot uint64 `json:"slot"`
}

type bubblegumErrorResponse struct {
	Error string `json:"error"`

	Message string `json:"message"`
}

// MintToken は Bubblegum V2 internal service を経由して
// productId 1件分の cNFT を mint します.
//
// POST {SOLANA_BUBBLEGUM_SERVICE_URL}/mint
//
// Request:
//
//	{
//	  "productId": "...",
//	  "toAddress": "...",
//	  "name": "...",
//	  "symbol": "...",
//	  "metadataUri": "...",
//	  "assetStandard": "BUBBLEGUM_V2"
//	}
//
// Response:
//
//	{
//	  "signature": "...",
//	  "assetId": "...",
//	  "treeAddress": "...",
//	  "leafIndex": 0,
//	  "slot": 0
//	}
//
// IMPORTANT:
// - productId は internal service 側の idempotency key として使用します。
// - leafIndex=0 は正当な値です。
// - HTTP transport error 時に Go 側で POST retry はしません。
// - retry / 重複排除は productId を使って service 側で保証します。
// - DAS indexing delay を理由に再mintしてはいけません。
func (c *MintClient) MintToken(
	ctx context.Context,
	params tokendom.MintParams,
) (*tokendom.MintResult, error) {
	if c == nil {
		return nil, errors.New(
			"bubblegum mint client is nil",
		)
	}

	if c.httpClient == nil {
		return nil, errors.New(
			"bubblegum mint http client is nil",
		)
	}

	if c.serviceURL == "" {
		return nil, errors.New(
			"bubblegum mint service URL is empty",
		)
	}

	if params.ProductID == "" {
		return nil, errors.New(
			"ProductID is empty",
		)
	}

	if params.ToAddress == "" {
		return nil, errors.New(
			"ToAddress is empty",
		)
	}

	if params.Name == "" {
		return nil, errors.New(
			"Name is empty",
		)
	}

	if params.Symbol == "" {
		return nil, errors.New(
			"Symbol is empty",
		)
	}

	if params.MetadataURI == "" {
		return nil, errors.New(
			"MetadataURI is empty",
		)
	}

	if params.Amount != 0 && params.Amount != 1 {
		return nil, fmt.Errorf(
			"Bubblegum V2 mint requires Amount=1: amount=%d",
			params.Amount,
		)
	}

	requestBody := bubblegumMintRequest{
		ProductID: params.ProductID,

		ToAddress: params.ToAddress,

		Name: params.Name,

		Symbol: params.Symbol,

		MetadataURI: params.MetadataURI,

		AssetStandard: string(
			tokendom.AssetStandardBubblegumV2,
		),
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal bubblegum mint request: %w",
			err,
		)
	}

	endpoint := c.serviceURL + bubblegumMintPath

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create bubblegum mint request: %w",
			err,
		)
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	req.Header.Set(
		"Accept",
		"application/json",
	)

	req.Header.Set(
		"Idempotency-Key",
		params.ProductID,
	)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"call bubblegum mint service: %w",
			err,
		)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(
		io.LimitReader(
			response.Body,
			maxBubblegumResponseBodyBytes+1,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"read bubblegum mint response: %w",
			err,
		)
	}

	if int64(len(responseBody)) >
		maxBubblegumResponseBodyBytes {
		return nil, errors.New(
			"bubblegum mint response body is too large",
		)
	}

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return nil, decodeBubblegumServiceError(
			response.StatusCode,
			responseBody,
		)
	}

	var result bubblegumMintResponse

	if err := json.Unmarshal(
		responseBody,
		&result,
	); err != nil {
		return nil, fmt.Errorf(
			"decode bubblegum mint response: %w",
			err,
		)
	}

	if result.Signature == "" {
		return nil, errors.New(
			"bubblegum mint response signature is empty",
		)
	}

	if result.AssetID == "" {
		return nil, errors.New(
			"bubblegum mint response assetId is empty",
		)
	}

	if result.TreeAddress == "" {
		return nil, errors.New(
			"bubblegum mint response treeAddress is empty",
		)
	}

	return &tokendom.MintResult{
		Signature: result.Signature,

		AssetID: result.AssetID,

		TreeAddress: result.TreeAddress,

		LeafIndex: result.LeafIndex,

		Slot: result.Slot,
	}, nil
}

func decodeBubblegumServiceError(
	statusCode int,
	body []byte,
) error {
	var serviceErr bubblegumErrorResponse

	if len(body) > 0 {
		if err := json.Unmarshal(
			body,
			&serviceErr,
		); err == nil {
			switch {
			case serviceErr.Error != "" &&
				serviceErr.Message != "":
				return fmt.Errorf(
					"bubblegum mint service status=%d error=%s message=%s",
					statusCode,
					serviceErr.Error,
					serviceErr.Message,
				)

			case serviceErr.Error != "":
				return fmt.Errorf(
					"bubblegum mint service status=%d error=%s",
					statusCode,
					serviceErr.Error,
				)

			case serviceErr.Message != "":
				return fmt.Errorf(
					"bubblegum mint service status=%d message=%s",
					statusCode,
					serviceErr.Message,
				)
			}
		}
	}

	return fmt.Errorf(
		"bubblegum mint service returned status=%d",
		statusCode,
	)
}
