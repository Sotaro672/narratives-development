// backend/internal/infra/solana/token_transfer_reader.go
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
	"sort"
	"strings"
	"time"

	"google.golang.org/api/idtoken"
)

var (
	ErrTokenTransferReaderNotConfigured = errors.New(
		"token_transfer_reader: not configured",
	)

	ErrTokenTransferReaderServiceURLEmpty = errors.New(
		"token_transfer_reader: bubblegum service URL is empty",
	)

	ErrTokenTransferReaderInvalidServiceURL = errors.New(
		"token_transfer_reader: bubblegum service URL is invalid",
	)

	ErrTokenTransferReaderAssetIDEmpty = errors.New(
		"token_transfer_reader: assetId is empty",
	)

	ErrTokenTransferReaderResponseTooLarge = errors.New(
		"token_transfer_reader: bubblegum service response is too large",
	)

	ErrTokenTransferReaderAssetMismatch = errors.New(
		"token_transfer_reader: bubblegum service returned unexpected assetId",
	)
)

const (
	envBubblegumTransferReaderServiceURL = "SOLANA_BUBBLEGUM_SERVICE_URL"

	envBubblegumTransferReaderServiceAudience = "SOLANA_BUBBLEGUM_SERVICE_AUDIENCE"

	bubblegumAssetTransfersPath = "/asset-transfers"

	defaultTokenTransferReaderLimit = 50

	defaultTokenTransferReaderHTTPTimeout = 20 * time.Second

	maxTokenTransferReaderResponseBodyBytes int64 = 512 * 1024
)

// TokenTransferReaderSolana reads Bubblegum V2 cNFT transfer history through
// the internal solana-bubblegum service.
//
// The Go backend does not inspect SPL token accounts or parse SPL Token
// transfer instructions. Bubblegum cNFT history is resolved by the internal
// service using DAS / indexed on-chain data.
type TokenTransferReaderSolana struct {
	ServiceURL string
	HTTPClient *http.Client
	Timeout    time.Duration

	initErr error
}

// NewTokenTransferReaderSolana keeps the existing constructor signature so
// current DI code continues to compile.
//
// The legacy argument used to be SOLANA_RPC_URL. It is intentionally ignored
// because this reader must no longer query a standard Solana RPC endpoint
// directly.
//
// Configure the reader with:
//
// SOLANA_BUBBLEGUM_SERVICE_URL
// SOLANA_BUBBLEGUM_SERVICE_AUDIENCE
func NewTokenTransferReaderSolana(
	_ string,
) *TokenTransferReaderSolana {
	timeout := defaultTokenTransferReaderHTTPTimeout

	serviceURL := strings.Trim(
		os.Getenv(
			envBubblegumTransferReaderServiceURL,
		),
		" \t\r\n",
	)

	reader := &TokenTransferReaderSolana{
		ServiceURL: serviceURL,
		Timeout:    timeout,
	}

	if serviceURL == "" {
		reader.initErr =
			ErrTokenTransferReaderServiceURLEmpty

		return reader
	}

	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		reader.initErr = fmt.Errorf(
			"%w: %v",
			ErrTokenTransferReaderInvalidServiceURL,
			err,
		)

		return reader
	}

	if parsedURL.Scheme != "http" &&
		parsedURL.Scheme != "https" {
		reader.initErr = fmt.Errorf(
			"%w: unsupported scheme=%s",
			ErrTokenTransferReaderInvalidServiceURL,
			parsedURL.Scheme,
		)

		return reader
	}

	if parsedURL.Host == "" {
		reader.initErr = fmt.Errorf(
			"%w: host is empty",
			ErrTokenTransferReaderInvalidServiceURL,
		)

		return reader
	}

	reader.ServiceURL = strings.TrimRight(
		serviceURL,
		"/",
	)

	audience := strings.Trim(
		os.Getenv(
			envBubblegumTransferReaderServiceAudience,
		),
		" \t\r\n",
	)

	if audience == "" {
		reader.HTTPClient = &http.Client{
			Timeout: timeout,
		}

		return reader
	}

	authenticatedClient, err := idtoken.NewClient(
		context.Background(),
		audience,
	)
	if err != nil {
		reader.initErr = fmt.Errorf(
			"token_transfer_reader: create authenticated HTTP client: %w",
			err,
		)

		return reader
	}

	authenticatedClient.Timeout = timeout
	reader.HTTPClient = authenticatedClient

	return reader
}

