// backend/internal/infra/solana/wallet_reader.go
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
)

var (
	ErrOnchainWalletReaderNotConfigured      = errors.New("solana wallet reader: not configured")
	ErrOnchainWalletReaderServiceURLEmpty    = errors.New("solana wallet reader: bubblegum service URL is empty")
	ErrOnchainWalletReaderInvalidServiceURL  = errors.New("solana wallet reader: bubblegum service URL is invalid")
	ErrOnchainWalletReaderWalletAddressEmpty = errors.New("solana wallet reader: walletAddress is empty")
	ErrOnchainWalletReaderResponseTooLarge   = errors.New("solana wallet reader: bubblegum service response is too large")
)

const (
	envBubblegumWalletReaderServiceURL      = "SOLANA_BUBBLEGUM_SERVICE_URL"
	envBubblegumWalletReaderServiceAudience = "SOLANA_BUBBLEGUM_SERVICE_AUDIENCE"

	bubblegumOwnedAssetsPath = "/owned-assets"

	defaultOnchainWalletReaderHTTPTimeout = 20 * time.Second

	maxOnchainWalletReaderResponseBodyBytes int64 = 512 * 1024
)

// OnchainWalletReaderImpl reads Bubblegum V2 cNFT ownership through
// the internal solana-bubblegum service.
//
// The Go backend does not use getTokenAccountsByOwner for cNFT ownership.
// Current ownership is resolved through DAS by the Bubblegum service.
type OnchainWalletReaderImpl struct {
	HTTPClient *http.Client
	ServiceURL string
	Timeout    time.Duration

	initErr error
}

// NewOnchainWalletReaderDevnet keeps the existing constructor name for DI
// compatibility. The actual Solana cluster and DAS endpoint are managed by
// the internal Bubblegum service.
//
// Required:
// SOLANA_BUBBLEGUM_SERVICE_URL
//
// Optional:
// SOLANA_BUBBLEGUM_SERVICE_AUDIENCE
func NewOnchainWalletReaderDevnet() *OnchainWalletReaderImpl {
	timeout := defaultOnchainWalletReaderHTTPTimeout
	serviceURL := strings.Trim(os.Getenv(envBubblegumWalletReaderServiceURL), " \t\r\n")

	reader := &OnchainWalletReaderImpl{
		ServiceURL: serviceURL,
		Timeout:    timeout,
	}

	if serviceURL == "" {
		reader.initErr = ErrOnchainWalletReaderServiceURLEmpty
		return reader
	}

	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		reader.initErr = fmt.Errorf("%w: %v", ErrOnchainWalletReaderInvalidServiceURL, err)
		return reader
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		reader.initErr = fmt.Errorf(
			"%w: unsupported scheme=%s",
			ErrOnchainWalletReaderInvalidServiceURL,
			parsedURL.Scheme,
		)
		return reader
	}

	if parsedURL.Host == "" {
		reader.initErr = fmt.Errorf("%w: host is empty", ErrOnchainWalletReaderInvalidServiceURL)
		return reader
	}

	reader.ServiceURL = strings.TrimRight(serviceURL, "/")

	audience := strings.Trim(os.Getenv(envBubblegumWalletReaderServiceAudience), " \t\r\n")
	if audience == "" {
		reader.HTTPClient = &http.Client{Timeout: timeout}
		return reader
	}

	authenticatedClient, err := idtoken.NewClient(context.Background(), audience)
	if err != nil {
		reader.initErr = fmt.Errorf(
			"solana wallet reader: create authenticated HTTP client: %w",
			err,
		)
		return reader
	}

	authenticatedClient.Timeout = timeout
	reader.HTTPClient = authenticatedClient

	return reader
}

type bubblegumOwnedAssetsRequest struct {
	AssetStandard string `json:"assetStandard"`
	WalletAddress string `json:"walletAddress"`
}

