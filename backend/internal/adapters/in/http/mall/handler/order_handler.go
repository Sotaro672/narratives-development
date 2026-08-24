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
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	orderdom "narratives/internal/domain/order"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
	transportationdom "narratives/internal/domain/transportation"
)

// OrderHandler handles:
//   - POST  /mall/me/orders
//   - GET   /mall/me/orders
//   - GET   /mall/me/orders/{orderId}
//   - PATCH /mall/me/orders/{orderId}/items/{itemIndex}/cancel
//   - PATCH /mall/me/orders/{orderId}/items/{itemIndex}/return
//
// The Mall return endpoint records a purchaser return request only.
// It must not execute Stripe Refund, Stripe Transfer Reversal, Payment refund
// state mutation, or Settlement cancellation/reversal.
//
// Financial refund execution belongs to the Console-side approval flow.
type OrderHandler struct {
	uc               *usecase.OrderUsecase
	historyQuery     OrderHistoryQuery
	orderDetailQuery OrderDetailQuery
}

type OrderHistoryQuery interface {
	EnrichOrderPage(ctx context.Context, in historydto.EnrichHistoryOrderPageInput) (historydto.HistoryOrderPage, error)
}

type OrderDetailQuery interface {
	EnrichOrderDetail(
		ctx context.Context,
		in orderdom.Order,
	) (historydto.OrderDetail, error)
}

