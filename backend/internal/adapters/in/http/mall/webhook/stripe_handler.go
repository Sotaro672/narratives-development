// backend/internal/adapters/in/http/mall/webhook/stripe_handler.go
package mallHandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

const stripeWebhookMaxBodyBytes int64 = 1 << 20 // 1 MiB

type StripeWebhookHandler struct {
	paymentUC                      *usecase.PaymentUsecase
	orderUC                        *usecase.OrderUsecase
	settlementUC                   *usecase.SettlementUsecase
	refundUC                       *usecase.RefundUsecase
	refundCompletionNotificationUC usecase.RefundCompletionNotificationUsecasePort

	// Stripe webhook signing secret (whsec_...).
	signingSecret string

	// Maximum difference between the current time and the timestamp in
	// Stripe-Signature.
	tolerance time.Duration

	now func() time.Time
}

func NewStripeWebhookHandler(
	paymentUC *usecase.PaymentUsecase,
	orderUC *usecase.OrderUsecase,
	settlementUC *usecase.SettlementUsecase,
	refundUC *usecase.RefundUsecase,
	refundCompletionNotificationUC usecase.RefundCompletionNotificationUsecasePort,
	signingSecret string,
) http.Handler {
	return &StripeWebhookHandler{
		paymentUC:                      paymentUC,
		orderUC:                        orderUC,
		settlementUC:                   settlementUC,
		refundUC:                       refundUC,
		refundCompletionNotificationUC: refundCompletionNotificationUC,
		signingSecret:                  signingSecret,
		tolerance:                      5 * time.Minute,
		now:                            time.Now,
	}
}

func (h *StripeWebhookHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	// Stripe itself does not require CORS preflight, but environments such
	// as Cloud Run health checks may send OPTIONS requests.
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		writeStripeWebhookJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "not_found",
			},
		)
		return
	}

	if h == nil || h.paymentUC == nil {
		writeStripeWebhookJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "payment_usecase_not_initialized",
			},
		)
		return
	}

	secret := strings.TrimSpace(
		h.signingSecret,
	)
	if secret == "" {
		writeStripeWebhookJSON(
			w,
			http.StatusNotImplemented,
			map[string]string{
				"error": "stripe_webhook_secret_not_configured",
			},
		)
		return
	}

	requestBody := http.MaxBytesReader(
		w,
		r.Body,
		stripeWebhookMaxBodyBytes,
	)

	body, err := io.ReadAll(requestBody)
	if err != nil {
		writeStripeWebhookJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_body",
			},
		)
		return
	}

	signatureHeader := strings.TrimSpace(
		r.Header.Get("Stripe-Signature"),
	)
	if signatureHeader == "" {
		writeStripeWebhookJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "missing_stripe_signature",
			},
		)
		return
	}

	now := time.Now().UTC()
	if h.now != nil {
		now = h.now().UTC()
	}

	if err := verifyStripeSignature(
		signatureHeader,
		body,
		secret,
		now,
		h.tolerance,
	); err != nil {
		writeStripeWebhookJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_signature",
			},
		)
		return
	}

	var event stripeEvent
	if err := json.Unmarshal(
		body,
		&event,
	); err != nil {
		writeStripeWebhookJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_json",
			},
		)
		return
	}

	refundInput, refundSupported, err :=
		extractStripeRefundEventInput(event)
	if err != nil {
		writeStripeWebhookJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_stripe_refund_event",
			},
		)
		return
	}

	if refundSupported {
		if err := h.applyStripeRefundEvent(
			r.Context(),
			refundInput,
		); err != nil {
			// Refund state is financial state. Return 500 so Stripe retries
			// transient persistence failures instead of acknowledging them.
			writeStripeWebhookJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "internal_error",
				},
			)
			return
		}

		// Re-read the authoritative Payment after applying the Stripe event.
		// If the persisted Refund is succeeded, always finish seller-side
		// Transfer Reversal and ensure the purchaser notification delivery.
		//
		// This intentionally also runs for duplicate or out-of-order webhook
		// deliveries so a previous crash after refund-state persistence can be
		// repaired safely.
		if err := h.completeSucceededRefund(
			r.Context(),
			refundInput.PaymentID,
		); err != nil {
			writeStripeWebhookJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "internal_error",
				},
			)
			return
		}

		writeStripeWebhookJSON(
			w,
			http.StatusOK,
			map[string]string{
				"status": "ok",
			},
		)
		return
	}

	input, supported, err :=
		extractStripePaymentEventInput(event)
	if err != nil {
		writeStripeWebhookJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_stripe_event",
			},
		)
		return
	}

	// Unsupported event types and Stripe objects that do not belong to this
	// application are acknowledged so Stripe does not retry them.
	if !supported {
		writeStripeWebhookJSON(
			w,
			http.StatusOK,
			map[string]string{
				"status": "ignored",
			},
		)
		return
	}

	// ApplyStripeEvent performs the following operations through a Firestore
	// Transaction:
	//
	// - event ID deduplication
	// - PaymentIntent ID verification
	// - Stripe Charge ID persistence
	// - status transition validation
	// - status update
	// - first-succeeded post-paid marker acquisition
	//
	// A duplicate event is a successful no-op. The returned Payment always
	// represents the current persisted state.
	payment, err := h.paymentUC.ApplyStripeEvent(
		r.Context(),
		input,
	)
	if err != nil {
		// Return 500 so Stripe retries the event.
		//
		// This is also important when Stripe sends the webhook after creating
		// the PaymentIntent but before the payments document has been created.
		writeStripeWebhookJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "internal_error",
			},
		)
		return
	}

	// A succeeded Payment must always have Account-level Settlement records.
	//
	// This is intentionally executed even when ApplyStripeEvent processed a
	// duplicate succeeded event. EnsureForSucceededPayment is idempotent because
	// each Settlement ID is deterministic:
	//
	//	paymentId + "_" + accountId
	//
	// If Order loading or Settlement creation fails, return 500 so Stripe
	// retries the webhook. Settlement creation is financial state and must not
	// be handled as a best-effort side effect.
	if payment != nil &&
		payment.Status == paymentdom.StatusSucceeded {

		if h.orderUC == nil {
			writeStripeWebhookJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "order_usecase_not_initialized",
				},
			)
			return
		}

		if h.settlementUC == nil {
			writeStripeWebhookJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "settlement_usecase_not_initialized",
				},
			)
			return
		}

		order, err := h.orderUC.GetByID(
			r.Context(),
			payment.PaymentID,
		)
		if err != nil {
			writeStripeWebhookJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "internal_error",
				},
			)
			return
		}

		if _, err := h.settlementUC.EnsureForSucceededPayment(
			r.Context(),
			order,
			*payment,
		); err != nil {
			writeStripeWebhookJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "internal_error",
				},
			)
			return
		}
	}

	writeStripeWebhookJSON(
		w,
		http.StatusOK,
		map[string]string{
			"status": "ok",
		},
	)
}

