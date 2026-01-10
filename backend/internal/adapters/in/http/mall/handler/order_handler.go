// backend/internal/adapters/in/http/mall/handler/order_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
)

// OrderHandler handles /mall/me/orders (GET/POST) and /mall/me/orders/{id} (GET).
// ✅ legacy /mall/orders is removed (returns 404).
type OrderHandler struct {
	uc *usecase.OrderUsecase
}

func NewOrderHandler(uc *usecase.OrderUsecase) http.Handler {
	return &OrderHandler{uc: uc}
}

func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	// ✅ /mall/me/orders POST (auth required)
	case r.Method == http.MethodPost && path == "/mall/me/orders":
		h.post(w, r, true /* requireAuth */)
		return

	// ✅ GET: /mall/me/orders/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/mall/me/orders/"):
		id := strings.TrimPrefix(path, "/mall/me/orders/")
		h.get(w, r, id)
		return

	// ✅ Explicitly deny legacy /mall/orders (404)
	case strings.HasPrefix(path, "/mall/orders"):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
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

	// 原則 auth uid を正。uid が取れない構成のみ body.userId を許容（/mall/me/orders は許容しない）
	UserID string `json:"userId"`

	// Order entity requires avatarId
	AvatarID string `json:"avatarId"`

	CartID string `json:"cartId"`

	ShippingSnapshot shippingSnapshotRequest `json:"shippingSnapshot"`
	BillingSnapshot  billingSnapshotRequest  `json:"billingSnapshot"`

	Items []orderItemSnapshotRequest `json:"items"`
}

func (h *OrderHandler) post(w http.ResponseWriter, r *http.Request, requireAuth bool) {
	ctx := r.Context()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_body"})
		return
	}

	var req createOrderRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	trim := func(s string) string { return strings.TrimSpace(s) }

	// ✅ Auth UID: middleware helper を正として使う（Context key mismatch を根絶）
	authUID, ok := middleware.CurrentUserUID(r)
	if !ok {
		authUID = ""
	}
	bodyUID := trim(req.UserID)

	// 一時ログ（原因切り分け用）
	log.Printf(`[order.post] requireAuth=%v authUID=%q bodyUID=%q`, requireAuth, authUID, bodyUID)

	// /mall/me/orders は auth 必須（body.userId 代替は禁止）
	if requireAuth && authUID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// --- Decide userID ---
	var userID string
	switch {
	case authUID != "":
		userID = authUID
		// auth が取れているのに body.userId が別UIDなら拒否（改ざん対策）
		if bodyUID != "" && bodyUID != authUID {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "userId_mismatch"})
			return
		}

	// ✅ legacy path removed: never accept bodyUID fallback
	default:
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// --- AvatarID required (body first, fallback to query param) ---
	avatarID := trim(req.AvatarID)
	if avatarID == "" {
		avatarID = trim(r.URL.Query().Get("avatarId"))
	}
	if avatarID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatarId is required"})
		return
	}

	cartID := trim(req.CartID)
	if cartID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "cartId is required"})
		return
	}

	if len(req.Items) == 0 {
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
	if ship.State == "" || ship.City == "" || ship.Street == "" || ship.Country == "" {
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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "billingSnapshot.last4 is required"})
		return
	}

	// ---- items validation ----
	items := make([]orderdom.OrderItemSnapshot, 0, len(req.Items))
	for _, it := range req.Items {
		mid := trim(it.ModelID)
		iid := trim(it.InventoryID)
		qty := it.Qty
		price := it.Price

		if mid == "" || iid == "" || qty <= 0 || price < 0 {
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
