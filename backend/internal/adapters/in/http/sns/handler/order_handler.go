// backend/internal/adapters/in/http/sns/handler/order_handler.go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
)

// OrderHandler は /sns/orders 関連のエンドポイントを担当します（GET/POST）。
type OrderHandler struct {
	uc *usecase.OrderUsecase
}

func NewOrderHandler(uc *usecase.OrderUsecase) http.Handler {
	return &OrderHandler{uc: uc}
}

func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/sns/orders"):
		path = strings.TrimPrefix(path, "/sns")
	case strings.HasPrefix(path, "/orders"):
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	switch {
	case r.Method == http.MethodPost && (path == "/orders" || path == "/orders/"):
		h.post(w, r)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/orders/"):
		id := strings.TrimPrefix(path, "/orders/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// --- POST request shape (snapshot-based) ---

type shippingSnapshotRequest struct {
	ZipCode string `json:"zipCode"`
	State   string `json:"state"`
	City    string `json:"city"`
	Street  string `json:"street"`
	Street2 string `json:"street2"`
	Country string `json:"country"`
}

type billingSnapshotRequest struct {
	Last4          string `json:"last4"`
	CardHolderName string `json:"cardHolderName"`
}

type orderItemSnapshotRequest struct {
	ModelID     string `json:"modelId"`
	InventoryID string `json:"inventoryId"`
	Qty         int    `json:"qty"`
	Price       int    `json:"price"`
}

type createOrderRequest struct {
	ID string `json:"id"`

	// 原則 auth uid を正。uid が取れない構成のみ body.userId を許容。
	UserID string `json:"userId"`

	CartID string `json:"cartId"`

	ShippingSnapshot shippingSnapshotRequest `json:"shippingSnapshot"`
	BillingSnapshot  billingSnapshotRequest  `json:"billingSnapshot"`

	Items     []orderItemSnapshotRequest `json:"items"`
	InvoiceID string                     `json:"invoiceId"`
	PaymentID string                     `json:"paymentId"`

	TransferedDate *string `json:"transferedDate"` // RFC3339
	UpdatedBy      *string `json:"updatedBy"`
}

func (h *OrderHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	trim := func(s string) string { return strings.TrimSpace(s) }

	// --- Auth UID (best-effort) ---
	authUID := trim(getAuthUID(ctx))
	bodyUID := trim(req.UserID)

	var userID string
	switch {
	case authUID != "":
		userID = authUID
		if bodyUID != "" && bodyUID != authUID {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "userId_mismatch"})
			return
		}
	case bodyUID != "":
		userID = bodyUID
	default:
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	cartID := trim(req.CartID)
	invoiceID := trim(req.InvoiceID)
	paymentID := trim(req.PaymentID)

	if cartID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "cartId is required"})
		return
	}
	if invoiceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invoiceId is required"})
		return
	}
	if paymentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "paymentId is required"})
		return
	}
	if len(req.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "items is required"})
		return
	}

	ship := orderdom.ShippingSnapshot{
		ZipCode: trim(req.ShippingSnapshot.ZipCode),
		State:   trim(req.ShippingSnapshot.State),
		City:    trim(req.ShippingSnapshot.City),
		Street:  trim(req.ShippingSnapshot.Street),
		Street2: trim(req.ShippingSnapshot.Street2),
		Country: trim(req.ShippingSnapshot.Country),
	}
	if ship.State == "" && ship.City == "" && ship.Street == "" && ship.Country == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "shippingSnapshot is required"})
		return
	}

	bill := orderdom.BillingSnapshot{
		Last4:          trim(req.BillingSnapshot.Last4),
		CardHolderName: trim(req.BillingSnapshot.CardHolderName),
	}
	if bill.Last4 == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "billingSnapshot.last4 is required"})
		return
	}

	items := make([]orderdom.OrderItemSnapshot, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, orderdom.OrderItemSnapshot{
			ModelID:     trim(it.ModelID),
			InventoryID: trim(it.InventoryID),
			Qty:         it.Qty,
			Price:       it.Price,
		})
	}

	var td *time.Time
	if req.TransferedDate != nil && trim(*req.TransferedDate) != "" {
		t, err := time.Parse(time.RFC3339, trim(*req.TransferedDate))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid transferedDate (must be RFC3339)"})
			return
		}
		utc := t.UTC()
		td = &utc
	}

	in := usecase.CreateOrderInput{
		ID:     trim(req.ID),
		UserID: userID,
		CartID: cartID,

		ShippingSnapshot: ship,
		BillingSnapshot:  bill,

		Items:     items,
		InvoiceID: invoiceID,
		PaymentID: paymentID,

		TransferedDate: td,
		UpdatedBy:      req.UpdatedBy,
	}

	out, err := h.uc.Create(ctx, in)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	out, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ------------------------------------------------------------
// Auth helpers
// ------------------------------------------------------------

func getAuthUID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	keys := []any{"uid", "userId", "firebase_uid", "firebaseUid"}
	for _, k := range keys {
		if v := ctx.Value(k); v != nil {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
			if sp, ok := v.(*string); ok && sp != nil {
				return strings.TrimSpace(*sp)
			}
		}
	}

	type ctxKey string
	for _, ks := range []string{"uid", "userId", "firebase_uid", "firebaseUid"} {
		if v := ctx.Value(ctxKey(ks)); v != nil {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
			if sp, ok := v.(*string); ok && sp != nil {
				return strings.TrimSpace(*sp)
			}
		}
	}

	return ""
}

// ------------------------------------------------------------
// Error mapping
// ------------------------------------------------------------

func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, orderdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, orderdom.ErrConflict):
		code = http.StatusConflict

	case errors.Is(err, orderdom.ErrInvalidID),
		errors.Is(err, orderdom.ErrInvalidUserID),
		errors.Is(err, orderdom.ErrInvalidCartID),
		errors.Is(err, orderdom.ErrInvalidShippingAddress),
		errors.Is(err, orderdom.ErrInvalidBillingAddress),
		errors.Is(err, orderdom.ErrInvalidItems),
		errors.Is(err, orderdom.ErrInvalidInvoiceID),
		errors.Is(err, orderdom.ErrInvalidPaymentID),
		errors.Is(err, orderdom.ErrInvalidTransferredDate),
		errors.Is(err, orderdom.ErrInvalidCreatedAt),
		errors.Is(err, orderdom.ErrInvalidUpdatedAt),
		errors.Is(err, orderdom.ErrInvalidUpdatedBy),
		errors.Is(err, orderdom.ErrInvalidItemSnapshot):
		code = http.StatusBadRequest
	default:
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if msg == "not_found" || strings.Contains(msg, "not found") || strings.Contains(msg, "not_found") {
			code = http.StatusNotFound
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
