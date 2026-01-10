// backend/internal/adapters/in/http/mall/handler/payment_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
)

type PaymentHandler struct {
	uc     *usecase.PaymentUsecase
	orderQ any
}

func NewPaymentHandler(uc *usecase.PaymentUsecase) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: nil}
}

func NewPaymentHandlerWithOrderQuery(uc *usecase.PaymentUsecase, orderQ any) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: orderQ}
}

func (h *PaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")
	if path0 == "" {
		path0 = "/"
	}

	if strings.HasPrefix(path0, "/mall/") {
		path0 = strings.TrimPrefix(path0, "/mall")
		if path0 == "" {
			path0 = "/"
		}
	}

	switch {
	case r.Method == http.MethodGet && path0 == "/me/payment":
		h.getPaymentContext(w, r)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

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

	out, err := callResolveByUID(h.orderQ, r.Context(), uid)
	if err != nil {
		if errors.Is(err, mallquery.ErrNotFound) || isNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	// ✅ Go struct のままだと BillingAddressID 等になりがちなので、
	//    map化して lowerCamel のキーを補う（フロント互換）
	m, convOK := toMap(out)
	if convOK {
		alias(m, "BillingAddressID", "billingAddressId")
		alias(m, "ShippingAddressID", "shippingAddressId")
		alias(m, "InvoiceID", "invoiceId")
		alias(m, "OrderID", "orderId")
		alias(m, "AvatarID", "avatarId")
		alias(m, "UserID", "userId")
		alias(m, "CreatedAt", "createdAt")
		alias(m, "UpdatedAt", "updatedAt")

		// もし "billingAddressID" という中途半端camelが来るケースも吸収
		alias(m, "billingAddressID", "billingAddressId")
		alias(m, "shippingAddressID", "shippingAddressId")
		alias(m, "invoiceID", "invoiceId")
		alias(m, "orderID", "orderId")
		alias(m, "avatarID", "avatarId")
		alias(m, "userID", "userId")

		_ = json.NewEncoder(w).Encode(m)
		return
	}

	// fallback
	_ = json.NewEncoder(w).Encode(out)
}

func toMap(v any) (map[string]any, bool) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false
	}
	return m, true
}

func alias(m map[string]any, from, to string) {
	if m == nil {
		return
	}
	// すでに期待キーがあり、値も入っているなら何もしない
	if v, ok := m[to]; ok {
		if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
			return
		}
	}
	// 元キーがあればコピー
	if v, ok := m[from]; ok {
		m[to] = v
	}
}

func callResolveByUID(orderQ any, ctx context.Context, uid string) (any, error) {
	if orderQ == nil {
		return nil, errors.New("order_query_not_initialized")
	}

	rv := reflect.ValueOf(orderQ)
	if !rv.IsValid() {
		return nil, errors.New("order_query_not_initialized")
	}

	m := rv.MethodByName("ResolveByUID")
	if !m.IsValid() {
		if rv.Kind() != reflect.Pointer && rv.CanAddr() {
			m = rv.Addr().MethodByName("ResolveByUID")
		}
	}
	if !m.IsValid() {
		return nil, errors.New("order_query_missing_method_ResolveByUID")
	}

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
