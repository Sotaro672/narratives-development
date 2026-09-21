// backend/internal/adapters/in/http/mall/handler/trade_dispatch_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"

	usecase "narratives/internal/application/usecase"
	tradedom "narratives/internal/domain/trade"
	transportationdom "narratives/internal/domain/transportation"
)

type dispatchTradeRequest struct {
	Carrier transportationdom.Carrier `json:"carrier"`
	BoxSize int                       `json:"boxSize"`
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
