// backend/internal/adapters/in/http/mall/handler/trade_return_receipt_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"

	usecase "narratives/internal/application/usecase"
	inquirydom "narratives/internal/domain/inquiry"
	refunddom "narratives/internal/domain/refund"
	tradedom "narratives/internal/domain/trade"
)

type receiveTradeReturnRequest struct {
	MerchandiseRefundAmount int  `json:"merchandiseRefundAmount"`
	RefundOutboundShipping  bool `json:"refundOutboundShipping"`
	CoverReturnShipping     bool `json:"coverReturnShipping"`
}

type receiveTradeReturnResponse struct {
	Data receiveTradeReturnResultResponse `json:"data"`
}

type receiveTradeReturnResultResponse struct {
	FinanciallyCompleted bool `json:"financiallyCompleted"`
	OrderCompleted       bool `json:"orderCompleted"`
	NotificationEnsured  bool `json:"notificationEnsured"`
	AlreadyCompleted     bool `json:"alreadyCompleted"`
}

// POST /mall/me/trades/{tradeId}/receive-return
//
// SellerAvatarID is never accepted from the client. The authenticated Avatar
// from AvatarContextMiddleware is authoritative.
//
// Request body for both unopened and opened returns:
//
//	{
//	  "merchandiseRefundAmount": 5000,
//	  "refundOutboundShipping": true,
//	  "coverReturnShipping": true
//	}
//
// MerchandiseRefundAmount is tax-inclusive and must not exceed the authoritative
// merchandise amount including tax. Consumption tax is automatically allocated
// by the backend from the persisted Order snapshot.
//
// The usecase resolves Trade, Order, Order item, return Inquiry, tax amounts and
// shipping amounts from authoritative persisted state. Order ID, item index,
// Inquiry ID, tax amount, shipping amount and seller identity are never accepted
// from the client.
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
		internalError(w, "resale trade return receipt usecase is nil")
		return
	}

	var req receiveTradeReturnRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}

	selection := refunddom.ReturnRefundSelection{
		MerchandiseRefundAmount: req.MerchandiseRefundAmount,
		RefundOutboundShipping:  req.RefundOutboundShipping,
		CoverReturnShipping:     req.CoverReturnShipping,
	}
	if err := refunddom.ValidateReturnRefundSelection(selection); err != nil {
		badRequest(w, err.Error())
		return
	}

	result, err := h.returnReceiptUC.ReceiveReturn(
		r.Context(),
		usecase.ReceiveResaleTradeReturnInput{
			TradeID:        tradeID,
			SellerAvatarID: avatarID,
			Selection:      selection,
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

	writeJSON(w, status, receiveTradeReturnResponse{
		Data: receiveTradeReturnResultResponse{
			FinanciallyCompleted: result.FinanciallyCompleted,
			OrderCompleted:       result.OrderCompleted,
			NotificationEnsured:  result.NotificationEnsured,
			AlreadyCompleted:     result.AlreadyCompleted,
		},
	})
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

	case errors.Is(err, refunddom.ErrInvalidReturnRefundAmount),
		errors.Is(err, refunddom.ErrInvalidReturnRefundAmounts):
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
