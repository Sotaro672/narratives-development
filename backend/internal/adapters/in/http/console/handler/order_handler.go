// backend/internal/adapters/in/http/console/handler/order_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	orderq "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	settlementdom "narratives/internal/domain/settlement"
)

// SettlementTransferEnqueuer is the minimal outbound contract required by
// OrderHandler after a seller's Order items have been dispatched.
//
// The concrete implementation is SettlementQueue.
//
// The queue payload must contain only SettlementID. Amount, Stripe Account,
// Charge ID, and TransferGroup are loaded from the authoritative Settlement
// document by the internal worker.
type SettlementTransferEnqueuer interface {
	EnqueueSettlementTransfer(
		ctx context.Context,
		settlementID string,
	) error
}

// OrderHandler handles:
//   - GET /orders/items
//   - GET /orders/undispatched-count
//   - PATCH /orders/{id}/dispatch
//   - GET /orders/{id}
type OrderHandler struct {
	uc                     *usecase.OrderUsecase
	paymentFlowUC          *usecase.PaymentFlowUsecase
	paymentUC              *usecase.PaymentUsecase
	settlementUC           *usecase.SettlementUsecase
	settlementQueue        SettlementTransferEnqueuer
	q                      *orderq.OrderManagementQuery
	detailQ                *orderq.OrderDetailQuery
	dispatchNotificationUC usecase.OrderDispatchNotificationUsecasePort
}

func NewOrderHandler(
	uc *usecase.OrderUsecase,
	paymentFlowUC *usecase.PaymentFlowUsecase,
	paymentUC *usecase.PaymentUsecase,
	settlementUC *usecase.SettlementUsecase,
	settlementQueue SettlementTransferEnqueuer,
	q *orderq.OrderManagementQuery,
	detailQ *orderq.OrderDetailQuery,
	dispatchNotificationUC usecase.OrderDispatchNotificationUsecasePort,
) http.Handler {
	return &OrderHandler{
		uc:                     uc,
		paymentFlowUC:          paymentFlowUC,
		paymentUC:              paymentUC,
		settlementUC:           settlementUC,
		settlementQueue:        settlementQueue,
		q:                      q,
		detailQ:                detailQ,
		dispatchNotificationUC: dispatchNotificationUC,
	}
}

func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/orders/items":
		h.listItemRows(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/orders/undispatched-count":
		h.countUndispatchedOrders(w, r)
		return

	case r.Method == http.MethodPatch &&
		strings.HasPrefix(r.URL.Path, "/orders/") &&
		strings.HasSuffix(r.URL.Path, "/dispatch"):
		id := strings.TrimSuffix(
			strings.TrimPrefix(
				r.URL.Path,
				"/orders/",
			),
			"/dispatch",
		)
		h.dispatch(w, r, id)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/orders/"):
		id := strings.TrimPrefix(r.URL.Path, "/orders/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.Trim(id, " \t\r\n/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if h == nil || h.detailQ == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_detail_query_not_wired"})
		return
	}

	dto, err := h.detailQ.GetByID(ctx, id)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(dto)
}