// stripeRefundEventInput is the application-side representation of a verified
// Stripe Refund webhook.
//
// Refund lifecycle is intentionally separate from PaymentIntent lifecycle.
type stripeRefundEventInput struct {
	PaymentID string

	StripeRefundID        string
	StripeChargeID        string
	StripePaymentIntentID string

	Status paymentdom.RefundStatus
	Amount int

	OccurredAt time.Time
}

// applyStripeRefundEvent validates that the Refund belongs to the authoritative
// Payment and then persists the refund state through PaymentUsecase.
//
// The current AMOL refund flow supports one full Refund per Payment.
//
// Stripe Refund webhooks must never change Payment.Status. The original
// PaymentIntent remains succeeded after a refund.
func (h *StripeWebhookHandler) applyStripeRefundEvent(
	ctx context.Context,
	in stripeRefundEventInput,
) error {
	if h == nil ||
		h.paymentUC == nil {
		return errors.New(
			"stripe webhook: payment usecase is not initialized",
		)
	}

	current, err := h.paymentUC.GetByPaymentID(
		ctx,
		in.PaymentID,
	)
	if err != nil {
		return err
	}

	if current == nil ||
		current.PaymentID != in.PaymentID {
		return paymentdom.ErrNotFound
	}

	if current.Status != paymentdom.StatusSucceeded {
		return paymentdom.ErrRefundRequiresSucceeded
	}

	if current.StripeChargeID == "" ||
		current.StripeChargeID !=
			in.StripeChargeID {
		return paymentdom.ErrInvalidStripeChargeID
	}

	if current.StripePaymentIntentID == "" ||
		current.StripePaymentIntentID !=
			in.StripePaymentIntentID {
		return paymentdom.ErrInvalidStripePaymentIntent
	}

	if in.Amount <= 0 ||
		in.Amount != current.Amount {
		// Partial refunds are intentionally rejected by the current Payment
		// model. Item-level partial refunds require a separate Refund aggregate.
		return paymentdom.ErrInvalidRefundedAmount
	}

	if current.RefundStatus != "" &&
		current.RefundStatus !=
			paymentdom.RefundStatusNone {

		if current.StripeRefundID == "" {
			return paymentdom.ErrInvalidRefundState
		}

		if current.StripeRefundID !=
			in.StripeRefundID {
			// The current Payment model stores one full Refund only.
			// A second Refund object must not silently overwrite the first.
			return paymentdom.ErrConflict
		}

		if current.RefundStatus == in.Status {
			// Duplicate webhook or an equivalent later snapshot.
			return nil
		}

		if isTerminalRefundStatus(
			current.RefundStatus,
		) {
			// Stripe webhook delivery order is not guaranteed. Do not allow an
			// older non-terminal snapshot to regress a terminal Refund state.
			return nil
		}
	}

	refundedAmount := 0
	var refundedAt *time.Time

	if in.Status ==
		paymentdom.RefundStatusSucceeded {
		refundedAmount = in.Amount

		value := in.OccurredAt.UTC()
		refundedAt = &value
	}

	_, err = h.paymentUC.UpdateRefundState(
		ctx,
		usecase.UpdatePaymentRefundStateInput{
			PaymentID: in.PaymentID,

			StripeRefundID: in.StripeRefundID,
			RefundStatus:   in.Status,
			RefundedAmount: refundedAmount,
			RefundedAt:     refundedAt,
		},
	)
	return err
}