func NewOrderHandler(
	uc *usecase.OrderUsecase,
	historyQuery OrderHistoryQuery,
	orderDetailQuery OrderDetailQuery,
) http.Handler {
	return &OrderHandler{
		uc:               uc,
		historyQuery:     historyQuery,
		orderDetailQuery: orderDetailQuery,
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

	case r.Method == http.MethodPatch &&
		strings.HasPrefix(path, "/mall/me/orders/") &&
		strings.HasSuffix(path, "/cancel"):
		h.cancelMe(w, r, path)
		return

	case r.Method == http.MethodPatch &&
		strings.HasPrefix(path, "/mall/me/orders/") &&
		strings.HasSuffix(path, "/return"):
		h.returnMe(w, r, path)
		return

	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/mall/me/orders/"):
		h.getMe(w, r, path)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
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
	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`
}

type createOrderRequest struct {
	ID string `json:"id"`

	ShippingAddressID string             `json:"shippingAddressId"`
	PaymentMethodID   string             `json:"paymentMethodId"`
	Items             []orderItemRequest `json:"items"`
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

	authUID, ok := middleware.CurrentUserUID(r)
	if !ok || authUID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing avatarId"})
		return
	}

	if req.ShippingAddressID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "shippingAddressId is required"})
		return
	}

	if req.PaymentMethodID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "paymentMethodId is required"})
		return
	}

	if len(req.Items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "items is required"})
		return
	}

	items := make([]usecase.CreateOrderItemInput, 0, len(req.Items))
	for _, requestItem := range req.Items {
		item, ok := orderItemRequestToInput(requestItem)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid order item"})
			return
		}

		items = append(items, item)
	}

	in := usecase.CreateOrderInput{
		ID:                req.ID,
		UserID:            authUID,
		AvatarID:          avatarID,
		CartID:            avatarID,
		ShippingAddressID: req.ShippingAddressID,
		PaymentMethodID:   req.PaymentMethodID,
		Items:             items,
	}

	out, err := h.uc.Create(ctx, in)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func orderItemRequestToInput(item orderItemRequest) (usecase.CreateOrderItemInput, bool) {
	itemType := orderdom.OrderItemType(item.Type)

	switch itemType {
	case orderdom.OrderItemTypeList:
		if item.ListID == "" || item.ModelID == "" || item.Qty <= 0 {
			return usecase.CreateOrderItemInput{}, false
		}

		return usecase.CreateOrderItemInput{
			Type:         orderdom.OrderItemTypeList,
			ListID:       item.ListID,
			ModelID:      item.ModelID,
			Qty:          item.Qty,
			IsCancelled:  item.IsCancelled,
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
			IsCancelled:  item.IsCancelled,
			IsDispatched: item.IsDispatched,
		}, true

	default:
		return usecase.CreateOrderItemInput{}, false
	}
}

func (h *OrderHandler) getMe(
	w http.ResponseWriter,
	r *http.Request,
	path string,
) {
	ctx := r.Context()

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing avatarId"})
		return
	}

	orderID := strings.TrimSpace(
		strings.TrimPrefix(
			path,
			"/mall/me/orders/",
		),
	)

	if orderID == "" || strings.Contains(orderID, "/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	out, err := h.uc.GetByID(ctx, orderID)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	if out.AvatarID != avatarID {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	if h == nil || h.orderDetailQuery == nil {
		writeOrderErr(
			w,
			errors.New(
				"order handler: order detail query not configured",
			),
		)
		return
	}

	detail, err := h.orderDetailQuery.EnrichOrderDetail(
		ctx,
		out,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(detail)
}

func (h *OrderHandler) cancelMe(
	w http.ResponseWriter,
	r *http.Request,
	path string,
) {
	ctx := r.Context()

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing avatarId"})
		return
	}

	itemPath := strings.TrimSpace(
		strings.TrimPrefix(
			path,
			"/mall/me/orders/",
		),
	)

	parts :=
		strings.Split(
			itemPath,
			"/",
		)

	if len(parts) != 4 ||
		parts[0] == "" ||
		parts[1] != "items" ||
		parts[2] == "" ||
		parts[3] != "cancel" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	orderID :=
		strings.TrimSpace(
			parts[0],
		)

	itemIndex, err :=
		strconv.Atoi(
			parts[2],
		)
	if err != nil ||
		itemIndex < 0 {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	out, err :=
		h.uc.CancelItem(
			ctx,
			usecase.CancelOrderItemInput{
				ID:        orderID,
				AvatarID:  avatarID,
				ItemIndex: itemIndex,
			},
		)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	if h == nil || h.orderDetailQuery == nil {
		writeOrderErr(
			w,
			errors.New(
				"order handler: order detail query not configured",
			),
		)
		return
	}

	detail, err :=
		h.orderDetailQuery.EnrichOrderDetail(
			ctx,
			out,
		)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(detail)
}

func (h *OrderHandler) returnMe(
	w http.ResponseWriter,
	r *http.Request,
	path string,
) {
	ctx := r.Context()

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing avatarId"})
		return
	}

	itemPath := strings.TrimSpace(
		strings.TrimPrefix(
			path,
			"/mall/me/orders/",
		),
	)

	parts :=
		strings.Split(
			itemPath,
			"/",
		)

	if len(parts) != 4 ||
		parts[0] == "" ||
		parts[1] != "items" ||
		parts[2] == "" ||
		parts[3] != "return" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	orderID :=
		strings.TrimSpace(
			parts[0],
		)

	itemIndex, err :=
		strconv.Atoi(
			parts[2],
		)
	if err != nil ||
		itemIndex < 0 {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	// Mallから行うのは返品申請の記録だけとする。
	//
	// ReturnItemはOrder itemのIsReturnRequestedを更新する責務だけを持ち、
	// Stripe Refund / Transfer Reversalは実行しない。
	//
	// 現在のRefundUsecaseはPayment単位の全額返金のみを扱うため、
	// item単位の返品申請をここから直接RefundByPaymentIDへ接続してはいけない。
	out, err :=
		h.uc.ReturnItem(
			ctx,
			usecase.ReturnOrderItemInput{
				ID:        orderID,
				AvatarID:  avatarID,
				ItemIndex: itemIndex,
			},
		)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	if h == nil || h.orderDetailQuery == nil {
		writeOrderErr(
			w,
			errors.New(
				"order handler: order detail query not configured",
			),
		)
		return
	}

	detail, err :=
		h.orderDetailQuery.EnrichOrderDetail(
			ctx,
			out,
		)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(detail)
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

	out, err := h.uc.ListByAvatarID(ctx, avatarID, sort, page)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	enriched, err := h.enrichOrderHistoryPage(ctx, out)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(enriched)
}

func (h *OrderHandler) enrichOrderHistoryPage(ctx context.Context, out any) (historydto.HistoryOrderPage, error) {
	if h == nil || h.historyQuery == nil {
		return historydto.HistoryOrderPage{}, errors.New("order handler: history query not configured")
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
	page := parsePositiveIntDefault(q.Get("page"), 1)
	perPage := parsePositiveIntDefault(q.Get("perPage"), 20)

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

func writeOrderErr(w http.ResponseWriter, err error) {
	message := "internal_error"
	if err != nil {
		message = err.Error()
	}

	writeJSON(w, orderHTTPStatus(err), map[string]string{"error": message})
}

func orderHTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusInternalServerError

	case errors.Is(err, context.Canceled):
		return 499

	case errors.Is(err, orderdom.ErrNotFound),
		errors.Is(err, inventorydom.ErrNotFound),
		errors.Is(err, listdom.ErrNotFound),
		errors.Is(err, modeldom.ErrNotFound),
		errors.Is(err, shippingaddressdom.ErrNotFound),
		errors.Is(err, transportationdom.ErrNotFound):
		return http.StatusNotFound

	case errors.Is(err, orderdom.ErrConflict),
		errors.Is(err, shippingaddressdom.ErrConflict):
		return http.StatusConflict

	case isInvalidOrderError(err),
		isInvalidShippingAddressError(err),
		isInvalidShippingQuoteError(err):
		return http.StatusBadRequest

	case isUnprocessableShippingQuoteError(err):
		return http.StatusUnprocessableEntity

	case isUnavailableShippingQuoteError(err):
		return http.StatusServiceUnavailable

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
		errors.Is(err, orderdom.ErrInvalidShippingQuote) ||
		errors.Is(err, orderdom.ErrInvalidShippingQuoteItem) ||
		errors.Is(err, orderdom.ErrInvalidPaymentMethod) ||
		errors.Is(err, orderdom.ErrInvalidItems) ||
		errors.Is(err, orderdom.ErrInvalidItemSnapshot) ||
		errors.Is(err, orderdom.ErrInvalidCreatedAt)
}

func isInvalidShippingQuoteError(err error) bool {
	return strings.HasPrefix(
		err.Error(),
		"usecase: invalid request",
	) ||
		errors.Is(err, inventorydom.ErrInvalidTransportationOption) ||
		errors.Is(err, inventorydom.ErrTransportationIDRequired) ||
		errors.Is(err, inventorydom.ErrTransportationIDNotAllowed) ||
		errors.Is(err, modeldom.ErrInvalidShippingPackage) ||
		errors.Is(err, transportationdom.ErrInvalidCarrier) ||
		errors.Is(err, transportationdom.ErrInvalidPackage) ||
		errors.Is(err, transportationdom.ErrInvalidAddress) ||
		errors.Is(err, transportationdom.ErrUnsupportedCountry) ||
		errors.Is(err, transportationdom.ErrInvalidPrefectureCode) ||
		errors.Is(err, transportationdom.ErrInvalidRateAmount)
}

func isUnprocessableShippingQuoteError(err error) bool {
	return errors.Is(err, transportationdom.ErrYamatoPackageTooLarge) ||
		errors.Is(err, transportationdom.ErrSagawaPackageTooLarge) ||
		errors.Is(err, transportationdom.ErrPostPackageTooLarge) ||
		errors.Is(err, transportationdom.ErrPostPackageTooHeavy) ||
		errors.Is(err, transportationdom.ErrSagawaIslandSurchargeRequired) ||
		errors.Is(err, transportationdom.ErrYamatoRateNotFound) ||
		errors.Is(err, transportationdom.ErrSagawaRateNotFound) ||
		errors.Is(err, transportationdom.ErrPostRateNotFound)
}

func isUnavailableShippingQuoteError(err error) bool {
	return errors.Is(err, transportationdom.ErrCarrierRateNotConfigured) ||
		errors.Is(err, transportationdom.ErrServiceUnavailable) ||
		strings.HasPrefix(
			err.Error(),
			"usecase: operation not supported",
		)
}
