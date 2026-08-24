// backend/internal/adapters/out/stripe/transfer_reversal_gateway.go
package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
)

var _ usecase.StripeTransferReversalGateway = (*TransferReversalGateway)(nil)

// ============================================================
// TransferReversalGateway
// ============================================================

// TransferReversalGateway creates Stripe Connect Transfer Reversals.
//
// AMOL uses Stripe Separate Charges and Transfers:
//
//	PaymentIntent
//		-> Charge
//		-> Transfer A
//		-> Transfer B
//
// A purchaser-side Charge refund does not by itself restore money already sent
// to a seller Connected Account. Each completed seller Transfer is therefore
// reversed explicitly through this gateway.
type TransferReversalGateway struct {
	secretKey string

	httpClient *http.Client
}

func NewTransferReversalGateway(
	secretKey string,
) *TransferReversalGateway {
	return &TransferReversalGateway{
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ============================================================
// Create Transfer Reversal
// ============================================================

// CreateTransferReversal creates one Stripe Connect Transfer Reversal.
//
// Stripe:
//
//	POST /v1/transfers/{transferId}/reversals
//
// Parameters:
//
//	amount
//
// RefundUsecase currently passes the complete Settlement TransferAmount, so
// AMOL's first refund flow performs a full seller Transfer reversal.
//
// The deterministic Idempotency-Key protects against duplicate reversals when
// Stripe accepted the request but the application did not receive or persist
// the result.
func (g *TransferReversalGateway) CreateTransferReversal(
	ctx context.Context,
	in usecase.CreateStripeTransferReversalInput,
) (*usecase.CreateStripeTransferReversalResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	stripeTransferID :=
		in.StripeTransferID

	idempotencyKey :=
		in.IdempotencyKey

	if stripeTransferID == "" ||
		!strings.HasPrefix(
			stripeTransferID,
			"tr_",
		) ||
		strings.Contains(
			stripeTransferID,
			"/",
		) {
		return nil,
			newTransferReversalError(
				0,
				"invalid_request",
				"invalid_transfer",
				"stripe transfer reversal transfer id is invalid",
				false,
			)
	}

	if in.Amount <= 0 {
		return nil,
			newTransferReversalError(
				0,
				"invalid_request",
				"invalid_amount",
				"stripe transfer reversal amount is invalid",
				false,
			)
	}

	if idempotencyKey == "" {
		return nil,
			newTransferReversalError(
				0,
				"invalid_request",
				"invalid_idempotency_key",
				"stripe transfer reversal idempotency key is empty",
				false,
			)
	}

	if in.PaymentID == "" {
		return nil,
			newTransferReversalError(
				0,
				"invalid_request",
				"invalid_payment_id",
				"stripe transfer reversal payment id is empty",
				false,
			)
	}

	if in.SettlementID == "" {
		return nil,
			newTransferReversalError(
				0,
				"invalid_request",
				"invalid_settlement_id",
				"stripe transfer reversal settlement id is empty",
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

	if in.OrderID != "" {
		form.Set(
			"metadata[orderId]",
			in.OrderID,
		)
	}

	form.Set(
		"metadata[paymentId]",
		in.PaymentID,
	)

	form.Set(
		"metadata[settlementId]",
		in.SettlementID,
	)

	if in.CompanyID != "" {
		form.Set(
			"metadata[companyId]",
			in.CompanyID,
		)
	}

	if in.AccountID != "" {
		form.Set(
			"metadata[accountId]",
			in.AccountID,
		)
	}

	var out stripeTransferReversalResponse

	if err := g.postTransferReversal(
		ctx,
		stripeTransferID,
		form,
		idempotencyKey,
		&out,
	); err != nil {
		return nil, err
	}

	stripeTransferReversalID :=
		out.ID

	if stripeTransferReversalID == "" ||
		!strings.HasPrefix(
			stripeTransferReversalID,
			"trr_",
		) {
		return nil,
			newTransferReversalError(
				http.StatusOK,
				"invalid_response",
				"invalid_transfer_reversal_id",
				"stripe transfer reversal id is empty or invalid",
				true,
			)
	}

	if out.Object != "transfer_reversal" {
		return nil,
			newTransferReversalError(
				http.StatusOK,
				"invalid_response",
				"invalid_object",
				"stripe transfer reversal object is invalid",
				true,
			)
	}

	if out.Amount != in.Amount {
		return nil,
			newTransferReversalError(
				http.StatusOK,
				"invalid_response",
				"amount_mismatch",
				"stripe transfer reversal amount does not match request",
				true,
			)
	}

	if out.Transfer != stripeTransferID {
		return nil,
			newTransferReversalError(
				http.StatusOK,
				"invalid_response",
				"transfer_mismatch",
				"stripe transfer reversal transfer does not match request",
				true,
			)
	}

	return &usecase.CreateStripeTransferReversalResult{
		StripeTransferReversalID: stripeTransferReversalID,
	}, nil
}

// ============================================================
// HTTP
// ============================================================

func (g *TransferReversalGateway) validateReady() error {
	if g == nil {
		return newTransferReversalError(
			0,
			"configuration_error",
			"gateway_nil",
			"stripe transfer reversal gateway is nil",
			false,
		)
	}

	if g.secretKey == "" {
		return newTransferReversalError(
			0,
			"configuration_error",
			"secret_key_empty",
			"stripe transfer reversal gateway secret key is empty",
			false,
		)
	}

	if !strings.HasPrefix(
		g.secretKey,
		"sk_",
	) {
		return newTransferReversalError(
			0,
			"configuration_error",
			"secret_key_invalid",
			"stripe transfer reversal gateway secret key is invalid",
			false,
		)
	}

	if g.httpClient == nil {
		return newTransferReversalError(
			0,
			"configuration_error",
			"http_client_nil",
			"stripe transfer reversal gateway http client is nil",
			false,
		)
	}

	return nil
}

func (g *TransferReversalGateway) postTransferReversal(
	ctx context.Context,
	stripeTransferID string,
	form url.Values,
	idempotencyKey string,
	dst *stripeTransferReversalResponse,
) error {
	if err := g.validateReady(); err != nil {
		return err
	}

	if stripeTransferID == "" ||
		!strings.HasPrefix(
			stripeTransferID,
			"tr_",
		) ||
		strings.Contains(
			stripeTransferID,
			"/",
		) {
		return newTransferReversalError(
			0,
			"invalid_request",
			"invalid_transfer",
			"stripe transfer reversal transfer id is invalid",
			false,
		)
	}

	if idempotencyKey == "" {
		return newTransferReversalError(
			0,
			"invalid_request",
			"invalid_idempotency_key",
			"stripe transfer reversal idempotency key is empty",
			false,
		)
	}

	endpoint :=
		stripeAPIBaseURL +
			"/transfers/" +
			url.PathEscape(
				stripeTransferID,
			) +
			"/reversals"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		strings.NewReader(
			form.Encode(),
		),
	)
	if err != nil {
		return newTransferReversalError(
			0,
			"request_error",
			"request_creation_failed",
			err.Error(),
			false,
		)
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+g.secretKey,
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
		idempotencyKey,
	)

	response, err :=
		g.httpClient.Do(req)
	if err != nil {
		return newTransferReversalError(
			0,
			"transport_error",
			"",
			err.Error(),
			true,
		)
	}
	defer response.Body.Close()

	body, err :=
		io.ReadAll(
			response.Body,
		)
	if err != nil {
		return newTransferReversalError(
			response.StatusCode,
			"transport_error",
			"response_read_failed",
			err.Error(),
			true,
		)
	}

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return stripeTransferReversalHTTPError(
			response.StatusCode,
			body,
		)
	}

	if dst == nil {
		return newTransferReversalError(
			response.StatusCode,
			"invalid_response",
			"destination_nil",
			"stripe transfer reversal response destination is nil",
			true,
		)
	}

	if err := json.Unmarshal(
		body,
		dst,
	); err != nil {
		return newTransferReversalError(
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

type stripeTransferReversalResponse struct {
	ID string `json:"id"`

	Object string `json:"object"`

	Amount int `json:"amount"`

	BalanceTransaction string `json:"balance_transaction"`

	Created int64 `json:"created"`

	Currency string `json:"currency"`

	DestinationPaymentRefund *string `json:"destination_payment_refund"`

	Metadata map[string]string `json:"metadata"`

	SourceRefund *string `json:"source_refund"`

	Transfer string `json:"transfer"`
}

type stripeTransferReversalErrorResponse struct {
	Error struct {
		Type string `json:"type"`
		Code string `json:"code"`

		Message string `json:"message"`

		Param string `json:"param"`

		RequestLogURL string `json:"request_log_url"`
	} `json:"error"`
}

// ============================================================
// TransferReversalError
// ============================================================

// TransferReversalError retains Stripe error metadata and whether retrying the
// same deterministic idempotent request is safe.
type TransferReversalError struct {
	statusCode int

	errorType string
	errorCode string

	message string

	retryable bool
}

func (e *TransferReversalError) Error() string {
	if e == nil {
		return "stripe transfer reversal error"
	}

	return e.message
}

func (e *TransferReversalError) StatusCode() int {
	if e == nil {
		return 0
	}

	return e.statusCode
}

func (e *TransferReversalError) ErrorType() string {
	if e == nil {
		return ""
	}

	return e.errorType
}

func (e *TransferReversalError) ErrorCode() string {
	if e == nil {
		return ""
	}

	return e.errorCode
}

func (e *TransferReversalError) Retryable() bool {
	if e == nil {
		return false
	}

	return e.retryable
}

func newTransferReversalError(
	statusCode int,
	errorType string,
	errorCode string,
	message string,
	retryable bool,
) *TransferReversalError {
	if message == "" {
		message = "stripe transfer reversal request failed"
	}

	return &TransferReversalError{
		statusCode: statusCode,
		errorType:  errorType,
		errorCode:  errorCode,
		message:    message,
		retryable:  retryable,
	}
}

func stripeTransferReversalHTTPError(
	statusCode int,
	body []byte,
) error {
	var stripeError stripeTransferReversalErrorResponse

	if err := json.Unmarshal(
		body,
		&stripeError,
	); err != nil {
		return newTransferReversalError(
			statusCode,
			"stripe_error",
			"invalid_error_response",
			fmt.Sprintf(
				"stripe transfer reversal request failed with status %d",
				statusCode,
			),
			isRetryableTransferReversalStatusCode(
				statusCode,
			),
		)
	}

	errorType := stripeError.Error.Type
	errorCode := stripeError.Error.Code
	message := stripeError.Error.Message

	if message == "" {
		message = fmt.Sprintf(
			"stripe transfer reversal request failed with status %d",
			statusCode,
		)
	}

	return newTransferReversalError(
		statusCode,
		errorType,
		errorCode,
		message,
		isRetryableTransferReversalStatusCode(
			statusCode,
		),
	)
}

func isRetryableTransferReversalStatusCode(
	statusCode int,
) bool {
	switch {
	case statusCode == http.StatusRequestTimeout:
		return true

	case statusCode == http.StatusConflict:
		return true

	case statusCode == http.StatusTooManyRequests:
		return true

	case statusCode >= 500:
		return true

	default:
		return false
	}
}
