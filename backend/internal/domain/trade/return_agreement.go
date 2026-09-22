// backend/internal/domain/trade/return_agreement.go

package trade

import (
	"errors"
	"strings"
	"time"
)

// ============================================================
// Types
// ============================================================

type ReturnStatus string
type ReturnConsultationReason string
type ReturnProposalAgreement string
type ReturnRequirement string

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

const (
	ReturnConsultationReasonNotAsDescribed ReturnConsultationReason = "not_as_described"
	ReturnConsultationReasonDamaged        ReturnConsultationReason = "damaged"
	ReturnConsultationReasonWrongItem      ReturnConsultationReason = "wrong_item"
	ReturnConsultationReasonOther          ReturnConsultationReason = "other"
)

const (
	ReturnProposalAgreementAgree    ReturnProposalAgreement = "agree"
	ReturnProposalAgreementDisagree ReturnProposalAgreement = "disagree"
)

const (
	ReturnRequirementRequired    ReturnRequirement = "required"
	ReturnRequirementNotRequired ReturnRequirement = "not_required"
)

// ============================================================
// Policy
// ============================================================

const (
	MaxReturnConsultationDetailLength = 5000
)

// ============================================================
// Errors
// ============================================================

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
	ErrInvalidReturnConsultationID = errors.New(
		"trade: invalid return consultation id",
	)
	ErrInvalidReturnConsultationReason = errors.New(
		"trade: invalid return consultation reason",
	)
	ErrInvalidReturnConsultationDetail = errors.New(
		"trade: invalid return consultation detail",
	)
	ErrInvalidReturnConsultationCreatedAt = errors.New(
		"trade: invalid return consultation createdAt",
	)
	ErrInvalidReturnProposalID = errors.New(
		"trade: invalid return proposal id",
	)
	ErrInvalidReturnProposalAgreement = errors.New(
		"trade: invalid return proposal agreement",
	)
	ErrInvalidReturnRequirement = errors.New(
		"trade: invalid return requirement",
	)
	ErrInvalidReturnRefundAmount = errors.New(
		"trade: invalid return refund amount",
	)
	ErrInvalidReturnProposalCreatedAt = errors.New(
		"trade: invalid return proposal createdAt",
	)
	ErrInvalidReturnProposalRejectedAt = errors.New(
		"trade: invalid return proposal rejectedAt",
	)
	ErrInvalidReturnAgreedAt = errors.New(
		"trade: invalid return agreedAt",
	)
	ErrInvalidReturnShippedAt = errors.New(
		"trade: invalid return shippedAt",
	)
	ErrInvalidReturnReceivedAt = errors.New(
		"trade: invalid return receivedAt",
	)
	ErrInvalidReturnRefundProcessingAt = errors.New(
		"trade: invalid return refundProcessingAt",
	)
	ErrInvalidReturnCompletedAt = errors.New(
		"trade: invalid return completedAt",
	)
	ErrInvalidReturnDisputedAt = errors.New(
		"trade: invalid return disputedAt",
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
	ErrReturnProposalNotAllowed = errors.New(
		"trade: return proposal is not allowed",
	)
	ErrReturnProposalNotFound = errors.New(
		"trade: return proposal not found",
	)
	ErrReturnProposalCannotBeAccepted = errors.New(
		"trade: return proposal cannot be accepted",
	)
	ErrReturnProposalCannotBeRejected = errors.New(
		"trade: return proposal cannot be rejected",
	)
	ErrReturnShipmentNotAllowed = errors.New(
		"trade: return shipment is not allowed",
	)
	ErrReturnReceiptNotAllowed = errors.New(
		"trade: return receipt is not allowed",
	)
	ErrReturnRefundProcessingNotAllowed = errors.New(
		"trade: return refund processing is not allowed",
	)
	ErrReturnCompletionNotAllowed = errors.New(
		"trade: return completion is not allowed",
	)
	ErrReturnDisputeNotAllowed = errors.New(
		"trade: return dispute is not allowed",
	)
)