type ListAssetTransfersInput struct {
	AssetID string

	Limit int

	BeforeSignature string
	UntilSignature  string
}

type ListAssetTransfersResult struct {
	AssetID   string                `json:"assetId"`
	Transfers []AssetTransferRecord `json:"transfers"`
}

type AssetTransferRecord struct {
	FromWalletAddress string     `json:"fromWalletAddress"`
	ToWalletAddress   string     `json:"toWalletAddress"`
	TransferredAt     *time.Time `json:"transferredAt,omitempty"`

	TxSignature string `json:"txSignature,omitempty"`
	Slot        uint64 `json:"slot,omitempty"`
}

type bubblegumAssetTransfersRequest struct {
	AssetStandard string `json:"assetStandard"`
	AssetID       string `json:"assetId"`

	Limit int `json:"limit,omitempty"`

	BeforeSignature string `json:"beforeSignature,omitempty"`
	UntilSignature  string `json:"untilSignature,omitempty"`
}

type bubblegumAssetTransfersResponse struct {
	AssetID   string                `json:"assetId"`
	Transfers []AssetTransferRecord `json:"transfers"`
}

type bubblegumAssetTransfersErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// ListAssetTransfers returns transfer history for one Bubblegum V2 cNFT.
//
// The internal Bubblegum service is responsible for:
// - resolving the asset by assetId
// - reading DAS / indexed on-chain transaction history
// - identifying transfer operations
// - resolving previous and next wallet owners
// - returning transfer timestamps and signatures
//
// The Go backend treats assetId as the canonical cNFT identifier.
func (e *TokenTransferReaderSolana) ListAssetTransfers(
	ctx context.Context,
	in ListAssetTransfersInput,
) (ListAssetTransfersResult, error) {
	if e == nil {
		return ListAssetTransfersResult{},
			ErrTokenTransferReaderNotConfigured
	}

	if e.initErr != nil {
		return ListAssetTransfersResult{},
			e.initErr
	}

	if e.ServiceURL == "" ||
		e.HTTPClient == nil {
		return ListAssetTransfersResult{},
			ErrTokenTransferReaderNotConfigured
	}

	assetID := strings.Trim(
		in.AssetID,
		" \t\r\n",
	)
	if assetID == "" {
		return ListAssetTransfersResult{},
			ErrTokenTransferReaderAssetIDEmpty
	}

	limit := in.Limit
	if limit <= 0 {
		limit = defaultTokenTransferReaderLimit
	}

	payload := bubblegumAssetTransfersRequest{
		AssetStandard: "BUBBLEGUM_V2",
		AssetID:       assetID,

		Limit: limit,

		BeforeSignature: strings.Trim(
			in.BeforeSignature,
			" \t\r\n",
		),
		UntilSignature: strings.Trim(
			in.UntilSignature,
			" \t\r\n",
		),
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return ListAssetTransfersResult{},
			fmt.Errorf(
				"token_transfer_reader: marshal request: %w",
				err,
			)
	}

	requestCtx := ctx
	var cancel context.CancelFunc

	if _, hasDeadline := ctx.Deadline(); !hasDeadline &&
		e.Timeout > 0 {
		requestCtx, cancel =
			context.WithTimeout(
				ctx,
				e.Timeout,
			)
		defer cancel()
	}

	endpoint :=
		e.ServiceURL +
			bubblegumAssetTransfersPath

	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return ListAssetTransfersResult{},
			fmt.Errorf(
				"token_transfer_reader: create request: %w",
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

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return ListAssetTransfersResult{},
			fmt.Errorf(
				"token_transfer_reader: bubblegum asset transfer request failed assetId=%s: %w",
				assetID,
				err,
			)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(
			resp.Body,
			maxTokenTransferReaderResponseBodyBytes+1,
		),
	)
	if err != nil {
		return ListAssetTransfersResult{},
			fmt.Errorf(
				"token_transfer_reader: read bubblegum response: %w",
				err,
			)
	}

	if int64(len(body)) >
		maxTokenTransferReaderResponseBodyBytes {
		return ListAssetTransfersResult{},
			ErrTokenTransferReaderResponseTooLarge
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return ListAssetTransfersResult{},
			bubblegumAssetTransfersHTTPError(
				resp.StatusCode,
				body,
			)
	}

	var serviceResult bubblegumAssetTransfersResponse

	if err := json.Unmarshal(
		body,
		&serviceResult,
	); err != nil {
		return ListAssetTransfersResult{},
			fmt.Errorf(
				"token_transfer_reader: decode bubblegum response: %w",
				err,
			)
	}

	serviceResult.AssetID = strings.Trim(
		serviceResult.AssetID,
		" \t\r\n",
	)

	if serviceResult.AssetID != "" &&
		serviceResult.AssetID != assetID {
		return ListAssetTransfersResult{},
			fmt.Errorf(
				"%w: requested=%s returned=%s",
				ErrTokenTransferReaderAssetMismatch,
				assetID,
				serviceResult.AssetID,
			)
	}

	transfers := normalizeAssetTransferRecords(
		serviceResult.Transfers,
	)

	return ListAssetTransfersResult{
		AssetID:   assetID,
		Transfers: transfers,
	}, nil
}

