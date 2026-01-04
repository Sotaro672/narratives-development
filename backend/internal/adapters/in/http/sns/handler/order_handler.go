// backend/internal/adapters/in/http/sns/handler/order_handler.go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

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

	// ✅ NEW: Order entity に avatarId が必須になったため受け取る
	// SNS 側は原則「遷移URL/画面が avatarId を持つ」前提。
	AvatarID string `json:"avatarId"`

	CartID string `json:"cartId"`

	ShippingSnapshot shippingSnapshotRequest `json:"shippingSnapshot"`
	BillingSnapshot  billingSnapshotRequest  `json:"billingSnapshot"`

	Items []orderItemSnapshotRequest `json:"items"`

	// ✅ order を先に起票するため、invoice/payment は order 作成では受け取らない
	// InvoiceID string `json:"invoiceId"`
	// PaymentID string `json:"paymentId"`
}

func (h *OrderHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := _pickReqID(r)

	// ---- read body (so we can log on decode failure) ----
	const maxBodyLog = 4096
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[sns_order] reqId=%s read_body_failed err=%v", reqID, err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_body"})
		return
	}

	var req createOrderRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf(
			"[sns_order] reqId=%s invalid_json err=%v bodyLen=%d bodySample=%q",
			reqID, err, len(bodyBytes), _sampleBytes(bodyBytes, maxBodyLog),
		)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	trim := func(s string) string { return strings.TrimSpace(s) }

	// --- Auth UID (best-effort) ---
	authUID := trim(getAuthUID(ctx))
	bodyUID := trim(req.UserID)

	// --- AvatarID (body first, fallback to query param) ---
	avatarID := trim(req.AvatarID)
	if avatarID == "" {
		avatarID = trim(r.URL.Query().Get("avatarId"))
	}

	// 入口の要約ログ（失敗時の比較用）
	log.Printf(
		"[sns_order] reqId=%s in authUid=%q bodyUserId=%q avatarId=%q cartId=%q items=%d ship(state=%t city=%t street=%t country=%t) bill(last4=%t)",
		reqID,
		_maskID(authUID),
		_maskID(bodyUID),
		_maskID(avatarID),
		_maskID(trim(req.CartID)),
		len(req.Items),
		trim(req.ShippingSnapshot.State) != "",
		trim(req.ShippingSnapshot.City) != "",
		trim(req.ShippingSnapshot.Street) != "",
		trim(req.ShippingSnapshot.Country) != "",
		trim(req.BillingSnapshot.Last4) != "",
	)

	// --- Decide userID ---
	var userID string
	switch {
	case authUID != "":
		userID = authUID
		if bodyUID != "" && bodyUID != authUID {
			log.Printf("[sns_order] reqId=%s bad_request reason=userId_mismatch authUid=%q bodyUserId=%q",
				reqID, _maskID(authUID), _maskID(bodyUID),
			)
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "userId_mismatch"})
			return
		}
	case bodyUID != "":
		userID = bodyUID
	default:
		log.Printf("[sns_order] reqId=%s unauthorized reason=missing_uid", reqID)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// --- AvatarID required (entity.go の必須前提) ---
	if avatarID == "" {
		log.Printf("[sns_order] reqId=%s bad_request reason=missing_avatarId userId=%q", reqID, _maskID(userID))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatarId is required"})
		return
	}

	cartID := trim(req.CartID)
	if cartID == "" {
		log.Printf("[sns_order] reqId=%s bad_request reason=missing_cartId avatarId=%q userId=%q", reqID, _maskID(avatarID), _maskID(userID))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "cartId is required"})
		return
	}

	if len(req.Items) == 0 {
		log.Printf("[sns_order] reqId=%s bad_request reason=missing_items avatarId=%q userId=%q", reqID, _maskID(avatarID), _maskID(userID))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "items is required"})
		return
	}

	// ---- build shipping ----
	ship := orderdom.ShippingSnapshot{
		ZipCode: trim(req.ShippingSnapshot.ZipCode),
		State:   trim(req.ShippingSnapshot.State),
		City:    trim(req.ShippingSnapshot.City),
		Street:  trim(req.ShippingSnapshot.Street),
		Street2: trim(req.ShippingSnapshot.Street2),
		Country: trim(req.ShippingSnapshot.Country),
	}

	// entity.go の validateShippingSnapshot: State/City/Street/Country が必須
	missShipState := ship.State == ""
	missShipCity := ship.City == ""
	missShipStreet := ship.Street == ""
	missShipCountry := ship.Country == ""
	if missShipState || missShipCity || missShipStreet || missShipCountry {
		log.Printf("[sns_order] reqId=%s bad_request reason=invalid_shippingSnapshot missing(state=%t city=%t street=%t country=%t) zip=%t avatarId=%q userId=%q",
			reqID,
			missShipState,
			missShipCity,
			missShipStreet,
			missShipCountry,
			ship.ZipCode != "",
			_maskID(avatarID),
			_maskID(userID),
		)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "shippingSnapshot is invalid"})
		return
	}

	// ---- build billing ----
	bill := orderdom.BillingSnapshot{
		Last4:          trim(req.BillingSnapshot.Last4),
		CardHolderName: trim(req.BillingSnapshot.CardHolderName),
	}
	if bill.Last4 == "" {
		log.Printf("[sns_order] reqId=%s bad_request reason=missing_billing_last4 avatarId=%q userId=%q", reqID, _maskID(avatarID), _maskID(userID))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "billingSnapshot.last4 is required"})
		return
	}

	// ---- items validation (ここが落ちやすいので詳細ログ) ----
	// entity.go の validateItems: price < 0 はNG（0 はOK）、qty <= 0 はNG
	items := make([]orderdom.OrderItemSnapshot, 0, len(req.Items))
	for i, it := range req.Items {
		mid := trim(it.ModelID)
		iid := trim(it.InventoryID)
		qty := it.Qty
		price := it.Price

		if mid == "" || iid == "" || qty <= 0 || price < 0 {
			log.Printf(
				"[sns_order] reqId=%s bad_request reason=invalid_item idx=%d modelId=%q inventoryId=%q qty=%d price=%d avatarId=%q userId=%q",
				reqID,
				i,
				_maskID(mid),
				_maskID(iid),
				qty,
				price,
				_maskID(avatarID),
				_maskID(userID),
			)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid item snapshot"})
			return
		}

		items = append(items, orderdom.OrderItemSnapshot{
			ModelID:     mid,
			InventoryID: iid,
			Qty:         qty,
			Price:       price,
		})
	}

	in := usecase.CreateOrderInput{
		ID:       trim(req.ID),
		UserID:   userID,
		AvatarID: avatarID,
		CartID:   cartID,

		ShippingSnapshot: ship,
		BillingSnapshot:  bill,

		Items: items,
	}

	out, err := h.uc.Create(ctx, in)
	if err != nil {
		log.Printf("[sns_order] reqId=%s create_failed err=%v userId=%q avatarId=%q cartId=%q items=%d",
			reqID, err, _maskID(userID), _maskID(avatarID), _maskID(cartID), len(items),
		)
		writeOrderErr(w, err)
		return
	}

	log.Printf("[sns_order] reqId=%s created ok userId=%q avatarId=%q cartId=%q items=%d", reqID, _maskID(userID), _maskID(avatarID), _maskID(cartID), len(items))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	reqID := _pickReqID(r)

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[sns_order] reqId=%s bad_request reason=invalid_id", reqID)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	out, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[sns_order] reqId=%s get_failed id=%q err=%v", reqID, _maskID(id), err)
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

	msg := strings.ToLower(strings.TrimSpace(err.Error()))

	switch {
	case errors.Is(err, context.Canceled):
		code = 499
	case msg == "not_found" || strings.Contains(msg, "not found") || strings.Contains(msg, "not_found"):
		code = http.StatusNotFound
	case strings.Contains(msg, "conflict") || strings.Contains(msg, "already exists"):
		code = http.StatusConflict
	case strings.Contains(msg, "invalid") || strings.Contains(msg, "required") || strings.Contains(msg, "missing"):
		code = http.StatusBadRequest
	default:
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// ------------------------------------------------------------
// logging helpers
// ------------------------------------------------------------

func _pickReqID(r *http.Request) string {
	if r == nil {
		return "-"
	}
	// Cloud Run / LB / 自前 middleware など、どれかに乗っている想定で複数見る
	for _, k := range []string{"X-Request-Id", "X-Cloud-Trace-Context", "X-Amzn-Trace-Id"} {
		if v := strings.TrimSpace(r.Header.Get(k)); v != "" {
			return v
		}
	}
	return "-"
}

func _sampleBytes(b []byte, limit int) string {
	if limit <= 0 || len(b) == 0 {
		return ""
	}
	if len(b) <= limit {
		return string(b)
	}
	return string(b[:limit]) + "...(truncated)"
}

func _maskID(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	// 先頭4 + 末尾4 だけ残す（長さが短ければ全体）
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
