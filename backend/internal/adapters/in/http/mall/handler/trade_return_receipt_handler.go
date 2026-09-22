// backend/internal/adapters/in/http/mall/handler/trade_return_receipt_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	refunddom "narratives/internal/domain/refund"
	tradedom "narratives/internal/domain/trade"
)

type receiveTradeReturnResponse struct {
	Data receiveTradeReturnResultResponse `json:"data"`
}

type receiveTradeReturnResultResponse struct {
	FinanciallyCompleted bool `json:"financiallyCompleted"`
	ReturnCompleted      bool `json:"returnCompleted"`
	NotificationEnsured  bool `json:"notificationEnsured"`
	AlreadyCompleted     bool `json:"alreadyCompleted"`
}

// POST /mall/me/trades/{tradeId}/receive-return
//
// SellerAvatarID is never accepted from the client. The authenticated Avatar
// from AvatarContextMiddleware is authoritative.
//
// This endpoint does not accept refund conditions in the request body.
//
// The accepted ReturnProposal is authoritative for:
//
//   - whether physical return is required
//   - merchandise refund amount
//
// The usecase resolves Trade, Order, Order item, ReturnAgreement,
// ReturnProposal and ReturnShipment from authoritative persisted state.
//
// The seller's explicit receipt confirmation is the authoritative local event
// while AMOL does not recognize carrier shipment notifications.
//
// Current flow:
//
//	agreed
//	-> seller confirms physical receipt
//	-> return_received
//	-> refund_processing
//	-> completed
//
// The frontend must never provide Order ID, item index, proposal ID, shipment
// identity, refund amount, tax amount, shipping amount or seller identity.
func (h *TradeHandler) receiveReturn(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	tradeID = strings.TrimSpace(tradeID)
	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}

	if h == nil || h.returnReceiptUC == nil {
		internalError(w, "resale trade return receipt usecase is nil")
		return
	}

	result, err := h.returnReceiptUC.ReceiveReturn(
		r.Context(),
		usecase.ReceiveResaleTradeReturnInput{
			TradeID:        tradeID,
			SellerAvatarID: avatarID,
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
			ReturnCompleted:      result.ReturnCompleted,
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
		errors.Is(err, tradedom.ErrReturnAgreementNotFound),
		errors.Is(err, tradedom.ErrReturnProposalNotFound),
		errors.Is(err, tradedom.ErrReturnShipmentNotFound),
		errors.Is(err, usecase.ErrResaleTradeReturnReceiptTradeMismatch):
		notFound(w)

	case errors.Is(
		err,
		usecase.ErrResaleTradeReturnReceiptInvalidSeller,
	):
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "avatar context is required",
		})

	case errors.Is(err, tradedom.ErrInvalidID),
		errors.Is(err, tradedom.ErrInvalidReturnRequirement),
		errors.Is(err, tradedom.ErrInvalidReturnRefundAmount),
		errors.Is(err, refunddom.ErrInvalidReturnRefundAmount),
		errors.Is(err, refunddom.ErrInvalidReturnRefundAmounts):
		badRequest(w, err.Error())

	case errors.Is(
		err,
		usecase.ErrResaleTradeReturnReceiptOrderNotPaid,
	),
		errors.Is(
			err,
			usecase.ErrResaleTradeReturnReceiptNotEligible,
		),
		errors.Is(
			err,
			usecase.ErrResaleTradeReturnReceiptAgreementNotReady,
		),
		errors.Is(
			err,
			usecase.ErrResaleTradeReturnReceiptPhysicalReturnNotRequired,
		),
		errors.Is(
			err,
			usecase.ErrResaleTradeReturnReceiptDisputed,
		),
		errors.Is(
			err,
			usecase.ErrResaleTradeReturnReceiptShipmentNotReady,
		),
		errors.Is(
			err,
			usecase.ErrResaleTradeReturnReceiptRefundMismatch,
		),
		errors.Is(
			err,
			usecase.ErrResaleTradeReturnReceiptAgreementCompletionMismatch,
		),
		errors.Is(err, tradedom.ErrReturnAgreementConflict),
		errors.Is(err, tradedom.ErrReturnShipmentConflict),
		errors.Is(err, tradedom.ErrReturnReceiptNotAllowed),
		errors.Is(err, tradedom.ErrReturnRefundProcessingNotAllowed),
		errors.Is(err, tradedom.ErrReturnCompletionNotAllowed):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(
		err,
		usecase.ErrResaleTradeReturnReceiptNotConfigured,
	):
		internalError(w, err.Error())

	default:
		writeOrderErr(w, err)
	}
}
