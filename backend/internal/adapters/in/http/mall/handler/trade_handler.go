// backend/internal/adapters/in/http/mall/handler/trade_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	tradedom "narratives/internal/domain/trade"
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
	query                    *mallquery.TradeQuery
	messageUC                *usecase.TradeMessageUsecase
	reportUC                 *usecase.ReportUsecase
	dispatchUC               *usecase.ResaleTradeDispatchUsecase
	returnConsultationUC     *usecase.ResaleTradeReturnConsultationUsecase
	returnProposalUC         *usecase.ResaleTradeReturnProposalUsecase
	returnProposalResponseUC *usecase.ResaleTradeReturnProposalResponseUsecase
	returnShipmentUC         *usecase.ResaleTradeReturnShipmentUsecase
	returnReceiptUC          *usecase.ResaleTradeReturnReceiptUsecase
}

func NewTradeHandler(
	query *mallquery.TradeQuery,
	messageUC *usecase.TradeMessageUsecase,
	reportUC *usecase.ReportUsecase,
	dispatchUC *usecase.ResaleTradeDispatchUsecase,
	returnConsultationUC *usecase.ResaleTradeReturnConsultationUsecase,
	returnProposalUC *usecase.ResaleTradeReturnProposalUsecase,
	returnProposalResponseUC *usecase.ResaleTradeReturnProposalResponseUsecase,
	returnShipmentUC *usecase.ResaleTradeReturnShipmentUsecase,
	returnReceiptUC *usecase.ResaleTradeReturnReceiptUsecase,
) http.Handler {
	return &TradeHandler{
		query:                    query,
		messageUC:                messageUC,
		reportUC:                 reportUC,
		dispatchUC:               dispatchUC,
		returnConsultationUC:     returnConsultationUC,
		returnProposalUC:         returnProposalUC,
		returnProposalResponseUC: returnProposalResponseUC,
		returnShipmentUC:         returnShipmentUC,
		returnReceiptUC:          returnReceiptUC,
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
//	POST /mall/me/trades/{tradeId}/reports
//	POST /mall/me/trades/{tradeId}/messages/{messageId}/reports
//	POST /mall/me/trades/{tradeId}/read
//	GET  /mall/me/trades/{tradeId}/unread-count
//	POST /mall/me/trades/{tradeId}/dispatch
//	POST /mall/me/trades/{tradeId}/return-consultations
//	POST /mall/me/trades/{tradeId}/return-proposals
//	POST /mall/me/trades/{tradeId}/return-proposals/{proposalId}/accept
//	POST /mall/me/trades/{tradeId}/return-proposals/{proposalId}/reject
//	POST /mall/me/trades/{tradeId}/return-shipment
//	GET  /mall/me/trades/{tradeId}/return-shipment
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

	if len(parts) == 4 &&
		parts[1] == "messages" &&
		parts[2] != "" &&
		parts[3] == "reports" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.reportMessage(w, r, tradeID, parts[2])
		return
	}

	if len(parts) == 4 &&
		parts[1] == "return-proposals" &&
		parts[2] != "" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		switch parts[3] {
		case "accept":
			h.acceptReturnProposal(w, r, tradeID, parts[2])
			return
		case "reject":
			h.rejectReturnProposal(w, r, tradeID, parts[2])
			return
		default:
			notFound(w)
			return
		}
	}

	if len(parts) != 2 || parts[1] == "" {
		notFound(w)
		return
	}

	switch parts[1] {
	case "reports":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.reportTrade(w, r, tradeID)

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

	case "return-consultations":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.createReturnConsultation(w, r, tradeID)

	case "return-proposals":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.createReturnProposal(w, r, tradeID)

	case "return-shipment":
		switch r.Method {
		case http.MethodPost:
			h.createReturnShipment(w, r, tradeID)
		case http.MethodGet:
			h.getReturnShipment(w, r, tradeID)
		default:
			methodNotAllowed(w)
		}

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

func (h *TradeHandler) reportTrade(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
) {
	if h == nil || h.reportUC == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"report service not configured",
		)
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	tradeID = strings.TrimSpace(tradeID)
	if tradeID == "" {
		badRequest(w, "tradeId is required")
		return
	}

	reason, detail, ok := decodeReportRequest(w, r)
	if !ok {
		return
	}

	result, err := h.reportUC.ReportTradeReturnDisputeByAvatar(
		r.Context(),
		usecase.ReportTradeReturnDisputeByAvatarInput{
			TradeID:  tradeID,
			AvatarID: avatarID,
			Reason:   reason,
			Detail:   detail,
		},
	)
	if err != nil {
		writeTradeReportErr(w, err)
		return
	}

	writeReportResult(w, result)
}

func (h *TradeHandler) acceptReturnProposal(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
	proposalID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	tradeID = strings.TrimSpace(tradeID)
	proposalID = strings.TrimSpace(proposalID)
	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}
	if proposalID == "" {
		badRequest(w, "invalid proposal id")
		return
	}

	if h == nil || h.returnProposalResponseUC == nil {
		internalError(w, "resale trade return proposal response usecase is nil")
		return
	}

	result, err := h.returnProposalResponseUC.Accept(
		r.Context(),
		usecase.RespondResaleTradeReturnProposalInput{
			TradeID:       tradeID,
			ProposalID:    proposalID,
			BuyerAvatarID: avatarID,
		},
	)
	if err != nil {
		writeTradeReturnProposalResponseErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result.Agreement,
	})
}

