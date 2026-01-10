// backend/internal/adapters/in/http/mall/handler/payment_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
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

	// Preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")
	if path0 == "" {
		path0 = "/"
	}

	// strip "/mall"
	if strings.HasPrefix(path0, "/mall/") {
		path0 = strings.TrimPrefix(path0, "/mall")
		if path0 == "" {
			path0 = "/"
		}
	}

	switch {
	// GET /mall/me/payment : payment context (uid -> avatarId + addresses + cart)
	case r.Method == http.MethodGet && path0 == "/me/payment":
		h.getPaymentContext(w, r)
		return

	// POST /mall/me/payments : create payment (buyer-flow dev)
	case r.Method == http.MethodPost && path0 == "/me/payments":
		h.postPayments(w, r)
		return

	// GET /mall/me/payments?invoiceId=... : list payments by invoiceId (optional)
	case r.Method == http.MethodGet && path0 == "/me/payments":
		h.getPayments(w, r)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// ------------------------------------------------------------
// GET /me/payment (payment context)
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

	out, err := payCallResolveByUID(h.orderQ, r.Context(), uid)
	if err != nil {
		if errors.Is(err, mallquery.ErrNotFound) || payIsNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
		log.Printf("[mall/payment] GET /me/payment failed uid=%q err=%v", payMaskUID(uid), err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "internal_error", "detail": err.Error()})
		return
	}

	// struct -> map (lowerCamel key aliasing, frontend compatibility)
	if m, ok := payToMap(out); ok {
		payAlias(m, "BillingAddressID", "billingAddressId")
		payAlias(m, "ShippingAddressID", "shippingAddressId")
		payAlias(m, "InvoiceID", "invoiceId")
		payAlias(m, "OrderID", "orderId")
		payAlias(m, "AvatarID", "avatarId")
		payAlias(m, "UserID", "userId")
		payAlias(m, "UID", "uid")

		// some intermediate camel variants (best-effort)
		payAlias(m, "billingAddressID", "billingAddressId")
		payAlias(m, "shippingAddressID", "shippingAddressId")
		payAlias(m, "invoiceID", "invoiceId")
		payAlias(m, "orderID", "orderId")
		payAlias(m, "avatarID", "avatarId")
		payAlias(m, "userID", "userId")

		_ = json.NewEncoder(w).Encode(m)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ------------------------------------------------------------
// POST /me/payments (create payment)
// ------------------------------------------------------------

type payCreateReq struct {
	InvoiceID        string
	BillingAddressID string
	Amount           int
	Status           string
}

func (h *PaymentHandler) postPayments(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payment_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	req := payCreateReq{
		// invoiceId docId=orderId 前提なら orderId を受けてもよい
		InvoiceID: payPickString(body, "invoiceId", "invoiceID", "invoice_id", "orderId", "orderID", "order_id"),
		BillingAddressID: payPickString(
			body,
			"billingAddressId", "billingAddressID", "billing_address_id",
			"billingId", "billingID",
			"addressId", "addressID",
			"id", // fallback
		),
		Amount: payPickInt(body, "amount", "total", "totalAmount", "total_amount"),
		Status: payPickString(body, "status", "paymentStatus"),
	}

	req.InvoiceID = strings.TrimSpace(req.InvoiceID)
	req.BillingAddressID = strings.TrimSpace(req.BillingAddressID)
	req.Status = strings.TrimSpace(req.Status)

	if req.InvoiceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invoiceId_required"})
		return
	}
	if req.BillingAddressID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "billingAddressId_required"})
		return
	}
	if req.Amount <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "amount_invalid"})
		return
	}

	// Build paymentdom.CreatePaymentInput WITHOUT assuming its fields at compile-time.
	var in paymentdom.CreatePaymentInput
	paySetStringFieldBestEffort(&in, req.InvoiceID, "InvoiceID", "InvoiceId", "invoiceId")
	paySetStringFieldBestEffort(&in, req.BillingAddressID, "BillingAddressID", "BillingAddressId", "billingAddressId")
	paySetIntFieldBestEffort(&in, req.Amount, "Amount", "Total", "TotalAmount")

	// dev default: if caller did not provide status, set "paid" best-effort
	// (PaymentUsecase marks invoice paid when created payment has paid/succeeded status)
	if req.Status == "" {
		req.Status = "paid"
	}
	paySetStatusFieldBestEffort(&in, req.Status, "Status", "PaymentStatus")

	p, err := h.uc.Create(r.Context(), in)
	if err != nil {
		log.Printf("[mall/payment] POST /me/payments failed uid=%q invoiceId=%q billingAddressId=%q amount=%d err=%v",
			payMaskUID(uid), payMaskUID(req.InvoiceID), payMaskUID(req.BillingAddressID), req.Amount, err)

		if errors.Is(err, mallquery.ErrNotFound) || payIsNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "detail": err.Error()})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "internal_error", "detail": err.Error()})
		return
	}

	// Response (best-effort alias)
	if m, ok := payToMap(p); ok {
		payAlias(m, "ID", "id")
		payAlias(m, "PaymentID", "paymentId")
		payAlias(m, "InvoiceID", "invoiceId")
		payAlias(m, "BillingAddressID", "billingAddressId")
		payAlias(m, "CreatedAt", "createdAt")
		payAlias(m, "UpdatedAt", "updatedAt")

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(m)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

// ------------------------------------------------------------
// GET /me/payments (list payments by invoiceId)
// ------------------------------------------------------------

func (h *PaymentHandler) getPayments(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payment_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	invoiceID := strings.TrimSpace(r.URL.Query().Get("invoiceId"))
	if invoiceID == "" {
		// accept orderId as alias
		invoiceID = strings.TrimSpace(r.URL.Query().Get("orderId"))
	}

	if invoiceID == "" {
		// keep it simple: caller should provide invoiceId/orderId
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invoiceId_required"})
		return
	}

	list, err := h.uc.GetByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		log.Printf("[mall/payment] GET /me/payments failed uid=%q invoiceId=%q err=%v",
			payMaskUID(uid), payMaskUID(invoiceID), err)

		if errors.Is(err, mallquery.ErrNotFound) || payIsNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "detail": err.Error()})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "internal_error", "detail": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(list)
}