func normalizeAssetTransferRecords(
	values []AssetTransferRecord,
) []AssetTransferRecord {
	if len(values) == 0 {
		return []AssetTransferRecord{}
	}

	seen := make(
		map[string]struct{},
		len(values),
	)

	result := make(
		[]AssetTransferRecord,
		0,
		len(values),
	)

	for _, value := range values {
		fromWallet := strings.Trim(
			value.FromWalletAddress,
			" \t\r\n",
		)

		toWallet := strings.Trim(
			value.ToWalletAddress,
			" \t\r\n",
		)

		txSignature := strings.Trim(
			value.TxSignature,
			" \t\r\n",
		)

		if fromWallet == "" ||
			toWallet == "" {
			continue
		}

		var transferredAt *time.Time
		if value.TransferredAt != nil &&
			!value.TransferredAt.IsZero() {
			normalizedTime :=
				value.TransferredAt.UTC()

			transferredAt = &normalizedTime
		}

		key := txSignature

		if key == "" {
			timestamp := ""

			if transferredAt != nil {
				timestamp =
					transferredAt.Format(
						time.RFC3339Nano,
					)
			}

			key =
				fromWallet +
					"|" +
					toWallet +
					"|" +
					timestamp +
					"|" +
					fmt.Sprintf(
						"%d",
						value.Slot,
					)
		}

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}

		result = append(
			result,
			AssetTransferRecord{
				FromWalletAddress: fromWallet,
				ToWalletAddress:   toWallet,
				TransferredAt:     transferredAt,
				TxSignature:       txSignature,
				Slot:              value.Slot,
			},
		)
	}

	sort.SliceStable(
		result,
		func(i, j int) bool {
			leftTime := int64(0)
			rightTime := int64(0)

			if result[i].TransferredAt != nil {
				leftTime =
					result[i].
						TransferredAt.
						UnixNano()
			}

			if result[j].TransferredAt != nil {
				rightTime =
					result[j].
						TransferredAt.
						UnixNano()
			}

			if leftTime != rightTime {
				return leftTime > rightTime
			}

			if result[i].Slot !=
				result[j].Slot {
				return result[i].Slot >
					result[j].Slot
			}

			if result[i].TxSignature !=
				result[j].TxSignature {
				return result[i].TxSignature >
					result[j].TxSignature
			}

			if result[i].FromWalletAddress !=
				result[j].FromWalletAddress {
				return result[i].
					FromWalletAddress >
					result[j].
						FromWalletAddress
			}

			return result[i].
				ToWalletAddress >
				result[j].
					ToWalletAddress
		},
	)

	return result
}

func bubblegumAssetTransfersHTTPError(
	statusCode int,
	body []byte,
) error {
	var errorResponse bubblegumAssetTransfersErrorResponse

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
			"token_transfer_reader: bubblegum service returned status=%d error=%s message=%s",
			statusCode,
			errorCode,
			message,
		)

	case errorCode != "":
		return fmt.Errorf(
			"token_transfer_reader: bubblegum service returned status=%d error=%s",
			statusCode,
			errorCode,
		)

	case message != "":
		return fmt.Errorf(
			"token_transfer_reader: bubblegum service returned status=%d message=%s",
			statusCode,
			message,
		)

	default:
		return fmt.Errorf(
			"token_transfer_reader: bubblegum service returned status=%d",
			statusCode,
		)
	}
}
