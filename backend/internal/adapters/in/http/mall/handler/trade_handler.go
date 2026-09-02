// backend/internal/adapters/in/http/mall/handler/trade_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	inquirydom "narratives/internal/domain/inquiry"
	refunddom "narratives/internal/domain/refund"
	tradedom "narratives/internal/domain/trade"
	transportationdom "narratives/internal/domain/transportation"
)

// TradeHandler handles private Resale Trade communication in Mall.
//
// Trade is limited to secondary-market transactions:
//
//	buyer Avatar <-> seller Avatar
//
// Avatar identity is always resolved from AvatarContextMiddleware and is never
// accepted from request body or query parameters.
type TradeHandler struct {
	query           *mallquery.TradeQuery
	messageUC       *usecase.TradeMessageUsecase
	dispatchUC      *usecase.ResaleTradeDispatchUsecase
	returnReceiptUC *usecase.ResaleTradeReturnReceiptUsecase
}

func NewTradeHandler(
	query *mallquery.TradeQuery,
	messageUC *usecase.TradeMessageUsecase,
	dispatchUC *usecase.ResaleTradeDispatchUsecase,
	returnReceiptUC *usecase.ResaleTradeReturnReceiptUsecase,
) http.Handler {
	return &TradeHandler{
		query:           query,
		messageUC:       messageUC,
		dispatchUC:      dispatchUC,
		returnReceiptUC: returnReceiptUC,
	}
}

// ServeHTTP is the routing entry point.
//
// Supported:
//
//	GET  /mall/me/trades
//	GET  /mall/me/trades/order-items/{orderId}/{itemIndex}
//	GET  /mall/me/trades/{tradeId}
//	POST /mall/me/trades/{tradeId}/messages
//	POST /mall/me/trades/{tradeId}/read
//	GET  /mall/me/trades/{tradeId}/unread-count
//	POST /mall/me/trades/{tradeId}/dispatch
//	POST /mall/me/trades/{tradeId}/receive-return
func (h *TradeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.query == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "trade query is nil",
		})
		return
	}
	if h.messageUC == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "trade message usecase is nil",
		})
		return
	}

	if r.URL.Path == "/mall/me/trades" || r.URL.Path == "/mall/me/trades/" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.list(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/mall/me/trades/order-items/") {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getByOrderItem(w, r)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/mall/me/trades/") {
		notFound(w)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/mall/me/trades/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		notFound(w)
		return
	}

	tradeID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getByID(w, r, tradeID)
		return
	}

	if len(parts) != 2 || parts[1] == "" {
		notFound(w)
		return
	}

	switch parts[1] {
	case "messages":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.createMessage(w, r, tradeID)

	case "read":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.markRead(w, r, tradeID)

	case "unread-count":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.countUnread(w, r, tradeID)

	case "dispatch":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.dispatch(w, r, tradeID)

	case "receive-return":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.receiveReturn(w, r, tradeID)

	default:
		notFound(w)
	}
}

type createTradeMessageRequest struct {
	Content string `json:"content"`
}

type dispatchTradeRequest struct {
	Carrier transportationdom.Carrier `json:"carrier"`
	BoxSize int                       `json:"boxSize"`
}

type receiveTradeReturnRequest struct {
	Policy refunddom.OpenedReturnRefundPolicy `json:"policy,omitempty"`
}