// ------------------------------------------------------------
// helpers (unique names to avoid collisions with other files)
// ------------------------------------------------------------

func payPickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}

func payPickInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		if n, ok2 := payParseIntAny(v); ok2 {
			return n
		}
	}
	return 0
}

func payParseIntAny(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case float32:
		return int(x), true
	case float64:
		return int(x), true
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		n := json.Number(s)
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func payToMap(v any) (map[string]any, bool) {
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

func payAlias(m map[string]any, from, to string) {
	if m == nil {
		return
	}
	// if already present and non-empty string, keep it
	if v, ok := m[to]; ok {
		if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
			return
		}
	}
	if v, ok := m[from]; ok {
		m[to] = v
	}
}

func payMaskUID(uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return ""
	}
	if len(uid) <= 6 {
		return "***"
	}
	return uid[:3] + "***" + uid[len(uid)-3:]
}

func payIsNotFoundLike(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found") || strings.Contains(msg, "404")
}

// ---- reflect setters for CreatePaymentInput (compile-safe) ----

func paySetStringFieldBestEffort(ptr any, val string, fieldNames ...string) {
	val = strings.TrimSpace(val)
	if ptr == nil || val == "" {
		return
	}
	rv := reflect.ValueOf(ptr)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	ev := rv.Elem()
	if !ev.IsValid() || ev.Kind() != reflect.Struct {
		return
	}

	for _, name := range fieldNames {
		f := ev.FieldByName(name)
		if !f.IsValid() || !f.CanSet() {
			continue
		}
		if f.Kind() == reflect.String {
			f.SetString(val)
			return
		}
		// support alias types: type Xxx string
		if f.Kind() == reflect.String && f.Type().ConvertibleTo(reflect.TypeOf("")) {
			f.SetString(val)
			return
		}
	}
}

func paySetIntFieldBestEffort(ptr any, val int, fieldNames ...string) {
	if ptr == nil || val == 0 {
		return
	}
	rv := reflect.ValueOf(ptr)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	ev := rv.Elem()
	if !ev.IsValid() || ev.Kind() != reflect.Struct {
		return
	}

	for _, name := range fieldNames {
		f := ev.FieldByName(name)
		if !f.IsValid() || !f.CanSet() {
			continue
		}
		switch f.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			f.SetInt(int64(val))
			return
		}
	}
}

func paySetStatusFieldBestEffort(ptr any, status string, fieldNames ...string) {
	status = strings.TrimSpace(status)
	if ptr == nil || status == "" {
		return
	}
	rv := reflect.ValueOf(ptr)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	ev := rv.Elem()
	if !ev.IsValid() || ev.Kind() != reflect.Struct {
		return
	}

	for _, name := range fieldNames {
		f := ev.FieldByName(name)
		if !f.IsValid() || !f.CanSet() {
			continue
		}

		// string
		if f.Kind() == reflect.String {
			f.SetString(status)
			return
		}

		// alias type (e.g. type PaymentStatus string)
		if f.Kind() == reflect.String && f.Type().ConvertibleTo(reflect.TypeOf("")) {
			f.SetString(status)
			return
		}

		// named string type (common for enums)
		if f.Kind() == reflect.String {
			f.SetString(status)
			return
		}

		// fallback: if it's a defined type whose underlying kind is string
		if f.Kind() == reflect.String {
			f.SetString(status)
			return
		}
	}
}

// ------------------------------------------------------------
// reflect call to OrderQuery.ResolveByUID (kept, because orderQ is any)
// ------------------------------------------------------------

func payCallResolveByUID(orderQ any, ctx context.Context, uid string) (any, error) {
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
