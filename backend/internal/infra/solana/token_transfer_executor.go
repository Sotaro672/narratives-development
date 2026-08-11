// backend/internal/infra/solana/token_transfer_executor.go
package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"google.golang.org/api/idtoken"

	usecase "narratives/internal/application/usecase"
)

var (
	ErrTokenTransferNotConfigured = errors.New(
		"token_transfer_executor: not configured",
	)

	ErrTokenTransferServiceURLEmpty = errors.New(
		"token_transfer_executor: bubblegum service URL is empty",
	)

	ErrTokenTransferInvalidServiceURL = errors.New(
		"token_transfer_executor: bubblegum service URL is invalid",
	)

	ErrTokenTransferProductIDEmpty = errors.New(
		"token_transfer_executor: productId is empty",
	)

	ErrTokenTransferAssetIDEmpty = errors.New(
		"token_transfer_executor: assetId is empty",
	)

	ErrTokenTransferSenderEmpty = errors.New(
		"token_transfer_executor: sender identity is empty",
	)

	ErrTokenTransferSenderAmbiguous = errors.New(
		"token_transfer_executor: both fromAvatarId and fromBrandId are set",
	)

	ErrTokenTransferToAvatarEmpty = errors.New(
		"token_transfer_executor: toAvatarId is empty",
	)

	ErrTokenTransferFromWalletEmpty = errors.New(
		"token_transfer_executor: fromWalletAddress is empty",
	)

	ErrTokenTransferToWalletEmpty = errors.New(
		"token_transfer_executor: toWalletAddress is empty",
	)

	ErrTokenTransferResponseTooLarge = errors.New(
		"token_transfer_executor: bubblegum service response is too large",
	)

	ErrTokenTransferEmptySignature = errors.New(
		"token_transfer_executor: bubblegum service returned empty signature",
	)

	ErrTokenTransferAssetMismatch = errors.New(
		"token_transfer_executor: bubblegum service returned unexpected assetId",
	)
)

const (
	envBubblegumTransferServiceURL = "SOLANA_BUBBLEGUM_SERVICE_URL"

	envBubblegumTransferServiceAudience = "SOLANA_BUBBLEGUM_SERVICE_AUDIENCE"

	bubblegumTransferPath = "/transfer"

	bubblegumTransferRequestTimeout = 45 * time.Second

	maxBubblegumTransferResponseBodyBytes int64 = 256 * 1024
)

// TokenTransferExecutorSolana delegates Bubblegum V2 cNFT transfer execution
// to the internal solana-bubblegum service.
//
// The Go backend does not:
// - load avatar or brand private keys
// - construct Bubblegum instructions
// - fetch DAS proofs
// - sign Solana transactions
// - manage a Solana fee payer
//
// Those responsibilities belong to the internal Bubblegum service.
type TokenTransferExecutorSolana struct {
	httpClient *http.Client
	serviceURL string
	initErr    error
}

var _ usecase.TokenTransferExecutor = (*TokenTransferExecutorSolana)(nil)

// NewTokenTransferExecutorSolana constructs a Bubblegum V2 transfer executor.
//
// The argument is now treated as an optional Bubblegum service URL.
// Existing DI currently calls NewTokenTransferExecutorSolana(""), so an empty
// value resolves from SOLANA_BUBBLEGUM_SERVICE_URL.
//
// The constructor intentionally keeps the existing one-argument signature so
// the current DI can migrate without an additional compile break.
func NewTokenTransferExecutorSolana(
	serviceURL string,
) *TokenTransferExecutorSolana {
	serviceURL = strings.Trim(
		serviceURL,
		" \t\r\n",
	)

	if serviceURL == "" {
		serviceURL = strings.Trim(
			os.Getenv(
				envBubblegumTransferServiceURL,
			),
			" \t\r\n",
		)
	}

	executor := &TokenTransferExecutorSolana{
		serviceURL: serviceURL,
	}

	if serviceURL == "" {
		executor.initErr =
			ErrTokenTransferServiceURLEmpty

		return executor
	}

	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		executor.initErr = fmt.Errorf(
			"%w: %v",
			ErrTokenTransferInvalidServiceURL,
			err,
		)

		return executor
	}

	if parsedURL.Scheme != "http" &&
		parsedURL.Scheme != "https" {
		executor.initErr = fmt.Errorf(
			"%w: unsupported scheme=%s",
			ErrTokenTransferInvalidServiceURL,
			parsedURL.Scheme,
		)

		return executor
	}

	if parsedURL.Host == "" {
		executor.initErr = fmt.Errorf(
			"%w: host is empty",
			ErrTokenTransferInvalidServiceURL,
		)

		return executor
	}

	executor.serviceURL = strings.TrimRight(
		serviceURL,
		"/",
	)

	audience := strings.Trim(
		os.Getenv(
			envBubblegumTransferServiceAudience,
		),
		" \t\r\n",
	)

	if audience == "" {
		executor.httpClient = &http.Client{
			Timeout: bubblegumTransferRequestTimeout,
		}

		return executor
	}

	authenticatedClient, err := idtoken.NewClient(
		context.Background(),
		audience,
	)
	if err != nil {
		executor.initErr = fmt.Errorf(
			"token_transfer_executor: create authenticated HTTP client: %w",
			err,
		)

		return executor
	}

	authenticatedClient.Timeout =
		bubblegumTransferRequestTimeout

	executor.httpClient = authenticatedClient

	return executor
}

