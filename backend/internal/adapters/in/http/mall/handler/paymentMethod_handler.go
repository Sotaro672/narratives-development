// backend/internal/adapters/in/http/mall/handler/paymentMethod_handler.go
package mallHandler

import (
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

const (
	paymentMethodHandlerTag = "[mall_payment_method_handler]"

	// PaymentMethod APIで受け付けるJSONリクエストの最大サイズです。
	paymentMethodMaxRequestBodyBytes int64 = 64 * 1024
)

// PaymentMethodHandlerは、/mall/me/payment-methods関連の
// HTTPエンドポイントを処理します。
type PaymentMethodHandler struct {
	uc *usecase.PaymentMethodUsecase

	stripePublicKeyOnce sync.Once
	stripePublicKey     string
	stripePublicKeyErr  error
}

// NewPaymentMethodHandlerはHTTPハンドラーを初期化します。
func NewPaymentMethodHandler(
	uc *usecase.PaymentMethodUsecase,
) http.Handler {
	return &PaymentMethodHandler{
		uc: uc,
	}
}

// ServeHTTPはHTTPルーティングの入口です。
func (h *PaymentMethodHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

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
	case r.Method == http.MethodGet &&
		path == "/mall/config/stripe":
		h.getStripeConfig(w, r)
		return

	// GET /mall/me/payment-methods
	case r.Method == http.MethodGet &&
		path == "/mall/me/payment-methods":
		h.list(w, r)
		return

	// GET /mall/me/payment-methods/default
	case r.Method == http.MethodGet &&
		path == "/mall/me/payment-methods/default":
		h.getDefault(w, r)
		return

	// POST /mall/me/payment-methods/setup-intent
	case r.Method == http.MethodPost &&
		path == "/mall/me/payment-methods/setup-intent":
		h.postSetupIntent(w, r)
		return

	// POST /mall/me/payment-methods
	case r.Method == http.MethodPost &&
		path == "/mall/me/payment-methods":
		h.post(w, r)
		return

	// PUT /mall/me/payment-methods/{id}/default
	case r.Method == http.MethodPut &&
		strings.HasPrefix(
			path,
			"/mall/me/payment-methods/",
		) &&
		strings.HasSuffix(path, "/default"):
		id := strings.TrimSuffix(
			strings.TrimPrefix(
				path,
				"/mall/me/payment-methods/",
			),
			"/default",
		)
		h.setDefault(w, r, id)
		return

	// GET /mall/me/payment-methods/{id}
	case r.Method == http.MethodGet &&
		strings.HasPrefix(
			path,
			"/mall/me/payment-methods/",
		):
		id := strings.TrimPrefix(
			path,
			"/mall/me/payment-methods/",
		)
		h.get(w, r, id)
		return

	// DELETE /mall/me/payment-methods/{id}
	case r.Method == http.MethodDelete &&
		strings.HasPrefix(
			path,
			"/mall/me/payment-methods/",
		):
		id := strings.TrimPrefix(
			path,
			"/mall/me/payment-methods/",
		)
		h.delete(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "not_found",
			},
		)
		return
	}
}

// ------------------------------------------------------------
// Authentication
// ------------------------------------------------------------

// requireUIDは、認証Middlewareがcontextへ設定したUIDを取得します。
func requireUID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	uid, ok := middleware.CurrentUserUID(r)
	uid = strings.TrimSpace(uid)

	if ok && uid != "" {
		return uid, true
	}

	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"error": "unauthorized",
		},
	)

	return "", false
}

// ------------------------------------------------------------
// Handlers
// ------------------------------------------------------------

// GET /mall/config/stripe
func (h *PaymentMethodHandler) getStripeConfig(
	w http.ResponseWriter,
	r *http.Request,
) {
	publicKey, err := h.getStripePublicKey(r.Context())
	if err != nil {
		log.Printf(
			"%s getStripeConfig failed err=%v",
			paymentMethodHandlerTag,
			err,
		)

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "stripe_public_key_not_configured",
			},
		)
		return
	}

	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"publishableKey": publicKey,
		},
	)
}

// GET /mall/me/payment-methods
func (h *PaymentMethodHandler) list(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"data": items,
		},
	)
}

// GET /mall/me/payment-methods/default
func (h *PaymentMethodHandler) getDefault(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"data": item,
		},
	)
}

// GET /mall/me/payment-methods/{id}
func (h *PaymentMethodHandler) get(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid id",
			},
		)
		return
	}

	item, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	// 他ユーザーのPaymentMethodは返却しません。
	if item.UserID != uid {
		writePaymentMethodErr(w, pm.ErrNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"data": item,
		},
	)
}

// POST /mall/me/payment-methods/setup-intent
func (h *PaymentMethodHandler) postSetupIntent(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	var in paymentMethodSetupIntentRequest

	// 空のbodyは許容します。
	// JSONが送信された場合、cardholderName以外のフィールドは拒否します。
	if err := decodePaymentMethodJSON(
		w,
		r,
		&in,
		true,
	); err != nil {
		log.Printf(
			"%s postSetupIntent decode failed uid=%q err=%v",
			paymentMethodHandlerTag,
			uid,
			err,
		)

		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid json",
			},
		)
		return
	}

	log.Printf(
		"%s postSetupIntent parsed uid=%q hasCardholderName=%v",
		paymentMethodHandlerTag,
		uid,
		strings.TrimSpace(in.CardholderName) != "",
	)

	result, err := h.uc.CreateSetupIntent(
		ctx,
		uid,
		in.CardholderName,
	)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"data": result,
		},
	)
}

