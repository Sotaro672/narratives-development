// backend/internal/adapters/out/firestore/trade_return_shipment_repository.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tradedom "narratives/internal/domain/trade"
)

var (
	ErrTradeReturnShipmentRepositoryNotConfigured = errors.New(
		"trade_return_shipment_repository_fs: not configured",
	)
	ErrInvalidTradeReturnShipmentDocumentData = errors.New(
		"trade_return_shipment_repository_fs: invalid document data",
	)
)

// TradeReturnShipmentRepositoryFS implements
// tradedom.ReturnShipmentRepository using Firestore.
//
// Firestore:
//
//	tradeReturnShipments/{tradeId}
//
// One Avatar-to-Avatar Resale Trade may have at most one ReturnShipment.
//
// The document ID, ReturnShipment.ID, ReturnShipment.TradeID and
// ReturnShipment.ReturnAgreementID are all identical to tradeId.
//
// Creation additionally verifies that:
//   - Trade exists and is an Avatar-to-Avatar Resale Trade.
//   - ReturnAgreement exists.
//   - ReturnAgreement is agreed.
//   - the accepted proposal requires a physical return.
//   - ReturnShipment.ProposalID matches the accepted proposal.
type TradeReturnShipmentRepositoryFS struct {
	Client *firestore.Client
}

var _ tradedom.ReturnShipmentRepository = (*TradeReturnShipmentRepositoryFS)(nil)

func NewTradeReturnShipmentRepositoryFS(
	client *firestore.Client,
) *TradeReturnShipmentRepositoryFS {
	return &TradeReturnShipmentRepositoryFS{
		Client: client,
	}
}

func (r *TradeReturnShipmentRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("tradeReturnShipments")
}

func (r *TradeReturnShipmentRepositoryFS) tradesCol() *firestore.CollectionRef {
	return r.Client.Collection("trades")
}

func (r *TradeReturnShipmentRepositoryFS) returnAgreementsCol() *firestore.CollectionRef {
	return r.Client.Collection("tradeReturnAgreements")
}

// ============================================================
// Read
// ============================================================

func (r *TradeReturnShipmentRepositoryFS) GetByTradeID(
	ctx context.Context,
	tradeID string,
) (tradedom.ReturnShipment, error) {
	if r == nil || r.Client == nil {
		return tradedom.ReturnShipment{},
			ErrTradeReturnShipmentRepositoryNotConfigured
	}
	if tradeID == "" {
		return tradedom.ReturnShipment{},
			tradedom.ErrReturnShipmentNotFound
	}

	snap, err := r.col().Doc(tradeID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tradedom.ReturnShipment{},
				tradedom.ErrReturnShipmentNotFound
		}
		return tradedom.ReturnShipment{}, err
	}

	shipment, err := docToTradeReturnShipment(snap)
	if err != nil {
		return tradedom.ReturnShipment{}, err
	}
	if shipment.ID != tradeID ||
		shipment.TradeID != tradeID ||
		shipment.ReturnAgreementID != tradeID {
		return tradedom.ReturnShipment{},
			ErrInvalidTradeReturnShipmentDocumentData
	}

	return shipment, nil
}

// ============================================================
// Create
// ============================================================

