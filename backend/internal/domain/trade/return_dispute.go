// backend/internal/domain/trade/return_dispute.go

package trade

import (
	"errors"
	"time"
)

var (
	ErrInvalidReturnDisputedAt = errors.New(
		"trade: invalid return disputedAt",
	)
	ErrReturnDisputeNotAllowed = errors.New(
		"trade: return dispute is not allowed",
	)
	ErrReturnDiscussionResumeNotAllowed = errors.New(
		"trade: return discussion resume is not allowed",
	)
)

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

// ResumeDiscussion returns an Admin-reviewed return dispute to the normal
// buyer/seller negotiation flow.
//
// The current dispute workflow is opened only after the seller declined the
// buyer's return request. Therefore only a disputed agreement whose latest
// seller proposal is "disagree" may be resumed.
//
// Resuming clears DisputedAt because the aggregate is no longer under platform
// adjudication.
func (a *ReturnAgreement) ResumeDiscussion(at time.Time) error {
	if a == nil ||
		a.Status != ReturnStatusDisputed ||
		a.DisputedAt == nil ||
		a.DisputedAt.IsZero() ||
		a.Proposal == nil ||
		a.Proposal.Agreement != ReturnProposalAgreementDisagree {
		return ErrReturnDiscussionResumeNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnAgreementUpdatedAt
	}

	at = at.UTC()
	if err := a.validateTransitionTime(at); err != nil ||
		at.Before(*a.DisputedAt) {
		return ErrInvalidReturnAgreementUpdatedAt
	}

	a.Status = ReturnStatusDiscussing
	a.DisputedAt = nil
	a.UpdatedAt = at
	return nil
}
