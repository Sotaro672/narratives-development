// backend/internal/domain/trade/return_agreement_validation.go

package trade

import "time"

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

func (a ReturnAgreement) validatePersistedState() error {
	if a.Status == ReturnStatusDiscussing {
		if a.AgreedAt != nil ||
			a.ReturnShippedAt != nil ||
			a.ReturnReceivedAt != nil ||
			a.RefundProcessingAt != nil ||
			a.CompletedAt != nil ||
			a.DisputedAt != nil {
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

func cloneReturnTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
