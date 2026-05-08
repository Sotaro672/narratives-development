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

type StripeWebhookHandler struct {
	paymentUC *usecase.PaymentUsecase

	// Stripe webhook signing secret (whsec_...)
	signingSecret string

	// signature tolerance (e.g. 5 minutes)
	tolerance time.Duration
}

func NewStripeWebhookHandler(paymentUC *usecase.PaymentUsecase, signingSecret string) http.Handler {
	return &StripeWebhookHandler{
		paymentUC:     paymentUC,
		signingSecret: signingSecret,
		tolerance:     5 * time.Minute,
	}
}

func (h *StripeWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Stripe はブラウザから叩かれない想定だが、Cloud Run 前段などで OPTIONS が来ても落とさない
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	if h == nil || h.paymentUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payment_usecase_not_initialized"})
		return
	}

	secret := strings.TrimSpace(h.signingSecret)
	if secret == "" {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "stripe_webhook_secret_not_configured"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_body"})
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	if sigHeader == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing_stripe_signature"})
		return
	}

	if err := verifyStripeSignature(sigHeader, body, secret, time.Now().UTC(), h.tolerance); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_signature"})
		return
	}

	var ev stripeEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	paymentID, paymentMethodID, amount, stripePaymentIntentID, stripeCustomerID, stripePaymentMethodID, ok := extractPaidInfoFromStripeEvent(ev)
	if !ok || strings.TrimSpace(paymentID) == "" {
		// paymentId が取れないイベントは「受け取ったが処理しない」。
		// Stripe には 2xx を返す（リトライを増やさない）。
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	paymentID = strings.TrimSpace(paymentID)
	paymentMethodID = strings.TrimSpace(paymentMethodID)
	stripePaymentIntentID = strings.TrimSpace(stripePaymentIntentID)
	stripeCustomerID = strings.TrimSpace(stripeCustomerID)
	stripePaymentMethodID = strings.TrimSpace(stripePaymentMethodID)

	if paymentMethodID == "" || stripeCustomerID == "" || stripePaymentMethodID == "" || stripePaymentIntentID == "" || amount <= 0 {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	p, err := paymentdom.New(
		paymentID,
		paymentMethodID,
		stripeCustomerID,
		stripePaymentMethodID,
		stripePaymentIntentID,
		amount,
		paymentdom.StatusSucceeded,
		nil,
		nil,
		nil,
		time.Now().UTC(),
	)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	if err := h.paymentUC.Update(r.Context(), p); err != nil {
		// Stripe へのリトライを促すなら 500 を返す
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ------------------------------------------------------------
// Stripe signature verification (no stripe-go dependency)
// ------------------------------------------------------------

// Stripe-Signature: t=timestamp,v1=signature(,v0=...)
// signed_payload = "{t}.{raw_body}"
// expected_sig = HMAC_SHA256(secret, signed_payload)
func verifyStripeSignature(sigHeader string, body []byte, secret string, now time.Time, tolerance time.Duration) error {
	t, sigs, err := parseStripeSignatureHeader(sigHeader)
	if err != nil {
		return err
	}

	ts := time.Unix(t, 0).UTC()
	if tolerance > 0 {
		d := now.Sub(ts)
		if d < 0 {
			d = -d
		}
		if d > tolerance {
			return errors.New("timestamp_out_of_tolerance")
		}
	}

	signed := fmt.Sprintf("%d.%s", t, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signed))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, s := range sigs {
		if subtleEqHex(expected, s) {
			return nil
		}
	}

	return errors.New("signature_mismatch")
}

func parseStripeSignatureHeader(h string) (timestamp int64, v1s []string, err error) {
	parts := strings.Split(h, ",")

	var tStr string
	sigs := make([]string, 0)

	for _, p := range parts {
		if strings.HasPrefix(p, "t=") {
			tStr = strings.TrimPrefix(p, "t=")
			continue
		}
		if strings.HasPrefix(p, "v1=") {
			sigs = append(sigs, strings.TrimPrefix(p, "v1="))
			continue
		}
	}

	if tStr == "" || len(sigs) == 0 {
		return 0, nil, errors.New("invalid_signature_header")
	}

	t, e := strconv.ParseInt(tStr, 10, 64)
	if e != nil {
		return 0, nil, errors.New("invalid_signature_timestamp")
	}

	return t, sigs, nil
}

// constant-time-ish hex compare (avoid leaking early mismatch)
func subtleEqHex(a, b string) bool {
	ab := []byte(strings.ToLower(a))
	bb := []byte(strings.ToLower(b))

	if len(ab) != len(bb) {
		return false
	}

	var v byte = 0
	for i := 0; i < len(ab); i++ {
		v |= ab[i] ^ bb[i]
	}

	return v == 0
}

// ------------------------------------------------------------
// Stripe event parsing (minimal)
// ------------------------------------------------------------

type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripePaymentIntent struct {
	ID            string            `json:"id"`
	Amount        int               `json:"amount"` // in smallest currency unit
	Currency      string            `json:"currency"`
	Customer      string            `json:"customer"`
	PaymentMethod string            `json:"payment_method"`
	Metadata      map[string]string `json:"metadata"`
}

type stripeCheckoutSession struct {
	ID            string            `json:"id"`
	Mode          string            `json:"mode"`
	Amount        int               `json:"amount_total"`
	Currency      string            `json:"currency"`
	Customer      string            `json:"customer"`
	Metadata      map[string]string `json:"metadata"`
	PaymentIntent string            `json:"payment_intent"`
}

// extractPaidInfoFromStripeEvent returns:
// paymentID, paymentMethodID(app), amount, stripePaymentIntentID,
// stripeCustomerID, stripePaymentMethodID, ok.
//
// paymentID must be the same value as order.ID.
func extractPaidInfoFromStripeEvent(ev stripeEvent) (
	paymentID string,
	paymentMethodID string,
	amount int,
	stripePaymentIntentID string,
	stripeCustomerID string,
	stripePaymentMethodID string,
	ok bool,
) {
	switch ev.Type {
	case "payment_intent.succeeded":
		var pi stripePaymentIntent
		if err := json.Unmarshal(ev.Data.Object, &pi); err != nil {
			return "", "", 0, "", "", "", false
		}

		paymentID = firstNonEmpty(pi.Metadata["paymentId"], pi.Metadata["orderId"])
		paymentMethodID = pi.Metadata["paymentMethodId"]
		amount = pi.Amount
		stripePaymentIntentID = pi.ID
		stripeCustomerID = pi.Customer
		stripePaymentMethodID = firstNonEmpty(pi.PaymentMethod, pi.Metadata["stripePaymentMethodId"])

		return paymentID, paymentMethodID, amount, stripePaymentIntentID, stripeCustomerID, stripePaymentMethodID, true

	case "checkout.session.completed":
		var cs stripeCheckoutSession
		if err := json.Unmarshal(ev.Data.Object, &cs); err != nil {
			return "", "", 0, "", "", "", false
		}

		if cs.Mode != "" && !strings.EqualFold(cs.Mode, "payment") {
			return "", "", 0, "", "", "", false
		}

		paymentID = firstNonEmpty(cs.Metadata["paymentId"], cs.Metadata["orderId"])
		paymentMethodID = cs.Metadata["paymentMethodId"]
		amount = cs.Amount
		stripePaymentIntentID = cs.PaymentIntent
		if stripePaymentIntentID == "" {
			stripePaymentIntentID = cs.ID
		}
		stripeCustomerID = cs.Customer
		stripePaymentMethodID = cs.Metadata["stripePaymentMethodId"]

		return paymentID, paymentMethodID, amount, stripePaymentIntentID, stripeCustomerID, stripePaymentMethodID, true

	default:
		return "", "", 0, "", "", "", false
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
