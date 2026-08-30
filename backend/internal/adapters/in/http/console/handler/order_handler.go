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

// SettlementReadinessMarker is the minimal application contract required by
// OrderHandler after dispatch has been persisted.
//
// Payment success creates Settlement as pending. Only an Account seller whose
// active Order items have been dispatched may move to ready.
//
// The complete immutable SellerIdentity is passed to the Settlement usecase so
// CompanyID, AccountID and StripeAccountID remain part of the readiness
// boundary.
type SettlementReadinessMarker interface {
	MarkReadyByPaymentAndSeller(
		ctx context.Context,
		paymentID string,
		seller settlementdom.SellerIdentity,
	) (settlementdom.Settlement, error)
}

// OrderHandler handles:
//   - GET /orders/items
//   - GET /orders/action-required-count
//   - PATCH /orders/{id}/dispatch
//   - PATCH /orders/{id}/refund
//   - GET /orders/{id}
type OrderHandler struct {
	uc                     *usecase.OrderUsecase
	paymentFlowUC          *usecase.PaymentFlowUsecase
	paymentUC              *usecase.PaymentUsecase
	settlementUC           *usecase.SettlementUsecase
	refundUC               *usecase.RefundUsecase
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
	refundUC *usecase.RefundUsecase,
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
		refundUC:               refundUC,
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

	case r.Method == http.MethodGet && r.URL.Path == "/orders/action-required-count":
		h.countActionRequiredOrders(w, r)
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

	case r.Method == http.MethodPatch &&
		strings.HasPrefix(r.URL.Path, "/orders/") &&
		strings.HasSuffix(r.URL.Path, "/refund"):
		id := strings.TrimSuffix(
			strings.TrimPrefix(
				r.URL.Path,
				"/orders/",
			),
			"/refund",
		)
		h.refund(w, r, id)
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
	// 場合がある。そのためWebhookによるSettlement PENDING作成を前提にせず、
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
	// SellerSnapshotとPaymentをsource of truthとしてSettlement PENDINGを作成する。
	paidOrder, err := h.uc.GetByID(
		ctx,
		id,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	_, err = h.settlementUC.EnsureForSucceededPayment(
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

	// このConsole企業が実際に発送したTargetItemsからAccount sellerを抽出する。
	//
	// Accountに属する有効なOrder itemがすべて発送済みであることを確認した後、
	// そのSellerのSettlementだけをPENDING -> READYへ進めてCloud Tasksへ投入する。
	//
	// Order全体のSettlementをREADYにしてはいけない。
	// Company Aの発送でCompany BのSettlementを送金しないため、
	// SellerSnapshotのCompanyID、AccountID、StripeAccountIDを含む完全な
	// SellerIdentityを照合する。
	err = enqueueDispatchedSettlementTransfers(
		ctx,
		payment.PaymentID,
		result.Order.Items,
		result.TargetItems,
		h.settlementUC,
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

func (h *OrderHandler) refund(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.Trim(id, " \t\r\n/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if h == nil ||
		h.uc == nil ||
		h.paymentUC == nil ||
		h.refundUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_refund_not_wired"})
		return
	}

	order, err := h.uc.GetByID(
		ctx,
		id,
	)
	if err != nil {
		writeOrderErr(w, err)
		return
	}

	if order.ID != id {
		writeOrderErr(
			w,
			orderdom.ErrNotFound,
		)
		return
	}

	if !order.Paid {
		writeOrderErr(
			w,
			orderdom.ErrConflict,
		)
		return
	}

	// 現在のRefundUsecaseはPayment単位の全額返金のみを扱う。
	//
	// MallのReturnItemは商品単位の返品申請なので、1商品だけの申請を
	// Payment全額返金へ直接変換してはいけない。
	//
	// キャンセル済み商品を除くすべての商品が返品申請済みの場合だけ、
	// ConsoleからPayment全額返金を実行できる。
	if err := validateWholeOrderRefundRequested(
		order.Items,
	); err != nil {
		writeOrderErr(w, err)
		return
	}

	result, refundErr :=
		h.refundUC.RefundByPaymentID(
			ctx,
			usecase.RefundByPaymentIDInput{
				PaymentID: id,
			},
		)

	// Stripeが返した実際のRefund stateをそのままPaymentへ保存する。
	// PaymentIntentのStatus=succeededは維持し、Refund lifecycleだけを更新する。
	//
	// pending / requires_action / failed / canceledをsucceededへ上書きしない。
	// Transfer Reversalで後続エラーが発生した場合でも、購入者側Refundの
	// 実際の状態は失われないようにする。
	if result != nil &&
		result.StripeRefundID != "" {
		_, stateErr :=
			h.paymentUC.UpdateRefundState(
				ctx,
				usecase.UpdatePaymentRefundStateInput{
					PaymentID: id,

					StripeRefundID: result.StripeRefundID,
					RefundStatus:   result.RefundStatus,
					RefundedAmount: result.RefundedAmount,
					RefundedAt:     result.RefundedAt,
				},
			)
		if stateErr != nil {
			writeOrderErr(w, stateErr)
			return
		}
	}

	if refundErr != nil {
		writeOrderErr(w, refundErr)
		return
	}

	if result == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "refund_result_empty"})
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

func validateWholeOrderRefundRequested(
	items []orderdom.OrderItemSnapshot,
) error {
	if len(items) == 0 {
		return orderdom.ErrNotFound
	}

	activeCount := 0

	for _, item := range items {
		if item.IsCancelled {
			continue
		}

		activeCount++

		if !item.IsReturnRequested {
			return orderdom.ErrConflict
		}
	}

	if activeCount == 0 {
		return orderdom.ErrConflict
	}

	return nil
}

func enqueueDispatchedSettlementTransfers(
	ctx context.Context,
	paymentID string,
	orderItems []orderdom.OrderItemSnapshot,
	targetItems []orderdom.OrderItemSnapshot,
	settlementReadiness SettlementReadinessMarker,
	queue SettlementTransferEnqueuer,
) error {
	if paymentID == "" {
		return paymentdom.ErrInvalidPaymentID
	}

	if settlementReadiness == nil {
		return errors.New(
			"order dispatch: settlement readiness is not configured",
		)
	}

	if queue == nil {
		return errors.New(
			"order dispatch: settlement queue is not configured",
		)
	}

	if len(orderItems) == 0 ||
		len(targetItems) == 0 {
		return orderdom.ErrNotFound
	}

	// AccountID -> SellerIdentity
	//
	// このConsole発送routeは一次販売Account seller専用。
	// 同一Accountに複数Brandが紐づく場合でも1回だけREADY化・enqueueする。
	targetSellers := make(
		map[string]settlementdom.SellerIdentity,
	)

	for _, item := range targetItems {
		if item.IsCancelled ||
			item.IsReturnRequested ||
			!item.IsDispatched {
			return orderdom.ErrConflict
		}

		snapshot := item.SellerSnapshot

		seller := settlementdom.SellerIdentity{
			Type:            settlementdom.SellerTypeAccount,
			CompanyID:       snapshot.CompanyID,
			AccountID:       snapshot.AccountID,
			StripeAccountID: snapshot.StripeAccountID,
		}

		if err := seller.Validate(); err != nil {
			return orderdom.ErrInvalidSellerSnapshot
		}

		if existing, exists :=
			targetSellers[seller.AccountID]; exists &&
			existing != seller {
			return orderdom.ErrInvalidSellerSnapshot
		}

		targetSellers[seller.AccountID] = seller
	}

	if len(targetSellers) == 0 {
		return orderdom.ErrNotFound
	}

	accountIDs := make(
		[]string,
		0,
		len(targetSellers),
	)

	for accountID := range targetSellers {
		accountIDs = append(
			accountIDs,
			accountID,
		)
	}

	sort.Strings(accountIDs)

	for _, accountID := range accountIDs {
		seller := targetSellers[accountID]

		allDispatched, err :=
			areSettlementAccountItemsDispatched(
				orderItems,
				seller.CompanyID,
				seller.AccountID,
			)
		if err != nil {
			return err
		}

		if !allDispatched {
			continue
		}

		settlement, err :=
			settlementReadiness.MarkReadyByPaymentAndSeller(
				ctx,
				paymentID,
				seller,
			)
		if err != nil {
			return err
		}

		if settlement.ID == "" {
			return settlementdom.ErrInvalidID
		}

		settlementSeller := settlement.SellerIdentity()
		if err := settlementSeller.Validate(); err != nil {
			return errors.New(
				"order dispatch: invalid settlement seller identity",
			)
		}

		if settlement.PaymentID != paymentID ||
			settlementSeller != seller {
			return errors.New(
				"order dispatch: settlement identity mismatch",
			)
		}

		switch settlement.Status {
		case settlementdom.StatusReady,
			settlementdom.StatusFailedRetryable,
			settlementdom.StatusTransferring:

		case settlementdom.StatusTransferred,
			settlementdom.StatusFailed:
			continue

		default:
			return settlementdom.ErrInvalidStatusTransition
		}

		if err := queue.EnqueueSettlementTransfer(
			ctx,
			settlement.ID,
		); err != nil {
			return err
		}
	}

	return nil
}

func areSettlementAccountItemsDispatched(
	orderItems []orderdom.OrderItemSnapshot,
	companyID string,
	accountID string,
) (bool, error) {
	if companyID == "" ||
		accountID == "" {
		return false,
			orderdom.ErrInvalidSellerSnapshot
	}

	matched := false

	for _, item := range orderItems {
		if item.IsCancelled {
			continue
		}

		if item.SellerSnapshot.AccountID != accountID {
			continue
		}

		if item.SellerSnapshot.CompanyID != companyID {
			return false,
				orderdom.ErrInvalidSellerSnapshot
		}

		matched = true

		if item.IsReturnRequested ||
			!item.IsDispatched {
			return false, nil
		}
	}

	if !matched {
		return false,
			orderdom.ErrNotFound
	}

	return true, nil
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

func (h *OrderHandler) countActionRequiredOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.q == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_management_query_not_wired"})
		return
	}

	count, err := h.q.CountActionRequiredOrders(ctx)
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
		errors.Is(err, paymentdom.ErrInvalidPaymentID),
		errors.Is(err, paymentdom.ErrInvalidRefundStatus),
		errors.Is(err, paymentdom.ErrInvalidStripeRefundID),
		errors.Is(err, paymentdom.ErrInvalidRefundedAmount),
		errors.Is(err, paymentdom.ErrInvalidRefundedAt),
		errors.Is(err, usecase.ErrPaymentFlowPaymentIDEmpty),
		errors.Is(err, usecase.ErrPaymentFlowAmountInvalid):
		code = http.StatusBadRequest

	case errors.Is(err, orderdom.ErrNotFound),
		errors.Is(err, paymentdom.ErrNotFound),
		errors.Is(err, usecase.ErrPaymentFlowOrderNotFound):
		code = http.StatusNotFound

	case errors.Is(err, orderdom.ErrConflict),
		errors.Is(err, paymentdom.ErrConflict),
		errors.Is(err, paymentdom.ErrRefundRequiresSucceeded),
		errors.Is(err, paymentdom.ErrInvalidRefundState),
		errors.Is(err, settlementdom.ErrInvalidStatusTransition),
		errors.Is(err, usecase.ErrRefundPaymentNotSucceeded),
		errors.Is(err, usecase.ErrRefundSettlementAmountMismatch),
		errors.Is(err, usecase.ErrRefundSettlementPaymentMismatch),
		errors.Is(err, usecase.ErrRefundSettlementDuplicate),
		errors.Is(err, usecase.ErrRefundSettlementTransferring),
		errors.Is(err, usecase.ErrRefundSettlementFailed),
		errors.Is(err, usecase.ErrRefundSettlementStatusUnsupported),
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
