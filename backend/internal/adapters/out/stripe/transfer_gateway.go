// backend/internal/adapters/out/stripe/transfer_gateway.go
package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
)

var _ usecase.StripeSettlementTransferGateway = (*TransferGateway)(nil)

// ============================================================
// TransferGateway
// ============================================================

// TransferGateway creates Stripe Connect Transfers.
//
// This gateway represents fiat settlement:
//
//	AMOL Stripe Platform
//		-> Stripe Connected Account
//
// It is unrelated to Solana token ownership transfer.
//
// AMOL uses Stripe Separate Charges and Transfers:
//
//	PaymentIntent
//		-> Charge
//		-> Transfer A
//		-> Transfer B
//
// Each Transfer uses:
//
// - destination
// - source_transaction
// - transfer_group
// - deterministic Idempotency-Key
type TransferGateway struct {
	secretKey string

	httpClient *http.Client
}

func NewTransferGateway(
	secretKey string,
) *TransferGateway {
	return &TransferGateway{
		secretKey: strings.TrimSpace(
			secretKey,
		),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ============================================================
// Create Transfer
// ============================================================

// CreateTransfer creates one Stripe Connect Transfer.
//
// Stripe:
//
//	POST /v1/transfers
//
// Parameters:
//
//	amount
//	currency
//	destination
//	source_transaction
//	transfer_group
//
// The deterministic Idempotency-Key protects against duplicate transfers when
// the request result is uncertain.
//
// source_transaction must be the Charge ID belonging to the platform-side
// PaymentIntent.
func (g *TransferGateway) CreateTransfer(
	ctx context.Context,
	in usecase.CreateStripeSettlementTransferInput,
) (*usecase.CreateStripeSettlementTransferResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	currency := strings.ToLower(
		strings.TrimSpace(
			in.Currency,
		),
	)

	destinationStripeAccountID := strings.TrimSpace(
		in.DestinationStripeAccountID,
	)

	sourceTransaction := strings.TrimSpace(
		in.SourceTransaction,
	)

	transferGroup := strings.TrimSpace(
		in.TransferGroup,
	)

	idempotencyKey := strings.TrimSpace(
		in.IdempotencyKey,
	)

	if in.Amount <= 0 {
		return nil,
			newSettlementTransferError(
				0,
				"invalid_request",
				"invalid_amount",
				"stripe transfer amount is invalid",
				false,
			)
	}

	if currency == "" || len(currency) != 3 {
		return nil,
			newSettlementTransferError(
				0,
				"invalid_request",
				"invalid_currency",
				"stripe transfer currency is invalid",
				false,
			)
	}

	if destinationStripeAccountID == "" ||
		!strings.HasPrefix(
			destinationStripeAccountID,
			"acct_",
		) {
		return nil,
			newSettlementTransferError(
				0,
				"invalid_request",
				"invalid_destination",
				"stripe transfer destination is invalid",
				false,
			)
	}

	if sourceTransaction == "" ||
		!strings.HasPrefix(
			sourceTransaction,
			"ch_",
		) {
		return nil,
			newSettlementTransferError(
				0,
				"invalid_request",
				"invalid_source_transaction",
				"stripe transfer source_transaction is invalid",
				false,
			)
	}

	if transferGroup == "" {
		return nil,
			newSettlementTransferError(
				0,
				"invalid_request",
				"invalid_transfer_group",
				"stripe transfer transfer_group is empty",
				false,
			)
	}

	if idempotencyKey == "" {
		return nil,
			newSettlementTransferError(
				0,
				"invalid_request",
				"invalid_idempotency_key",
				"stripe transfer idempotency key is empty",
				false,
			)
	}

	seller := in.Seller
	if err := seller.Validate(); err != nil {
		return nil,
			newSettlementTransferError(
				0,
				"invalid_request",
				"invalid_seller",
				err.Error(),
				false,
			)
	}

	if strings.TrimSpace(seller.StripeAccountID) != destinationStripeAccountID {
		return nil,
			newSettlementTransferError(
				0,
				"invalid_request",
				"seller_destination_mismatch",
				"seller Stripe account does not match transfer destination",
				false,
			)
	}

	form := url.Values{}

	form.Set(
		"amount",
		fmt.Sprintf(
			"%d",
			in.Amount,
		),
	)

	form.Set(
		"currency",
		currency,
	)

	form.Set(
		"destination",
		destinationStripeAccountID,
	)

	form.Set(
		"source_transaction",
		sourceTransaction,
	)

	form.Set(
		"transfer_group",
		transferGroup,
	)

	if value := strings.TrimSpace(
		in.OrderID,
	); value != "" {
		form.Set(
			"metadata[orderId]",
			value,
		)
	}

	if value := strings.TrimSpace(
		in.PaymentID,
	); value != "" {
		form.Set(
			"metadata[paymentId]",
			value,
		)
	}

	if value := strings.TrimSpace(
		in.SettlementID,
	); value != "" {
		form.Set(
			"metadata[settlementId]",
			value,
		)
	}

	form.Set(
		"metadata[sellerType]",
		string(seller.Type),
	)

	form.Set(
		"metadata[companyId]",
		strings.TrimSpace(seller.CompanyID),
	)

	form.Set(
		"metadata[accountId]",
		strings.TrimSpace(seller.AccountID),
	)

	var out stripeTransferResponse

	if err := g.postTransfer(
		ctx,
		form,
		idempotencyKey,
		&out,
	); err != nil {
		return nil, err
	}

	stripeTransferID := strings.TrimSpace(
		out.ID,
	)

	if stripeTransferID == "" ||
		!strings.HasPrefix(
			stripeTransferID,
			"tr_",
		) {
		return nil,
			newSettlementTransferError(
				http.StatusOK,
				"invalid_response",
				"invalid_transfer_id",
				"stripe transfer id is empty or invalid",
				true,
			)
	}

	if out.Amount != in.Amount {
		return nil,
			newSettlementTransferError(
				http.StatusOK,
				"invalid_response",
				"amount_mismatch",
				"stripe transfer amount does not match request",
				true,
			)
	}

	if strings.ToLower(
		strings.TrimSpace(
			out.Currency,
		),
	) != currency {
		return nil,
			newSettlementTransferError(
				http.StatusOK,
				"invalid_response",
				"currency_mismatch",
				"stripe transfer currency does not match request",
				true,
			)
	}

	if strings.TrimSpace(
		out.Destination,
	) != destinationStripeAccountID {
		return nil,
			newSettlementTransferError(
				http.StatusOK,
				"invalid_response",
				"destination_mismatch",
				"stripe transfer destination does not match request",
				true,
			)
	}

	if strings.TrimSpace(
		out.SourceTransaction,
	) != sourceTransaction {
		return nil,
			newSettlementTransferError(
				http.StatusOK,
				"invalid_response",
				"source_transaction_mismatch",
				"stripe transfer source_transaction does not match request",
				true,
			)
	}

	if strings.TrimSpace(
		out.TransferGroup,
	) != transferGroup {
		return nil,
			newSettlementTransferError(
				http.StatusOK,
				"invalid_response",
				"transfer_group_mismatch",
				"stripe transfer transfer_group does not match request",
				true,
			)
	}

	return &usecase.CreateStripeSettlementTransferResult{
		StripeTransferID: stripeTransferID,
	}, nil
}

// ============================================================
// HTTP
// ============================================================

func (g *TransferGateway) validateReady() error {
	if g == nil {
		return newSettlementTransferError(
			0,
			"configuration_error",
			"gateway_nil",
			"stripe transfer gateway is nil",
			false,
		)
	}

	secretKey := strings.TrimSpace(
		g.secretKey,
	)

	if secretKey == "" {
		return newSettlementTransferError(
			0,
			"configuration_error",
			"secret_key_empty",
			"stripe transfer gateway secret key is empty",
			false,
		)
	}

	if !strings.HasPrefix(
		secretKey,
		"sk_",
	) {
		return newSettlementTransferError(
			0,
			"configuration_error",
			"secret_key_invalid",
			"stripe transfer gateway secret key is invalid",
			false,
		)
	}

	if g.httpClient == nil {
		return newSettlementTransferError(
			0,
			"configuration_error",
			"http_client_nil",
			"stripe transfer gateway http client is nil",
			false,
		)
	}

	return nil
}

func (g *TransferGateway) postTransfer(
	ctx context.Context,
	form url.Values,
	idempotencyKey string,
	dst *stripeTransferResponse,
) error {
	if err := g.validateReady(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		stripeAPIBaseURL+
			"/transfers",
		strings.NewReader(
			form.Encode(),
		),
	)
	if err != nil {
		return newSettlementTransferError(
			0,
			"request_error",
			"request_creation_failed",
			err.Error(),
			false,
		)
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+
			strings.TrimSpace(
				g.secretKey,
			),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	req.Header.Set(
		"Accept",
		"application/json",
	)

	req.Header.Set(
		"Idempotency-Key",
		strings.TrimSpace(
			idempotencyKey,
		),
	)

	response, err := g.httpClient.Do(req)
	if err != nil {
		return newSettlementTransferError(
			0,
			"transport_error",
			"",
			err.Error(),
			true,
		)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(
		response.Body,
	)
	if err != nil {
		return newSettlementTransferError(
			response.StatusCode,
			"transport_error",
			"response_read_failed",
			err.Error(),
			true,
		)
	}

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return stripeTransferHTTPError(
			response.StatusCode,
			body,
		)
	}

	if dst == nil {
		return newSettlementTransferError(
			response.StatusCode,
			"invalid_response",
			"destination_nil",
			"stripe transfer response destination is nil",
			true,
		)
	}

	if err := json.Unmarshal(
		body,
		dst,
	); err != nil {
		return newSettlementTransferError(
			response.StatusCode,
			"invalid_response",
			"invalid_json",
			err.Error(),
			true,
		)
	}

	return nil
}

// ============================================================
// Stripe response
// ============================================================

type stripeTransferResponse struct {
	ID string `json:"id"`

	Object string `json:"object"`

	Amount         int `json:"amount"`
	AmountReversed int `json:"amount_reversed"`

	BalanceTransaction string `json:"balance_transaction"`

	Created int64 `json:"created"`

	Currency string `json:"currency"`

	Description string `json:"description"`

	Destination string `json:"destination"`

	DestinationPayment string `json:"destination_payment"`

	Livemode bool `json:"livemode"`

	Reversed bool `json:"reversed"`

	SourceTransaction string `json:"source_transaction"`
	SourceType        string `json:"source_type"`

	TransferGroup string `json:"transfer_group"`

	Metadata map[string]string `json:"metadata"`
}

type stripeTransferErrorResponse struct {
	Error struct {
		Type string `json:"type"`
		Code string `json:"code"`

		Message string `json:"message"`

		Param string `json:"param"`

		RequestLogURL string `json:"request_log_url"`
	} `json:"error"`
}

// ============================================================
// SettlementTransferError
// ============================================================

// SettlementTransferError exposes retryability and Stripe error metadata to
// SettlementUsecase without coupling the application layer to Stripe response
// DTOs.
type SettlementTransferError struct {
	statusCode int

	errorType string
	errorCode string

	message string

	retryable bool
}

var _ usecase.RetryableStripeSettlementError = (*SettlementTransferError)(nil)

var _ usecase.StripeSettlementErrorMetadata = (*SettlementTransferError)(nil)

func (e *SettlementTransferError) Error() string {
	if e == nil {
		return "stripe transfer error"
	}

	return e.message
}

func (e *SettlementTransferError) Retryable() bool {
	if e == nil {
		return false
	}

	return e.retryable
}

func (e *SettlementTransferError) ErrorType() string {
	if e == nil {
		return ""
	}

	return e.errorType
}

func (e *SettlementTransferError) ErrorCode() string {
	if e == nil {
		return ""
	}

	return e.errorCode
}

func newSettlementTransferError(
	statusCode int,
	errorType string,
	errorCode string,
	message string,
	retryable bool,
) *SettlementTransferError {
	errorType = strings.TrimSpace(
		errorType,
	)

	errorCode = strings.TrimSpace(
		errorCode,
	)

	message = strings.TrimSpace(
		message,
	)

	if message == "" {
		message = "stripe transfer failed"
	}

	return &SettlementTransferError{
		statusCode: statusCode,
		errorType:  errorType,
		errorCode:  errorCode,
		message:    message,
		retryable:  retryable,
	}
}

// ============================================================
// Stripe error mapping
// ============================================================

func stripeTransferHTTPError(
	statusCode int,
	body []byte,
) error {
	var response stripeTransferErrorResponse

	if err := json.Unmarshal(
		body,
		&response,
	); err != nil {
		message := fmt.Sprintf(
			"stripe transfer http %d: %s",
			statusCode,
			strings.TrimSpace(
				string(body),
			),
		)

		return newSettlementTransferError(
			statusCode,
			"",
			"",
			message,
			isRetryableStripeTransferFailure(
				statusCode,
				"",
				"",
			),
		)
	}

	errorType := strings.TrimSpace(
		response.Error.Type,
	)

	errorCode := strings.TrimSpace(
		response.Error.Code,
	)

	errorMessage := strings.TrimSpace(
		response.Error.Message,
	)

	if errorMessage == "" {
		errorMessage = "Stripe Transfer request failed"
	}

	message := fmt.Sprintf(
		"stripe transfer http %d: %s",
		statusCode,
		errorMessage,
	)

	return newSettlementTransferError(
		statusCode,
		errorType,
		errorCode,
		message,
		isRetryableStripeTransferFailure(
			statusCode,
			errorType,
			errorCode,
		),
	)
}

// isRetryableStripeTransferFailure classifies failures that can reasonably be
// retried with the same deterministic Stripe Idempotency-Key.
//
// Retryable:
//
// - request timeout
// - conflict
// - rate limit
// - Stripe 5xx
// - Stripe api_error
// - insufficient platform balance
//
// Most other 4xx failures are configuration or business-rule errors and must
// not be retried automatically.
func isRetryableStripeTransferFailure(
	statusCode int,
	errorType string,
	errorCode string,
) bool {
	errorType = strings.ToLower(
		strings.TrimSpace(
			errorType,
		),
	)

	errorCode = strings.ToLower(
		strings.TrimSpace(
			errorCode,
		),
	)

	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusTooManyRequests:
		return true
	}

	if statusCode >= 500 {
		return true
	}

	switch errorType {
	case "api_error",
		"rate_limit_error":
		return true
	}

	switch errorCode {
	case "balance_insufficient":
		return true
	}

	return false
}

// ============================================================
// Optional helpers
// ============================================================

// SetHTTPClient is primarily useful for tests.
//
// Production DI normally uses the default client created by
// NewTransferGateway.
func (g *TransferGateway) SetHTTPClient(
	client *http.Client,
) {
	if g == nil || client == nil {
		return
	}

	g.httpClient = client
}

// IsSettlementTransferError can be used by adapter-level tests or diagnostics
// without exposing Stripe response DTOs.
func IsSettlementTransferError(
	err error,
) bool {
	if err == nil {
		return false
	}

	var target *SettlementTransferError

	return errors.As(
		err,
		&target,
	)
}