// GET /mall/me/trades
//
// Returns all Resale Trades in which the authenticated Avatar participates as
// buyer or seller. The result is intended for ChatListPage and includes latest
// message, unread count and latest activity information.
func (h *TradeHandler) list(w http.ResponseWriter, r *http.Request) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	result, err := h.query.ListForAvatar(
		r.Context(),
		avatarID,
	)
	if err != nil {
		writeTradeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GET /mall/me/trades/order-items/{orderId}/{itemIndex}
//
// Resolves one Trade from its authoritative Order item identity and returns the
// Trade chat detail including messages. This endpoint is used when OrderDetail
// knows orderId + itemIndex but does not yet know tradeId.
func (h *TradeHandler) getByOrderItem(w http.ResponseWriter, r *http.Request) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	rest := strings.TrimPrefix(
		r.URL.Path,
		"/mall/me/trades/order-items/",
	)

	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		badRequest(w, "invalid order item path")
		return
	}

	orderID := parts[0]

	itemIndex, err := strconv.Atoi(parts[1])
	if err != nil || itemIndex < 0 {
		badRequest(w, "invalid order item index")
		return
	}

	messageLimit, beforeCreatedAt, afterCreatedAt, ok :=
		parseTradeMessageListQuery(w, r)
	if !ok {
		return
	}

	detail, err := h.query.GetByOrderItem(
		r.Context(),
		mallquery.GetTradeByOrderItemInput{
			AvatarID:        avatarID,
			OrderID:         orderID,
			OrderItemIndex:  itemIndex,
			MessageLimit:    messageLimit,
			BeforeCreatedAt: beforeCreatedAt,
			AfterCreatedAt:  afterCreatedAt,
		},
	)
	if err != nil {
		writeTradeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": detail,
	})
}

// GET /mall/me/trades/{tradeId}
//
// Returns one Trade chat detail for ChatDetailPage when Trade ID is known.
func (h *TradeHandler) getByID(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}

	messageLimit, beforeCreatedAt, afterCreatedAt, ok :=
		parseTradeMessageListQuery(w, r)
	if !ok {
		return
	}

	detail, err := h.query.GetByID(
		r.Context(),
		mallquery.GetTradeByIDInput{
			AvatarID:        avatarID,
			TradeID:         tradeID,
			MessageLimit:    messageLimit,
			BeforeCreatedAt: beforeCreatedAt,
			AfterCreatedAt:  afterCreatedAt,
		},
	)
	if err != nil {
		writeTradeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": detail,
	})
}

// POST /mall/me/trades/{tradeId}/messages
//
// Body:
//
//	{
//	  "content": "発送ありがとうございます"
//	}
//
// SenderSide, SenderType and SenderID are never accepted from the client.
// TradeMessageUsecase derives them from the authenticated Avatar and Trade.
func (h *TradeHandler) createMessage(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}

	var req createTradeMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}

	if req.Content == "" {
		badRequest(w, "content is required")
		return
	}

	created, err := h.messageUC.CreateMessage(
		r.Context(),
		usecase.CreateTradeMessageInput{
			TradeID:  tradeID,
			AvatarID: avatarID,
			Content:  req.Content,
		},
	)
	if err != nil {
		writeTradeErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"data": created,
	})
}

// POST /mall/me/trades/{tradeId}/read
//
// Marks messages sent by the opposite side as read by the authenticated Avatar.
func (h *TradeHandler) markRead(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}

	err := h.messageUC.MarkRead(
		r.Context(),
		usecase.MarkTradeMessagesReadInput{
			TradeID:  tradeID,
			AvatarID: avatarID,
		},
	)
	if err != nil {
		writeTradeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}

// GET /mall/me/trades/{tradeId}/unread-count
//
// Returns unread messages for the currently authenticated participant.
func (h *TradeHandler) countUnread(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}

	count, err := h.messageUC.CountUnread(
		r.Context(),
		usecase.CountUnreadTradeMessagesInput{
			TradeID:  tradeID,
			AvatarID: avatarID,
		},
	)
	if err != nil {
		writeTradeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"count": count,
	})
}

