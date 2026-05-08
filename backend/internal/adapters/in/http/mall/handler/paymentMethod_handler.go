// backend/internal/adapters/in/http/mall/handler/paymentMethod_handler.go
package mallHandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	pm "narratives/internal/domain/paymentMethod"
)

const paymentMethodHandlerTag = "[mall_payment_method_handler]"

// PaymentMethodHandler は /mall/me/payment-methods 関連のエンドポイントを担当します。
type PaymentMethodHandler struct {
	uc *usecase.PaymentMethodUsecase

	stripePublicKeyOnce sync.Once
	stripePublicKey     string
	stripePublicKeyErr  error
}

// NewPaymentMethodHandler は HTTP ハンドラを初期化します。
func NewPaymentMethodHandler(uc *usecase.PaymentMethodUsecase) http.Handler {
	return &PaymentMethodHandler{uc: uc}
}

// ServeHTTP は HTTP ルーティングの入口です。
func (h *PaymentMethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	log.Printf(
		"%s enter method=%s path=%s trace=%q contentType=%q contentLen=%d",
		paymentMethodHandlerTag,
		strings.ToUpper(r.Method),
		r.URL.Path,
		r.Header.Get("X-Cloud-Trace-Context"),
		r.Header.Get("Content-Type"),
		r.ContentLength,
	)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch {
	// GET /mall/config/stripe
	case r.Method == http.MethodGet && path == "/mall/config/stripe":
		h.getStripeConfig(w, r)
		return

	// GET /mall/me/payment-methods
	case r.Method == http.MethodGet && path == "/mall/me/payment-methods":
		h.list(w, r)
		return

	// GET /mall/me/payment-methods/default
	case r.Method == http.MethodGet && path == "/mall/me/payment-methods/default":
		h.getDefault(w, r)
		return

	// POST /mall/me/payment-methods/setup-intent
	case r.Method == http.MethodPost && path == "/mall/me/payment-methods/setup-intent":
		h.postSetupIntent(w, r)
		return

	// POST /mall/me/payment-methods
	case r.Method == http.MethodPost && path == "/mall/me/payment-methods":
		h.post(w, r)
		return

	// PUT /mall/me/payment-methods/{id}/default
	case r.Method == http.MethodPut && strings.HasPrefix(path, "/mall/me/payment-methods/") && strings.HasSuffix(path, "/default"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/mall/me/payment-methods/"), "/default")
		h.setDefault(w, r, id)
		return

	// GET /mall/me/payment-methods/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/mall/me/payment-methods/"):
		id := strings.TrimPrefix(path, "/mall/me/payment-methods/")
		h.get(w, r, id)
		return

	// DELETE /mall/me/payment-methods/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/mall/me/payment-methods/"):
		id := strings.TrimPrefix(path, "/mall/me/payment-methods/")
		h.delete(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// ------------------------------------------------------------
// auth helpers
// ------------------------------------------------------------

// UserAuthMiddleware が context に入れた uid を取得して userId として使う
func requireUID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid, ok := middleware.CurrentUserUID(r)
	if ok && uid != "" {
		return uid, true
	}
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	return "", false
}

// ------------------------------------------------------------
// handlers
// ------------------------------------------------------------

// GET /mall/config/stripe
func (h *PaymentMethodHandler) getStripeConfig(w http.ResponseWriter, r *http.Request) {
	publicKey, err := h.getStripePublicKey(r.Context())
	if err != nil {
		log.Printf("%s getStripeConfig failed err=%v", paymentMethodHandlerTag, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "stripe_public_key_not_configured",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"publishableKey": publicKey,
	})
}

// GET /mall/me/payment-methods
func (h *PaymentMethodHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	items, err := h.uc.GetByUser(ctx, uid)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": items,
	})
}

// GET /mall/me/payment-methods/default
func (h *PaymentMethodHandler) getDefault(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	item, err := h.uc.GetDefaultByUser(ctx, uid)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": item,
	})
}

// GET /mall/me/payment-methods/{id}
func (h *PaymentMethodHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	if id == "" || strings.Contains(id, "/") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	item, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	if item.UserID != uid {
		writePaymentMethodErr(w, pm.ErrNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": item,
	})
}

// POST /mall/me/payment-methods/setup-intent
func (h *PaymentMethodHandler) postSetupIntent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	raw, head, err := readPaymentMethodBodyWithHead(r, 220)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	log.Printf("%s postSetupIntent raw body len=%d head=%q uid=%q", paymentMethodHandlerTag, len(raw), head, uid)

	var in paymentMethodSetupIntentRequest
	if len(raw) > 0 {
		if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&in); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
			return
		}
	}

	log.Printf(
		"%s postSetupIntent parsed uid=%q cardholderName=%q",
		paymentMethodHandlerTag,
		uid,
		in.CardholderName,
	)

	result, err := h.uc.CreateSetupIntent(ctx, uid, in.CardholderName)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": result,
	})
}