// completeSucceededRefund performs post-refund processing from the authoritative
// persisted Payment state.
//
// It is safe to call for every supported Refund webhook:
//   - non-succeeded Refunds are a no-op
//   - seller-side reversal is idempotent in RefundUsecase
//   - notification delivery uses a deterministic delivery ID
func (h *StripeWebhookHandler) completeSucceededRefund(
	ctx context.Context,
	paymentID string,
) error {
	if h == nil || h.paymentUC == nil {
		return errors.New("stripe webhook: payment usecase is not initialized")
	}

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return paymentdom.ErrInvalidPaymentID
	}

	payment, err := h.paymentUC.GetByPaymentID(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment == nil || payment.PaymentID != paymentID {
		return paymentdom.ErrNotFound
	}

	if payment.RefundStatus != paymentdom.RefundStatusSucceeded {
		return nil
	}

	if h.refundUC == nil {
		return errors.New("stripe webhook: refund usecase is not initialized")
	}
	if h.orderUC == nil {
		return errors.New("stripe webhook: order usecase is not initialized")
	}
	if h.refundCompletionNotificationUC == nil {
		return errors.New(
			"stripe webhook: refund completion notification usecase is not initialized",
		)
	}

	stripeRefundID := strings.TrimSpace(payment.StripeRefundID)
	if stripeRefundID == "" {
		return paymentdom.ErrInvalidRefundState
	}
	if payment.RefundedAmount <= 0 || payment.RefundedAt == nil {
		return paymentdom.ErrInvalidRefundState
	}

	if _, err := h.refundUC.CompleteSucceededRefund(
		ctx,
		usecase.CompleteSucceededRefundInput{
			PaymentID:      payment.PaymentID,
			StripeRefundID: stripeRefundID,
		},
	); err != nil {
		return err
	}

	order, err := h.orderUC.GetByID(
		ctx,
		payment.PaymentID,
	)
	if err != nil {
		return err
	}

	orderID := strings.TrimSpace(order.ID)
	if orderID == "" || orderID != payment.PaymentID {
		return errors.New("stripe webhook: refund order does not match payment")
	}

	userID := strings.TrimSpace(order.UserID)
	if userID == "" {
		return errors.New("stripe webhook: refund order userId is empty")
	}

	_, err = h.refundCompletionNotificationUC.EnsureDelivery(
		ctx,
		usecase.EnsureRefundCompletionNotificationInput{
			PaymentID:      payment.PaymentID,
			OrderID:        orderID,
			UserID:         userID,
			StripeRefundID: stripeRefundID,
			RefundedAmount: payment.RefundedAmount,
		},
	)
	return err
}

func isTerminalRefundStatus(
	status paymentdom.RefundStatus,
) bool {
	switch status {
	case paymentdom.RefundStatusSucceeded,
		paymentdom.RefundStatusFailed,
		paymentdom.RefundStatusCanceled:
		return true

	default:
		return false
	}
}