func (r *TradeReturnShipmentRepositoryFS) Create(
	ctx context.Context,
	shipment tradedom.ReturnShipment,
) (tradedom.ReturnShipment, error) {
	if r == nil || r.Client == nil {
		return tradedom.ReturnShipment{},
			ErrTradeReturnShipmentRepositoryNotConfigured
	}
	if shipment.TradeID == "" {
		return tradedom.ReturnShipment{},
			tradedom.ErrInvalidReturnShipmentTradeID
	}

	tradeID := shipment.TradeID

	if shipment.ID == "" {
		shipment.ID = tradeID
	}
	if shipment.ID != tradeID ||
		shipment.ReturnAgreementID != tradeID {
		return tradedom.ReturnShipment{},
			tradedom.ErrReturnShipmentConflict
	}

	now := time.Now().UTC()

	if shipment.CreatedAt.IsZero() {
		shipment.CreatedAt = now
	} else {
		shipment.CreatedAt = shipment.CreatedAt.UTC()
	}

	if shipment.UpdatedAt.IsZero() {
		shipment.UpdatedAt = shipment.CreatedAt
	} else {
		shipment.UpdatedAt = shipment.UpdatedAt.UTC()
	}

	normalizeTradeReturnShipmentTimes(&shipment)

	if err := shipment.ValidateForPersist(); err != nil {
		return tradedom.ReturnShipment{}, err
	}

	ref := r.col().Doc(tradeID)
	tradeRef := r.tradesCol().Doc(tradeID)
	agreementRef := r.returnAgreementsCol().Doc(tradeID)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			tradeSnap, err := tx.Get(tradeRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return tradedom.ErrNotFound
				}
				return err
			}

			trade, err := docToTrade(tradeSnap)
			if err != nil {
				return err
			}
			if trade.ID != tradeID ||
				trade.SellerType != tradedom.SellerTypeAvatar ||
				trade.SellerAvatarID == "" {
				return tradedom.ErrConflict
			}

			agreementSnap, err := tx.Get(agreementRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return tradedom.ErrReturnAgreementNotFound
				}
				return err
			}

			agreement, err :=
				docToTradeReturnAgreement(agreementSnap)
			if err != nil {
				return err
			}

			if agreement.ID != tradeID ||
				agreement.TradeID != tradeID ||
				agreement.ID != shipment.ReturnAgreementID ||
				agreement.Status != tradedom.ReturnStatusAgreed ||
				agreement.AgreedAt == nil ||
				agreement.Proposal == nil ||
				agreement.Proposal.ID != shipment.ProposalID ||
				agreement.Proposal.Agreement !=
					tradedom.ReturnProposalAgreementAgree ||
				agreement.Proposal.RejectedAt != nil ||
				agreement.Proposal.ReturnRequirement !=
					tradedom.ReturnRequirementRequired {
				return tradedom.ErrReturnShipmentConflict
			}

			existingSnap, err := tx.Get(ref)
			if err == nil && existingSnap.Exists() {
				return tradedom.ErrReturnShipmentAlreadyExists
			}
			if err != nil &&
				status.Code(err) != codes.NotFound {
				return err
			}

			return tx.Create(
				ref,
				tradeReturnShipmentToDoc(shipment),
			)
		},
	)
	if err != nil {
		if errors.Is(
			err,
			tradedom.ErrReturnShipmentAlreadyExists,
		) || status.Code(err) == codes.AlreadyExists {
			return tradedom.ReturnShipment{},
				tradedom.ErrReturnShipmentAlreadyExists
		}

		return tradedom.ReturnShipment{}, err
	}

	return shipment, nil
}

// ============================================================
// Update
// ============================================================

func (r *TradeReturnShipmentRepositoryFS) Update(
	ctx context.Context,
	tradeID string,
	shipment tradedom.ReturnShipment,
) (tradedom.ReturnShipment, error) {
	if r == nil || r.Client == nil {
		return tradedom.ReturnShipment{},
			ErrTradeReturnShipmentRepositoryNotConfigured
	}
	if tradeID == "" {
		return tradedom.ReturnShipment{},
			tradedom.ErrReturnShipmentNotFound
	}

	if shipment.ID != tradeID ||
		shipment.TradeID != tradeID ||
		shipment.ReturnAgreementID != tradeID {
		return tradedom.ReturnShipment{},
			tradedom.ErrReturnShipmentConflict
	}

	normalizeTradeReturnShipmentTimes(&shipment)

	ref := r.col().Doc(tradeID)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snap, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return tradedom.ErrReturnShipmentNotFound
				}
				return err
			}

			current, err := docToTradeReturnShipment(snap)
			if err != nil {
				return err
			}

			if !sameTradeReturnShipmentIdentity(
				current,
				shipment,
				tradeID,
			) {
				return tradedom.ErrReturnShipmentConflict
			}

			if !shipment.UpdatedAt.IsZero() &&
				shipment.UpdatedAt.Before(current.UpdatedAt) {
				return tradedom.ErrReturnShipmentConflict
			}

			updated := current
			updated.Status = shipment.Status
			updated.ProviderShipmentID =
				shipment.ProviderShipmentID
			updated.QRCodePayload =
				shipment.QRCodePayload
			updated.TrackingNumber =
				shipment.TrackingNumber
			updated.CarrierStatus =
				shipment.CarrierStatus
			updated.QRCodeExpiresAt =
				cloneTradeReturnShipmentTimePtr(
					shipment.QRCodeExpiresAt,
				)
			updated.ReadyAt =
				cloneTradeReturnShipmentTimePtr(
					shipment.ReadyAt,
				)
			updated.ShippedAt =
				cloneTradeReturnShipmentTimePtr(
					shipment.ShippedAt,
				)
			updated.DeliveredAt =
				cloneTradeReturnShipmentTimePtr(
					shipment.DeliveredAt,
				)
			updated.CancelledAt =
				cloneTradeReturnShipmentTimePtr(
					shipment.CancelledAt,
				)

			if shipment.UpdatedAt.IsZero() {
				updated.UpdatedAt = time.Now().UTC()
			} else {
				updated.UpdatedAt =
					shipment.UpdatedAt.UTC()
			}

			normalizeTradeReturnShipmentTimes(&updated)

			if err := updated.ValidateForPersist(); err != nil {
				return err
			}

			return tx.Set(
				ref,
				tradeReturnShipmentToDoc(updated),
			)
		},
	)
	if err != nil {
		return tradedom.ReturnShipment{}, err
	}

	return r.GetByTradeID(ctx, tradeID)
}

