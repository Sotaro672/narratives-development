// backend/internal/domain/trade/information_request.go

package trade

import (
	"errors"
	"strings"
	"time"
)

type InformationRequestStatus string
type InformationRequestTargetSide string

const (
	InformationRequestStatusPending   InformationRequestStatus = "pending"
	InformationRequestStatusSubmitted InformationRequestStatus = "submitted"
)

const (
	InformationRequestTargetBuyer  InformationRequestTargetSide = "buyer"
	InformationRequestTargetSeller InformationRequestTargetSide = "seller"
)

const (
	MaxInformationRequestReasonLength   = 5000
	MaxInformationRequestResponseLength = 5000
)

var (
	ErrInvalidInformationRequestID = errors.New(
		"trade: invalid information request id",
	)
	ErrInvalidInformationRequestTradeID = errors.New(
		"trade: invalid information request tradeId",
	)
	ErrInvalidInformationRequestTargetSide = errors.New(
		"trade: invalid information request target side",
	)
	ErrInvalidInformationRequestTargetAvatarID = errors.New(
		"trade: invalid information request target avatarId",
	)
	ErrInvalidInformationRequestReason = errors.New(
		"trade: invalid information request reason",
	)
	ErrInvalidInformationRequestResponse = errors.New(
		"trade: invalid information request response",
	)
	ErrInvalidInformationRequestStatus = errors.New(
		"trade: invalid information request status",
	)
	ErrInvalidInformationRequestCreatedAt = errors.New(
		"trade: invalid information request createdAt",
	)
	ErrInvalidInformationRequestUpdatedAt = errors.New(
		"trade: invalid information request updatedAt",
	)
	ErrInvalidInformationRequestSubmittedAt = errors.New(
		"trade: invalid information request submittedAt",
	)
	ErrInformationRequestAlreadySubmitted = errors.New(
		"trade: information request already submitted",
	)
	ErrInformationRequestNotPending = errors.New(
		"trade: information request is not pending",
	)
)

