// backend/internal/adapters/out/stripe/refund_gateway.go
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
	paymentdom "narratives/internal/domain/payment"
)

var _ usecase.StripeRefundGateway = (*RefundGateway)(nil)

// ============================================================
// RefundGateway
// ============================================================

// RefundGateway creates Stripe refunds against the platform-side Charge.
//
// AMOL uses Stripe Separate Charges and Transfers:
//
//	PaymentIntent
//		-> Charge
//		-> Transfer A
//		-> Transfer B
//
// Refunding the Charge does not represent seller-side payout reversal in AMOL.
// Completed seller Transfers are reversed separately through
// StripeTransferReversalGateway.
type RefundGateway struct {
	secretKey  string
	httpClient *http.Client
}

func NewRefundGateway(
	secretKey string,
) *RefundGateway {
	return &RefundGateway{
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ============================================================
// Create Refund
// ============================================================

// CreateRefund creates one full or partial Stripe Charge refund.
//
// Stripe:
//
//	POST /v1/refunds
//
// Parameters:
//
//	charge
//	amount
//	metadata[paymentId]
//	metadata[refundId] (item-level refund only)
//
// RefundUsecase passes the complete Payment amount and leaves RefundID empty.
//
// ItemRefundUsecase passes the item-level partial refund amount together with
// the deterministic RefundID. That RefundID is persisted to Stripe metadata so
// the webhook can distinguish an item-level refund from a full Payment refund.
//
// reverse_transfer is intentionally not set here. AMOL uses Separate Charges
// and Transfers and coordinates each seller Transfer Reversal explicitly.
//
// The deterministic Idempotency-Key protects against duplicate refunds when
// the request result is uncertain.
func (g *RefundGateway) CreateRefund(
	ctx context.Context,
	in usecase.CreateStripeRefundInput,
) (*usecase.CreateStripeRefundResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	stripeChargeID := in.StripeChargeID
	idempotencyKey := in.IdempotencyKey
	paymentID := in.PaymentID
	refundID := strings.TrimSpace(in.RefundID)

	if stripeChargeID == "" ||
		!strings.HasPrefix(
			stripeChargeID,
			"ch_",
		) {
		return nil,
			newRefundError(
				0,
				"invalid_request",
				"invalid_charge",
				"stripe refund charge is invalid",
				false,
			)
	}

	if in.Amount <= 0 {
		return nil,
			newRefundError(
				0,
				"invalid_request",
				"invalid_amount",
				"stripe refund amount is invalid",
				false,
			)
	}

	if idempotencyKey == "" {
		return nil,
			newRefundError(
				0,
				"invalid_request",
				"invalid_idempotency_key",
				"stripe refund idempotency key is empty",
				false,
			)
	}

	if paymentID == "" {
		return nil,
			newRefundError(
				0,
				"invalid_request",
				"invalid_payment_id",
				"stripe refund payment id is empty",
				false,
			)
	}

	form := url.Values{}

	form.Set(
		"charge",
		stripeChargeID,
	)

	form.Set(
		"amount",
		fmt.Sprintf(
			"%d",
			in.Amount,
		),
	)

	form.Set(
		"metadata[paymentId]",
		paymentID,
	)

	if refundID != "" {
		form.Set(
			"metadata[refundId]",
			refundID,
		)
	}

	var out stripeRefundResponse

	if err := g.postRefund(
		ctx,
		form,
		idempotencyKey,
		&out,
	); err != nil {
		return nil, err
	}

	stripeRefundID :=
		out.ID

	if stripeRefundID == "" ||
		!strings.HasPrefix(
			stripeRefundID,
			"re_",
		) {
		return nil,
			newRefundError(
				http.StatusOK,
				"invalid_response",
				"invalid_refund_id",
				"stripe refund id is empty or invalid",
				true,
			)
	}

	if out.Object != "refund" {
		return nil,
			newRefundError(
				http.StatusOK,
				"invalid_response",
				"invalid_object",
				"stripe refund object is invalid",
				true,
			)
	}

	if out.Amount != in.Amount {
		return nil,
			newRefundError(
				http.StatusOK,
				"invalid_response",
				"amount_mismatch",
				"stripe refund amount does not match request",
				true,
			)
	}

	if out.Charge != stripeChargeID {
		return nil,
			newRefundError(
				http.StatusOK,
				"invalid_response",
				"charge_mismatch",
				"stripe refund charge does not match request",
				true,
			)
	}

	refundStatus :=
		paymentdom.RefundStatus(
			out.Status,
		)

	if !paymentdom.IsValidRefundStatus(
		refundStatus,
	) ||
		refundStatus ==
			paymentdom.RefundStatusNone {
		return nil,
			newRefundError(
				http.StatusOK,
				"invalid_response",
				"invalid_refund_status",
				"stripe refund status is invalid",
				true,
			)
	}

	if out.Created <= 0 {
		return nil,
			newRefundError(
				http.StatusOK,
				"invalid_response",
				"invalid_created_at",
				"stripe refund created timestamp is invalid",
				true,
			)
	}

	createdAt :=
		time.Unix(
			out.Created,
			0,
		).UTC()

	return &usecase.CreateStripeRefundResult{
		StripeRefundID: stripeRefundID,
		Status:         refundStatus,
		CreatedAt:      createdAt,
	}, nil
}

// ============================================================
// HTTP
// ============================================================

func (g *RefundGateway) validateReady() error {
	if g == nil {
		return newRefundError(
			0,
			"configuration_error",
			"gateway_nil",
			"stripe refund gateway is nil",
			false,
		)
	}

	if g.secretKey == "" {
		return newRefundError(
			0,
			"configuration_error",
			"secret_key_empty",
			"stripe refund gateway secret key is empty",
			false,
		)
	}

	if !strings.HasPrefix(
		g.secretKey,
		"sk_",
	) {
		return newRefundError(
			0,
			"configuration_error",
			"secret_key_invalid",
			"stripe refund gateway secret key is invalid",
			false,
		)
	}

	if g.httpClient == nil {
		return newRefundError(
			0,
			"configuration_error",
			"http_client_nil",
			"stripe refund gateway http client is nil",
			false,
		)
	}

	return nil
}

func (g *RefundGateway) postRefund(
	ctx context.Context,
	form url.Values,
	idempotencyKey string,
	dst *stripeRefundResponse,
) error {
	if err := g.validateReady(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		stripeAPIBaseURL+
			"/refunds",
		strings.NewReader(
			form.Encode(),
		),
	)
	if err != nil {
		return newRefundError(
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
		return newRefundError(
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
		return newRefundError(
			response.StatusCode,
			"transport_error",
			"response_read_failed",
			err.Error(),
			true,
		)
	}

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return stripeRefundHTTPError(
			response.StatusCode,
			body,
		)
	}

	if dst == nil {
		return newRefundError(
			response.StatusCode,
			"invalid_response",
			"destination_nil",
			"stripe refund response destination is nil",
			true,
		)
	}

	if err := json.Unmarshal(
		body,
		dst,
	); err != nil {
		return newRefundError(
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

type stripeRefundResponse struct {
	ID                     string            `json:"id"`
	Object                 string            `json:"object"`
	Amount                 int               `json:"amount"`
	BalanceTransaction     string            `json:"balance_transaction"`
	Charge                 string            `json:"charge"`
	Created                int64             `json:"created"`
	Currency               string            `json:"currency"`
	Metadata               map[string]string `json:"metadata"`
	PaymentIntent          string            `json:"payment_intent"`
	Reason                 *string           `json:"reason"`
	ReceiptNumber          *string           `json:"receipt_number"`
	SourceTransferReversal *string           `json:"source_transfer_reversal"`
	Status                 string            `json:"status"`
	TransferReversal       *string           `json:"transfer_reversal"`
}

type stripeRefundErrorResponse struct {
	Error struct {
		Type          string `json:"type"`
		Code          string `json:"code"`
		Message       string `json:"message"`
		Param         string `json:"param"`
		RequestLogURL string `json:"request_log_url"`
	} `json:"error"`
}

// ============================================================
// RefundError
// ============================================================

// RefundError retains Stripe error metadata and whether retrying the same
// deterministic idempotent request is safe.
type RefundError struct {
	statusCode int
	errorType  string
	errorCode  string
	message    string
	retryable  bool
}

func (e *RefundError) Error() string {
	if e == nil {
		return "stripe refund error"
	}

	return e.message
}

func (e *RefundError) StatusCode() int {
	if e == nil {
		return 0
	}

	return e.statusCode
}

func (e *RefundError) ErrorType() string {
	if e == nil {
		return ""
	}

	return e.errorType
}

func (e *RefundError) ErrorCode() string {
	if e == nil {
		return ""
	}

	return e.errorCode
}

func (e *RefundError) Retryable() bool {
	if e == nil {
		return false
	}

	return e.retryable
}

func newRefundError(
	statusCode int,
	errorType string,
	errorCode string,
	message string,
	retryable bool,
) *RefundError {
	if message == "" {
		message = "stripe refund request failed"
	}

	return &RefundError{
		statusCode: statusCode,
		errorType:  errorType,
		errorCode:  errorCode,
		message:    message,
		retryable:  retryable,
	}
}

func stripeRefundHTTPError(
	statusCode int,
	body []byte,
) error {
	var stripeError stripeRefundErrorResponse

	if err := json.Unmarshal(
		body,
		&stripeError,
	); err != nil {
		return newRefundError(
			statusCode,
			"stripe_error",
			"invalid_error_response",
			fmt.Sprintf(
				"stripe refund request failed with status %d",
				statusCode,
			),
			isRetryableRefundStatusCode(
				statusCode,
			),
		)
	}

	errorType :=
		stripeError.Error.Type

	errorCode :=
		stripeError.Error.Code

	message :=
		stripeError.Error.Message

	if message == "" {
		message = fmt.Sprintf(
			"stripe refund request failed with status %d",
			statusCode,
		)
	}

	return newRefundError(
		statusCode,
		errorType,
		errorCode,
		message,
		isRetryableRefundStatusCode(
			statusCode,
		),
	)
}

func isRetryableRefundStatusCode(
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