// POST /mall/me/trades/{tradeId}/dispatch
//
// Body:
//
//	{
//	  "carrier": "yamato",
//	  "boxSize": 80
//	}
//
// SellerAvatarID is never accepted from the client. The authenticated Avatar
// from AvatarContextMiddleware is used as the seller identity.
//
// Shipping amount is never accepted from the client. The usecase resolves the
// authoritative flat rate from carrier and boxSize before payment.
//
// The usecase performs:
//   - Trade seller authorization
//   - authoritative Order item validation
//   - authoritative Resale shipping rate calculation
//   - ShippingQuoteSnapshot update
//   - off-session payment
//   - SalesReceivable pending creation
//   - Order item dispatch state update
func (h *TradeHandler) dispatch(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}

	if h == nil || h.dispatchUC == nil {
		internalError(
			w,
			"resale trade dispatch usecase is nil",
		)
		return
	}

	var req dispatchTradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}

	result, err := h.dispatchUC.Dispatch(
		r.Context(),
		usecase.DispatchResaleTradeInput{
			TradeID:        tradeID,
			SellerAvatarID: avatarID,
			Carrier:        req.Carrier,
			BoxSize:        req.BoxSize,
		},
	)
	if err != nil {
		writeTradeDispatchErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

// POST /mall/me/trades/{tradeId}/receive-return
//
// SellerAvatarID is never accepted from the client. The authenticated Avatar
// from AvatarContextMiddleware is authoritative.
//
// Unopened return:
//
//	Request body may be omitted or sent as {}.
//
// Opened return:
//
//	{
//	  "policy": "half_merchandise"
//	}
//
// The usecase resolves Trade, Order, Order item and return Inquiry from
// authoritative persisted state. Order ID, item index, Inquiry ID, refund amount
// and seller identity are never accepted from the client.
func (h *TradeHandler) receiveReturn(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}

	if h == nil || h.returnReceiptUC == nil {
		internalError(
			w,
			"resale trade return receipt usecase is nil",
		)
		return
	}

	var req receiveTradeReturnRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		badRequest(w, "invalid json")
		return
	}

	result, err := h.returnReceiptUC.ReceiveReturn(
		r.Context(),
		usecase.ReceiveResaleTradeReturnInput{
			TradeID:        tradeID,
			SellerAvatarID: avatarID,
			Policy:         req.Policy,
		},
	)
	if err != nil {
		writeTradeReturnReceiptErr(w, err)
		return
	}

	status := http.StatusOK
	if !result.FinanciallyCompleted {
		status = http.StatusAccepted
	}

	writeJSON(w, status, map[string]any{
		"data": result,
	})
}

func parseTradeMessageListQuery(
	w http.ResponseWriter,
	r *http.Request,
) (
	int,
	*time.Time,
	*time.Time,
	bool,
) {
	limit := tradedom.DefaultMessageListLimit

	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value <= 0 {
			badRequest(w, "invalid limit")
			return 0, nil, nil, false
		}

		if value > tradedom.MaxMessageListLimit {
			value = tradedom.MaxMessageListLimit
		}

		limit = value
	}

	beforeCreatedAt, ok := parseTradeMessageTimeQuery(
		w,
		r.URL.Query().Get("beforeCreatedAt"),
		"beforeCreatedAt",
	)
	if !ok {
		return 0, nil, nil, false
	}

	afterCreatedAt, ok := parseTradeMessageTimeQuery(
		w,
		r.URL.Query().Get("afterCreatedAt"),
		"afterCreatedAt",
	)
	if !ok {
		return 0, nil, nil, false
	}

	if beforeCreatedAt != nil &&
		afterCreatedAt != nil &&
		!afterCreatedAt.Before(*beforeCreatedAt) {
		badRequest(w, "invalid message time range")
		return 0, nil, nil, false
	}

	return limit, beforeCreatedAt, afterCreatedAt, true
}

func parseTradeMessageTimeQuery(
	w http.ResponseWriter,
	raw string,
	field string,
) (*time.Time, bool) {
	if raw == "" {
		return nil, true
	}

	value, err := time.Parse(
		time.RFC3339Nano,
		raw,
	)
	if err != nil {
		badRequest(w, "invalid "+field)
		return nil, false
	}

	value = value.UTC()
	return &value, true
}