// ============================================================
// Firestore documents
// ============================================================

type tradeReturnShipmentDoc struct {
	ID                string `firestore:"id"`
	TradeID           string `firestore:"tradeId"`
	ReturnAgreementID string `firestore:"returnAgreementId"`
	ProposalID        string `firestore:"proposalId"`

	Status        string `firestore:"status"`
	Carrier       string `firestore:"carrier"`
	DropOffMethod string `firestore:"dropOffMethod"`

	ProviderShipmentID string `firestore:"providerShipmentId,omitempty"`
	QRCodePayload      string `firestore:"qrCodePayload,omitempty"`
	TrackingNumber     string `firestore:"trackingNumber,omitempty"`
	CarrierStatus      string `firestore:"carrierStatus,omitempty"`

	QRCodeExpiresAt *time.Time `firestore:"qrCodeExpiresAt,omitempty"`
	ReadyAt         *time.Time `firestore:"readyAt,omitempty"`
	ShippedAt       *time.Time `firestore:"shippedAt,omitempty"`
	DeliveredAt     *time.Time `firestore:"deliveredAt,omitempty"`
	CancelledAt     *time.Time `firestore:"cancelledAt,omitempty"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

func tradeReturnShipmentToDoc(
	shipment tradedom.ReturnShipment,
) tradeReturnShipmentDoc {
	return tradeReturnShipmentDoc{
		ID:                shipment.ID,
		TradeID:           shipment.TradeID,
		ReturnAgreementID: shipment.ReturnAgreementID,
		ProposalID:        shipment.ProposalID,

		Status:        string(shipment.Status),
		Carrier:       string(shipment.Carrier),
		DropOffMethod: string(shipment.DropOffMethod),

		ProviderShipmentID: shipment.ProviderShipmentID,
		QRCodePayload:      shipment.QRCodePayload,
		TrackingNumber:     shipment.TrackingNumber,
		CarrierStatus:      shipment.CarrierStatus,

		QRCodeExpiresAt: cloneTradeReturnShipmentTimePtr(
			shipment.QRCodeExpiresAt,
		),
		ReadyAt: cloneTradeReturnShipmentTimePtr(
			shipment.ReadyAt,
		),
		ShippedAt: cloneTradeReturnShipmentTimePtr(
			shipment.ShippedAt,
		),
		DeliveredAt: cloneTradeReturnShipmentTimePtr(
			shipment.DeliveredAt,
		),
		CancelledAt: cloneTradeReturnShipmentTimePtr(
			shipment.CancelledAt,
		),

		CreatedAt: shipment.CreatedAt.UTC(),
		UpdatedAt: shipment.UpdatedAt.UTC(),
	}
}

func docToTradeReturnShipment(
	snap *firestore.DocumentSnapshot,
) (tradedom.ReturnShipment, error) {
	if snap == nil ||
		snap.Ref == nil ||
		!snap.Exists() {
		return tradedom.ReturnShipment{},
			tradedom.ErrReturnShipmentNotFound
	}

	var doc tradeReturnShipmentDoc
	if err := snap.DataTo(&doc); err != nil {
		return tradedom.ReturnShipment{}, err
	}

	shipment := tradedom.ReturnShipment{
		ID:                doc.ID,
		TradeID:           doc.TradeID,
		ReturnAgreementID: doc.ReturnAgreementID,
		ProposalID:        doc.ProposalID,

		Status: tradedom.ReturnShipmentStatus(
			doc.Status,
		),
		Carrier: tradedom.ReturnShipmentCarrier(
			doc.Carrier,
		),
		DropOffMethod: tradedom.ReturnShipmentDropOffMethod(
			doc.DropOffMethod,
		),

		ProviderShipmentID: doc.ProviderShipmentID,
		QRCodePayload:      doc.QRCodePayload,
		TrackingNumber:     doc.TrackingNumber,
		CarrierStatus:      doc.CarrierStatus,

		QRCodeExpiresAt: cloneTradeReturnShipmentTimePtr(
			doc.QRCodeExpiresAt,
		),
		ReadyAt: cloneTradeReturnShipmentTimePtr(
			doc.ReadyAt,
		),
		ShippedAt: cloneTradeReturnShipmentTimePtr(
			doc.ShippedAt,
		),
		DeliveredAt: cloneTradeReturnShipmentTimePtr(
			doc.DeliveredAt,
		),
		CancelledAt: cloneTradeReturnShipmentTimePtr(
			doc.CancelledAt,
		),

		CreatedAt: doc.CreatedAt.UTC(),
		UpdatedAt: doc.UpdatedAt.UTC(),
	}

	if shipment.ID != snap.Ref.ID ||
		shipment.TradeID != snap.Ref.ID ||
		shipment.ReturnAgreementID != snap.Ref.ID {
		return tradedom.ReturnShipment{},
			fmt.Errorf(
				"trade return shipment %s: %w: document id mismatch",
				snap.Ref.ID,
				ErrInvalidTradeReturnShipmentDocumentData,
			)
	}

	if err := shipment.ValidateForPersist(); err != nil {
		return tradedom.ReturnShipment{},
			fmt.Errorf(
				"trade return shipment %s: %w: %v",
				snap.Ref.ID,
				ErrInvalidTradeReturnShipmentDocumentData,
				err,
			)
	}

	return shipment, nil
}

// ============================================================
// Helpers
// ============================================================

func sameTradeReturnShipmentIdentity(
	current tradedom.ReturnShipment,
	incoming tradedom.ReturnShipment,
	tradeID string,
) bool {
	if current.ID != tradeID ||
		current.TradeID != tradeID ||
		current.ReturnAgreementID != tradeID ||
		incoming.ID != tradeID ||
		incoming.TradeID != tradeID ||
		incoming.ReturnAgreementID != tradeID {
		return false
	}

	return current.ProposalID == incoming.ProposalID &&
		current.Carrier == incoming.Carrier &&
		current.DropOffMethod == incoming.DropOffMethod &&
		current.CreatedAt.Equal(incoming.CreatedAt)
}

func normalizeTradeReturnShipmentTimes(
	shipment *tradedom.ReturnShipment,
) {
	if shipment == nil {
		return
	}

	if !shipment.CreatedAt.IsZero() {
		shipment.CreatedAt = shipment.CreatedAt.UTC()
	}
	if !shipment.UpdatedAt.IsZero() {
		shipment.UpdatedAt = shipment.UpdatedAt.UTC()
	}

	shipment.QRCodeExpiresAt =
		cloneTradeReturnShipmentTimePtr(
			shipment.QRCodeExpiresAt,
		)
	shipment.ReadyAt =
		cloneTradeReturnShipmentTimePtr(
			shipment.ReadyAt,
		)
	shipment.ShippedAt =
		cloneTradeReturnShipmentTimePtr(
			shipment.ShippedAt,
		)
	shipment.DeliveredAt =
		cloneTradeReturnShipmentTimePtr(
			shipment.DeliveredAt,
		)
	shipment.CancelledAt =
		cloneTradeReturnShipmentTimePtr(
			shipment.CancelledAt,
		)
}

func cloneTradeReturnShipmentTimePtr(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