// POST /mall/me/payment-methods
func (h *PaymentMethodHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	raw, head, err := readPaymentMethodBodyWithHead(r, 220)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	log.Printf("%s post raw body len=%d head=%q", paymentMethodHandlerTag, len(raw), head)

	var in pm.CreatePaymentMethodInput
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// anti-spoof: userId は middleware の uid を強制
	in.UserID = uid

	maskedStripeCustomerID := maskIDForLog(in.StripeCustomerID)
	maskedStripePaymentMethodID := maskIDForLog(in.StripePaymentMethodID)

	log.Printf(
		"%s post parsed userId=%q stripeCustomerId=%q stripePaymentMethodId=%q brand=%q expMonth=%d expYear=%d cardholderName=%q isDefault=%v hasCardNumber=%v hasCVC=%v",
		paymentMethodHandlerTag,
		in.UserID,
		maskedStripeCustomerID,
		maskedStripePaymentMethodID,
		in.Brand,
		in.ExpMonth,
		in.ExpYear,
		in.CardholderName,
		in.IsDefault,
		strings.TrimSpace(in.CardNumber) != "",
		strings.TrimSpace(in.CVC) != "",
	)

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": created,
	})
}

// PUT /mall/me/payment-methods/{id}/default
func (h *PaymentMethodHandler) setDefault(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	if id == "" || strings.Contains(id, "/") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	existing, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}
	if existing.UserID != uid {
		writePaymentMethodErr(w, pm.ErrNotFound)
		return
	}

	log.Printf("%s setDefault start id=%q userId=%q", paymentMethodHandlerTag, id, uid)
	updated, err := h.uc.SetDefault(ctx, id, uid)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	log.Printf("%s setDefault ok id=%q userId=%q", paymentMethodHandlerTag, updated.ID, updated.UserID)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": updated,
	})
}

// DELETE /mall/me/payment-methods/{id}
func (h *PaymentMethodHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	if id == "" || strings.Contains(id, "/") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	existing, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}
	if existing.UserID != uid {
		writePaymentMethodErr(w, pm.ErrNotFound)
		return
	}

	log.Printf("%s delete start id=%q", paymentMethodHandlerTag, id)
	if err := h.uc.Delete(ctx, id); err != nil {
		writePaymentMethodErr(w, err)
		return
	}
	log.Printf("%s delete ok id=%q", paymentMethodHandlerTag, id)

	w.WriteHeader(http.StatusNoContent)
}

// ------------------------------------------------------------
// error handling
// ------------------------------------------------------------

func writePaymentMethodErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, pm.ErrInvalidID):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidUserID):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidStripeCustomerID):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidStripePaymentMethod):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidBrand):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidLast4):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidExpMonth):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidExpYear):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidCardholderName):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidCreatedAt):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrInvalidUpdatedAt):
		code = http.StatusBadRequest
	case errors.Is(err, pm.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, pm.ErrConflict):
		code = http.StatusConflict
	case errors.Is(err, usecase.ErrInvalidCardNumber):
		code = http.StatusBadRequest
	case errors.Is(err, usecase.ErrInvalidCVC):
		code = http.StatusBadRequest
	case errors.Is(err, usecase.ErrSetupIntentNotImplemented):
		code = http.StatusNotImplemented
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// ============================================================
// helpers (local)
// ============================================================

type paymentMethodSetupIntentRequest struct {
	CardholderName string `json:"cardholderName"`
}

func (h *PaymentMethodHandler) getStripePublicKey(ctx context.Context) (string, error) {
	h.stripePublicKeyOnce.Do(func() {
		h.stripePublicKey, h.stripePublicKeyErr = accessSecretVersion(
			ctx,
			"stripe-public-key",
		)
	})

	return h.stripePublicKey, h.stripePublicKeyErr
}

func accessSecretVersion(ctx context.Context, secretID string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
	}
	if projectID == "" {
		projectID = os.Getenv("PROJECT_ID")
	}
	if projectID == "" {
		return "", errors.New("google cloud project id is not configured")
	}

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	name := "projects/" + projectID + "/secrets/" + secretID + "/versions/latest"

	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(string(result.Payload.Data))
	if value == "" {
		return "", errors.New(secretID + " is empty")
	}

	return value, nil
}

func readPaymentMethodBodyWithHead(r *http.Request, headN int) (raw []byte, head string, err error) {
	if r.Body == nil {
		return []byte{}, "", nil
	}
	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, "", err
	}

	h := string(b)
	if headN <= 0 {
		headN = 200
	}
	if len(h) > headN {
		h = h[:headN]
	}
	return b, h, nil
}

func maskIDForLog(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "***" + s[len(s)-3:]
}