func writeStripeWebhookJSON(
	w http.ResponseWriter,
	statusCode int,
	value any,
) {
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

// ============================================================
// Stripe signature verification
// ============================================================

// Stripe-Signature:
//
//	t=timestamp,v1=signature(,v0=...)
//
// Signed payload:
//
//	"{timestamp}.{raw request body}"
//
// Expected signature:
//
//	HMAC-SHA256(webhook signing secret, signed payload)
func verifyStripeSignature(
	signatureHeader string,
	body []byte,
	secret string,
	now time.Time,
	tolerance time.Duration,
) error {
	timestamp, signatures, err :=
		parseStripeSignatureHeader(
			signatureHeader,
		)
	if err != nil {
		return err
	}

	signedAt := time.Unix(
		timestamp,
		0,
	).UTC()

	if tolerance > 0 {
		difference := now.Sub(signedAt)
		if difference < 0 {
			difference = -difference
		}

		if difference > tolerance {
			return errors.New(
				"timestamp_out_of_tolerance",
			)
		}
	}

	signedPayload := fmt.Sprintf(
		"%d.%s",
		timestamp,
		string(body),
	)

	mac := hmac.New(
		sha256.New,
		[]byte(secret),
	)
	_, _ = mac.Write(
		[]byte(signedPayload),
	)

	expected := hex.EncodeToString(
		mac.Sum(nil),
	)

	for _, signature := range signatures {
		if subtleEqHex(
			expected,
			signature,
		) {
			return nil
		}
	}

	return errors.New("signature_mismatch")
}

func parseStripeSignatureHeader(
	header string,
) (
	timestamp int64,
	v1Signatures []string,
	err error,
) {
	parts := strings.Split(
		header,
		",",
	)

	var timestampText string
	signatures := make(
		[]string,
		0,
	)

	for _, part := range parts {
		part = strings.TrimSpace(part)

		switch {
		case strings.HasPrefix(part, "t="):
			timestampText = strings.TrimSpace(
				strings.TrimPrefix(
					part,
					"t=",
				),
			)

		case strings.HasPrefix(part, "v1="):
			signature := strings.TrimSpace(
				strings.TrimPrefix(
					part,
					"v1=",
				),
			)
			if signature != "" {
				signatures = append(
					signatures,
					signature,
				)
			}
		}
	}

	if timestampText == "" ||
		len(signatures) == 0 {
		return 0, nil,
			errors.New(
				"invalid_signature_header",
			)
	}

	timestamp, err = strconv.ParseInt(
		timestampText,
		10,
		64,
	)
	if err != nil {
		return 0, nil,
			errors.New(
				"invalid_signature_timestamp",
			)
	}

	return timestamp, signatures, nil
}

// subtleEqHex compares lowercase hexadecimal strings without returning on the
// first mismatch.
func subtleEqHex(
	left string,
	right string,
) bool {
	leftBytes := []byte(
		strings.ToLower(
			strings.TrimSpace(left),
		),
	)
	rightBytes := []byte(
		strings.ToLower(
			strings.TrimSpace(right),
		),
	)

	if len(leftBytes) != len(rightBytes) {
		return false
	}

	var difference byte
	for index := 0; index < len(leftBytes); index++ {
		difference |=
			leftBytes[index] ^
				rightBytes[index]
	}

	return difference == 0
}

// ============================================================
// Stripe event parsing
// ============================================================

type stripeEvent struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Created int64           `json:"created"`
	Data    stripeEventData `json:"data"`
}

type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

type stripeRefund struct {
	ID            string            `json:"id"`
	Object        string            `json:"object"`
	Amount        int               `json:"amount"`
	Charge        string            `json:"charge"`
	Created       int64             `json:"created"`
	Status        string            `json:"status"`
	PaymentIntent string            `json:"payment_intent"`
	Metadata      map[string]string `json:"metadata"`
}

type stripePaymentIntent struct {
	ID                 string              `json:"id"`
	Status             string              `json:"status"`
	Created            int64               `json:"created"`
	Metadata           map[string]string   `json:"metadata"`
	LatestCharge       string              `json:"latest_charge"`
	LastPaymentError   *stripePaymentError `json:"last_payment_error"`
	CancellationReason string              `json:"cancellation_reason"`
}

