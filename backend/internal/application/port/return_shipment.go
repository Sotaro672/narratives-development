// backend/internal/application/port/return_shipment.go
package port

import (
	"context"
	"time"

	tradedom "narratives/internal/domain/trade"
)

// ReturnShipmentGateway is the external reverse-logistics boundary used for
// Avatar-to-Avatar Resale Trade returns.
//
// The current AMOL flow is:
//
//	buyer
//	  -> PUDO drop-off
//	  -> Yamato delivery
//	  -> seller
//
// Domain and application layers must not depend on provider-specific API
// request/response structures. A concrete adapter is responsible for
// translating this contract to Yamato/PUDO or another future provider.
//
// ReturnAgreement remains authoritative for the mutually agreed financial
// conditions. ReturnShipmentGateway deals only with physical logistics.
type ReturnShipmentGateway interface {
	// CreateReturnShipment creates or resolves one provider-side anonymous
	// return shipment.
	//
	// IdempotencyKey must be stable across retries for the same Trade return.
	// Repeated calls with the same key must not intentionally create multiple
	// provider shipments.
	CreateReturnShipment(
		ctx context.Context,
		in CreateReturnShipmentInput,
	) (CreateReturnShipmentResult, error)

	// GetReturnShipmentStatus retrieves the latest provider-side logistics
	// state.
	//
	// The application layer translates ProviderState into ReturnShipment domain
	// transitions such as MarkShipped or MarkDelivered.
	GetReturnShipmentStatus(
		ctx context.Context,
		in GetReturnShipmentStatusInput,
	) (ReturnShipmentStatusResult, error)

	// CancelReturnShipment cancels a provider-side shipment before carrier
	// acceptance when supported.
	//
	// Once the parcel has already been accepted by the carrier, implementations
	// should return an appropriate terminal/provider error rather than pretending
	// that the physical shipment was cancelled.
	CancelReturnShipment(
		ctx context.Context,
		in CancelReturnShipmentInput,
	) (CancelReturnShipmentResult, error)
}

// CreateReturnShipmentInput identifies one immutable return-shipment request.
//
// BuyerAvatarID and SellerAvatarID identify the transaction participants without
// exposing postal addresses to the application port. A concrete anonymous
// shipping adapter may resolve or exchange provider-specific anonymous delivery
// information behind this boundary.
//
// ProposalID must be the seller proposal explicitly accepted by the buyer.
type CreateReturnShipmentInput struct {
	TradeID           string
	ReturnAgreementID string
	ProposalID        string

	OrderID        string
	OrderItemIndex int

	BuyerAvatarID  string
	SellerAvatarID string

	Carrier       tradedom.ReturnShipmentCarrier
	DropOffMethod tradedom.ReturnShipmentDropOffMethod

	IdempotencyKey string
}

// CreateReturnShipmentResult represents a successfully prepared provider-side
// return shipment.
//
// ProviderShipmentID is the provider's stable shipment identifier.
//
// QRCodePayload contains the value required to render or otherwise present the
// PUDO return QR to the buyer. It may be an opaque token, URL, or provider
// payload depending on the adapter.
//
// TrackingNumber may be empty until the parcel is physically handed to the
// carrier.
//
// CarrierStatus is the provider's raw/opaque status string. Business decisions
// should use ProviderState rather than interpreting CarrierStatus directly.
type CreateReturnShipmentResult struct {
	ProviderShipmentID string

	ProviderState ReturnShipmentProviderState

	QRCodePayload   string
	QRCodeExpiresAt *time.Time

	TrackingNumber string
	CarrierStatus  string

	PreparedAt time.Time
}

// GetReturnShipmentStatusInput identifies the provider shipment to refresh.
//
// TradeID is included to preserve transaction context and allow adapters to
// verify that a provider shipment belongs to the expected Trade.
type GetReturnShipmentStatusInput struct {
	TradeID            string
	ProviderShipmentID string
}

// ReturnShipmentStatusResult is the provider-independent observation returned
// by the gateway.
//
// ObservedAt represents when this state was retrieved or reported by the
// provider.
//
// ShippedAt and DeliveredAt should be populated only when the provider exposes
// authoritative event timestamps. If unavailable, the application layer may
// use ObservedAt when applying the corresponding domain transition.
type ReturnShipmentStatusResult struct {
	ProviderShipmentID string

	ProviderState ReturnShipmentProviderState

	TrackingNumber string
	CarrierStatus  string

	ShippedAt   *time.Time
	DeliveredAt *time.Time
	CancelledAt *time.Time

	ObservedAt time.Time
}

// CancelReturnShipmentInput identifies a shipment that should be cancelled.
//
// IdempotencyKey must remain stable for retries of the same cancellation
// operation.
type CancelReturnShipmentInput struct {
	TradeID            string
	ProviderShipmentID string
	IdempotencyKey     string
}

// CancelReturnShipmentResult represents the provider state after a successful
// cancellation request.
type CancelReturnShipmentResult struct {
	ProviderShipmentID string
	ProviderState      ReturnShipmentProviderState
	CarrierStatus      string
	CancelledAt        time.Time
}

// ReturnShipmentProviderState is a normalized provider-independent logistics
// state.
//
// Raw Yamato/PUDO status codes must be translated by the concrete adapter into
// one of these values.
//
// Typical flow:
//
//	ready_for_dropoff
//	  -> shipped
//	  -> delivered
//
// Before carrier acceptance:
//
//	ready_for_dropoff -> cancelled
type ReturnShipmentProviderState string

const (
	ReturnShipmentProviderStateReadyForDropOff ReturnShipmentProviderState = "ready_for_dropoff"
	ReturnShipmentProviderStateShipped         ReturnShipmentProviderState = "shipped"
	ReturnShipmentProviderStateDelivered       ReturnShipmentProviderState = "delivered"
	ReturnShipmentProviderStateCancelled       ReturnShipmentProviderState = "cancelled"
)

// IsValidReturnShipmentProviderState reports whether state is understood by the
// application layer.
func IsValidReturnShipmentProviderState(
	state ReturnShipmentProviderState,
) bool {
	switch state {
	case ReturnShipmentProviderStateReadyForDropOff,
		ReturnShipmentProviderStateShipped,
		ReturnShipmentProviderStateDelivered,
		ReturnShipmentProviderStateCancelled:
		return true

	default:
		return false
	}
}

// ReturnShipmentGatewayError exposes structured provider failure metadata.
//
// Adapter implementations should use this interface when the application must
// distinguish retryable provider/network failures from terminal failures.
//
// Error() must not contain addresses, phone numbers, QR payloads, or other
// sensitive shipment data.
type ReturnShipmentGatewayError interface {
	error

	ErrorType() string
	ErrorCode() string
	Retryable() bool
}