// ============================================================
// Return Agreement
// ============================================================

// ReturnAgreement represents the negotiation and agreement state for one
// Avatar-to-Avatar Resale Trade return.
//
// The aggregate is created when the buyer starts a return consultation.
// Absence of this aggregate means ReturnStatusNone.
//
// Order remains authoritative for the transaction itself. ReturnAgreement owns
// only the negotiation and mutually agreed return/refund conditions.
//
// Typical flow when carrier shipment state is available:
//
//	discussing
//	  -> proposed
//	  -> agreed
//	  -> return_shipped
//	  -> return_received
//	  -> refund_processing
//	  -> completed
//
// Current flow without carrier-state recognition may use the seller's explicit
// receipt confirmation instead:
//
//	agreed -> return_received -> refund_processing -> completed
//
// If the seller rejects the requested return, the aggregate remains discussing.
// If the buyer rejects a seller proposal, it also returns to discussing.
//
// A return that does not require the physical item to be sent back skips:
//
//	return_shipped
//	return_received
//
// and proceeds:
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

// ReturnConsultation is the buyer's initial reason for starting a return
// discussion.
type ReturnConsultation struct {
	ID        string                   `json:"id"`
	Reason    ReturnConsultationReason `json:"reason"`
	Detail    string                   `json:"detail"`
	CreatedAt time.Time                `json:"createdAt"`
}