func (h *TradeHandler) rejectReturnProposal(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
	proposalID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	tradeID = strings.TrimSpace(tradeID)
	proposalID = strings.TrimSpace(proposalID)
	if tradeID == "" {
		badRequest(w, "invalid trade id")
		return
	}
	if proposalID == "" {
		badRequest(w, "invalid proposal id")
		return
	}

	if h == nil || h.returnProposalResponseUC == nil {
		internalError(w, "resale trade return proposal response usecase is nil")
		return
	}

	result, err := h.returnProposalResponseUC.Reject(
		r.Context(),
		usecase.RespondResaleTradeReturnProposalInput{
			TradeID:       tradeID,
			ProposalID:    proposalID,
			BuyerAvatarID: avatarID,
		},
	)
	if err != nil {
		writeTradeReturnProposalResponseErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result.Agreement,
	})
}

func (h *TradeHandler) createReturnShipment(
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

	if h == nil || h.returnShipmentUC == nil {
		internalError(w, "resale trade return shipment usecase is nil")
		return
	}

	result, err := h.returnShipmentUC.Create(
		r.Context(),
		usecase.CreateResaleTradeReturnShipmentInput{
			TradeID:       tradeID,
			BuyerAvatarID: avatarID,
		},
	)
	if err != nil {
		writeTradeReturnShipmentErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": buildTradeReturnShipmentResponse(result.Shipment),
	})
}

func (h *TradeHandler) getReturnShipment(
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

	if h == nil || h.returnShipmentUC == nil {
		internalError(w, "resale trade return shipment usecase is nil")
		return
	}

	result, err := h.returnShipmentUC.Get(
		r.Context(),
		usecase.GetResaleTradeReturnShipmentInput{
			TradeID:       tradeID,
			BuyerAvatarID: avatarID,
		},
	)
	if err != nil {
		writeTradeReturnShipmentErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": buildTradeReturnShipmentResponse(result.Shipment),
	})
}

func buildTradeReturnShipmentResponse(
	shipment tradedom.ReturnShipment,
) map[string]any {
	return map[string]any{
		"status":        shipment.Status,
		"dropOffMethod": shipment.DropOffMethod,
		"qrCodePayload": shipment.QRCodePayload,
		"readyAt":       shipment.ReadyAt,
	}
}

func writeTradeReportErr(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		internalError(w, "unknown error")

	case errors.Is(err, tradedom.ErrNotFound),
		errors.Is(err, tradedom.ErrReturnAgreementNotFound):
		notFound(w)

	case errors.Is(err, tradedom.ErrReturnDisputeNotAllowed),
		errors.Is(err, tradedom.ErrReturnAgreementConflict):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	default:
		writeReportError(w, err)
	}
}

func writeTradeReturnShipmentErr(
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
		errors.Is(err, usecase.ErrResaleTradeReturnShipmentTradeMismatch):
		notFound(w)

	case errors.Is(err, usecase.ErrResaleTradeReturnShipmentInvalidBuyer):
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "avatar context is required",
		})

	case errors.Is(err, tradedom.ErrInvalidID),
		errors.Is(err, tradedom.ErrInvalidReturnProposalID),
		errors.Is(err, tradedom.ErrInvalidReturnRequirement),
		errors.Is(err, tradedom.ErrInvalidReturnRefundAmount):
		badRequest(w, err.Error())

	case errors.Is(err, tradedom.ErrTradeAlreadyClosed),
		errors.Is(err, tradedom.ErrConflict),
		errors.Is(err, tradedom.ErrReturnAgreementConflict),
		errors.Is(err, tradedom.ErrReturnShipmentConflict),
		errors.Is(err, tradedom.ErrReturnShipmentAlreadyExists),
		errors.Is(err, usecase.ErrResaleTradeReturnShipmentOrderNotPaid),
		errors.Is(err, usecase.ErrResaleTradeReturnShipmentNotEligible),
		errors.Is(err, usecase.ErrResaleTradeReturnShipmentAgreementNotReady),
		errors.Is(err, usecase.ErrResaleTradeReturnShipmentPhysicalReturnNotRequired),
		errors.Is(err, usecase.ErrResaleTradeReturnShipmentDisputed),
		errors.Is(err, usecase.ErrResaleTradeReturnShipmentStateConflict):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(err, usecase.ErrResaleTradeReturnShipmentNotConfigured):
		internalError(w, err.Error())

	default:
		writeOrderErr(w, err)
	}
}

func writeTradeReturnProposalResponseErr(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		return

	case errors.Is(err, tradedom.ErrNotFound),
		errors.Is(err, tradedom.ErrReturnAgreementNotFound),
		errors.Is(err, tradedom.ErrReturnProposalNotFound),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalResponseTradeMismatch):
		notFound(w)

	case errors.Is(err, usecase.ErrResaleTradeReturnProposalResponseInvalidBuyer):
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "avatar context is required",
		})

	case errors.Is(err, tradedom.ErrInvalidID),
		errors.Is(err, tradedom.ErrInvalidReturnProposalID),
		errors.Is(err, tradedom.ErrInvalidReturnRequirement),
		errors.Is(err, tradedom.ErrInvalidReturnRefundAmount):
		badRequest(w, err.Error())

	case errors.Is(err, tradedom.ErrTradeAlreadyClosed),
		errors.Is(err, tradedom.ErrConflict),
		errors.Is(err, tradedom.ErrReturnAgreementConflict),
		errors.Is(err, tradedom.ErrReturnProposalCannotBeAccepted),
		errors.Is(err, tradedom.ErrReturnProposalCannotBeRejected),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalResponseOrderNotPaid),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalResponseNotEligible),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalResponseProposalMismatch),
		errors.Is(err, usecase.ErrResaleTradeReturnProposalResponseRefundAmountInvalid):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(err, usecase.ErrResaleTradeReturnProposalResponseNotConfigured):
		internalError(w, err.Error())

	default:
		writeOrderErr(w, err)
	}
}
