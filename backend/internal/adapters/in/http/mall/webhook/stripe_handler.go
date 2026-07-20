// backend/internal/adapters/in/http/mall/webhook/stripe_handler.go
package mallHandler

import (
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
	paymentUC *usecase.PaymentUsecase

	// Stripe webhook signing secret (whsec_...).
	signingSecret string

	// Maximum difference between the current time and the timestamp in
	// Stripe-Signature.
	tolerance time.Duration

	now func() time.Time
}

func NewStripeWebhookHandler(
	paymentUC *usecase.PaymentUsecase,
	signingSecret string,
) http.Handler {
	return &StripeWebhookHandler{
		paymentUC:     paymentUC,
		signingSecret: signingSecret,
		tolerance:     5 * time.Minute,
		now:           time.Now,
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

	// Unsupported event types and PaymentIntents that do not belong to this
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
	// - status transition validation
	// - status update
	// - first-succeeded post-paid marker acquisition
	//
	// A duplicate event is a successful no-op.
	if _, err := h.paymentUC.ApplyStripeEvent(
		r.Context(),
		input,
	); err != nil {
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

	writeStripeWebhookJSON(
		w,
		http.StatusOK,
		map[string]string{
			"status": "ok",
		},
	)
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

type stripePaymentIntent struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Created int64  `json:"created"`

	Metadata map[string]string `json:"metadata"`

	LastPaymentError *stripePaymentError `json:"last_payment_error"`

	CancellationReason string `json:"cancellation_reason"`
}

type stripePaymentError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
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
		EventID: eventID,

		PaymentID: paymentID,

		StripePaymentIntentID: stripePaymentIntentID,

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
// types to the Payment Domain status.
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
