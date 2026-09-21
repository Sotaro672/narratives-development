// backend/internal/adapters/in/http/mall/handler/trade_return_proposal_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	tradedom "narratives/internal/domain/trade"
)

type createTradeReturnProposalRequest struct {
	Agreement         tradedom.ReturnProposalAgreement `json:"agreement"`
	ReturnRequirement tradedom.ReturnRequirement       `json:"returnRequirement,omitempty"`
	RefundAmount      int                              `json:"refundAmount,omitempty"`
}

// POST /mall/me/trades/{tradeId}/return-proposals
//
// Records the authenticated seller Avatar's response to a buyer return
// consultation.
//
// Body when agreeing:
//
//	{
//	  "agreement": "agree",
//	  "returnRequirement": "required",
//	  "refundAmount": 5000
//	}
//
// Body when disagreeing:
//
//	{
//	  "agreement": "disagree"
//	}
//
// SellerAvatarID is never accepted from the client. The authenticated Avatar
// from AvatarContextMiddleware is authoritative.
//
// refundAmount is validated against the authoritative tax-inclusive merchandise
// refund maximum calculated from the persisted Order item. The client cannot
// supply or override that maximum.
//
// agreement=agree moves ReturnAgreement from discussing to proposed.
// agreement=disagree keeps ReturnAgreement in discussing so the parties can
// continue negotiating or escalate the Trade.
func (h *TradeHandler) createReturnProposal(
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

	if h == nil || h.returnProposalUC == nil {
		internalError(w, "resale trade return proposal usecase is nil")
		return
	}

	var req createTradeReturnProposalRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}

	result, err := h.returnProposalUC.Create(
		r.Context(),
		usecase.CreateResaleTradeReturnProposalInput{
			TradeID:           tradeID,
			SellerAvatarID:    avatarID,
			Agreement:         req.Agreement,
			ReturnRequirement: req.ReturnRequirement,
			RefundAmount:      req.RefundAmount,
		},
	)
	if err != nil {
		writeTradeReturnProposalErr(w, err)
		return
	}

	statusCode := http.StatusOK
	if result.Changed {
		statusCode = http.StatusCreated
	}

	writeJSON(w, statusCode, map[string]any{
		"data": result.Agreement,
	})
}

func writeTradeReturnProposalErr(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		return

	case errors.Is(err, tradedom.ErrNotFound),
		errors.Is(err, tradedom.ErrReturnAgreementNotFound),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalTradeMismatch):
		notFound(w)

	case errors.Is(err, usecase.ErrResaleTradeReturnProposalInvalidSeller):
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "avatar context is required",
		})

	case errors.Is(err, tradedom.ErrInvalidID),
		errors.Is(err, tradedom.ErrInvalidReturnProposalAgreement),
		errors.Is(err, tradedom.ErrInvalidReturnRequirement),
		errors.Is(err, tradedom.ErrInvalidReturnRefundAmount):
		badRequest(w, err.Error())

	case errors.Is(err, tradedom.ErrTradeAlreadyClosed),
		errors.Is(err, tradedom.ErrConflict),
		errors.Is(err, tradedom.ErrReturnAgreementConflict),
		errors.Is(err, tradedom.ErrReturnProposalNotAllowed),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalOrderNotPaid),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalNotEligible),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalRefundAmountExceedsMaximum):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(err, usecase.ErrResaleTradeReturnProposalNotConfigured):
		internalError(w, err.Error())

	default:
		writeOrderErr(w, err)
	}
}
