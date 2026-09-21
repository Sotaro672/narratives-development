// backend/internal/domain/trade/return_agreement_repository_port.go

package trade

import (
	"context"
	"errors"
)

// ============================================================
// Contract errors
// ============================================================

var (
	ErrReturnAgreementNotFound = errors.New(
		"trade: return agreement not found",
	)
	ErrReturnAgreementAlreadyExists = errors.New(
		"trade: return agreement already exists",
	)
	ErrReturnAgreementConflict = errors.New(
		"trade: return agreement conflict",
	)
)

// ============================================================
// Return Agreement repository
// ============================================================

// ReturnAgreementRepository is the repository port for the return negotiation
// aggregate of one Avatar-to-Avatar Resale Trade.
//
// A Trade may have at most one active ReturnAgreement.
//
// Absence of a ReturnAgreement represents:
//
//	returnStatus = none
//
// Firestore:
//
//	tradeReturnAgreements/{tradeId}
//
// The document ID is expected to be the Trade ID so the one-to-one relationship
// is enforced structurally:
//
//	tradeId -> ReturnAgreement
//
// ReturnAgreement owns:
//   - buyer return consultation
//   - seller return proposal
//   - buyer agreement/rejection
//   - return shipment state
//   - return receipt state
//   - refund-processing state
//   - return completion state
//   - dispute transition
//
// Order remains authoritative for the underlying purchase, dispatch, token
// transfer, refund execution, settlement, and cancellation state.
type ReturnAgreementRepository interface {
	// GetByTradeID retrieves the ReturnAgreement associated with one Trade.
	//
	// Implementations should return ErrReturnAgreementNotFound when no
	// ReturnAgreement exists for the supplied Trade ID.
	GetByTradeID(
		ctx context.Context,
		tradeID string,
	) (ReturnAgreement, error)

	// Create persists a newly started return consultation.
	//
	// There must be at most one ReturnAgreement for one Trade.
	//
	// Implementations should return ErrReturnAgreementAlreadyExists when an
	// agreement already exists for tradeID.
	//
	// Immutable identity fields:
	//   - ID
	//   - TradeID
	//   - Consultation
	//   - CreatedAt
	Create(
		ctx context.Context,
		agreement ReturnAgreement,
	) (ReturnAgreement, error)

	// Update persists mutable ReturnAgreement state.
	//
	// Expected mutable fields include:
	//   - Status
	//   - Proposal
	//   - AgreedAt
	//   - ReturnShippedAt
	//   - ReturnReceivedAt
	//   - RefundProcessingAt
	//   - CompletedAt
	//   - DisputedAt
	//   - UpdatedAt
	//
	// Immutable fields must not be changed:
	//   - ID
	//   - TradeID
	//   - Consultation
	//   - CreatedAt
	//
	// Implementations should return:
	//   - ErrReturnAgreementNotFound when the aggregate does not exist
	//   - ErrReturnAgreementConflict when immutable identity differs from the
	//     persisted aggregate
	Update(
		ctx context.Context,
		tradeID string,
		agreement ReturnAgreement,
	) (ReturnAgreement, error)
}
