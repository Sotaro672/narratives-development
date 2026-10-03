// backend/internal/domain/trade/return_lifecycle.go

package trade

import (
	"errors"
	"time"
)

var (
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
)

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
