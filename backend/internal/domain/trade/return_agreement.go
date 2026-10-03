// backend/internal/domain/trade/return_agreement.go

package trade

import (
	"errors"
	"strings"
	"time"
)

type ReturnStatus string

const (
	ReturnStatusNone             ReturnStatus = "none"
	ReturnStatusDiscussing       ReturnStatus = "discussing"
	ReturnStatusProposed         ReturnStatus = "proposed"
	ReturnStatusAgreed           ReturnStatus = "agreed"
	ReturnStatusReturnShipped    ReturnStatus = "return_shipped"
	ReturnStatusReturnReceived   ReturnStatus = "return_received"
	ReturnStatusRefundProcessing ReturnStatus = "refund_processing"
	ReturnStatusCompleted        ReturnStatus = "completed"
	ReturnStatusDisputed         ReturnStatus = "disputed"
)

var (
	ErrInvalidReturnAgreementID = errors.New(
		"trade: invalid return agreement id",
	)
	ErrInvalidReturnAgreementTradeID = errors.New(
		"trade: invalid return agreement tradeId",
	)
	ErrInvalidReturnStatus = errors.New(
		"trade: invalid return status",
	)
	ErrInvalidReturnAgreementCreatedAt = errors.New(
		"trade: invalid return agreement createdAt",
	)
	ErrInvalidReturnAgreementUpdatedAt = errors.New(
		"trade: invalid return agreement updatedAt",
	)
	ErrInvalidReturnAgreementState = errors.New(
		"trade: invalid return agreement state",
	)
)

// ReturnAgreement represents the negotiation and agreement state for one
// Avatar-to-Avatar Resale Trade return.
//
// The aggregate is created when the buyer starts a return consultation.
// Absence of this aggregate means ReturnStatusNone.
//
// Order remains authoritative for the transaction itself. ReturnAgreement owns
// only the negotiation and mutually agreed return/refund conditions.
//
// Typical flow:
//
//	discussing
//	  -> proposed
//	  -> agreed
//	  -> return_shipped
//	  -> return_received
//	  -> refund_processing
//	  -> completed
//
// Current flow without carrier-state recognition may use:
//
//	agreed -> return_received -> refund_processing -> completed
//
// A return that does not require physical return skips shipment/receipt:
//
//	agreed -> refund_processing -> completed
//
// Firestore:
//
//	tradeReturnAgreements/{tradeId}
type ReturnAgreement struct {
	ID      string `json:"id"`
	TradeID string `json:"tradeId"`

	Status ReturnStatus `json:"status"`

	Consultation ReturnConsultation `json:"consultation"`
	Proposal     *ReturnProposal    `json:"proposal,omitempty"`

	AgreedAt           *time.Time `json:"agreedAt,omitempty"`
	ReturnShippedAt    *time.Time `json:"returnShippedAt,omitempty"`
	ReturnReceivedAt   *time.Time `json:"returnReceivedAt,omitempty"`
	RefundProcessingAt *time.Time `json:"refundProcessingAt,omitempty"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	DisputedAt         *time.Time `json:"disputedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (a ReturnAgreement) GetID() string {
	return a.ID
}

// NewReturnAgreementForCreate creates a return negotiation when the buyer opens
// a return consultation.
//
// ID, Consultation.ID and timestamps may be empty/zero before repository
// persistence. The repository may populate them.
func NewReturnAgreementForCreate(
	id string,
	tradeID string,
	consultationID string,
	reason ReturnConsultationReason,
	detail string,
) (ReturnAgreement, error) {
	agreement := ReturnAgreement{
		ID:      strings.TrimSpace(id),
		TradeID: strings.TrimSpace(tradeID),
		Status:  ReturnStatusDiscussing,
		Consultation: ReturnConsultation{
			ID:     strings.TrimSpace(consultationID),
			Reason: reason,
			Detail: strings.TrimSpace(detail),
		},
	}

	if err := agreement.ValidateForCreate(); err != nil {
		return ReturnAgreement{}, err
	}

	return agreement, nil
}