func (h *OrderHandler) dispatch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.Trim(id, " \t\r\n/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if h == nil ||
		h.uc == nil ||
		h.paymentFlowUC == nil ||
		h.paymentUC == nil ||
		h.settlementUC == nil ||
		h.settlementQueue == nil ||
		h.q == nil ||
		h.detailQ == nil ||
		h.dispatchNotificationUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_dispatch_not_wired"})
		return
	}

	allowedInventoryIDs, err := h.q.AllowedInventoryIDSet(ctx)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	dispatchInput := usecase.DispatchOrderItemsInput{
		ID:                  id,
		AllowedInventoryIDs: allowedInventoryIDs,
	}

	// 決済前に、このConsole企業が発送できる対象商品を持つことを確認する。
	// この時点ではOrderの発送状態を変更しない。
	_, err = h.uc.PrepareDispatchItems(
		ctx,
		dispatchInput,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	// Orderに保存されたPaymentMethodSnapshotを使って発送時決済を行う。
	// 決済が成功し、order.Paid=trueが永続化されるまで発送状態へ進めない。
	err = h.paymentFlowUC.EnsureOrderPaidForDispatch(
		ctx,
		id,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	// 発送時off-session決済は、Stripe webhookより先にsucceededが確定する
	// 場合がある。そのためWebhookによるSettlement READY作成を前提にせず、
	// ここでも現在のsucceeded Paymentを取得してSettlementを冪等に保証する。
	payment, err := h.paymentUC.GetByPaymentID(
		ctx,
		id,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	if payment == nil ||
		payment.PaymentID != id ||
		payment.Status != paymentdom.StatusSucceeded {
		writeOrderErr(
			w,
			usecase.ErrPaymentFlowDispatchNotSucceeded,
		)
		return
	}

	// EnsureOrderPaidForDispatch後の最新Orderを再取得する。
	// SellerSnapshotとPaymentをsource of truthとしてSettlement READYを作成する。
	paidOrder, err := h.uc.GetByID(
		ctx,
		id,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	settlements, err := h.settlementUC.EnsureForSucceededPayment(
		ctx,
		paidOrder,
		*payment,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	result, err := h.uc.DispatchItems(
		ctx,
		dispatchInput,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	// このConsole企業が実際に発送したTargetItemsからAccountIDを抽出し、
	// そのAccountに対応するSettlementだけをCloud Tasksへ投入する。
	//
	// Order全体のSettlementを投入してはいけない。
	// Company Aの発送でCompany BのSettlementを送金しないため、
	// SellerSnapshot.CompanyIDとAccountIDの両方を照合する。
	err = enqueueDispatchedSettlementTransfers(
		ctx,
		payment.PaymentID,
		result.TargetItems,
		settlements,
		h.settlementQueue,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	// DispatchItemsが既に発送済みを返した場合も必ずEnsureDeliveryを呼ぶ。
	// 発送状態保存後、通知outbox作成前に処理が停止したケースを再実行で修復する。
	_, err = h.dispatchNotificationUC.EnsureDelivery(
		ctx,
		result.Order,
		result.TargetItems,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	dto, err := h.detailQ.GetByID(ctx, id)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(dto)
}

func enqueueDispatchedSettlementTransfers(
	ctx context.Context,
	paymentID string,
	targetItems []orderdom.OrderItemSnapshot,
	settlements []settlementdom.Settlement,
	queue SettlementTransferEnqueuer,
) error {
	if paymentID == "" {
		return paymentdom.ErrInvalidPaymentID
	}

	if queue == nil {
		return errors.New(
			"order dispatch: settlement queue is not configured",
		)
	}

	if len(targetItems) == 0 {
		return orderdom.ErrNotFound
	}

	// AccountID -> CompanyID
	//
	// 同一Accountに複数Brandが紐づく場合でも1回だけenqueueする。
	targetAccounts := make(
		map[string]string,
	)

	for _, item := range targetItems {
		if item.IsCancelled ||
			item.IsReturnRequested ||
			!item.IsDispatched {
			return orderdom.ErrConflict
		}

		accountID :=
			item.SellerSnapshot.AccountID

		companyID :=
			item.SellerSnapshot.CompanyID

		if accountID == "" ||
			companyID == "" {
			return orderdom.ErrInvalidSellerSnapshot
		}

		if existingCompanyID, exists :=
			targetAccounts[accountID]; exists &&
			existingCompanyID != companyID {
			return orderdom.ErrInvalidSellerSnapshot
		}

		targetAccounts[accountID] =
			companyID
	}

	if len(targetAccounts) == 0 {
		return orderdom.ErrNotFound
	}

	settlementByAccount := make(
		map[string]settlementdom.Settlement,
		len(targetAccounts),
	)

	for _, settlement := range settlements {
		if settlement.PaymentID != paymentID {
			return errors.New(
				"order dispatch: settlement payment mismatch",
			)
		}

		accountID :=
			settlement.AccountID

		expectedCompanyID, target :=
			targetAccounts[accountID]
		if !target {
			continue
		}

		if settlement.CompanyID != expectedCompanyID {
			return errors.New(
				"order dispatch: settlement company mismatch",
			)
		}

		if _, exists :=
			settlementByAccount[accountID]; exists {
			return usecase.ErrSettlementDuplicateAccount
		}

		settlementByAccount[accountID] =
			settlement
	}

	accountIDs := make(
		[]string,
		0,
		len(targetAccounts),
	)

	for accountID := range targetAccounts {
		accountIDs = append(
			accountIDs,
			accountID,
		)
	}

	sort.Strings(accountIDs)

	for _, accountID := range accountIDs {
		settlement, ok :=
			settlementByAccount[accountID]
		if !ok {
			return errors.New(
				"order dispatch: settlement for dispatched account not found",
			)
		}

		settlementID :=
			settlement.ID

		if settlementID == "" {
			return settlementdom.ErrInvalidID
		}

		if err := queue.EnqueueSettlementTransfer(
			ctx,
			settlementID,
		); err != nil {
			return err
		}
	}

	return nil
}

func (h *OrderHandler) listItemRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.q == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_management_query_not_wired"})
		return
	}

	filter, page, err := parseOrderListParams(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var sort common.Sort

	pr, err := h.q.ListItemInventoryRows(ctx, filter, sort, page)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(pr)
}

func (h *OrderHandler) countUndispatchedOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.q == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_management_query_not_wired"})
		return
	}

	count, err := h.q.CountUndispatchedOrders(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]int{"count": count})
}

// ============================================================
// Query param parsing
// ============================================================

func parseOrderListParams(r *http.Request) (orderdom.Filter, common.Page, error) {
	q := r.URL.Query()

	pageNum := parseIntDefault(q.Get("page"), 1)
	perPage := parseIntDefault(q.Get("perPage"), 20)

	f := orderdom.Filter{
		ID: q.Get("id"),
	}

	if v := q.Get("userId"); v != "" {
		f.UserID = v
	}
	if v := q.Get("avatarId"); v != "" {
		f.AvatarID = v
	}
	if v := q.Get("cartId"); v != "" {
		f.CartID = v
	}
	if v := q.Get("modelId"); v != "" {
		f.ModelID = v
	}
	if v := q.Get("inventoryId"); v != "" {
		f.InventoryID = v
	}
	if v := q.Get("listId"); v != "" {
		f.ListID = v
	}

	if v := q.Get("createdFrom"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return orderdom.Filter{}, common.Page{}, errors.New("invalid createdFrom (expected RFC3339)")
		}
		f.CreatedFrom = &t
	}
	if v := q.Get("createdTo"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return orderdom.Filter{}, common.Page{}, errors.New("invalid createdTo (expected RFC3339)")
		}
		f.CreatedTo = &t
	}

	p := common.Page{
		Number:  pageNum,
		PerPage: perPage,
	}

	return f, p, nil
}

// ============================================================
// Error handling
// ============================================================

func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, orderdom.ErrInvalidID),
		errors.Is(err, usecase.ErrPaymentFlowPaymentIDEmpty),
		errors.Is(err, usecase.ErrPaymentFlowAmountInvalid):
		code = http.StatusBadRequest

	case errors.Is(err, orderdom.ErrNotFound),
		errors.Is(err, usecase.ErrPaymentFlowOrderNotFound):
		code = http.StatusNotFound

	case errors.Is(err, orderdom.ErrConflict),
		errors.Is(err, usecase.ErrPaymentFlowOrderAlreadyPaid),
		errors.Is(err, usecase.ErrPaymentFlowPaymentMethodMismatch),
		errors.Is(err, usecase.ErrPaymentFlowDispatchRequiresAction),
		errors.Is(err, usecase.ErrPaymentFlowDispatchProcessing),
		errors.Is(err, usecase.ErrPaymentFlowDispatchPending),
		errors.Is(err, usecase.ErrPaymentFlowDispatchNotSucceeded),
		errors.Is(err, usecase.ErrPaymentFlowDispatchPaymentMismatch),
		errors.Is(err, usecase.ErrPaymentFlowDispatchPaidStateInvalid),
		errors.Is(err, usecase.ErrPaymentFlowStripePaymentIntentFailed),
		errors.Is(err, usecase.ErrPaymentFlowStripePaymentIntentCanceled):
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