type bubblegumOwnedAssetsResponse struct {
	WalletAddress string   `json:"walletAddress,omitempty"`
	AssetIDs      []string `json:"assetIds"`
}

type bubblegumOwnedAssetsErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// ListOwnedAssetIDs returns Bubblegum V2 cNFT asset IDs currently owned by
// the specified wallet.
//
// DAS is the source of truth. Firestore wallet.assetIds is only a synchronized
// local projection.
func (r *OnchainWalletReaderImpl) ListOwnedAssetIDs(
	ctx context.Context,
	walletAddress string,
) ([]string, error) {
	if r == nil {
		return nil, ErrOnchainWalletReaderNotConfigured
	}
	if r.initErr != nil {
		return nil, r.initErr
	}
	if r.HTTPClient == nil || r.ServiceURL == "" {
		return nil, ErrOnchainWalletReaderNotConfigured
	}

	walletAddress = strings.Trim(walletAddress, " \t\r\n")
	if walletAddress == "" {
		return nil, ErrOnchainWalletReaderWalletAddressEmpty
	}

	payload := bubblegumOwnedAssetsRequest{
		AssetStandard: "BUBBLEGUM_V2",
		WalletAddress: walletAddress,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("solana wallet reader: marshal request: %w", err)
	}

	requestCtx := ctx
	var cancel context.CancelFunc

	if _, hasDeadline := ctx.Deadline(); !hasDeadline && r.Timeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}

	endpoint := r.ServiceURL + bubblegumOwnedAssetsPath

	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("solana wallet reader: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"solana wallet reader: bubblegum owned-assets request failed walletAddress=%s: %w",
			walletAddress,
			err,
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(
			resp.Body,
			maxOnchainWalletReaderResponseBodyBytes+1,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("solana wallet reader: read bubblegum response: %w", err)
	}

	if int64(len(body)) > maxOnchainWalletReaderResponseBodyBytes {
		return nil, ErrOnchainWalletReaderResponseTooLarge
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, bubblegumOwnedAssetsHTTPError(resp.StatusCode, body)
	}

	var serviceResult bubblegumOwnedAssetsResponse

	if err := json.Unmarshal(body, &serviceResult); err != nil {
		return nil, fmt.Errorf("solana wallet reader: decode bubblegum response: %w", err)
	}

	return normalizeAssetIDs(serviceResult.AssetIDs), nil
}

func normalizeAssetIDs(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		assetID := strings.Trim(value, " \t\r\n")
		if assetID == "" {
			continue
		}

		if _, ok := seen[assetID]; ok {
			continue
		}

		seen[assetID] = struct{}{}
		result = append(result, assetID)
	}

	return result
}

func bubblegumOwnedAssetsHTTPError(
	statusCode int,
	body []byte,
) error {
	var errorResponse bubblegumOwnedAssetsErrorResponse

	if len(body) > 0 {
		_ = json.Unmarshal(body, &errorResponse)
	}

	errorCode := strings.Trim(errorResponse.Error, " \t\r\n")
	message := strings.Trim(errorResponse.Message, " \t\r\n")

	if message == "" {
		message = strings.Trim(string(body), " \t\r\n")
	}
	if len(message) > 1024 {
		message = message[:1024]
	}

	switch {
	case errorCode != "" && message != "":
		return fmt.Errorf(
			"solana wallet reader: bubblegum service returned status=%d error=%s message=%s",
			statusCode,
			errorCode,
			message,
		)
	case errorCode != "":
		return fmt.Errorf(
			"solana wallet reader: bubblegum service returned status=%d error=%s",
			statusCode,
			errorCode,
		)
	case message != "":
		return fmt.Errorf(
			"solana wallet reader: bubblegum service returned status=%d message=%s",
			statusCode,
			message,
		)
	default:
		return fmt.Errorf(
			"solana wallet reader: bubblegum service returned status=%d",
			statusCode,
		)
	}
}