type bubblegumTransferRequest struct {
	ProductID string `json:"productId"`

	AssetStandard string `json:"assetStandard"`
	AssetID       string `json:"assetId"`

	FromAvatarID string `json:"fromAvatarId,omitempty"`
	FromBrandID  string `json:"fromBrandId,omitempty"`
	ToAvatarID   string `json:"toAvatarId"`

	BrandID          string `json:"brandId,omitempty"`
	ModelID          string `json:"modelId,omitempty"`
	TokenBlueprintID string `json:"tokenBlueprintId,omitempty"`

	FromWalletAddress string `json:"fromWalletAddress"`
	ToWalletAddress   string `json:"toWalletAddress"`
}

type bubblegumTransferResponse struct {
	Signature string `json:"signature"`
	AssetID   string `json:"assetId,omitempty"`
	Slot      uint64 `json:"slot,omitempty"`
}

type bubblegumTransferErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// ExecuteTransfer delegates a Bubblegum V2 cNFT transfer to the internal
// solana-bubblegum service.
//
// The Bubblegum service is responsible for:
// - resolving the current cNFT owner through DAS
// - resolving the sender signer from Secret Manager
// - fetching the Bubblegum asset proof
// - building transferV2
// - paying the Solana transaction fee
// - signing and submitting the transaction
//
// The Go backend sends only public identifiers and wallet addresses.
// Private keys and signer objects must never be included in this request.
func (e *TokenTransferExecutorSolana) ExecuteTransfer(
	ctx context.Context,
	in usecase.ExecuteTransferInput,
) (
	usecase.ExecuteTransferResult,
	error,
) {
	if e == nil {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferNotConfigured
	}

	if e.initErr != nil {
		return usecase.ExecuteTransferResult{},
			e.initErr
	}

	if e.httpClient == nil ||
		e.serviceURL == "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferNotConfigured
	}

	if in.ProductID == "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferProductIDEmpty
	}

	if in.AssetID == "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferAssetIDEmpty
	}

	if in.FromAvatarID == "" &&
		in.FromBrandID == "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferSenderEmpty
	}

	if in.FromAvatarID != "" &&
		in.FromBrandID != "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferSenderAmbiguous
	}

	if in.ToAvatarID == "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferToAvatarEmpty
	}

	if in.FromWalletAddress == "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferFromWalletEmpty
	}

	if in.ToWalletAddress == "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferToWalletEmpty
	}

	payload := bubblegumTransferRequest{
		ProductID: in.ProductID,

		AssetStandard: "BUBBLEGUM_V2",
		AssetID:       in.AssetID,

		FromAvatarID: in.FromAvatarID,
		FromBrandID:  in.FromBrandID,
		ToAvatarID:   in.ToAvatarID,

		BrandID:          in.BrandID,
		ModelID:          in.ModelID,
		TokenBlueprintID: in.TokenBlueprintID,

		FromWalletAddress: in.FromWalletAddress,
		ToWalletAddress:   in.ToWalletAddress,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return usecase.ExecuteTransferResult{},
			fmt.Errorf(
				"token_transfer_executor: marshal request: %w",
				err,
			)
	}

	requestCtx := ctx
	var cancel context.CancelFunc

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		requestCtx, cancel =
			context.WithTimeout(
				ctx,
				bubblegumTransferRequestTimeout,
			)
		defer cancel()
	}

	endpoint :=
		e.serviceURL +
			bubblegumTransferPath

	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return usecase.ExecuteTransferResult{},
			fmt.Errorf(
				"token_transfer_executor: create request: %w",
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

	log.Printf(
		"[token_transfer_executor] transfer start productId=%s assetId=%s fromAvatarId=%s fromBrandId=%s toAvatarId=%s fromWallet=%s toWallet=%s",
		in.ProductID,
		in.AssetID,
		in.FromAvatarID,
		in.FromBrandID,
		in.ToAvatarID,
		in.FromWalletAddress,
		in.ToWalletAddress,
	)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return usecase.ExecuteTransferResult{},
			fmt.Errorf(
				"token_transfer_executor: bubblegum transfer request failed productId=%s assetId=%s: %w",
				in.ProductID,
				in.AssetID,
				err,
			)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(
			resp.Body,
			maxBubblegumTransferResponseBodyBytes+1,
		),
	)
	if err != nil {
		return usecase.ExecuteTransferResult{},
			fmt.Errorf(
				"token_transfer_executor: read bubblegum transfer response: %w",
				err,
			)
	}

	if int64(len(body)) >
		maxBubblegumTransferResponseBodyBytes {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferResponseTooLarge
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return usecase.ExecuteTransferResult{},
			bubblegumTransferHTTPError(
				resp.StatusCode,
				body,
			)
	}

	var result bubblegumTransferResponse

	if err := json.Unmarshal(
		body,
		&result,
	); err != nil {
		return usecase.ExecuteTransferResult{},
			fmt.Errorf(
				"token_transfer_executor: decode bubblegum transfer response: %w",
				err,
			)
	}

	result.Signature = strings.Trim(
		result.Signature,
		" \t\r\n",
	)

	result.AssetID = strings.Trim(
		result.AssetID,
		" \t\r\n",
	)

	if result.Signature == "" {
		return usecase.ExecuteTransferResult{},
			ErrTokenTransferEmptySignature
	}

	if result.AssetID != "" &&
		result.AssetID != in.AssetID {
		return usecase.ExecuteTransferResult{},
			fmt.Errorf(
				"%w: requested=%s returned=%s",
				ErrTokenTransferAssetMismatch,
				in.AssetID,
				result.AssetID,
			)
	}

	log.Printf(
		"[token_transfer_executor] transfer succeeded productId=%s assetId=%s signature=%s slot=%d",
		in.ProductID,
		in.AssetID,
		result.Signature,
		result.Slot,
	)

	return usecase.ExecuteTransferResult{
		TxSignature: result.Signature,
	}, nil
}

