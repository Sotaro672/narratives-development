// backend/internal/adapters/in/http/mall/handler/payment_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	// ✅ buyer auth context (uid)
	"narratives/internal/adapters/in/http/middleware"

	// keep for sentinel check if available
	mallquery "narratives/internal/application/query/mall"

	usecase "narratives/internal/application/usecase"
)

// PaymentHandler handles ONLY:
// - GET /mall/me/payment  ✅ (uid -> avatarId + shipping/billing + etc via order query)
//
// IMPORTANT:
// - /mall/payment は存在しない（受けない）
// - /payments/{id} もここでは受けない
type PaymentHandler struct {
	uc *usecase.PaymentUsecase // 互換のため残す（現状は未使用でもOK）
	// ✅ accept any (mall) and call ResolveByUID via reflection
	orderQ any
}

// NewPaymentHandler initializes handler.
// NOTE: /mall/me/payment を使うなら orderQ が必須（NewPaymentHandlerWithOrderQuery 推奨）
func NewPaymentHandler(uc *usecase.PaymentUsecase) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: nil}
}

// ✅ inject order query (for /mall/me/payment).
func NewPaymentHandlerWithOrderQuery(uc *usecase.PaymentUsecase, orderQ any) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: orderQ}
}

// ServeHTTP routes requests.
func (h *PaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ✅ Allow CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// normalize path (drop trailing slash)
	path0 := strings.TrimSuffix(r.URL.Path, "/")
	if path0 == "" {
		path0 = "/"
	}

	// ✅ support /mall/* mounts:
	// - /mall/me/payment -> /me/payment
	// - if router already stripped "/mall", it may already be "/me/payment"
	if strings.HasPrefix(path0, "/mall/") {
		path0 = strings.TrimPrefix(path0, "/mall")
		if path0 == "" {
			path0 = "/"
		}
	}

	switch {
	// ✅ ONLY: GET /mall/me/payment (normalized to /me/payment)
	case r.Method == http.MethodGet && path0 == "/me/payment":
		h.getPaymentContext(w, r)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// ------------------------------------------------------------
// GET /mall/me/payment  (uid -> avatarId + shipping/billing)
// ------------------------------------------------------------
func (h *PaymentHandler) getPaymentContext(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.orderQ == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_query_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	ctx := r.Context()

	out, err := callResolveByUID(h.orderQ, ctx, uid)
	if err != nil {
		// best-effort not found mapping
		if errors.Is(err, mallquery.ErrNotFound) || isNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

func callResolveByUID(orderQ any, ctx context.Context, uid string) (any, error) {
	if orderQ == nil {
		return nil, errors.New("order_query_not_initialized")
	}

	rv := reflect.ValueOf(orderQ)
	if !rv.IsValid() {
		return nil, errors.New("order_query_not_initialized")
	}

	// ResolveByUID(ctx, uid)
	m := rv.MethodByName("ResolveByUID")
	if !m.IsValid() {
		// try pointer receiver
		if rv.Kind() != reflect.Pointer && rv.CanAddr() {
			m = rv.Addr().MethodByName("ResolveByUID")
		}
	}
	if !m.IsValid() {
		return nil, errors.New("order_query_missing_method_ResolveByUID")
	}

	// arg check
	if m.Type().NumIn() != 2 || m.Type().NumOut() != 2 {
		return nil, errors.New("order_query_invalid_signature")
	}

	outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(uid)})
	if len(outs) != 2 {
		return nil, errors.New("order_query_invalid_signature")
	}

	var err error
	if !outs[1].IsNil() {
		if e, ok := outs[1].Interface().(error); ok {
			err = e
		} else {
			err = errors.New("order_query_returned_non_error")
		}
	}

	return outs[0].Interface(), err
}
