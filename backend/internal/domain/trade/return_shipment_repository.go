// backend/internal/domain/trade/return_shipment_repository.go
package trade

import (
	"context"
	"errors"
)

// ============================================================
// Contract errors
// ============================================================

var (
	ErrReturnShipmentNotFound = errors.New(
		"trade: return shipment not found",
	)
	ErrReturnShipmentAlreadyExists = errors.New(
		"trade: return shipment already exists",
	)
	ErrReturnShipmentConflict = errors.New(
		"trade: return shipment conflict",
	)
)

// ============================================================
// Return Shipment repository
// ============================================================

// ReturnShipmentRepository is the repository port for the physical return
// shipment of one Avatar-to-Avatar Resale Trade.
//
// A Trade may have at most one ReturnShipment.
//
// ReturnAgreement owns the mutually agreed return/refund conditions.
// ReturnShipment owns only reverse-logistics state after the buyer has accepted
// a proposal requiring physical return.
//
// Firestore:
//
//	tradeReturnShipments/{tradeId}
//
// The document ID is expected to be the Trade ID so the one-to-one relationship
// is structurally enforced:
//
//	tradeId -> ReturnShipment
//
// ReturnShipment owns:
//   - external provider shipment ID
//   - PUDO QR payload and expiry
//   - Yamato tracking number
//   - carrier-specific status
//   - drop-off readiness
//   - carrier acceptance
//   - delivery completion
//   - shipment cancellation
//
// ReturnAgreement remains authoritative for:
//
//	agreed
//	-> return_shipped
//	-> return_received
//	-> refund_processing
//	-> completed
//
// The application layer is responsible for coordinating both aggregates.
//
// For example:
//
//	ReturnShipment.MarkShipped()
//	-> ReturnAgreement.MarkReturnShipped()
//
//	ReturnShipment.MarkDelivered()
//	-> ReturnAgreement.MarkReturnReceived()
type ReturnShipmentRepository interface {
	// GetByTradeID retrieves the ReturnShipment associated with one Trade.
	//
	// Implementations should return ErrReturnShipmentNotFound when no
	// ReturnShipment exists for the supplied Trade ID.
	GetByTradeID(
		ctx context.Context,
		tradeID string,
	) (ReturnShipment, error)

	// Create persists a newly started return shipment.
	//
	// There must be at most one ReturnShipment for one Trade.
	//
	// Implementations should return ErrReturnShipmentAlreadyExists when a
	// shipment already exists for tradeID.
	//
	// Before persistence the application layer will normally create the
	// aggregate in:
	//
	//	pending
	//
	// state. The repository may populate ID, CreatedAt, and UpdatedAt when they
	// have not yet been assigned.
	//
	// Immutable identity fields:
	//   - ID
	//   - TradeID
	//   - ReturnAgreementID
	//   - ProposalID
	//   - Carrier
	//   - DropOffMethod
	//   - CreatedAt
	Create(
		ctx context.Context,
		shipment ReturnShipment,
	) (ReturnShipment, error)

	// Update persists mutable ReturnShipment state.
	//
	// Expected mutable fields include:
	//   - Status
	//   - ProviderShipmentID
	//   - QRCodePayload
	//   - QRCodeExpiresAt
	//   - TrackingNumber
	//   - CarrierStatus
	//   - ReadyAt
	//   - ShippedAt
	//   - DeliveredAt
	//   - CancelledAt
	//   - UpdatedAt
	//
	// Immutable fields must not be changed:
	//   - ID
	//   - TradeID
	//   - ReturnAgreementID
	//   - ProposalID
	//   - Carrier
	//   - DropOffMethod
	//   - CreatedAt
	//
	// Implementations should return:
	//   - ErrReturnShipmentNotFound when the aggregate does not exist
	//   - ErrReturnShipmentConflict when immutable identity differs from the
	//     persisted aggregate
	Update(
		ctx context.Context,
		tradeID string,
		shipment ReturnShipment,
	) (ReturnShipment, error)
}