func bubblegumTransferHTTPError(
	statusCode int,
	body []byte,
) error {
	var errorResponse bubblegumTransferErrorResponse

	if len(body) > 0 {
		_ = json.Unmarshal(
			body,
			&errorResponse,
		)
	}

	errorCode := strings.Trim(
		errorResponse.Error,
		" \t\r\n",
	)

	message := strings.Trim(
		errorResponse.Message,
		" \t\r\n",
	)

	if message == "" {
		message = strings.Trim(
			string(body),
			" \t\r\n",
		)
	}

	if len(message) > 1024 {
		message = message[:1024]
	}

	switch {
	case errorCode != "" &&
		message != "":
		return fmt.Errorf(
			"token_transfer_executor: bubblegum service returned status=%d error=%s message=%s",
			statusCode,
			errorCode,
			message,
		)

	case errorCode != "":
		return fmt.Errorf(
			"token_transfer_executor: bubblegum service returned status=%d error=%s",
			statusCode,
			errorCode,
		)

	case message != "":
		return fmt.Errorf(
			"token_transfer_executor: bubblegum service returned status=%d message=%s",
			statusCode,
			message,
		)

	default:
		return fmt.Errorf(
			"token_transfer_executor: bubblegum service returned status=%d",
			statusCode,
		)
	}
}
