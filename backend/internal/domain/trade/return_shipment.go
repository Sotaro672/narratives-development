// backend/internal/domain/trade/return_shipment.go
package trade

import (
	"errors"
	"strings"
	"time"
)

// ============================================================
// Types
// ============================================================

type ReturnShipmentStatus string
type ReturnShipmentCarrier string
type ReturnShipmentDropOffMethod string

const (
	ReturnShipmentStatusPending         ReturnShipmentStatus = "pending"
	ReturnShipmentStatusReadyForDropOff ReturnShipmentStatus = "ready_for_dropoff"
	ReturnShipmentStatusShipped         ReturnShipmentStatus = "shipped"
	ReturnShipmentStatusDelivered       ReturnShipmentStatus = "delivered"
	ReturnShipmentStatusCancelled       ReturnShipmentStatus = "cancelled"
)

const (
	ReturnShipmentCarrierYamato ReturnShipmentCarrier = "yamato"
)

const (
	ReturnShipmentDropOffMethodPUDO ReturnShipmentDropOffMethod = "pudo"
)

// ============================================================
// Policy
// ============================================================

const (
	MaxReturnShipmentProviderShipmentIDLength = 512
	MaxReturnShipmentQRCodePayloadLength      = 8192
	MaxReturnShipmentTrackingNumberLength     = 256
	MaxReturnShipmentCarrierStatusLength      = 256
)

// ============================================================
// Errors
// ============================================================

var (
	ErrInvalidReturnShipmentID = errors.New(
		"trade: invalid return shipment id",
	)
	ErrInvalidReturnShipmentTradeID = errors.New(
		"trade: invalid return shipment tradeId",
	)
	ErrInvalidReturnShipmentAgreementID = errors.New(
		"trade: invalid return shipment agreementId",
	)
	ErrInvalidReturnShipmentProposalID = errors.New(
		"trade: invalid return shipment proposalId",
	)
	ErrInvalidReturnShipmentStatus = errors.New(
		"trade: invalid return shipment status",
	)
	ErrInvalidReturnShipmentCarrier = errors.New(
		"trade: invalid return shipment carrier",
	)
	ErrInvalidReturnShipmentDropOffMethod = errors.New(
		"trade: invalid return shipment dropOffMethod",
	)
	ErrInvalidReturnShipmentProviderShipmentID = errors.New(
		"trade: invalid return shipment providerShipmentId",
	)
	ErrInvalidReturnShipmentQRCodePayload = errors.New(
		"trade: invalid return shipment qrCodePayload",
	)
	ErrInvalidReturnShipmentQRCodeExpiresAt = errors.New(
		"trade: invalid return shipment qrCodeExpiresAt",
	)
	ErrInvalidReturnShipmentTrackingNumber = errors.New(
		"trade: invalid return shipment trackingNumber",
	)
	ErrInvalidReturnShipmentCarrierStatus = errors.New(
		"trade: invalid return shipment carrierStatus",
	)
	ErrInvalidReturnShipmentReadyAt = errors.New(
		"trade: invalid return shipment readyAt",
	)
	ErrInvalidReturnShipmentShippedAt = errors.New(
		"trade: invalid return shipment shippedAt",
	)
	ErrInvalidReturnShipmentDeliveredAt = errors.New(
		"trade: invalid return shipment deliveredAt",
	)
	ErrInvalidReturnShipmentCancelledAt = errors.New(
		"trade: invalid return shipment cancelledAt",
	)
	ErrInvalidReturnShipmentCreatedAt = errors.New(
		"trade: invalid return shipment createdAt",
	)
	ErrInvalidReturnShipmentUpdatedAt = errors.New(
		"trade: invalid return shipment updatedAt",
	)
	ErrInvalidReturnShipmentState = errors.New(
		"trade: invalid return shipment state",
	)
	ErrReturnShipmentPreparationNotAllowed = errors.New(
		"trade: return shipment preparation is not allowed",
	)
	ErrReturnShipmentCarrierUpdateNotAllowed = errors.New(
		"trade: return shipment carrier update is not allowed",
	)
	ErrReturnShipmentDropOffNotAllowed = errors.New(
		"trade: return shipment drop-off is not allowed",
	)
	ErrReturnShipmentDeliveryNotAllowed = errors.New(
		"trade: return shipment delivery is not allowed",
	)
	ErrReturnShipmentCancellationNotAllowed = errors.New(
		"trade: return shipment cancellation is not allowed",
	)
	ErrReturnShipmentTrackingNumberMismatch = errors.New(
		"trade: return shipment tracking number mismatch",
	)
)