// InformationRequest represents an information request created by AMOL Admin
// for one participant in a Trade.
//
// Admin selects only TargetSide. The application layer must resolve the actual
// TargetAvatarID from the authoritative Trade before creating this aggregate.
// The client must never be allowed to choose an arbitrary Avatar ID.
//
// Pending requests remain actionable in Mall until the target Avatar submits a
// response. Reading the request alone does not complete or dismiss it.
//
// Firestore:
//
//	tradeInformationRequests/{requestId}
type InformationRequest struct {
	ID      string `json:"id"`
	TradeID string `json:"tradeId"`

	TargetSide     InformationRequestTargetSide `json:"targetSide"`
	TargetAvatarID string                       `json:"targetAvatarId"`

	Reason   string `json:"reason"`
	Response string `json:"response,omitempty"`

	Status InformationRequestStatus `json:"status"`

	CreatedAt   time.Time  `json:"createdAt"`
	SubmittedAt *time.Time `json:"submittedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (r InformationRequest) GetID() string {
	return r.ID
}

// NewInformationRequestForCreate creates a pending Admin information request.
//
// ID and timestamps may be empty/zero before repository persistence. The
// application layer is responsible for resolving TargetAvatarID from Trade:
//   - buyer  -> Trade.BuyerAvatarID
//   - seller -> Trade.SellerAvatarID
func NewInformationRequestForCreate(
	id string,
	tradeID string,
	targetSide InformationRequestTargetSide,
	targetAvatarID string,
	reason string,
) (InformationRequest, error) {
	request := InformationRequest{
		ID:             strings.TrimSpace(id),
		TradeID:        strings.TrimSpace(tradeID),
		TargetSide:     targetSide,
		TargetAvatarID: strings.TrimSpace(targetAvatarID),
		Reason:         strings.TrimSpace(reason),
		Response:       "",
		Status:         InformationRequestStatusPending,
	}

	if err := request.ValidateForCreate(); err != nil {
		return InformationRequest{}, err
	}

	return request, nil
}

// Submit records the requested Avatar's response and completes the pending
// action.
//
// Authorization is not performed by the entity. The application layer must
// confirm that the authenticated Avatar matches TargetAvatarID before calling
// Submit.
func (r *InformationRequest) Submit(
	response string,
	at time.Time,
) error {
	if r == nil {
		return ErrInformationRequestNotPending
	}
	if r.Status == InformationRequestStatusSubmitted {
		return ErrInformationRequestAlreadySubmitted
	}
	if r.Status != InformationRequestStatusPending {
		return ErrInformationRequestNotPending
	}
	if at.IsZero() {
		return ErrInvalidInformationRequestSubmittedAt
	}

	response = strings.TrimSpace(response)
	if response == "" ||
		len([]rune(response)) > MaxInformationRequestResponseLength {
		return ErrInvalidInformationRequestResponse
	}

	at = at.UTC()
	if !r.CreatedAt.IsZero() && at.Before(r.CreatedAt.UTC()) {
		return ErrInvalidInformationRequestSubmittedAt
	}
	if !r.UpdatedAt.IsZero() && at.Before(r.UpdatedAt.UTC()) {
		return ErrInvalidInformationRequestSubmittedAt
	}

	r.Response = response
	r.Status = InformationRequestStatusSubmitted
	r.SubmittedAt = informationRequestTimePtr(at)
	r.UpdatedAt = at

	return nil
}

// ValidateForCreate validates a newly created information request before
// persistence.
//
// Status must be pending and response/submittedAt must still be empty.
func (r InformationRequest) ValidateForCreate() error {
	if r.ID != "" && !isValidReferenceID(r.ID) {
		return ErrInvalidInformationRequestID
	}
	if !isValidReferenceID(r.TradeID) {
		return ErrInvalidInformationRequestTradeID
	}
	if !IsValidInformationRequestTargetSide(r.TargetSide) {
		return ErrInvalidInformationRequestTargetSide
	}
	if !isValidReferenceID(r.TargetAvatarID) {
		return ErrInvalidInformationRequestTargetAvatarID
	}

	reason := strings.TrimSpace(r.Reason)
	if reason == "" ||
		len([]rune(reason)) > MaxInformationRequestReasonLength {
		return ErrInvalidInformationRequestReason
	}

	if r.Status != InformationRequestStatusPending {
		return ErrInvalidInformationRequestStatus
	}
	if strings.TrimSpace(r.Response) != "" {
		return ErrInvalidInformationRequestResponse
	}
	if r.SubmittedAt != nil {
		return ErrInvalidInformationRequestSubmittedAt
	}

	if !r.CreatedAt.IsZero() &&
		!r.UpdatedAt.IsZero() &&
		r.UpdatedAt.Before(r.CreatedAt) {
		return ErrInvalidInformationRequestUpdatedAt
	}

	return nil
}

// ValidateForPersist validates an information request loaded from or written to
// persistence.
func (r InformationRequest) ValidateForPersist() error {
	if !isValidReferenceID(r.ID) {
		return ErrInvalidInformationRequestID
	}
	if !isValidReferenceID(r.TradeID) {
		return ErrInvalidInformationRequestTradeID
	}
	if !IsValidInformationRequestTargetSide(r.TargetSide) {
		return ErrInvalidInformationRequestTargetSide
	}
	if !isValidReferenceID(r.TargetAvatarID) {
		return ErrInvalidInformationRequestTargetAvatarID
	}

	reason := strings.TrimSpace(r.Reason)
	if reason == "" ||
		len([]rune(reason)) > MaxInformationRequestReasonLength {
		return ErrInvalidInformationRequestReason
	}

	if !IsValidInformationRequestStatus(r.Status) {
		return ErrInvalidInformationRequestStatus
	}
	if r.CreatedAt.IsZero() {
		return ErrInvalidInformationRequestCreatedAt
	}
	if r.UpdatedAt.IsZero() ||
		r.UpdatedAt.Before(r.CreatedAt) {
		return ErrInvalidInformationRequestUpdatedAt
	}

	switch r.Status {
	case InformationRequestStatusPending:
		if strings.TrimSpace(r.Response) != "" {
			return ErrInvalidInformationRequestResponse
		}
		if r.SubmittedAt != nil {
			return ErrInvalidInformationRequestSubmittedAt
		}

	case InformationRequestStatusSubmitted:
		response := strings.TrimSpace(r.Response)
		if response == "" ||
			len([]rune(response)) > MaxInformationRequestResponseLength {
			return ErrInvalidInformationRequestResponse
		}
		if r.SubmittedAt == nil ||
			r.SubmittedAt.IsZero() ||
			r.SubmittedAt.Before(r.CreatedAt) ||
			r.SubmittedAt.After(r.UpdatedAt) {
			return ErrInvalidInformationRequestSubmittedAt
		}
	}

	return nil
}

func IsValidInformationRequestStatus(
	status InformationRequestStatus,
) bool {
	switch status {
	case InformationRequestStatusPending,
		InformationRequestStatusSubmitted:
		return true
	default:
		return false
	}
}

func IsValidInformationRequestTargetSide(
	side InformationRequestTargetSide,
) bool {
	switch side {
	case InformationRequestTargetBuyer,
		InformationRequestTargetSeller:
		return true
	default:
		return false
	}
}

func informationRequestTimePtr(
	value time.Time,
) *time.Time {
	normalized := value.UTC()
	return &normalized
}
