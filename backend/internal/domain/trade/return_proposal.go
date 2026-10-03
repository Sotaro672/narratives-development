// backend/internal/domain/trade/return_proposal.go

package trade

import (
	"errors"
	"strings"
	"time"
)

type ReturnProposalAgreement string
type ReturnRequirement string

const (
	ReturnProposalAgreementAgree    ReturnProposalAgreement = "agree"
	ReturnProposalAgreementDisagree ReturnProposalAgreement = "disagree"
)

const (
	ReturnRequirementRequired    ReturnRequirement = "required"
	ReturnRequirementNotRequired ReturnRequirement = "not_required"
)

const (
	MaxReturnProposalReasonLength = 5000
)

var (
	ErrInvalidReturnProposalID = errors.New(
		"trade: invalid return proposal id",
	)
	ErrInvalidReturnProposalAgreement = errors.New(
		"trade: invalid return proposal agreement",
	)
	ErrInvalidReturnProposalReason = errors.New(
		"trade: invalid return proposal reason",
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
)

// ReturnProposal is the seller's latest response to the buyer's return
// consultation.
//
// Reason is required for both agreement and disagreement.
//
// When Agreement is disagree:
//   - ReturnRequirement must be empty.
//   - RefundAmount must be 0.
//   - ReturnAgreement remains discussing.
//
// When Agreement is agree:
//   - ReturnRequirement must be required or not_required.
//   - RefundAmount must be greater than 0.
//   - ReturnAgreement moves to proposed.
//
// RejectedAt is set when the buyer rejects an agreed seller proposal. If the
// buyer later changes the decision and accepts the same latest proposal,
// RejectedAt is cleared.
type ReturnProposal struct {
	ID string `json:"id"`

	Agreement         ReturnProposalAgreement `json:"agreement"`
	Reason            string                  `json:"reason"`
	ReturnRequirement ReturnRequirement       `json:"returnRequirement,omitempty"`
	RefundAmount      int                     `json:"refundAmount,omitempty"`

	CreatedAt  time.Time  `json:"createdAt"`
	RejectedAt *time.Time `json:"rejectedAt,omitempty"`
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
	reason string,
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
		Reason:            strings.TrimSpace(reason),
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
//
// The buyer may also change a previous rejection of the same latest proposal.
// In that case the aggregate is discussing with RejectedAt set. Acceptance
// clears RejectedAt and moves the aggregate to agreed.
func (a *ReturnAgreement) AcceptProposal(at time.Time) error {
	if a == nil || a.Proposal == nil {
		return ErrReturnProposalCannotBeAccepted
	}
	if a.Proposal.Agreement != ReturnProposalAgreementAgree {
		return ErrReturnProposalCannotBeAccepted
	}

	pendingProposal :=
		a.Status == ReturnStatusProposed &&
			a.Proposal.RejectedAt == nil

	previouslyRejectedProposal :=
		a.Status == ReturnStatusDiscussing &&
			a.Proposal.RejectedAt != nil

	if !pendingProposal && !previouslyRejectedProposal {
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

	a.Proposal.RejectedAt = nil
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

func (p ReturnProposal) ValidateForCreate() error {
	if p.ID != "" && !isValidReferenceID(p.ID) {
		return ErrInvalidReturnProposalID
	}
	if !IsValidReturnProposalAgreement(p.Agreement) {
		return ErrInvalidReturnProposalAgreement
	}

	reason := strings.TrimSpace(p.Reason)
	if reason == "" ||
		len([]rune(reason)) > MaxReturnProposalReasonLength {
		return ErrInvalidReturnProposalReason
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

	reason := strings.TrimSpace(p.Reason)
	if reason == "" ||
		len([]rune(reason)) > MaxReturnProposalReasonLength {
		return ErrInvalidReturnProposalReason
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
