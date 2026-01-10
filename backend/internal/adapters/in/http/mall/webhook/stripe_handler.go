// backend\internal\adapters\in\http\mall\webhook\stripe_handler.go
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
	"reflect"
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
		signingSecret: strings.TrimSpace(signingSecret),
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

	// Stripe event minimal parse
	var ev stripeEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	// 支払い成功系のみを扱う（必要に応じて増やす）
	// - payment_intent.succeeded
	// - checkout.session.completed (mode=payment の場合)
	invoiceID, billingAddressID, amount, providerRef, ok := extractPaidInfoFromStripeEvent(ev)
	if !ok || strings.TrimSpace(invoiceID) == "" {
		// invoiceId が取れないイベントは「受け取ったが処理しない」
		// Stripe には 2xx を返す（リトライを増やさない）
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	// CreatePaymentInput を「存在するフィールドだけ」セットして Create
	in := buildCreatePaymentInputBestEffort(invoiceID, billingAddressID, amount, "stripe", providerRef, ev.Type)

	ctx := r.Context()
	_, cErr := h.paymentUC.Create(ctx, in)
	if cErr != nil {
		// Stripe へのリトライを促すなら 500 を返す（ただし二重起票の冪等性が必要）
		// まずは運用上の安全性を優先し、ここは 500 にするのが無難。
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

	// tolerance check
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

	// constant-time compare against any v1 signatures
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
	var sigs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
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
	ab := []byte(strings.ToLower(strings.TrimSpace(a)))
	bb := []byte(strings.ToLower(strings.TrimSpace(b)))
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

// payment_intent
type stripePaymentIntent struct {
	ID       string            `json:"id"`
	Amount   int               `json:"amount"` // in smallest currency unit
	Currency string            `json:"currency"`
	Metadata map[string]string `json:"metadata"`
}

// checkout.session
type stripeCheckoutSession struct {
	ID       string            `json:"id"`
	Mode     string            `json:"mode"`
	Amount   int               `json:"amount_total"`
	Currency string            `json:"currency"`
	Metadata map[string]string `json:"metadata"`
	// sometimes: payment_intent: "pi_..."
	PaymentIntent string `json:"payment_intent"`
}

func extractPaidInfoFromStripeEvent(ev stripeEvent) (invoiceID string, billingAddressID string, amount int, providerRef string, ok bool) {
	typ := strings.TrimSpace(ev.Type)

	switch typ {
	case "payment_intent.succeeded":
		var pi stripePaymentIntent
		if err := json.Unmarshal(ev.Data.Object, &pi); err != nil {
			return "", "", 0, "", false
		}
		invoiceID = strings.TrimSpace(pi.Metadata["invoiceId"])
		if invoiceID == "" {
			// orderId を使う実装なら fallback
			invoiceID = strings.TrimSpace(pi.Metadata["orderId"])
		}
		billingAddressID = strings.TrimSpace(pi.Metadata["billingAddressId"])
		amount = pi.Amount
		providerRef = strings.TrimSpace(pi.ID)
		return invoiceID, billingAddressID, amount, providerRef, true

	case "checkout.session.completed":
		var cs stripeCheckoutSession
		if err := json.Unmarshal(ev.Data.Object, &cs); err != nil {
			return "", "", 0, "", false
		}
		// mode=payment のみ扱う（subscription などはここでは無視）
		if strings.TrimSpace(cs.Mode) != "" && !strings.EqualFold(strings.TrimSpace(cs.Mode), "payment") {
			return "", "", 0, "", false
		}
		invoiceID = strings.TrimSpace(cs.Metadata["invoiceId"])
		if invoiceID == "" {
			invoiceID = strings.TrimSpace(cs.Metadata["orderId"])
		}
		billingAddressID = strings.TrimSpace(cs.Metadata["billingAddressId"])
		amount = cs.Amount
		// payment_intent が取れるならそれを providerRef にする
		if strings.TrimSpace(cs.PaymentIntent) != "" {
			providerRef = strings.TrimSpace(cs.PaymentIntent)
		} else {
			providerRef = strings.TrimSpace(cs.ID)
		}
		return invoiceID, billingAddressID, amount, providerRef, true

	default:
		return "", "", 0, "", false
	}
}

// ------------------------------------------------------------
// CreatePaymentInput builder (best-effort via reflection)
// ------------------------------------------------------------

func buildCreatePaymentInputBestEffort(invoiceID, billingAddressID string, amount int, provider, providerRef, eventType string) paymentdom.CreatePaymentInput {
	in := paymentdom.CreatePaymentInput{}

	rv := reflect.ValueOf(&in).Elem()
	setStringField(rv, "InvoiceID", invoiceID)
	setStringField(rv, "BillingAddressID", billingAddressID)

	// amount (int/int64/float64 などの可能性を吸収)
	setIntField(rv, "Amount", int64(amount))

	// status: paid
	// PaymentStatus が独自型(string)でも Kind() は string なので set できる
	setStringField(rv, "Status", "paid")

	// optional fields (存在すれば入る)
	setStringField(rv, "Provider", provider)
	setStringField(rv, "ProviderRef", providerRef)
	setStringField(rv, "ProviderPaymentID", providerRef)
	setStringField(rv, "EventType", eventType)

	// timestamps (存在すれば入る)
	now := time.Now().UTC()
	setTimeField(rv, "CreatedAt", now)
	setTimePtrField(rv, "UpdatedAt", now)

	return in
}

func setStringField(rv reflect.Value, name string, val string) {
	f := rv.FieldByName(name)
	if !f.IsValid() || !f.CanSet() {
		return
	}
	if f.Kind() == reflect.String {
		f.SetString(val)
		return
	}
	// named string type also appears as Kind()==String, so above is enough
}

func setIntField(rv reflect.Value, name string, val int64) {
	f := rv.FieldByName(name)
	if !f.IsValid() || !f.CanSet() {
		return
	}
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val >= 0 {
			f.SetUint(uint64(val))
		}
	}
}

func setTimeField(rv reflect.Value, name string, t time.Time) {
	f := rv.FieldByName(name)
	if !f.IsValid() || !f.CanSet() {
		return
	}
	// time.Time
	if f.Type() == reflect.TypeOf(time.Time{}) && f.Kind() == reflect.Struct {
		f.Set(reflect.ValueOf(t))
	}
}

func setTimePtrField(rv reflect.Value, name string, t time.Time) {
	f := rv.FieldByName(name)
	if !f.IsValid() || !f.CanSet() {
		return
	}
	// *time.Time
	if f.Kind() == reflect.Pointer && f.Type().Elem() == reflect.TypeOf(time.Time{}) {
		tt := t
		f.Set(reflect.ValueOf(&tt))
	}
}
