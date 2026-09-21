// backend/internal/adapters/in/http/mall/handler/trade_message_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"

	usecase "narratives/internal/application/usecase"
)

type createTradeMessageRequest struct {
	Content string `json:"content"`
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
