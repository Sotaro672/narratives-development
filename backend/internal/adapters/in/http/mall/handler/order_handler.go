// backend/internal/adapters/in/http/mall/handler/order_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	mallQuery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// OrderHandler handles:
//   - POST /mall/me/orders
//   - GET  /mall/me/orders
//   - GET  /mall/me/orders/{id}
type OrderHandler struct {
	uc           *usecase.OrderUsecase
	historyQuery OrderHistoryQuery
}

type OrderHistoryQuery interface {
	EnrichOrderPage(
		ctx context.Context,
		in mallQuery.EnrichHistoryOrderPageInput,
	) (mallQuery.HistoryOrderPage, error)
}

func NewOrderHandler(uc *usecase.OrderUsecase) http.Handler {
	return &OrderHandler{uc: uc}
}

func NewOrderHandlerWithHistoryQuery(
	uc *usecase.OrderUsecase,
	historyQuery OrderHistoryQuery,
) http.Handler {
	return &OrderHandler{
		uc:           uc,
		historyQuery: historyQuery,
	}
}

func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodPost && path == "/mall/me/orders":
		h.post(w, r)
		return

	case r.Method == http.MethodGet && path == "/mall/me/orders":
		h.listMe(w, r)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/mall/me/orders/"):
		id := strings.TrimPrefix(path, "/mall/me/orders/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

type shippingSnapshotRequest struct {
	ZipCode string `json:"zipCode"`
	State   string `json:"state"`
	City    string `json:"city"`
	Street  string `json:"street"`
	Street2 string `json:"street2"`
	Country string `json:"country"`
}

type paymentMethodSnapshotRequest struct {
	CustomerID     string `json:"customerId"`
	Brand          string `json:"brand"`
	Last4          string `json:"last4"`
	ExpMonth       int    `json:"expMonth"`
	ExpYear        int    `json:"expYear"`
	CardholderName string `json:"cardholderName"`
	IsDefault      bool   `json:"isDefault"`
}

type orderItemSnapshotRequest struct {
	ModelID     string `json:"modelId"`
	InventoryID string `json:"inventoryId"`
	ListID      string `json:"listId"`
	Qty         int    `json:"qty"`
	Price       int    `json:"price"`

	IsCanceled   bool `json:"isCanceled"`
	IsDispatched bool `json:"isDispatched"`
}

type createOrderRequest struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	ShippingSnapshot      shippingSnapshotRequest      `json:"shippingSnapshot"`
	PaymentMethodSnapshot paymentMethodSnapshotRequest `json:"paymentMethodSnapshot"`

	Items []orderItemSnapshotRequest `json:"items"`
}