type stripePaymentError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// extractStripeRefundEventInput converts a supported, verified Stripe Refund
// event into the application-level refund input.
//
// Supported events:
//
// - refund.created
// - refund.failed
// - refund.updated
//
// Refund events without this application's paymentId/orderId metadata are
// acknowledged as unsupported because Payment cannot be resolved safely.
func extractStripeRefundEventInput(
	event stripeEvent,
) (
	input stripeRefundEventInput,
	supported bool,
	err error,
) {
	eventType := strings.TrimSpace(
		event.Type,
	)

	switch eventType {
	case "refund.created",
		"refund.failed",
		"refund.updated":

	default:
		return stripeRefundEventInput{},
			false,
			nil
	}

	eventID := strings.TrimSpace(
		event.ID,
	)
	if eventID == "" {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe event id is empty",
			)
	}

	var refund stripeRefund
	if err := json.Unmarshal(
		event.Data.Object,
		&refund,
	); err != nil {
		return stripeRefundEventInput{},
			false,
			fmt.Errorf(
				"decode Stripe Refund: %w",
				err,
			)
	}

	if strings.TrimSpace(
		refund.Object,
	) != "refund" {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe refund object type is invalid",
			)
	}

	stripeRefundID := strings.TrimSpace(
		refund.ID,
	)
	if stripeRefundID == "" ||
		!strings.HasPrefix(
			stripeRefundID,
			"re_",
		) {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe refund id is invalid",
			)
	}

	paymentID := firstNonEmpty(
		refund.Metadata["paymentId"],
		refund.Metadata["orderId"],
	)
	if paymentID == "" {
		// A Refund created outside the AMOL flow might not contain AMOL
		// metadata. It cannot be mapped safely with the current repository
		// contract, so acknowledge it as unrelated instead of guessing.
		return stripeRefundEventInput{},
			false,
			nil
	}

	stripeChargeID := strings.TrimSpace(
		refund.Charge,
	)
	if stripeChargeID == "" ||
		!strings.HasPrefix(
			stripeChargeID,
			"ch_",
		) {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe refund charge id is invalid",
			)
	}

	stripePaymentIntentID := strings.TrimSpace(
		refund.PaymentIntent,
	)
	if stripePaymentIntentID == "" ||
		!strings.HasPrefix(
			stripePaymentIntentID,
			"pi_",
		) {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe refund payment intent id is invalid",
			)
	}

	if refund.Amount <= 0 {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe refund amount is invalid",
			)
	}

	refundStatus, ok :=
		refundStatusFromStripe(
			refund.Status,
		)
	if !ok {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe refund status is invalid",
			)
	}

	if eventType == "refund.failed" &&
		refundStatus != paymentdom.RefundStatusFailed {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe refund.failed status mismatch",
			)
	}

	occurredUnix := event.Created
	if occurredUnix <= 0 {
		occurredUnix = refund.Created
	}
	if occurredUnix <= 0 {
		return stripeRefundEventInput{},
			false,
			errors.New(
				"Stripe refund event created timestamp is invalid",
			)
	}

	return stripeRefundEventInput{
		PaymentID: paymentID,

		StripeRefundID:        stripeRefundID,
		StripeChargeID:        stripeChargeID,
		StripePaymentIntentID: stripePaymentIntentID,

		Status: refundStatus,
		Amount: refund.Amount,

		OccurredAt: time.Unix(
			occurredUnix,
			0,
		).UTC(),
	}, true, nil
}

func refundStatusFromStripe(
	status string,
) (
	paymentdom.RefundStatus,
	bool,
) {
	switch strings.TrimSpace(
		status,
	) {
	case "pending":
		return paymentdom.RefundStatusPending,
			true

	case "requires_action":
		return paymentdom.RefundStatusRequiresAction,
			true

	case "succeeded":
		return paymentdom.RefundStatusSucceeded,
			true

	case "failed":
		return paymentdom.RefundStatusFailed,
			true

	case "canceled":
		return paymentdom.RefundStatusCanceled,
			true

	default:
		return "",
			false
	}
}