// ============================================================
// Return Shipment
// ============================================================

// ReturnShipment represents the physical reverse-logistics state for one
// Avatar-to-Avatar Resale Trade return.
//
// ReturnAgreement owns the mutually agreed return/refund conditions.
// ReturnShipment owns only physical logistics after an accepted proposal that
// requires the item to be returned.
//
// The shipment is bound to the exact ReturnAgreement and ReturnProposal that
// authorized it. This prevents a shipment generated for one proposal from being
// reused for another negotiation.
//
// Current AMOL return logistics:
//
//	carrier       = yamato
//	dropOffMethod = pudo
//
// Typical flow:
//
//	pending
//	  -> ready_for_dropoff
//	  -> shipped
//	  -> delivered
//
// A shipment may be cancelled before carrier acceptance:
//
//	pending -> cancelled
//	ready_for_dropoff -> cancelled
//
// ReturnAgreement state is updated separately by the application layer:
//
//	ReturnShipment shipped
//	  -> ReturnAgreement.MarkReturnShipped()
//
//	ReturnShipment delivered
//	  -> ReturnAgreement.MarkReturnReceived()
//
// Carrier delivery state should ultimately be authoritative for physical
// receipt. Seller input must not alter previously agreed financial conditions.
//
// QRCodePayload is temporary operational data. It is cleared after carrier
// acceptance or cancellation.
//
// Firestore:
//
//	tradeReturnShipments/{tradeId}
type ReturnShipment struct {
	ID                string `json:"id"`
	TradeID           string `json:"tradeId"`
	ReturnAgreementID string `json:"returnAgreementId"`
	ProposalID        string `json:"proposalId"`

	Status        ReturnShipmentStatus        `json:"status"`
	Carrier       ReturnShipmentCarrier       `json:"carrier"`
	DropOffMethod ReturnShipmentDropOffMethod `json:"dropOffMethod"`

	ProviderShipmentID string `json:"providerShipmentId,omitempty"`
	QRCodePayload      string `json:"qrCodePayload,omitempty"`
	TrackingNumber     string `json:"trackingNumber,omitempty"`
	CarrierStatus      string `json:"carrierStatus,omitempty"`

	QRCodeExpiresAt *time.Time `json:"qrCodeExpiresAt,omitempty"`
	ReadyAt         *time.Time `json:"readyAt,omitempty"`
	ShippedAt       *time.Time `json:"shippedAt,omitempty"`
	DeliveredAt     *time.Time `json:"deliveredAt,omitempty"`
	CancelledAt     *time.Time `json:"cancelledAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s ReturnShipment) GetID() string {
	return s.ID
}

// NewReturnShipmentForCreate creates a local ReturnShipment before the
// external carrier shipment is registered.
//
// Persisting pending first allows the application layer to establish an
// idempotent local record before calling an external shipping provider.
//
// ID and timestamps may be empty/zero before repository persistence.
func NewReturnShipmentForCreate(
	id string,
	tradeID string,
	returnAgreementID string,
	proposalID string,
) (ReturnShipment, error) {
	shipment := ReturnShipment{
		ID:                strings.TrimSpace(id),
		TradeID:           strings.TrimSpace(tradeID),
		ReturnAgreementID: strings.TrimSpace(returnAgreementID),
		ProposalID:        strings.TrimSpace(proposalID),
		Status:            ReturnShipmentStatusPending,
		Carrier:           ReturnShipmentCarrierYamato,
		DropOffMethod:     ReturnShipmentDropOffMethodPUDO,
	}

	if err := shipment.ValidateForCreate(); err != nil {
		return ReturnShipment{}, err
	}

	return shipment, nil
}

// MarkReady records successful external shipment preparation and makes the
// PUDO QR available to the buyer.
//
// trackingNumber may be empty when the carrier assigns it only after drop-off.
// qrCodeExpiresAt may be nil when the provider does not expose an expiry.
func (s *ReturnShipment) MarkReady(
	providerShipmentID string,
	qrCodePayload string,
	qrCodeExpiresAt *time.Time,
	trackingNumber string,
	carrierStatus string,
	at time.Time,
) error {
	if s == nil || s.Status != ReturnShipmentStatusPending {
		return ErrReturnShipmentPreparationNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnShipmentReadyAt
	}

	providerShipmentID = strings.TrimSpace(providerShipmentID)
	qrCodePayload = strings.TrimSpace(qrCodePayload)
	trackingNumber = strings.TrimSpace(trackingNumber)
	carrierStatus = strings.TrimSpace(carrierStatus)

	if !isValidReturnShipmentProviderShipmentID(providerShipmentID) {
		return ErrInvalidReturnShipmentProviderShipmentID
	}
	if !isValidReturnShipmentQRCodePayload(qrCodePayload) {
		return ErrInvalidReturnShipmentQRCodePayload
	}
	if trackingNumber != "" &&
		!isValidReturnShipmentTrackingNumber(trackingNumber) {
		return ErrInvalidReturnShipmentTrackingNumber
	}
	if carrierStatus != "" &&
		!isValidReturnShipmentCarrierStatus(carrierStatus) {
		return ErrInvalidReturnShipmentCarrierStatus
	}

	at = at.UTC()
	if err := s.validateTransitionTime(at); err != nil {
		return ErrInvalidReturnShipmentReadyAt
	}

	var expiresAt *time.Time
	if qrCodeExpiresAt != nil {
		if qrCodeExpiresAt.IsZero() {
			return ErrInvalidReturnShipmentQRCodeExpiresAt
		}

		normalized := qrCodeExpiresAt.UTC()
		if !normalized.After(at) {
			return ErrInvalidReturnShipmentQRCodeExpiresAt
		}
		expiresAt = cloneReturnShipmentTimePtr(&normalized)
	}

	s.Status = ReturnShipmentStatusReadyForDropOff
	s.ProviderShipmentID = providerShipmentID
	s.QRCodePayload = qrCodePayload
	s.QRCodeExpiresAt = expiresAt
	s.TrackingNumber = trackingNumber
	s.CarrierStatus = carrierStatus
	s.ReadyAt = cloneReturnShipmentTimePtr(&at)
	s.UpdatedAt = at

	return nil
}

// UpdateCarrierStatus records a provider-specific logistics state without
// changing the AMOL ReturnShipment state.
//
// CarrierStatus is intentionally opaque to this domain. Translation from
// Yamato/PUDO provider states into MarkShipped / MarkDelivered transitions
// belongs to the application/adapter boundary.
func (s *ReturnShipment) UpdateCarrierStatus(
	carrierStatus string,
	at time.Time,
) error {
	if s == nil ||
		s.Status == ReturnShipmentStatusCancelled ||
		s.Status == ReturnShipmentStatusDelivered {
		return ErrReturnShipmentCarrierUpdateNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnShipmentUpdatedAt
	}

	carrierStatus = strings.TrimSpace(carrierStatus)
	if !isValidReturnShipmentCarrierStatus(carrierStatus) {
		return ErrInvalidReturnShipmentCarrierStatus
	}

	at = at.UTC()
	if err := s.validateTransitionTime(at); err != nil {
		return ErrInvalidReturnShipmentUpdatedAt
	}

	s.CarrierStatus = carrierStatus
	s.UpdatedAt = at

	return nil
}

// MarkShipped records authoritative carrier acceptance of the returned item.
//
// This transition should be called only after Yamato confirms that the parcel
// has actually been handed over. Displaying or issuing a QR alone must not mark
// the return as shipped.
//
// The QR is cleared because it is no longer needed after carrier acceptance.
func (s *ReturnShipment) MarkShipped(
	trackingNumber string,
	carrierStatus string,
	at time.Time,
) error {
	if s == nil ||
		s.Status != ReturnShipmentStatusReadyForDropOff ||
		s.ReadyAt == nil ||
		s.ProviderShipmentID == "" {
		return ErrReturnShipmentDropOffNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnShipmentShippedAt
	}

	trackingNumber = strings.TrimSpace(trackingNumber)
	carrierStatus = strings.TrimSpace(carrierStatus)

	if !isValidReturnShipmentTrackingNumber(trackingNumber) {
		return ErrInvalidReturnShipmentTrackingNumber
	}
	if s.TrackingNumber != "" &&
		s.TrackingNumber != trackingNumber {
		return ErrReturnShipmentTrackingNumberMismatch
	}
	if carrierStatus != "" &&
		!isValidReturnShipmentCarrierStatus(carrierStatus) {
		return ErrInvalidReturnShipmentCarrierStatus
	}

	at = at.UTC()
	if err := s.validateTransitionTime(at); err != nil ||
		at.Before(*s.ReadyAt) {
		return ErrInvalidReturnShipmentShippedAt
	}

	s.Status = ReturnShipmentStatusShipped
	s.TrackingNumber = trackingNumber
	s.CarrierStatus = carrierStatus
	s.QRCodePayload = ""
	s.QRCodeExpiresAt = nil
	s.ShippedAt = cloneReturnShipmentTimePtr(&at)
	s.UpdatedAt = at

	return nil
}

// MarkDelivered records authoritative carrier delivery to the return
// destination.
//
// The application layer can use this transition as the physical evidence for
// ReturnAgreement.MarkReturnReceived().
func (s *ReturnShipment) MarkDelivered(
	carrierStatus string,
	at time.Time,
) error {
	if s == nil ||
		s.Status != ReturnShipmentStatusShipped ||
		s.ShippedAt == nil ||
		s.TrackingNumber == "" {
		return ErrReturnShipmentDeliveryNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnShipmentDeliveredAt
	}

	carrierStatus = strings.TrimSpace(carrierStatus)
	if carrierStatus != "" &&
		!isValidReturnShipmentCarrierStatus(carrierStatus) {
		return ErrInvalidReturnShipmentCarrierStatus
	}

	at = at.UTC()
	if err := s.validateTransitionTime(at); err != nil ||
		at.Before(*s.ShippedAt) {
		return ErrInvalidReturnShipmentDeliveredAt
	}

	s.Status = ReturnShipmentStatusDelivered
	s.CarrierStatus = carrierStatus
	s.DeliveredAt = cloneReturnShipmentTimePtr(&at)
	s.UpdatedAt = at

	return nil
}

// Cancel cancels a return shipment before carrier acceptance.
//
// Once the parcel has been accepted by the carrier, cancellation must be
// handled as a logistics exception rather than rewinding this aggregate.
func (s *ReturnShipment) Cancel(
	carrierStatus string,
	at time.Time,
) error {
	if s == nil ||
		(s.Status != ReturnShipmentStatusPending &&
			s.Status != ReturnShipmentStatusReadyForDropOff) {
		return ErrReturnShipmentCancellationNotAllowed
	}
	if at.IsZero() {
		return ErrInvalidReturnShipmentCancelledAt
	}

	carrierStatus = strings.TrimSpace(carrierStatus)
	if carrierStatus != "" &&
		!isValidReturnShipmentCarrierStatus(carrierStatus) {
		return ErrInvalidReturnShipmentCarrierStatus
	}

	at = at.UTC()
	if err := s.validateTransitionTime(at); err != nil {
		return ErrInvalidReturnShipmentCancelledAt
	}

	s.Status = ReturnShipmentStatusCancelled
	s.CarrierStatus = carrierStatus
	s.QRCodePayload = ""
	s.QRCodeExpiresAt = nil
	s.CancelledAt = cloneReturnShipmentTimePtr(&at)
	s.UpdatedAt = at

	return nil
}

func (s ReturnShipment) ValidateForCreate() error {
	if s.ID != "" && !isValidReferenceID(s.ID) {
		return ErrInvalidReturnShipmentID
	}
	if !isValidReferenceID(s.TradeID) {
		return ErrInvalidReturnShipmentTradeID
	}
	if !isValidReferenceID(s.ReturnAgreementID) {
		return ErrInvalidReturnShipmentAgreementID
	}
	if !isValidReferenceID(s.ProposalID) {
		return ErrInvalidReturnShipmentProposalID
	}
	if s.Status != ReturnShipmentStatusPending {
		return ErrInvalidReturnShipmentState
	}
	if !IsValidReturnShipmentCarrier(s.Carrier) {
		return ErrInvalidReturnShipmentCarrier
	}
	if !IsValidReturnShipmentDropOffMethod(s.DropOffMethod) {
		return ErrInvalidReturnShipmentDropOffMethod
	}
	if s.ProviderShipmentID != "" ||
		s.QRCodePayload != "" ||
		s.QRCodeExpiresAt != nil ||
		s.TrackingNumber != "" ||
		s.CarrierStatus != "" ||
		s.ReadyAt != nil ||
		s.ShippedAt != nil ||
		s.DeliveredAt != nil ||
		s.CancelledAt != nil {
		return ErrInvalidReturnShipmentState
	}
	if !s.CreatedAt.IsZero() &&
		!s.UpdatedAt.IsZero() &&
		s.UpdatedAt.Before(s.CreatedAt) {
		return ErrInvalidReturnShipmentUpdatedAt
	}

	return nil
}

func (s ReturnShipment) ValidateForPersist() error {
	if !isValidReferenceID(s.ID) {
		return ErrInvalidReturnShipmentID
	}
	if !isValidReferenceID(s.TradeID) {
		return ErrInvalidReturnShipmentTradeID
	}
	if !isValidReferenceID(s.ReturnAgreementID) {
		return ErrInvalidReturnShipmentAgreementID
	}
	if !isValidReferenceID(s.ProposalID) {
		return ErrInvalidReturnShipmentProposalID
	}
	if !IsValidReturnShipmentStatus(s.Status) {
		return ErrInvalidReturnShipmentStatus
	}
	if !IsValidReturnShipmentCarrier(s.Carrier) {
		return ErrInvalidReturnShipmentCarrier
	}
	if !IsValidReturnShipmentDropOffMethod(s.DropOffMethod) {
		return ErrInvalidReturnShipmentDropOffMethod
	}
	if s.ProviderShipmentID != "" &&
		!isValidReturnShipmentProviderShipmentID(s.ProviderShipmentID) {
		return ErrInvalidReturnShipmentProviderShipmentID
	}
	if s.QRCodePayload != "" &&
		!isValidReturnShipmentQRCodePayload(s.QRCodePayload) {
		return ErrInvalidReturnShipmentQRCodePayload
	}
	if s.TrackingNumber != "" &&
		!isValidReturnShipmentTrackingNumber(s.TrackingNumber) {
		return ErrInvalidReturnShipmentTrackingNumber
	}
	if s.CarrierStatus != "" &&
		!isValidReturnShipmentCarrierStatus(s.CarrierStatus) {
		return ErrInvalidReturnShipmentCarrierStatus
	}
	if s.CreatedAt.IsZero() {
		return ErrInvalidReturnShipmentCreatedAt
	}
	if s.UpdatedAt.IsZero() ||
		s.UpdatedAt.Before(s.CreatedAt) {
		return ErrInvalidReturnShipmentUpdatedAt
	}

	if err := s.validatePersistedState(); err != nil {
		return err
	}

	return nil
}

func (s ReturnShipment) validatePersistedState() error {
	switch s.Status {
	case ReturnShipmentStatusPending:
		if s.ProviderShipmentID != "" ||
			s.QRCodePayload != "" ||
			s.QRCodeExpiresAt != nil ||
			s.TrackingNumber != "" ||
			s.CarrierStatus != "" ||
			s.ReadyAt != nil ||
			s.ShippedAt != nil ||
			s.DeliveredAt != nil ||
			s.CancelledAt != nil {
			return ErrInvalidReturnShipmentState
		}

	case ReturnShipmentStatusReadyForDropOff:
		if !isValidReturnShipmentProviderShipmentID(
			s.ProviderShipmentID,
		) ||
			!isValidReturnShipmentQRCodePayload(
				s.QRCodePayload,
			) ||
			s.ReadyAt == nil ||
			s.ReadyAt.IsZero() ||
			s.ReadyAt.Before(s.CreatedAt) ||
			s.ReadyAt.After(s.UpdatedAt) ||
			s.ShippedAt != nil ||
			s.DeliveredAt != nil ||
			s.CancelledAt != nil {
			return ErrInvalidReturnShipmentState
		}

		if s.QRCodeExpiresAt != nil {
			if s.QRCodeExpiresAt.IsZero() ||
				!s.QRCodeExpiresAt.After(*s.ReadyAt) {
				return ErrInvalidReturnShipmentQRCodeExpiresAt
			}
		}

	case ReturnShipmentStatusShipped:
		if !isValidReturnShipmentProviderShipmentID(
			s.ProviderShipmentID,
		) ||
			!isValidReturnShipmentTrackingNumber(
				s.TrackingNumber,
			) ||
			s.QRCodePayload != "" ||
			s.QRCodeExpiresAt != nil ||
			s.ReadyAt == nil ||
			s.ShippedAt == nil ||
			s.ReadyAt.IsZero() ||
			s.ShippedAt.IsZero() ||
			s.ReadyAt.Before(s.CreatedAt) ||
			s.ShippedAt.Before(*s.ReadyAt) ||
			s.ShippedAt.After(s.UpdatedAt) ||
			s.DeliveredAt != nil ||
			s.CancelledAt != nil {
			return ErrInvalidReturnShipmentState
		}

	case ReturnShipmentStatusDelivered:
		if !isValidReturnShipmentProviderShipmentID(
			s.ProviderShipmentID,
		) ||
			!isValidReturnShipmentTrackingNumber(
				s.TrackingNumber,
			) ||
			s.QRCodePayload != "" ||
			s.QRCodeExpiresAt != nil ||
			s.ReadyAt == nil ||
			s.ShippedAt == nil ||
			s.DeliveredAt == nil ||
			s.ReadyAt.IsZero() ||
			s.ShippedAt.IsZero() ||
			s.DeliveredAt.IsZero() ||
			s.ReadyAt.Before(s.CreatedAt) ||
			s.ShippedAt.Before(*s.ReadyAt) ||
			s.DeliveredAt.Before(*s.ShippedAt) ||
			s.DeliveredAt.After(s.UpdatedAt) ||
			s.CancelledAt != nil {
			return ErrInvalidReturnShipmentState
		}

	case ReturnShipmentStatusCancelled:
		if s.QRCodePayload != "" ||
			s.QRCodeExpiresAt != nil ||
			s.ShippedAt != nil ||
			s.DeliveredAt != nil ||
			s.CancelledAt == nil ||
			s.CancelledAt.IsZero() ||
			s.CancelledAt.Before(s.CreatedAt) ||
			s.CancelledAt.After(s.UpdatedAt) {
			return ErrInvalidReturnShipmentState
		}

		if s.ReadyAt != nil {
			if s.ReadyAt.IsZero() ||
				s.ReadyAt.Before(s.CreatedAt) ||
				s.ReadyAt.After(*s.CancelledAt) ||
				!isValidReturnShipmentProviderShipmentID(
					s.ProviderShipmentID,
				) {
				return ErrInvalidReturnShipmentState
			}
		} else if s.ProviderShipmentID != "" ||
			s.TrackingNumber != "" {
			return ErrInvalidReturnShipmentState
		}

	default:
		return ErrInvalidReturnShipmentState
	}

	return nil
}

func (s ReturnShipment) validateTransitionTime(
	at time.Time,
) error {
	if at.IsZero() {
		return ErrInvalidReturnShipmentUpdatedAt
	}

	at = at.UTC()

	if !s.CreatedAt.IsZero() &&
		at.Before(s.CreatedAt) {
		return ErrInvalidReturnShipmentUpdatedAt
	}
	if !s.UpdatedAt.IsZero() &&
		at.Before(s.UpdatedAt) {
		return ErrInvalidReturnShipmentUpdatedAt
	}

	return nil
}

// ============================================================
// Validation helpers
// ============================================================

func IsValidReturnShipmentStatus(
	status ReturnShipmentStatus,
) bool {
	switch status {
	case ReturnShipmentStatusPending,
		ReturnShipmentStatusReadyForDropOff,
		ReturnShipmentStatusShipped,
		ReturnShipmentStatusDelivered,
		ReturnShipmentStatusCancelled:
		return true

	default:
		return false
	}
}

func IsValidReturnShipmentCarrier(
	carrier ReturnShipmentCarrier,
) bool {
	switch carrier {
	case ReturnShipmentCarrierYamato:
		return true

	default:
		return false
	}
}

func IsValidReturnShipmentDropOffMethod(
	method ReturnShipmentDropOffMethod,
) bool {
	switch method {
	case ReturnShipmentDropOffMethodPUDO:
		return true

	default:
		return false
	}
}

func isValidReturnShipmentProviderShipmentID(
	value string,
) bool {
	value = strings.TrimSpace(value)

	return value != "" &&
		len([]rune(value)) <=
			MaxReturnShipmentProviderShipmentIDLength &&
		!strings.ContainsAny(value, "\r\n")
}

func isValidReturnShipmentQRCodePayload(
	value string,
) bool {
	value = strings.TrimSpace(value)

	return value != "" &&
		len([]rune(value)) <=
			MaxReturnShipmentQRCodePayloadLength
}

func isValidReturnShipmentTrackingNumber(
	value string,
) bool {
	value = strings.TrimSpace(value)

	if value == "" ||
		len([]rune(value)) >
			MaxReturnShipmentTrackingNumberLength {
		return false
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return false
	}

	return true
}

func isValidReturnShipmentCarrierStatus(
	value string,
) bool {
	value = strings.TrimSpace(value)

	return value != "" &&
		len([]rune(value)) <=
			MaxReturnShipmentCarrierStatusLength &&
		!strings.ContainsAny(value, "\r\n")
}

func cloneReturnShipmentTimePtr(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
