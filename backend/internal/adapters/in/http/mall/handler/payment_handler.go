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
	orderQ OrderQuery
}

// OrderQuery is the typed contract PaymentHandler needs.
// ✅ ResolveByUID returns mallquery.OrderContextDTO (typed), so we can avoid best-effort.
type OrderQuery interface {
	ResolveByUID(ctx context.Context, uid string) (mallquery.OrderContextDTO, error)
}

func NewPaymentHandler(uc *usecase.PaymentUsecase) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: nil}
}

func NewPaymentHandlerWithOrderQuery(uc *usecase.PaymentUsecase, orderQ OrderQuery) http.Handler {
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

	// GET /mall/me/payments?invoiceId=... : list payments by invoiceId
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
	if !ok || uid == "" { // ✅ no TrimSpace for id
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	out, err := h.orderQ.ResolveByUID(r.Context(), uid) // ✅ typed call (no reflect)
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
	if !ok || uid == "" { // ✅ no TrimSpace for id
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

	// ✅ canonical keys only (Firestore実データを正)
	req := payCreateReq{
		InvoiceID:        payPickString(body, "invoiceId"),
		BillingAddressID: payPickString(body, "billingAddressId"),
		Amount:           payPickInt(body, "amount"),
		Status:           payPickString(body, "status"),
	}

	// ✅ IDs: no TrimSpace (per requirement)
	// req.InvoiceID = strings.TrimSpace(req.InvoiceID)
	// req.BillingAddressID = strings.TrimSpace(req.BillingAddressID)

	// status is not an id; trimming is OK
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
	paySetIntFieldBestEffort(&in, req.Amount, "Amount")

	// ✅ cartId (= avatarId) inject (typed orderQ; no reflect)
	{
		aid := ""
		if h.orderQ != nil {
			ctxDTO, qerr := h.orderQ.ResolveByUID(r.Context(), uid)
			if qerr == nil {
				// ✅ no TrimSpace for id
				aid = ctxDTO.AvatarID
			}
		}
		if aid != "" {
			paySetStringFieldBestEffort(&in, aid, "CartID", "CartId", "cartId")
			paySetStringFieldBestEffort(&in, aid, "AvatarID", "AvatarId", "avatarId")
		}
	}

	// dev default: if caller did not provide status, set "paid" best-effort
	if req.Status == "" {
		req.Status = "paid"
	}
	paySetStatusFieldBestEffort(&in, req.Status, "Status", "PaymentStatus")

	p, err := h.uc.Create(r.Context(), in)
	if err != nil {
		log.Printf("[mall/payment] POST /me/payments failed uid=%q invoiceId=%q billingAddressId=%q amount=%d err=%v",
			payMaskUID(uid), payMaskID(req.InvoiceID), payMaskID(req.BillingAddressID), req.Amount, err)

		if errors.Is(err, mallquery.ErrNotFound) || payIsNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "not_found", "detail": err.Error()})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "internal_error", "detail": err.Error()})
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
	if !ok || uid == "" { // ✅ no TrimSpace for id
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// ✅ invoiceId is id: no TrimSpace
	invoiceID := r.URL.Query().Get("invoiceId")
	if invoiceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invoiceId_required"})
		return
	}

	list, err := h.uc.GetByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		log.Printf("[mall/payment] GET /me/payments failed uid=%q invoiceId=%q err=%v",
			payMaskUID(uid), payMaskID(invoiceID), err)

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
// helpers
// ------------------------------------------------------------

func payPickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		// ✅ no TrimSpace (ids must not be trimmed)
		s := fmt.Sprint(v)
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
		// amount is not an id; trim is OK here to parse numeric
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

func payMaskUID(uid string) string {
	// ✅ no TrimSpace for id
	if uid == "" {
		return ""
	}
	if len(uid) <= 6 {
		return "***"
	}
	return uid[:3] + "***" + uid[len(uid)-3:]
}

func payMaskID(id string) string {
	// ✅ no TrimSpace for id
	if id == "" {
		return ""
	}
	if len(id) <= 6 {
		return "***"
	}
	return id[:3] + "***" + id[len(id)-3:]
}

func payIsNotFoundLike(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found") || strings.Contains(msg, "404")
}

// ---- reflect setters for CreatePaymentInput (kept) ----
// NOTE: We removed TrimSpace for id-ish values here as well.

func paySetStringFieldBestEffort(ptr any, val string, fieldNames ...string) {
	// ✅ no TrimSpace for ids
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
	// status is not id; trimming OK
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

		if f.Kind() == reflect.String {
			f.SetString(status)
			return
		}

		if f.Kind() == reflect.String && f.Type().ConvertibleTo(reflect.TypeOf("")) {
			f.SetString(status)
			return
		}
	}
}