// extractStripePaymentEventInput converts a supported, verified Stripe event
// into the application-level event input.
//
// Unsupported events return supported=false with no error.
//
// PaymentIntent events without this application's paymentId/orderId metadata
// are also ignored because they do not belong to this Payment flow.
func extractStripePaymentEventInput(
	event stripeEvent,
) (
	input usecase.ApplyStripePaymentEventInput,
	supported bool,
	err error,
) {
	eventID := strings.TrimSpace(event.ID)
	if eventID == "" {
		return usecase.ApplyStripePaymentEventInput{},
			false,
			errors.New(
				"Stripe event id is empty",
			)
	}

	eventType := strings.TrimSpace(
		event.Type,
	)

	paymentStatus, supported :=
		paymentStatusFromStripeEventType(
			eventType,
		)
	if !supported {
		return usecase.ApplyStripePaymentEventInput{},
			false,
			nil
	}

	var paymentIntent stripePaymentIntent
	if err := json.Unmarshal(
		event.Data.Object,
		&paymentIntent,
	); err != nil {
		return usecase.ApplyStripePaymentEventInput{},
			false,
			fmt.Errorf(
				"decode Stripe PaymentIntent: %w",
				err,
			)
	}

	paymentID := firstNonEmpty(
		paymentIntent.Metadata["paymentId"],
		paymentIntent.Metadata["orderId"],
	)
	stripePaymentIntentID := strings.TrimSpace(
		paymentIntent.ID,
	)

	// This is either another application's PaymentIntent or malformed event
	// metadata. Acknowledge it without starting Stripe retries.
	if paymentID == "" ||
		stripePaymentIntentID == "" {
		return usecase.ApplyStripePaymentEventInput{},
			false,
			nil
	}

	occurredUnix := event.Created
	if occurredUnix <= 0 {
		occurredUnix = paymentIntent.Created
	}
	if occurredUnix <= 0 {
		return usecase.ApplyStripePaymentEventInput{},
			false,
			errors.New(
				"Stripe event created timestamp is invalid",
			)
	}

	errorType, errorCode, errorMessage :=
		stripePaymentErrorFields(
			eventType,
			paymentIntent,
		)

	return usecase.ApplyStripePaymentEventInput{
		EventID:               eventID,
		PaymentID:             paymentID,
		StripePaymentIntentID: stripePaymentIntentID,
		StripeChargeID: strings.TrimSpace(
			paymentIntent.LatestCharge,
		),

		Status: paymentStatus,

		ErrorType: errorType,
		ErrorCode: errorCode,
		ErrorMsg:  errorMessage,

		OccurredAt: time.Unix(
			occurredUnix,
			0,
		).UTC(),
	}, true, nil
}

// paymentStatusFromStripeEventType maps supported Stripe PaymentIntent event
// types to Payment Domain status.
//
// payment_intent.requires_action is accepted for environments/API versions
// that emit it. The initial PaymentFlow response also persists
// requires_action immediately.
func paymentStatusFromStripeEventType(
	eventType string,
) (
	paymentdom.PaymentStatus,
	bool,
) {
	switch eventType {
	case "payment_intent.succeeded":
		return paymentdom.StatusSucceeded, true

	case "payment_intent.requires_action":
		return paymentdom.StatusRequiresAction, true

	case "payment_intent.processing":
		return paymentdom.StatusProcessing, true

	case "payment_intent.payment_failed":
		return paymentdom.StatusFailed, true

	case "payment_intent.canceled":
		return paymentdom.StatusCanceled, true

	default:
		return "", false
	}
}

func stripePaymentErrorFields(
	eventType string,
	paymentIntent stripePaymentIntent,
) (
	errorType *string,
	errorCode *string,
	errorMessage *string,
) {
	switch eventType {
	case "payment_intent.payment_failed":
		if paymentIntent.LastPaymentError != nil {
			errorType = optionalNonEmptyString(
				paymentIntent.LastPaymentError.Type,
			)
			errorCode = optionalNonEmptyString(
				paymentIntent.LastPaymentError.Code,
			)
			errorMessage = optionalNonEmptyString(
				paymentIntent.LastPaymentError.Message,
			)
		}

		if errorMessage == nil {
			value := "Stripe PaymentIntent payment failed"
			errorMessage = &value
		}

	case "payment_intent.canceled":
		value := "canceled"
		errorType = &value

		reason := strings.TrimSpace(
			paymentIntent.CancellationReason,
		)
		if reason != "" {
			errorCode = &reason

			message := fmt.Sprintf(
				"Stripe PaymentIntent was canceled: %s",
				reason,
			)
			errorMessage = &message
		} else {
			message :=
				"Stripe PaymentIntent was canceled"
			errorMessage = &message
		}
	}

	return errorType, errorCode, errorMessage
}

func optionalNonEmptyString(
	value string,
) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}

func firstNonEmpty(
	values ...string,
) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