func writeTradeReturnReceiptErr(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		return

	case errors.Is(err, tradedom.ErrNotFound),
		errors.Is(err, inquirydom.ErrNotFound),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptTradeMismatch),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptInquiryMismatch):
		notFound(w)

	case errors.Is(err, usecase.ErrResaleTradeReturnReceiptInvalidSeller):
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "avatar context is required",
		})

	case errors.Is(err, refunddom.ErrInvalidOpenedReturnRefundPolicy),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptUnexpectedPolicy):
		badRequest(w, err.Error())

	case errors.Is(err, usecase.ErrResaleTradeReturnReceiptOrderNotPaid),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptReturnNotRequested),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptInquiryClosed),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptInquiryResolved),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptReturnKindMismatch),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptUnopenedStateInvalid),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptOrderCompletionMismatch):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(err, usecase.ErrResaleTradeReturnReceiptNotConfigured):
		internalError(w, err.Error())

	default:
		writeOrderErr(w, err)
	}
}

func writeTradeDispatchErr(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		return

	case errors.Is(err, tradedom.ErrNotFound),
		errors.Is(err, usecase.ErrPaymentFlowOrderNotFound):
		notFound(w)

	case errors.Is(err, tradedom.ErrConflict),
		errors.Is(err, tradedom.ErrTradeAlreadyClosed),
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
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(err, tradedom.ErrInvalidID),
		errors.Is(err, tradedom.ErrInvalidSellerAvatarID),
		errors.Is(err, transportationdom.ErrInvalidCarrier),
		errors.Is(err, transportationdom.ErrInvalidResaleBoxSize):
		badRequest(w, err.Error())

	default:
		writeOrderErr(w, err)
	}
}

func writeTradeErr(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		return

	case errors.Is(err, tradedom.ErrNotFound),
		errors.Is(err, tradedom.ErrMessageNotFound):
		notFound(w)

	case errors.Is(err, mallquery.ErrTradeQueryAvatarIDEmpty),
		errors.Is(err, usecase.ErrTradeMessageAvatarIDEmpty):
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "avatar context is required",
		})

	case errors.Is(err, mallquery.ErrTradeQueryUnsupportedTrade),
		errors.Is(err, usecase.ErrTradeMessageUnsupportedTrade):
		notFound(w)

	case errors.Is(err, tradedom.ErrTradeAlreadyClosed):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(err, tradedom.ErrMessageAlreadyExists),
		errors.Is(err, tradedom.ErrAlreadyExists),
		errors.Is(err, tradedom.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(err, tradedom.ErrInvalidID),
		errors.Is(err, tradedom.ErrInvalidOrderID),
		errors.Is(err, tradedom.ErrInvalidOrderItemIndex),
		errors.Is(err, tradedom.ErrInvalidSellerAvatarID),
		errors.Is(err, tradedom.ErrInvalidMessageID),
		errors.Is(err, tradedom.ErrInvalidMessageTradeID),
		errors.Is(err, tradedom.ErrInvalidMessageSenderSide),
		errors.Is(err, tradedom.ErrInvalidMessageSenderType),
		errors.Is(err, tradedom.ErrInvalidMessageSenderID),
		errors.Is(err, tradedom.ErrInvalidMessageContent),
		errors.Is(err, tradedom.ErrMessageContentOrImageRequired),
		errors.Is(err, tradedom.ErrTooManyMessageImages),
		errors.Is(err, tradedom.ErrInvalidMessageCreatedAt),
		errors.Is(err, tradedom.ErrInvalidBuyerReadAt),
		errors.Is(err, tradedom.ErrInvalidSellerReadAt),
		errors.Is(err, tradedom.ErrInvalidStatus):
		badRequest(w, err.Error())

	case errors.Is(err, mallquery.ErrTradeQueryNotConfigured),
		errors.Is(err, usecase.ErrTradeMessageUsecaseNotConfigured):
		internalError(w, err.Error())

	default:
		internalError(w, "trade operation failed")
	}
}
