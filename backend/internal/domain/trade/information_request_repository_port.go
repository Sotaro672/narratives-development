// backend/internal/domain/trade/information_request_repository_port.go

package trade

import (
	"context"
	"errors"
)

// ============================================================
// Contract errors
// ============================================================

var (
	ErrInformationRequestNotFound = errors.New(
		"trade: information request not found",
	)
	ErrInformationRequestAlreadyExists = errors.New(
		"trade: information request already exists",
	)
	ErrInformationRequestConflict = errors.New(
		"trade: information request conflict",
	)
)

// ============================================================
// Information Request repository
// ============================================================

// InformationRequestRepository is the repository port for an Admin-created
// information request addressed to one participant in a Trade.
//
// Firestore:
//
//	tradeInformationRequests/{tradeId}
//
// One Trade has at most one InformationRequest. The document ID,
// InformationRequest.ID and InformationRequest.TradeID are expected to be
// identical to tradeId:
//
//	tradeId -> InformationRequest
//
// The request remains persisted after submission so Admin can review both the
// original request reason and the participant response.
//
// Mall does not list InformationRequests independently by Avatar. An
// InformationRequest is surfaced as part of its Trade and the application/query
// layer may inspect the request for that Trade to determine whether the
// authenticated participant still has a pending information action.
//
// InformationRequest itself does not own Trade message read state. The
// application layer creates the corresponding Trade system message and uses the
// request status to keep the Trade action counter active until a response is
// submitted.
type InformationRequestRepository interface {
	// GetByTradeID retrieves the InformationRequest associated with one Trade.
	//
	// Because one Trade has at most one InformationRequest, tradeID is also the
	// Firestore document ID.
	//
	// Implementations should return ErrInformationRequestNotFound when no
	// InformationRequest exists for the supplied Trade ID.
	GetByTradeID(
		ctx context.Context,
		tradeID string,
	) (InformationRequest, error)

	// Create persists one newly created pending InformationRequest.
	//
	// The document ID must be tradeID, structurally enforcing at most one
	// InformationRequest per Trade.
	//
	// Implementations should return ErrInformationRequestAlreadyExists when an
	// InformationRequest already exists for the Trade, regardless of whether
	// the existing request is pending or submitted.
	//
	// Expected immutable fields after creation:
	//   - ID
	//   - TradeID
	//   - TargetSide
	//   - TargetAvatarID
	//   - Reason
	//   - CreatedAt
	Create(
		ctx context.Context,
		request InformationRequest,
	) (InformationRequest, error)

	// Update persists the requested participant's response and completion state.
	//
	// Expected mutable fields:
	//   - Response
	//   - Status
	//   - SubmittedAt
	//   - UpdatedAt
	//
	// Immutable fields must not be changed:
	//   - ID
	//   - TradeID
	//   - TargetSide
	//   - TargetAvatarID
	//   - Reason
	//   - CreatedAt
	//
	// Implementations should return:
	//   - ErrInformationRequestNotFound when the request does not exist
	//   - ErrInformationRequestConflict when immutable identity differs from the
	//     persisted request or the persisted state changed incompatibly
	Update(
		ctx context.Context,
		tradeID string,
		request InformationRequest,
	) (InformationRequest, error)
}