// ReturnProposal is the seller's latest response to the buyer's return
// consultation.
//
// When Agreement is disagree:
//   - ReturnRequirement must be empty.
//   - RefundAmount must be 0.
//   - ReturnAgreement remains discussing.
//
// When Agreement is agree:
//   - ReturnRequirement is required.
//   - RefundAmount must be greater than 0.
//   - ReturnAgreement moves to proposed.
//
// RejectedAt is set when the buyer rejects an agreed seller proposal.
type ReturnProposal struct {
	ID string `json:"id"`

	Agreement         ReturnProposalAgreement `json:"agreement"`
	ReturnRequirement ReturnRequirement       `json:"returnRequirement,omitempty"`
	RefundAmount      int                     `json:"refundAmount,omitempty"`

	CreatedAt  time.Time  `json:"createdAt"`
	RejectedAt *time.Time `json:"rejectedAt,omitempty"`
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

// Propose records the seller's latest answer to the return consultation.
//
// A seller disagreement does not terminate the negotiation. The status remains
// discussing so both parties may continue communicating or escalate the Trade.
//
// A seller agreement creates a proposal that must later be explicitly accepted
// by the buyer.
func (a *ReturnAgreement) Propose(
	proposalID string,
	agreement ReturnProposalAgreement,
	returnRequirement ReturnRequirement,
	refundAmount int,
	at time.Time,
) error {
	if a == nil {
		return ErrReturnProposalNotAllowed
	}
	if a.Status != ReturnStatusDiscussing {
		return ErrReturnProposalNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnProposalCreatedAt
	}
	if !IsValidReturnProposalAgreement(agreement) {
		return ErrInvalidReturnProposalAgreement
	}

	proposal := ReturnProposal{
		ID:                strings.TrimSpace(proposalID),
		Agreement:         agreement,
		ReturnRequirement: returnRequirement,
		RefundAmount:      refundAmount,
		CreatedAt:         at.UTC(),
	}

	if err := proposal.ValidateForCreate(); err != nil {
		return err
	}
	if err := a.validateTransitionTime(proposal.CreatedAt); err != nil {
		return ErrInvalidReturnProposalCreatedAt
	}

	a.Proposal = &proposal

	switch agreement {
	case ReturnProposalAgreementAgree:
		a.Status = ReturnStatusProposed

	case ReturnProposalAgreementDisagree:
		a.Status = ReturnStatusDiscussing
	}

	a.UpdatedAt = proposal.CreatedAt
	return nil
}

// AcceptProposal records explicit buyer acceptance of the latest seller
// proposal and locks the proposal as the agreed return conditions.
func (a *ReturnAgreement) AcceptProposal(at time.Time) error {
	if a == nil ||
		a.Status != ReturnStatusProposed ||
		a.Proposal == nil {
		return ErrReturnProposalCannotBeAccepted
	}
	if a.Proposal.Agreement != ReturnProposalAgreementAgree ||
		a.Proposal.RejectedAt != nil {
		return ErrReturnProposalCannotBeAccepted
	}
	if at.IsZero() {
		return ErrInvalidReturnAgreedAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil ||
		at.Before(a.Proposal.CreatedAt) {
		return ErrInvalidReturnAgreedAt
	}

	a.Status = ReturnStatusAgreed
	a.AgreedAt = cloneReturnTimePtr(&at)
	a.UpdatedAt = at
	return nil
}

// RejectProposal records buyer rejection of the current seller proposal and
// returns the negotiation to discussing.
func (a *ReturnAgreement) RejectProposal(at time.Time) error {
	if a == nil ||
		a.Status != ReturnStatusProposed ||
		a.Proposal == nil {
		return ErrReturnProposalCannotBeRejected
	}
	if a.Proposal.Agreement != ReturnProposalAgreementAgree ||
		a.Proposal.RejectedAt != nil {
		return ErrReturnProposalCannotBeRejected
	}
	if at.IsZero() {
		return ErrInvalidReturnProposalRejectedAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil ||
		at.Before(a.Proposal.CreatedAt) {
		return ErrInvalidReturnProposalRejectedAt
	}

	a.Proposal.RejectedAt = cloneReturnTimePtr(&at)
	a.Status = ReturnStatusDiscussing
	a.UpdatedAt = at
	return nil
}

// MarkReturnShipped records that the buyer handed the return shipment to the
// carrier.
//
// Only an agreed proposal requiring physical return may enter this state.
func (a *ReturnAgreement) MarkReturnShipped(at time.Time) error {
	if a == nil ||
		a.Status != ReturnStatusAgreed ||
		a.Proposal == nil ||
		a.Proposal.Agreement != ReturnProposalAgreementAgree ||
		a.Proposal.ReturnRequirement != ReturnRequirementRequired {
		return ErrReturnShipmentNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnShippedAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil ||
		a.AgreedAt == nil ||
		at.Before(*a.AgreedAt) {
		return ErrInvalidReturnShippedAt
	}

	a.Status = ReturnStatusReturnShipped
	a.ReturnShippedAt = cloneReturnTimePtr(&at)
	a.UpdatedAt = at
	return nil
}

// MarkReturnReceived records physical receipt of a required returned item.
func (a *ReturnAgreement) MarkReturnReceived(at time.Time) error {
	if a == nil ||
		a.Status != ReturnStatusReturnShipped ||
		a.Proposal == nil ||
		a.Proposal.ReturnRequirement != ReturnRequirementRequired ||
		a.ReturnShippedAt == nil {
		return ErrReturnReceiptNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnReceivedAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil ||
		at.Before(*a.ReturnShippedAt) {
		return ErrInvalidReturnReceivedAt
	}

	a.Status = ReturnStatusReturnReceived
	a.ReturnReceivedAt = cloneReturnTimePtr(&at)
	a.UpdatedAt = at
	return nil
}

// MarkReturnReceivedBySeller records the seller's explicit confirmation that a
// physically required returned item has been received.
//
// This transition is used while AMOL does not recognize carrier shipment
// notifications. It allows an agreed physical return to move directly from
// agreed to return_received without fabricating return_shipped.
//
// ReturnShippedAt intentionally remains nil. Future carrier integration may use
// MarkReturnShipped followed by MarkReturnReceived instead.
func (a *ReturnAgreement) MarkReturnReceivedBySeller(at time.Time) error {
	if a == nil ||
		a.Status != ReturnStatusAgreed ||
		a.Proposal == nil ||
		a.Proposal.Agreement != ReturnProposalAgreementAgree ||
		a.Proposal.ReturnRequirement != ReturnRequirementRequired ||
		a.AgreedAt == nil {
		return ErrReturnReceiptNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnReceivedAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil ||
		at.Before(*a.AgreedAt) {
		return ErrInvalidReturnReceivedAt
	}

	a.Status = ReturnStatusReturnReceived
	a.ReturnReceivedAt = cloneReturnTimePtr(&at)
	a.UpdatedAt = at
	return nil
}

// MarkRefundProcessing starts financial refund processing using the immutable
// conditions accepted by the buyer.
//
// Physical return required:
//
//	return_received -> refund_processing
//
// Physical return not required:
//
//	agreed -> refund_processing
func (a *ReturnAgreement) MarkRefundProcessing(at time.Time) error {
	if a == nil ||
		a.Proposal == nil ||
		a.Proposal.Agreement != ReturnProposalAgreementAgree {
		return ErrReturnRefundProcessingNotAllowed
	}

	switch a.Proposal.ReturnRequirement {
	case ReturnRequirementRequired:
		if a.Status != ReturnStatusReturnReceived ||
			a.ReturnReceivedAt == nil {
			return ErrReturnRefundProcessingNotAllowed
		}

	case ReturnRequirementNotRequired:
		if a.Status != ReturnStatusAgreed ||
			a.AgreedAt == nil {
			return ErrReturnRefundProcessingNotAllowed
		}

	default:
		return ErrReturnRefundProcessingNotAllowed
	}

	if at.IsZero() {
		return ErrInvalidReturnRefundProcessingAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil {
		return ErrInvalidReturnRefundProcessingAt
	}

	a.Status = ReturnStatusRefundProcessing
	a.RefundProcessingAt = cloneReturnTimePtr(&at)
	a.UpdatedAt = at
	return nil
}

// Complete marks the agreed return/refund flow as fully completed.
func (a *ReturnAgreement) Complete(at time.Time) error {
	if a == nil ||
		a.Status != ReturnStatusRefundProcessing ||
		a.RefundProcessingAt == nil {
		return ErrReturnCompletionNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnCompletedAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil ||
		at.Before(*a.RefundProcessingAt) {
		return ErrInvalidReturnCompletedAt
	}

	a.Status = ReturnStatusCompleted
	a.CompletedAt = cloneReturnTimePtr(&at)
	a.UpdatedAt = at
	return nil
}

// MarkDisputed moves an unresolved return flow under platform adjudication.
//
// Completed returns cannot be disputed through this transition. Dispute
// resolution itself belongs to the dispute/admin workflow rather than this
// aggregate.
func (a *ReturnAgreement) MarkDisputed(at time.Time) error {
	if a == nil ||
		a.Status == ReturnStatusNone ||
		a.Status == ReturnStatusCompleted ||
		a.Status == ReturnStatusDisputed {
		return ErrReturnDisputeNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnDisputedAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil {
		return ErrInvalidReturnDisputedAt
	}

	a.Status = ReturnStatusDisputed
	a.DisputedAt = cloneReturnTimePtr(&at)
	a.UpdatedAt = at
	return nil
}

func (a ReturnAgreement) ValidateForCreate() error {
	if a.ID != "" && !isValidReferenceID(a.ID) {
		return ErrInvalidReturnAgreementID
	}
	if !isValidReferenceID(a.TradeID) {
		return ErrInvalidReturnAgreementTradeID
	}
	if a.Status != ReturnStatusDiscussing {
		return ErrInvalidReturnAgreementState
	}
	if err := a.Consultation.ValidateForCreate(); err != nil {
		return err
	}
	if a.Proposal != nil ||
		a.AgreedAt != nil ||
		a.ReturnShippedAt != nil ||
		a.ReturnReceivedAt != nil ||
		a.RefundProcessingAt != nil ||
		a.CompletedAt != nil ||
		a.DisputedAt != nil {
		return ErrInvalidReturnAgreementState
	}
	if !a.CreatedAt.IsZero() &&
		!a.UpdatedAt.IsZero() &&
		a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidReturnAgreementUpdatedAt
	}

	return nil
}

func (a ReturnAgreement) ValidateForPersist() error {
	if !isValidReferenceID(a.ID) {
		return ErrInvalidReturnAgreementID
	}
	if !isValidReferenceID(a.TradeID) {
		return ErrInvalidReturnAgreementTradeID
	}
	if !IsValidReturnStatus(a.Status) ||
		a.Status == ReturnStatusNone {
		return ErrInvalidReturnStatus
	}
	if a.CreatedAt.IsZero() {
		return ErrInvalidReturnAgreementCreatedAt
	}
	if a.UpdatedAt.IsZero() ||
		a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidReturnAgreementUpdatedAt
	}
	if err := a.Consultation.ValidateForPersist(); err != nil {
		return err
	}
	if a.Consultation.CreatedAt.Before(a.CreatedAt) ||
		a.Consultation.CreatedAt.After(a.UpdatedAt) {
		return ErrInvalidReturnConsultationCreatedAt
	}
	if a.Proposal != nil {
		if err := a.Proposal.ValidateForPersist(); err != nil {
			return err
		}
		if a.Proposal.CreatedAt.Before(a.Consultation.CreatedAt) ||
			a.Proposal.CreatedAt.After(a.UpdatedAt) {
			return ErrInvalidReturnProposalCreatedAt
		}
	}

	if err := a.validatePersistedState(); err != nil {
		return err
	}

	return nil
}

func (c ReturnConsultation) ValidateForCreate() error {
	if c.ID != "" && !isValidReferenceID(c.ID) {
		return ErrInvalidReturnConsultationID
	}
	if !IsValidReturnConsultationReason(c.Reason) {
		return ErrInvalidReturnConsultationReason
	}

	detail := strings.TrimSpace(c.Detail)
	if detail == "" ||
		len([]rune(detail)) > MaxReturnConsultationDetailLength {
		return ErrInvalidReturnConsultationDetail
	}

	return nil
}

func (c ReturnConsultation) ValidateForPersist() error {
	if !isValidReferenceID(c.ID) {
		return ErrInvalidReturnConsultationID
	}
	if err := c.ValidateForCreate(); err != nil {
		return err
	}
	if c.CreatedAt.IsZero() {
		return ErrInvalidReturnConsultationCreatedAt
	}

	return nil
}

func (p ReturnProposal) ValidateForCreate() error {
	if p.ID != "" && !isValidReferenceID(p.ID) {
		return ErrInvalidReturnProposalID
	}
	if !IsValidReturnProposalAgreement(p.Agreement) {
		return ErrInvalidReturnProposalAgreement
	}

	switch p.Agreement {
	case ReturnProposalAgreementAgree:
		if !IsValidReturnRequirement(p.ReturnRequirement) {
			return ErrInvalidReturnRequirement
		}
		if p.RefundAmount <= 0 {
			return ErrInvalidReturnRefundAmount
		}

	case ReturnProposalAgreementDisagree:
		if p.ReturnRequirement != "" {
			return ErrInvalidReturnRequirement
		}
		if p.RefundAmount != 0 {
			return ErrInvalidReturnRefundAmount
		}
	}

	if p.RejectedAt != nil {
		return ErrInvalidReturnProposalRejectedAt
	}

	return nil
}

func (p ReturnProposal) ValidateForPersist() error {
	if !isValidReferenceID(p.ID) {
		return ErrInvalidReturnProposalID
	}
	if !IsValidReturnProposalAgreement(p.Agreement) {
		return ErrInvalidReturnProposalAgreement
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidReturnProposalCreatedAt
	}

	switch p.Agreement {
	case ReturnProposalAgreementAgree:
		if !IsValidReturnRequirement(p.ReturnRequirement) {
			return ErrInvalidReturnRequirement
		}
		if p.RefundAmount <= 0 {
			return ErrInvalidReturnRefundAmount
		}

	case ReturnProposalAgreementDisagree:
		if p.ReturnRequirement != "" {
			return ErrInvalidReturnRequirement
		}
		if p.RefundAmount != 0 {
			return ErrInvalidReturnRefundAmount
		}
		if p.RejectedAt != nil {
			return ErrInvalidReturnProposalRejectedAt
		}
	}

	if p.RejectedAt != nil {
		if p.RejectedAt.IsZero() ||
			p.RejectedAt.Before(p.CreatedAt) {
			return ErrInvalidReturnProposalRejectedAt
		}
	}

	return nil
}

func (a ReturnAgreement) validatePersistedState() error {
	if a.Status == ReturnStatusDiscussing {
		if a.AgreedAt != nil ||
			a.ReturnShippedAt != nil ||
			a.ReturnReceivedAt != nil ||
			a.RefundProcessingAt != nil ||
			a.CompletedAt != nil {
			return ErrInvalidReturnAgreementState
		}

		if a.Proposal != nil &&
			a.Proposal.Agreement == ReturnProposalAgreementAgree &&
			a.Proposal.RejectedAt == nil {
			return ErrInvalidReturnAgreementState
		}

		return nil
	}

	if a.Status == ReturnStatusDisputed {
		if a.DisputedAt == nil ||
			a.DisputedAt.IsZero() {
			return ErrInvalidReturnDisputedAt
		}

		return nil
	}

	if a.DisputedAt != nil {
		return ErrInvalidReturnAgreementState
	}

	if a.Proposal == nil ||
		a.Proposal.Agreement != ReturnProposalAgreementAgree ||
		a.Proposal.RejectedAt != nil {
		return ErrInvalidReturnAgreementState
	}

	if a.Status == ReturnStatusProposed {
		if a.AgreedAt != nil ||
			a.ReturnShippedAt != nil ||
			a.ReturnReceivedAt != nil ||
			a.RefundProcessingAt != nil ||
			a.CompletedAt != nil {
			return ErrInvalidReturnAgreementState
		}

		return nil
	}

	if a.AgreedAt == nil ||
		a.AgreedAt.IsZero() ||
		a.AgreedAt.Before(a.Proposal.CreatedAt) {
		return ErrInvalidReturnAgreedAt
	}

	switch a.Status {
	case ReturnStatusAgreed:
		if a.ReturnShippedAt != nil ||
			a.ReturnReceivedAt != nil ||
			a.RefundProcessingAt != nil ||
			a.CompletedAt != nil {
			return ErrInvalidReturnAgreementState
		}

	case ReturnStatusReturnShipped:
		if a.Proposal.ReturnRequirement != ReturnRequirementRequired ||
			a.ReturnShippedAt == nil ||
			a.ReturnShippedAt.IsZero() ||
			a.ReturnShippedAt.Before(*a.AgreedAt) ||
			a.ReturnReceivedAt != nil ||
			a.RefundProcessingAt != nil ||
			a.CompletedAt != nil {
			return ErrInvalidReturnAgreementState
		}

	case ReturnStatusReturnReceived:
		if a.Proposal.ReturnRequirement != ReturnRequirementRequired ||
			a.ReturnReceivedAt == nil ||
			a.ReturnReceivedAt.IsZero() ||
			a.ReturnReceivedAt.Before(*a.AgreedAt) ||
			a.RefundProcessingAt != nil ||
			a.CompletedAt != nil {
			return ErrInvalidReturnAgreementState
		}

		if a.ReturnShippedAt != nil &&
			(a.ReturnShippedAt.IsZero() ||
				a.ReturnShippedAt.Before(*a.AgreedAt) ||
				a.ReturnReceivedAt.Before(*a.ReturnShippedAt)) {
			return ErrInvalidReturnAgreementState
		}

	case ReturnStatusRefundProcessing:
		if a.RefundProcessingAt == nil ||
			a.RefundProcessingAt.IsZero() ||
			a.CompletedAt != nil {
			return ErrInvalidReturnAgreementState
		}

		switch a.Proposal.ReturnRequirement {
		case ReturnRequirementRequired:
			if a.ReturnReceivedAt == nil ||
				a.ReturnReceivedAt.IsZero() ||
				a.ReturnReceivedAt.Before(*a.AgreedAt) ||
				a.RefundProcessingAt.Before(*a.ReturnReceivedAt) {
				return ErrInvalidReturnAgreementState
			}

			if a.ReturnShippedAt != nil &&
				(a.ReturnShippedAt.IsZero() ||
					a.ReturnShippedAt.Before(*a.AgreedAt) ||
					a.ReturnReceivedAt.Before(*a.ReturnShippedAt)) {
				return ErrInvalidReturnAgreementState
			}

		case ReturnRequirementNotRequired:
			if a.ReturnShippedAt != nil ||
				a.ReturnReceivedAt != nil ||
				a.RefundProcessingAt.Before(*a.AgreedAt) {
				return ErrInvalidReturnAgreementState
			}

		default:
			return ErrInvalidReturnRequirement
		}

	case ReturnStatusCompleted:
		if a.RefundProcessingAt == nil ||
			a.CompletedAt == nil ||
			a.RefundProcessingAt.IsZero() ||
			a.CompletedAt.IsZero() ||
			a.CompletedAt.Before(*a.RefundProcessingAt) {
			return ErrInvalidReturnAgreementState
		}

	default:
		return ErrInvalidReturnAgreementState
	}

	return nil
}

func (a ReturnAgreement) validateTransitionTime(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidReturnAgreementUpdatedAt
	}

	at = at.UTC()

	if !a.CreatedAt.IsZero() &&
		at.Before(a.CreatedAt) {
		return ErrInvalidReturnAgreementUpdatedAt
	}
	if !a.UpdatedAt.IsZero() &&
		at.Before(a.UpdatedAt) {
		return ErrInvalidReturnAgreementUpdatedAt
	}

	return nil
}

// ============================================================
// Validation helpers
// ============================================================

func IsValidReturnStatus(status ReturnStatus) bool {
	switch status {
	case ReturnStatusNone,
		ReturnStatusDiscussing,
		ReturnStatusProposed,
		ReturnStatusAgreed,
		ReturnStatusReturnShipped,
		ReturnStatusReturnReceived,
		ReturnStatusRefundProcessing,
		ReturnStatusCompleted,
		ReturnStatusDisputed:
		return true

	default:
		return false
	}
}

func IsValidReturnConsultationReason(
	reason ReturnConsultationReason,
) bool {
	switch reason {
	case ReturnConsultationReasonNotAsDescribed,
		ReturnConsultationReasonDamaged,
		ReturnConsultationReasonWrongItem,
		ReturnConsultationReasonOther:
		return true

	default:
		return false
	}
}

func IsValidReturnProposalAgreement(
	agreement ReturnProposalAgreement,
) bool {
	switch agreement {
	case ReturnProposalAgreementAgree,
		ReturnProposalAgreementDisagree:
		return true

	default:
		return false
	}
}

func IsValidReturnRequirement(
	requirement ReturnRequirement,
) bool {
	switch requirement {
	case ReturnRequirementRequired,
		ReturnRequirementNotRequired:
		return true

	default:
		return false
	}
}

func cloneReturnTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
