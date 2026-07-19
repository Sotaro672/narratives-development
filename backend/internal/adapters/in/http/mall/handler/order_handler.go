// backend/internal/adapters/in/http/mall/handler/order_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	historydto "narratives/internal/application/query/mall/dto"
	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// OrderHandler handles:
//   - POST /mall/me/orders
//   - GET  /mall/me/orders
type OrderHandler struct {
	uc           *usecase.OrderUsecase
	historyQuery OrderHistoryQuery
}

type OrderHistoryQuery interface {
	EnrichOrderPage(
		ctx context.Context,
		in historydto.EnrichHistoryOrderPageInput,
	) (historydto.HistoryOrderPage, error)
}

func NewOrderHandler(
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

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_found",
		})
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

type orderItemRequest struct {
	Type string `json:"type"`

	// list item identifiers
	ListID  string `json:"listId"`
	ModelID string `json:"modelId"`

	// resale item identifier
	ResaleID string `json:"resaleId"`

	Qty int `json:"qty"`

	// Reserved for future order creation behavior.
	IsCanceled   bool `json:"isCanceled"`
	IsDispatched bool `json:"isDispatched"`
}

type createOrderRequest struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`

	ShippingSnapshot shippingSnapshotRequest `json:"shippingSnapshot"`
	PaymentMethodID  string                  `json:"paymentMethodId"`

	Items []orderItemRequest `json:"items"`
}

func (h *OrderHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_body",
		})
		return
	}

	var req createOrderRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_json",
		})
		return
	}

	authUID, ok := middleware.CurrentUserUID(r)
	if !ok || authUID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized",
		})
		return
	}

	bodyUID := req.UserID
	if bodyUID != "" && bodyUID != authUID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "userId_mismatch",
		})
		return
	}

	userID := authUID

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized: missing avatarId",
		})
		return
	}

	cartID := avatarID

	if req.PaymentMethodID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "paymentMethodId is required",
		})
		return
	}

	if len(req.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "items is required",
		})
		return
	}

	shipping := orderdom.ShippingSnapshot{
		ZipCode: req.ShippingSnapshot.ZipCode,
		State:   req.ShippingSnapshot.State,
		City:    req.ShippingSnapshot.City,
		Street:  req.ShippingSnapshot.Street,
		Street2: req.ShippingSnapshot.Street2,
		Country: req.ShippingSnapshot.Country,
	}

	if shipping.State == "" ||
		shipping.City == "" ||
		shipping.Street == "" ||
		shipping.Country == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "shippingSnapshot is invalid",
		})
		return
	}

	items := make(
		[]usecase.CreateOrderItemInput,
		0,
		len(req.Items),
	)

	for _, requestItem := range req.Items {
		item, ok := orderItemRequestToInput(requestItem)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid order item",
			})
			return
		}

		items = append(items, item)
	}

	in := usecase.CreateOrderInput{
		ID:       req.ID,
		UserID:   userID,
		AvatarID: avatarID,
		CartID:   cartID,

		ShippingSnapshot: shipping,
		PaymentMethodID:  req.PaymentMethodID,
		Items:            items,
	}

	out, err := h.uc.Create(ctx, in)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func orderItemRequestToInput(
	item orderItemRequest,
) (usecase.CreateOrderItemInput, bool) {
	itemType := orderdom.OrderItemType(item.Type)

	switch itemType {
	case orderdom.OrderItemTypeList:
		if item.ListID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 {
			return usecase.CreateOrderItemInput{}, false
		}

		return usecase.CreateOrderItemInput{
			Type:         orderdom.OrderItemTypeList,
			ListID:       item.ListID,
			ModelID:      item.ModelID,
			Qty:          item.Qty,
			IsCanceled:   item.IsCanceled,
			IsDispatched: item.IsDispatched,
		}, true

	case orderdom.OrderItemTypeResale:
		if item.ResaleID == "" {
			return usecase.CreateOrderItemInput{}, false
		}

		return usecase.CreateOrderItemInput{
			Type:         orderdom.OrderItemTypeResale,
			ResaleID:     item.ResaleID,
			Qty:          1,
			IsCanceled:   item.IsCanceled,
			IsDispatched: item.IsDispatched,
		}, true

	default:
		return usecase.CreateOrderItemInput{}, false
	}
}

func (h *OrderHandler) listMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized: missing avatarId",
		})
		return
	}

	page := parseOrderPage(r)
	sort := parseOrderSort(r)

	out, err := h.uc.ListByAvatarID(
		ctx,
		avatarID,
		sort,
		page,
	)
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
) (historydto.HistoryOrderPage, error) {
	if h == nil || h.historyQuery == nil {
		return historydto.HistoryOrderPage{},
			errors.New("order handler: history query not configured")
	}

	body, err := json.Marshal(out)
	if err != nil {
		return historydto.HistoryOrderPage{}, err
	}

	var in historydto.EnrichHistoryOrderPageInput
	if err := json.Unmarshal(body, &in); err != nil {
		return historydto.HistoryOrderPage{}, err
	}

	return h.historyQuery.EnrichOrderPage(ctx, in)
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

	column := q.Get("sort")
	if column == "" {
		column = "createdAt"
	}

	order := strings.ToLower(q.Get("order"))
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
	code := orderHTTPStatus(err)

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

func orderHTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusInternalServerError

	case errors.Is(err, context.Canceled):
		return 499

	case errors.Is(err, orderdom.ErrNotFound):
		return http.StatusNotFound

	case errors.Is(err, orderdom.ErrConflict):
		return http.StatusConflict

	case isInvalidOrderError(err):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

func isInvalidOrderError(err error) bool {
	return errors.Is(err, orderdom.ErrInvalidID) ||
		errors.Is(err, orderdom.ErrInvalidUserID) ||
		errors.Is(err, orderdom.ErrInvalidAvatarID) ||
		errors.Is(err, orderdom.ErrInvalidCartID) ||
		errors.Is(err, orderdom.ErrInvalidShippingSnapshot) ||
		errors.Is(err, orderdom.ErrInvalidPaymentMethod) ||
		errors.Is(err, orderdom.ErrInvalidItems) ||
		errors.Is(err, orderdom.ErrInvalidItemSnapshot) ||
		errors.Is(err, orderdom.ErrInvalidCreatedAt)
}