// POST /mall/me/payment-methods
//
// Stripe.js / ElementsによるSetupIntent完了後、
// Stripeが発行したPaymentMethod IDと表示用カード情報を保存します。
// cardNumberおよびcvcは受け付けません。
func (h *PaymentMethodHandler) post(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	var in pm.CreatePaymentMethodInput
	if err := decodePaymentMethodJSON(
		w,
		r,
		&in,
		false,
	); err != nil {
		log.Printf(
			"%s post decode failed uid=%q err=%v",
			paymentMethodHandlerTag,
			uid,
			err,
		)

		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid json",
			},
		)
		return
	}

	// userIdはクライアント値を信用せず、
	// 認証Middlewareが設定したUIDで上書きします。
	in.UserID = uid

	maskedStripeCustomerID := maskIDForLog(
		in.StripeCustomerID,
	)
	maskedStripePaymentMethodID := maskIDForLog(
		in.StripePaymentMethodID,
	)

	// 生のリクエスト本文、カード番号、CVCはログへ出力しません。
	log.Printf(
		"%s post parsed userId=%q stripeCustomerId=%q stripePaymentMethodId=%q brand=%q expMonth=%d expYear=%d hasCardholderName=%v isDefault=%v",
		paymentMethodHandlerTag,
		in.UserID,
		maskedStripeCustomerID,
		maskedStripePaymentMethodID,
		in.Brand,
		in.ExpMonth,
		in.ExpYear,
		strings.TrimSpace(in.CardholderName) != "",
		in.IsDefault,
	)

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"data": created,
		},
	)
}

// PUT /mall/me/payment-methods/{id}/default
func (h *PaymentMethodHandler) setDefault(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid id",
			},
		)
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

	log.Printf(
		"%s setDefault start id=%q userId=%q",
		paymentMethodHandlerTag,
		id,
		uid,
	)

	updated, err := h.uc.SetDefault(
		ctx,
		id,
		uid,
	)
	if err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	log.Printf(
		"%s setDefault ok id=%q userId=%q",
		paymentMethodHandlerTag,
		updated.ID,
		updated.UserID,
	)

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"data": updated,
		},
	)
}

// DELETE /mall/me/payment-methods/{id}
func (h *PaymentMethodHandler) delete(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	ctx := r.Context()

	uid, ok := requireUID(w, r)
	if !ok {
		return
	}

	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "invalid id",
			},
		)
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

	log.Printf(
		"%s delete start id=%q",
		paymentMethodHandlerTag,
		id,
	)

	if err := h.uc.Delete(ctx, id); err != nil {
		writePaymentMethodErr(w, err)
		return
	}

	log.Printf(
		"%s delete ok id=%q",
		paymentMethodHandlerTag,
		id,
	)

	w.WriteHeader(http.StatusNoContent)
}

// ------------------------------------------------------------
// Error handling
// ------------------------------------------------------------

func writePaymentMethodErr(
	w http.ResponseWriter,
	err error,
) {
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

	case errors.Is(
		err,
		usecase.ErrSetupIntentNotImplemented,
	):
		code = http.StatusNotImplemented
	}

	if code == http.StatusInternalServerError {
		log.Printf(
			"%s internal error err=%v",
			paymentMethodHandlerTag,
			err,
		)
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"error": err.Error(),
		},
	)
}

// ============================================================
// Local helpers
// ============================================================

type paymentMethodSetupIntentRequest struct {
	CardholderName string `json:"cardholderName"`
}

func (h *PaymentMethodHandler) getStripePublicKey(
	ctx context.Context,
) (string, error) {
	h.stripePublicKeyOnce.Do(func() {
		h.stripePublicKey,
			h.stripePublicKeyErr = accessSecretVersion(
			ctx,
			"stripe-public-key",
		)
	})

	return h.stripePublicKey,
		h.stripePublicKeyErr
}

func accessSecretVersion(
	ctx context.Context,
	secretID string,
) (string, error) {
	projectID := strings.TrimSpace(
		os.Getenv("GOOGLE_CLOUD_PROJECT"),
	)
	if projectID == "" {
		projectID = strings.TrimSpace(
			os.Getenv("GCP_PROJECT"),
		)
	}
	if projectID == "" {
		projectID = strings.TrimSpace(
			os.Getenv("PROJECT_ID"),
		)
	}
	if projectID == "" {
		return "", errors.New(
			"google cloud project id is not configured",
		)
	}

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	name := "projects/" +
		projectID +
		"/secrets/" +
		secretID +
		"/versions/latest"

	result, err := client.AccessSecretVersion(
		ctx,
		&secretmanagerpb.AccessSecretVersionRequest{
			Name: name,
		},
	)
	if err != nil {
		return "", err
	}
	if result.Payload == nil {
		return "", errors.New(
			secretID + " payload is nil",
		)
	}

	value := strings.TrimSpace(
		string(result.Payload.Data),
	)
	if value == "" {
		return "", errors.New(
			secretID + " is empty",
		)
	}

	return value, nil
}

// decodePaymentMethodJSONは、リクエスト本文をストリームで読み取ります。
//
// 次の条件を保証します。
//
//   - 最大リクエストサイズを制限する
//   - CreatePaymentMethodInputに存在しないフィールドを拒否する
//   - cardNumberおよびcvcが送られた場合は拒否する
//   - 複数のJSON値を含むリクエストを拒否する
//   - リクエスト本文をログへ出力しない
func decodePaymentMethodJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
	allowEmpty bool,
) error {
	if r.Body == nil {
		if allowEmpty {
			return nil
		}
		return io.EOF
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		paymentMethodMaxRequestBodyBytes,
	)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if allowEmpty && errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	var extra any
	err := decoder.Decode(&extra)

	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}

	return errors.New(
		"request body must contain exactly one JSON value",
	)
}

func maskIDForLog(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return "***"
	}

	return value[:3] +
		"***" +
		value[len(value)-3:]
}
