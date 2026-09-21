// backend/internal/adapters/in/http/mall/handler/trade_return_consultation_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	tradedom "narratives/internal/domain/trade"
)

type createTradeReturnConsultationRequest struct {
	Reason tradedom.ReturnConsultationReason `json:"reason"`
	Detail string                            `json:"detail"`
}

// POST /mall/me/trades/{tradeId}/return-consultations
//
// Starts a return consultation for the authenticated buyer Avatar.
//
// Body:
//
//	{
//	  "reason": "not_as_described",
//	  "detail": "商品説明と実物の状態が異なります"
//	}
//
// BuyerAvatarID is never accepted from the client. The authenticated Avatar
// from AvatarContextMiddleware is authoritative. This endpoint does not mutate
// the legacy Order return-request fields; return negotiation state is persisted
// in ReturnAgreement.
func (h *TradeHandler) createReturnConsultation(
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

	if h == nil || h.returnConsultationUC == nil {
		internalError(w, "resale trade return consultation usecase is nil")
		return
	}

	var req createTradeReturnConsultationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}

	result, err := h.returnConsultationUC.Create(
		r.Context(),
		usecase.CreateResaleTradeReturnConsultationInput{
			TradeID:       tradeID,
			BuyerAvatarID: avatarID,
			Reason:        req.Reason,
			Detail:        req.Detail,
		},
	)
	if err != nil {
		writeTradeReturnConsultationErr(w, err)
		return
	}

	statusCode := http.StatusOK
	if result.Created {
		statusCode = http.StatusCreated
	}

	writeJSON(w, statusCode, map[string]any{
		"data": result.Agreement,
	})
}

func writeTradeReturnConsultationErr(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		return

	case errors.Is(err, tradedom.ErrNotFound),
		errors.Is(err, usecase.ErrResaleTradeReturnConsultationTradeMismatch):
		notFound(w)

	case errors.Is(err, usecase.ErrResaleTradeReturnConsultationInvalidBuyer):
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "avatar context is required",
		})

	case errors.Is(err, tradedom.ErrInvalidID),
		errors.Is(err, tradedom.ErrInvalidReturnConsultationReason),
		errors.Is(err, tradedom.ErrInvalidReturnConsultationDetail):
		badRequest(w, err.Error())

	case errors.Is(err, tradedom.ErrTradeAlreadyClosed),
		errors.Is(err, tradedom.ErrConflict),
		errors.Is(err, tradedom.ErrReturnAgreementAlreadyExists),
		errors.Is(err, tradedom.ErrReturnAgreementConflict),
		errors.Is(err, usecase.ErrResaleTradeReturnConsultationOrderNotPaid),
		errors.Is(err, usecase.ErrResaleTradeReturnConsultationNotEligible),
		errors.Is(err, usecase.ErrResaleTradeReturnConsultationAlreadyExists):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(),
		})

	case errors.Is(err, usecase.ErrResaleTradeReturnConsultationNotConfigured):
		internalError(w, err.Error())

	default:
		writeOrderErr(w, err)
	}
}