func (h *OrderHandler) post(w http.ResponseWriter, r *http.Request) {
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

	trim := strings.TrimSpace

	authUID, ok := middleware.CurrentUserUID(r)
	if !ok || authUID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	bodyUID := trim(req.UserID)
	if bodyUID != "" && bodyUID != authUID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "userId_mismatch"})
		return
	}

	userID := authUID

	log.Printf(
		`[order.post] start path=%q authUID=%q bodyUID=%q avatarId(body)=%q cartId(body)=%q items=%d`,
		r.URL.Path,
		authUID,
		bodyUID,
		trim(req.AvatarID),
		trim(req.CartID),
		len(req.Items),
	)

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

	paymentMethod := orderdom.PaymentMethodSnapshot{
		CustomerID:     trim(req.PaymentMethodSnapshot.CustomerID),
		Brand:          trim(req.PaymentMethodSnapshot.Brand),
		Last4:          trim(req.PaymentMethodSnapshot.Last4),
		ExpMonth:       req.PaymentMethodSnapshot.ExpMonth,
		ExpYear:        req.PaymentMethodSnapshot.ExpYear,
		CardholderName: trim(req.PaymentMethodSnapshot.CardholderName),
		IsDefault:      req.PaymentMethodSnapshot.IsDefault,
	}
	if paymentMethod.CustomerID == "" ||
		paymentMethod.Brand == "" ||
		paymentMethod.Last4 == "" ||
		paymentMethod.ExpMonth < 1 ||
		paymentMethod.ExpMonth > 12 ||
		paymentMethod.ExpYear < 2000 ||
		paymentMethod.ExpYear > 9999 ||
		paymentMethod.CardholderName == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "paymentMethodSnapshot is invalid"})
		return
	}

	items := make([]orderdom.OrderItemSnapshot, 0, len(req.Items))
	for idx, it := range req.Items {
		mid := trim(it.ModelID)
		iid := trim(it.InventoryID)
		lid := trim(it.ListID)
		qty := it.Qty
		price := it.Price

		log.Printf(
			`[order.post] item[%d] mid=%q iid=%q lid=%q qty=%d price=%d isCanceled=%v isDispatched=%v`,
			idx,
			mid,
			iid,
			lid,
			qty,
			price,
			it.IsCanceled,
			it.IsDispatched,
		)

		if mid == "" || iid == "" || lid == "" || qty <= 0 || price < 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid item snapshot"})
			return
		}

		items = append(items, orderdom.OrderItemSnapshot{
			ModelID:      mid,
			InventoryID:  iid,
			ListID:       lid,
			Qty:          qty,
			Price:        price,
			IsCanceled:   it.IsCanceled,
			IsDispatched: it.IsDispatched,
		})
	}

	in := usecase.CreateOrderInput{
		ID:       trim(req.ID),
		UserID:   userID,
		AvatarID: avatarID,
		CartID:   cartID,

		ShippingSnapshot:      ship,
		PaymentMethodSnapshot: paymentMethod,

		Items: items,
	}

	log.Printf(
		`[order.post] calling uc.Create userId=%q avatarId=%q cartId=%q items=%d`,
		userID,
		avatarID,
		cartID,
		len(items),
	)

	out, err := h.uc.Create(ctx, in)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *OrderHandler) listMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing avatarId"})
		return
	}

	page := parseOrderPage(r)
	sort := parseOrderSort(r)

	filter := usecase.OrderFilter{
		AvatarID: &avatarID,
	}

	out, err := h.uc.List(ctx, filter, sort, page)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	if h.historyQuery == nil {
		_ = json.NewEncoder(w).Encode(out)
		return
	}

	enriched, err := h.enrichOrderHistoryPage(ctx, out)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(enriched)
}

func (h *OrderHandler) enrichOrderHistoryPage(
	ctx context.Context,
	out any,
) (mallQuery.HistoryOrderPage, error) {
	if h == nil || h.historyQuery == nil {
		return mallQuery.HistoryOrderPage{}, errors.New("order handler: history query not configured")
	}

	body, err := json.Marshal(out)
	if err != nil {
		return mallQuery.HistoryOrderPage{}, err
	}

	var in mallQuery.EnrichHistoryOrderPageInput
	if err := json.Unmarshal(body, &in); err != nil {
		return mallQuery.HistoryOrderPage{}, err
	}

	return h.historyQuery.EnrichOrderPage(ctx, in)
}

func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

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

func parseOrderPage(r *http.Request) common.Page {
	q := r.URL.Query()

	page := parsePositiveInt(q.Get("page"), 1)
	perPage := parsePositiveInt(q.Get("perPage"), 20)

	if perPage > 100 {
		perPage = 100
	}

	return common.Page{
		Number:  page,
		PerPage: perPage,
	}
}

func parseOrderSort(r *http.Request) common.Sort {
	q := r.URL.Query()

	column := strings.TrimSpace(q.Get("sort"))
	if column == "" {
		column = "createdAt"
	}

	order := strings.ToLower(strings.TrimSpace(q.Get("order")))
	if order == "" {
		order = string(common.SortDesc)
	}

	sortOrder := common.SortDesc
	if order == string(common.SortAsc) {
		sortOrder = common.SortAsc
	}

	return common.Sort{
		Column: column,
		Order:  sortOrder,
	}
}

func parsePositiveInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}

func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	msg := strings.ToLower(err.Error())

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
