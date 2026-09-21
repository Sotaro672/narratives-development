// backend/internal/adapters/in/http/mall/handler/trade_query_handler.go
package mallHandler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	mallquery "narratives/internal/application/query/mall"
	tradedom "narratives/internal/domain/trade"
)

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
