// backend/internal/adapters/in/http/mall/handler/trade_error.go
package mallHandler

import (
	"errors"
	"net/http"

	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	tradedom "narratives/internal/domain/trade"
)

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
