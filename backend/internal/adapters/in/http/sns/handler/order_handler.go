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

// NewOrderHandler はHTTPハンドラを初期化します。
func NewOrderHandler(uc *usecase.OrderUsecase) http.Handler {
	return &OrderHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ルータ実装差分吸収:
	// - 直マウント: /sns/orders, /sns/orders/{id}
	// - prefix strip される構成: /orders, /orders/{id}
	path := r.URL.Path

	// normalize: accept both /sns/orders and /orders
	switch {
	case strings.HasPrefix(path, "/sns/orders"):
		path = strings.TrimPrefix(path, "/sns")
	case strings.HasPrefix(path, "/orders"):
		// ok
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	// now path is like /orders or /orders/{id}
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
	// 互換キーを増やしたい場合は handler ではなく frontend 側を揃える方が安全
}

type billingSnapshotRequest struct {
	Last4 string `json:"last4"`

	// JSON キー揺れ吸収（どれかが入っていればOK）
	CardHolderName string `json:"cardHolderName"`
	CardholderName string `json:"cardholderName"`
	HolderName     string `json:"holderName"`
}

type createOrderRequest struct {
	ID string `json:"id"`

	// ⚠️ セキュリティ上、注文の userId は原則「認証済み uid」から確定する。
	// 互換のため残すが、uid が取れる場合は無視/一致チェックする。
	UserID string `json:"userId"`

	CartID string `json:"cartId"`

	ShippingSnapshot shippingSnapshotRequest `json:"shippingSnapshot"`
	BillingSnapshot  billingSnapshotRequest  `json:"billingSnapshot"`

	ListID    string   `json:"listId"`
	Items     []string `json:"items"`
	InvoiceID string   `json:"invoiceId"`
	PaymentID string   `json:"paymentId"`

	// optional
	TransferedDate *string `json:"transferedDate"` // RFC3339

	UpdatedBy *string `json:"updatedBy"`
}

// POST /sns/orders
//
// ✅ 住所/カードは order テーブルへスナップショット保存する前提。
// - shippingSnapshot: 住所スナップショット（そのまま保存）
// - billingSnapshot: last4 + cardHolderName のみ保存
//
// ✅ userId は「認証済み uid」から確定（可能なら req.userId は無視/一致チェック）。
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

	// uid が取れるなら uid を正として確定。body の userId がある場合は一致チェック。
	// uid が取れない（開発/未配線）場合のみ body.userId を使用。
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
		// 互換: 認証uidが取れない構成では body.userId を許容（本番では middleware で uid を入れること）
		userID = bodyUID
	default:
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// required (domain validate で落ちる前に、最低限 handler で落とす)
	cartID := trim(req.CartID)
	listID := trim(req.ListID)
	invoiceID := trim(req.InvoiceID)
	paymentID := trim(req.PaymentID)

	if cartID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "cartId is required"})
		return
	}
	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "listId is required"})
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

	// shipping は entity 側 validate に任せるが、最低限の空チェック
	if ship.State == "" && ship.City == "" && ship.Street == "" && ship.Country == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "shippingSnapshot is required"})
		return
	}

	holder := trim(req.BillingSnapshot.CardHolderName)
	if holder == "" {
		holder = trim(req.BillingSnapshot.CardholderName)
	}
	if holder == "" {
		holder = trim(req.BillingSnapshot.HolderName)
	}

	bill := orderdom.BillingSnapshot{
		Last4:          trim(req.BillingSnapshot.Last4),
		CardHolderName: holder,
	}

	// ✅ 仕様に合わせて last4 / cardHolderName を両方必須
	if bill.Last4 == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "billingSnapshot.last4 is required"})
		return
	}
	if strings.TrimSpace(bill.CardHolderName) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "billingSnapshot.cardHolderName is required"})
		return
	}

	// transferedDate (optional)
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

		ListID:    listID,
		Items:     req.Items,
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

// GET /sns/orders/{id}
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
// Auth helpers (best-effort)
// ------------------------------------------------------------

// getAuthUID tries to read uid injected by middleware into request context.
// Since this package should not depend on middleware internals, it checks common key patterns.
func getAuthUID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Most common patterns in Go codebases:
	// - ctx.Value("uid") / ctx.Value("userId")
	// - typed keys (e.g., type ctxKey string) with the same strings
	keys := []any{
		"uid",
		"userId",
		"firebase_uid",
		"firebaseUid",
	}

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

	// Try a little harder: some middlewares use typed string keys.
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
// エラーハンドリング
// ------------------------------------------------------------

func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	// domain/usecase の sentinel がある前提で可能な限りマップ
	switch {
	case errors.Is(err, orderdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, orderdom.ErrConflict):
		code = http.StatusConflict

	// 代表的な「入力が悪い」系（増えてもここに足す）
	case errors.Is(err, orderdom.ErrInvalidID),
		errors.Is(err, orderdom.ErrInvalidUserID),
		errors.Is(err, orderdom.ErrInvalidCartID),
		errors.Is(err, orderdom.ErrInvalidListID),
		errors.Is(err, orderdom.ErrInvalidItems),
		errors.Is(err, orderdom.ErrInvalidInvoiceID),
		errors.Is(err, orderdom.ErrInvalidPaymentID),
		errors.Is(err, orderdom.ErrInvalidTransferredDate),
		errors.Is(err, orderdom.ErrInvalidCreatedAt),
		errors.Is(err, orderdom.ErrInvalidUpdatedAt),
		errors.Is(err, orderdom.ErrInvalidUpdatedBy),
		errors.Is(err, orderdom.ErrInvalidItemID):
		code = http.StatusBadRequest
	default:
		// fallback: メッセージから not found を拾う（念のため）
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if msg == "not_found" || strings.Contains(msg, "not found") || strings.Contains(msg, "not_found") {
			code = http.StatusNotFound
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
