// backend/internal/adapters/out/firestore/trade_return_agreement_repository_fs.go

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
	ErrTradeReturnAgreementRepositoryNotConfigured = errors.New(
		"trade_return_agreement_repository_fs: not configured",
	)
	ErrInvalidTradeReturnAgreementDocumentData = errors.New(
		"trade_return_agreement_repository_fs: invalid document data",
	)
)

// TradeReturnAgreementRepositoryFS implements
// tradedom.ReturnAgreementRepository using Firestore.
//
// Firestore:
//
//	tradeReturnAgreements/{tradeId}
//
// One Avatar-to-Avatar Resale Trade may have at most one ReturnAgreement.
// The document ID, ReturnAgreement.ID and ReturnAgreement.TradeID are all
// identical to tradeId.
type TradeReturnAgreementRepositoryFS struct {
	Client *firestore.Client
}

var _ tradedom.ReturnAgreementRepository = (*TradeReturnAgreementRepositoryFS)(nil)

func NewTradeReturnAgreementRepositoryFS(
	client *firestore.Client,
) *TradeReturnAgreementRepositoryFS {
	return &TradeReturnAgreementRepositoryFS{
		Client: client,
	}
}

func (r *TradeReturnAgreementRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("tradeReturnAgreements")
}

func (r *TradeReturnAgreementRepositoryFS) tradesCol() *firestore.CollectionRef {
	return r.Client.Collection("trades")
}

// ============================================================
// Read
// ============================================================

func (r *TradeReturnAgreementRepositoryFS) GetByTradeID(
	ctx context.Context,
	tradeID string,
) (tradedom.ReturnAgreement, error) {
	if r == nil || r.Client == nil {
		return tradedom.ReturnAgreement{},
			ErrTradeReturnAgreementRepositoryNotConfigured
	}
	if tradeID == "" {
		return tradedom.ReturnAgreement{},
			tradedom.ErrReturnAgreementNotFound
	}

	snap, err := r.col().Doc(tradeID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tradedom.ReturnAgreement{},
				tradedom.ErrReturnAgreementNotFound
		}
		return tradedom.ReturnAgreement{}, err
	}

	agreement, err := docToTradeReturnAgreement(snap)
	if err != nil {
		return tradedom.ReturnAgreement{}, err
	}
	if agreement.ID != tradeID ||
		agreement.TradeID != tradeID {
		return tradedom.ReturnAgreement{},
			ErrInvalidTradeReturnAgreementDocumentData
	}

	return agreement, nil
}

// ============================================================
// Create
// ============================================================

func (r *TradeReturnAgreementRepositoryFS) Create(
	ctx context.Context,
	agreement tradedom.ReturnAgreement,
) (tradedom.ReturnAgreement, error) {
	if r == nil || r.Client == nil {
		return tradedom.ReturnAgreement{},
			ErrTradeReturnAgreementRepositoryNotConfigured
	}
	if agreement.TradeID == "" {
		return tradedom.ReturnAgreement{},
			tradedom.ErrInvalidReturnAgreementTradeID
	}

	tradeID := agreement.TradeID

	if agreement.ID == "" {
		agreement.ID = tradeID
	}
	if agreement.ID != tradeID {
		return tradedom.ReturnAgreement{},
			tradedom.ErrReturnAgreementConflict
	}

	if agreement.Consultation.ID == "" {
		agreement.Consultation.ID = r.col().NewDoc().ID
	}

	now := time.Now().UTC()

	if agreement.CreatedAt.IsZero() {
		agreement.CreatedAt = now
	} else {
		agreement.CreatedAt = agreement.CreatedAt.UTC()
	}

	if agreement.Consultation.CreatedAt.IsZero() {
		agreement.Consultation.CreatedAt = agreement.CreatedAt
	} else {
		agreement.Consultation.CreatedAt =
			agreement.Consultation.CreatedAt.UTC()
	}

	if agreement.UpdatedAt.IsZero() {
		agreement.UpdatedAt = agreement.CreatedAt
	} else {
		agreement.UpdatedAt = agreement.UpdatedAt.UTC()
	}

	normalizeTradeReturnAgreementTimes(&agreement)

	if err := agreement.ValidateForPersist(); err != nil {
		return tradedom.ReturnAgreement{}, err
	}

	ref := r.col().Doc(tradeID)
	tradeRef := r.tradesCol().Doc(tradeID)

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

			existingSnap, err := tx.Get(ref)
			if err == nil && existingSnap.Exists() {
				return tradedom.ErrReturnAgreementAlreadyExists
			}
			if err != nil &&
				status.Code(err) != codes.NotFound {
				return err
			}

			return tx.Create(
				ref,
				tradeReturnAgreementToDoc(agreement),
			)
		},
	)
	if err != nil {
		if errors.Is(
			err,
			tradedom.ErrReturnAgreementAlreadyExists,
		) || status.Code(err) == codes.AlreadyExists {
			return tradedom.ReturnAgreement{},
				tradedom.ErrReturnAgreementAlreadyExists
		}

		return tradedom.ReturnAgreement{}, err
	}

	return agreement, nil
}

// ============================================================
// Update
// ============================================================

func (r *TradeReturnAgreementRepositoryFS) Update(
	ctx context.Context,
	tradeID string,
	agreement tradedom.ReturnAgreement,
) (tradedom.ReturnAgreement, error) {
	if r == nil || r.Client == nil {
		return tradedom.ReturnAgreement{},
			ErrTradeReturnAgreementRepositoryNotConfigured
	}
	if tradeID == "" {
		return tradedom.ReturnAgreement{},
			tradedom.ErrReturnAgreementNotFound
	}

	if agreement.ID != tradeID ||
		agreement.TradeID != tradeID {
		return tradedom.ReturnAgreement{},
			tradedom.ErrReturnAgreementConflict
	}

	normalizeTradeReturnAgreementTimes(&agreement)

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
					return tradedom.ErrReturnAgreementNotFound
				}
				return err
			}

			current, err := docToTradeReturnAgreement(snap)
			if err != nil {
				return err
			}

			if !sameTradeReturnAgreementIdentity(
				current,
				agreement,
				tradeID,
			) {
				return tradedom.ErrReturnAgreementConflict
			}

			if !agreement.UpdatedAt.IsZero() &&
				agreement.UpdatedAt.Before(current.UpdatedAt) {
				return tradedom.ErrReturnAgreementConflict
			}

			updated := current
			updated.Status = agreement.Status
			updated.Proposal =
				cloneTradeReturnProposal(agreement.Proposal)
			updated.AgreedAt =
				cloneTradeReturnAgreementTimePtr(
					agreement.AgreedAt,
				)
			updated.ReturnShippedAt =
				cloneTradeReturnAgreementTimePtr(
					agreement.ReturnShippedAt,
				)
			updated.ReturnReceivedAt =
				cloneTradeReturnAgreementTimePtr(
					agreement.ReturnReceivedAt,
				)
			updated.RefundProcessingAt =
				cloneTradeReturnAgreementTimePtr(
					agreement.RefundProcessingAt,
				)
			updated.CompletedAt =
				cloneTradeReturnAgreementTimePtr(
					agreement.CompletedAt,
				)
			updated.DisputedAt =
				cloneTradeReturnAgreementTimePtr(
					agreement.DisputedAt,
				)

			if updated.Proposal != nil &&
				updated.Proposal.ID == "" {
				updated.Proposal.ID = r.col().NewDoc().ID
			}

			if agreement.UpdatedAt.IsZero() {
				updated.UpdatedAt = time.Now().UTC()
			} else {
				updated.UpdatedAt =
					agreement.UpdatedAt.UTC()
			}

			normalizeTradeReturnAgreementTimes(&updated)

			if err := updated.ValidateForPersist(); err != nil {
				return err
			}

			return tx.Set(
				ref,
				tradeReturnAgreementToDoc(updated),
			)
		},
	)
	if err != nil {
		return tradedom.ReturnAgreement{}, err
	}

	return r.GetByTradeID(ctx, tradeID)
}

// ============================================================
// Firestore documents
// ============================================================

type tradeReturnAgreementDoc struct {
	ID      string `firestore:"id"`
	TradeID string `firestore:"tradeId"`

	Status string `firestore:"status"`

	Consultation tradeReturnConsultationDoc `firestore:"consultation"`
	Proposal     *tradeReturnProposalDoc    `firestore:"proposal,omitempty"`

	AgreedAt           *time.Time `firestore:"agreedAt,omitempty"`
	ReturnShippedAt    *time.Time `firestore:"returnShippedAt,omitempty"`
	ReturnReceivedAt   *time.Time `firestore:"returnReceivedAt,omitempty"`
	RefundProcessingAt *time.Time `firestore:"refundProcessingAt,omitempty"`
	CompletedAt        *time.Time `firestore:"completedAt,omitempty"`
	DisputedAt         *time.Time `firestore:"disputedAt,omitempty"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

type tradeReturnConsultationDoc struct {
	ID        string    `firestore:"id"`
	Reason    string    `firestore:"reason"`
	Detail    string    `firestore:"detail"`
	CreatedAt time.Time `firestore:"createdAt"`
}

type tradeReturnProposalDoc struct {
	ID string `firestore:"id"`

	Agreement         string `firestore:"agreement"`
	ReturnRequirement string `firestore:"returnRequirement,omitempty"`
	RefundAmount      int    `firestore:"refundAmount,omitempty"`

	CreatedAt  time.Time  `firestore:"createdAt"`
	RejectedAt *time.Time `firestore:"rejectedAt,omitempty"`
}

func tradeReturnAgreementToDoc(
	agreement tradedom.ReturnAgreement,
) tradeReturnAgreementDoc {
	return tradeReturnAgreementDoc{
		ID:      agreement.ID,
		TradeID: agreement.TradeID,
		Status:  string(agreement.Status),

		Consultation: tradeReturnConsultationDoc{
			ID:        agreement.Consultation.ID,
			Reason:    string(agreement.Consultation.Reason),
			Detail:    agreement.Consultation.Detail,
			CreatedAt: agreement.Consultation.CreatedAt.UTC(),
		},
		Proposal: tradeReturnProposalToDoc(
			agreement.Proposal,
		),

		AgreedAt: cloneTradeReturnAgreementTimePtr(
			agreement.AgreedAt,
		),
		ReturnShippedAt: cloneTradeReturnAgreementTimePtr(
			agreement.ReturnShippedAt,
		),
		ReturnReceivedAt: cloneTradeReturnAgreementTimePtr(
			agreement.ReturnReceivedAt,
		),
		RefundProcessingAt: cloneTradeReturnAgreementTimePtr(
			agreement.RefundProcessingAt,
		),
		CompletedAt: cloneTradeReturnAgreementTimePtr(
			agreement.CompletedAt,
		),
		DisputedAt: cloneTradeReturnAgreementTimePtr(
			agreement.DisputedAt,
		),
		CreatedAt: agreement.CreatedAt.UTC(),
		UpdatedAt: agreement.UpdatedAt.UTC(),
	}
}

func tradeReturnProposalToDoc(
	proposal *tradedom.ReturnProposal,
) *tradeReturnProposalDoc {
	if proposal == nil {
		return nil
	}

	return &tradeReturnProposalDoc{
		ID:                proposal.ID,
		Agreement:         string(proposal.Agreement),
		ReturnRequirement: string(proposal.ReturnRequirement),
		RefundAmount:      proposal.RefundAmount,
		CreatedAt:         proposal.CreatedAt.UTC(),
		RejectedAt: cloneTradeReturnAgreementTimePtr(
			proposal.RejectedAt,
		),
	}
}

func docToTradeReturnAgreement(
	snap *firestore.DocumentSnapshot,
) (tradedom.ReturnAgreement, error) {
	if snap == nil ||
		snap.Ref == nil ||
		!snap.Exists() {
		return tradedom.ReturnAgreement{},
			tradedom.ErrReturnAgreementNotFound
	}

	var doc tradeReturnAgreementDoc
	if err := snap.DataTo(&doc); err != nil {
		return tradedom.ReturnAgreement{}, err
	}

	agreement := tradedom.ReturnAgreement{
		ID:      doc.ID,
		TradeID: doc.TradeID,
		Status:  tradedom.ReturnStatus(doc.Status),

		Consultation: tradedom.ReturnConsultation{
			ID: doc.Consultation.ID,
			Reason: tradedom.ReturnConsultationReason(
				doc.Consultation.Reason,
			),
			Detail:    doc.Consultation.Detail,
			CreatedAt: doc.Consultation.CreatedAt.UTC(),
		},
		Proposal: docToTradeReturnProposal(
			doc.Proposal,
		),

		AgreedAt: cloneTradeReturnAgreementTimePtr(
			doc.AgreedAt,
		),
		ReturnShippedAt: cloneTradeReturnAgreementTimePtr(
			doc.ReturnShippedAt,
		),
		ReturnReceivedAt: cloneTradeReturnAgreementTimePtr(
			doc.ReturnReceivedAt,
		),
		RefundProcessingAt: cloneTradeReturnAgreementTimePtr(
			doc.RefundProcessingAt,
		),
		CompletedAt: cloneTradeReturnAgreementTimePtr(
			doc.CompletedAt,
		),
		DisputedAt: cloneTradeReturnAgreementTimePtr(
			doc.DisputedAt,
		),
		CreatedAt: doc.CreatedAt.UTC(),
		UpdatedAt: doc.UpdatedAt.UTC(),
	}

	if agreement.ID != snap.Ref.ID ||
		agreement.TradeID != snap.Ref.ID {
		return tradedom.ReturnAgreement{},
			fmt.Errorf(
				"trade return agreement %s: %w: document id mismatch",
				snap.Ref.ID,
				ErrInvalidTradeReturnAgreementDocumentData,
			)
	}

	if err := agreement.ValidateForPersist(); err != nil {
		return tradedom.ReturnAgreement{},
			fmt.Errorf(
				"trade return agreement %s: %w: %v",
				snap.Ref.ID,
				ErrInvalidTradeReturnAgreementDocumentData,
				err,
			)
	}

	return agreement, nil
}

func docToTradeReturnProposal(
	doc *tradeReturnProposalDoc,
) *tradedom.ReturnProposal {
	if doc == nil {
		return nil
	}

	return &tradedom.ReturnProposal{
		ID: doc.ID,
		Agreement: tradedom.ReturnProposalAgreement(
			doc.Agreement,
		),
		ReturnRequirement: tradedom.ReturnRequirement(
			doc.ReturnRequirement,
		),
		RefundAmount: doc.RefundAmount,
		CreatedAt:    doc.CreatedAt.UTC(),
		RejectedAt: cloneTradeReturnAgreementTimePtr(
			doc.RejectedAt,
		),
	}
}

// ============================================================
// Helpers
// ============================================================

func sameTradeReturnAgreementIdentity(
	current tradedom.ReturnAgreement,
	incoming tradedom.ReturnAgreement,
	tradeID string,
) bool {
	if current.ID != tradeID ||
		current.TradeID != tradeID ||
		incoming.ID != tradeID ||
		incoming.TradeID != tradeID {
		return false
	}

	if !current.CreatedAt.Equal(incoming.CreatedAt) {
		return false
	}

	return sameTradeReturnConsultation(
		current.Consultation,
		incoming.Consultation,
	)
}

func sameTradeReturnConsultation(
	current tradedom.ReturnConsultation,
	incoming tradedom.ReturnConsultation,
) bool {
	return current.ID == incoming.ID &&
		current.Reason == incoming.Reason &&
		current.Detail == incoming.Detail &&
		current.CreatedAt.Equal(incoming.CreatedAt)
}

func normalizeTradeReturnAgreementTimes(
	agreement *tradedom.ReturnAgreement,
) {
	if agreement == nil {
		return
	}

	if !agreement.CreatedAt.IsZero() {
		agreement.CreatedAt = agreement.CreatedAt.UTC()
	}
	if !agreement.UpdatedAt.IsZero() {
		agreement.UpdatedAt = agreement.UpdatedAt.UTC()
	}
	if !agreement.Consultation.CreatedAt.IsZero() {
		agreement.Consultation.CreatedAt =
			agreement.Consultation.CreatedAt.UTC()
	}

	agreement.AgreedAt =
		cloneTradeReturnAgreementTimePtr(
			agreement.AgreedAt,
		)
	agreement.ReturnShippedAt =
		cloneTradeReturnAgreementTimePtr(
			agreement.ReturnShippedAt,
		)
	agreement.ReturnReceivedAt =
		cloneTradeReturnAgreementTimePtr(
			agreement.ReturnReceivedAt,
		)
	agreement.RefundProcessingAt =
		cloneTradeReturnAgreementTimePtr(
			agreement.RefundProcessingAt,
		)
	agreement.CompletedAt =
		cloneTradeReturnAgreementTimePtr(
			agreement.CompletedAt,
		)
	agreement.DisputedAt =
		cloneTradeReturnAgreementTimePtr(
			agreement.DisputedAt,
		)

	if agreement.Proposal != nil {
		if !agreement.Proposal.CreatedAt.IsZero() {
			agreement.Proposal.CreatedAt =
				agreement.Proposal.CreatedAt.UTC()
		}
		agreement.Proposal.RejectedAt =
			cloneTradeReturnAgreementTimePtr(
				agreement.Proposal.RejectedAt,
			)
	}
}

func cloneTradeReturnProposal(
	proposal *tradedom.ReturnProposal,
) *tradedom.ReturnProposal {
	if proposal == nil {
		return nil
	}

	return &tradedom.ReturnProposal{
		ID:                proposal.ID,
		Agreement:         proposal.Agreement,
		ReturnRequirement: proposal.ReturnRequirement,
		RefundAmount:      proposal.RefundAmount,
		CreatedAt:         proposal.CreatedAt.UTC(),
		RejectedAt: cloneTradeReturnAgreementTimePtr(
			proposal.RejectedAt,
		),
	}
}

func cloneTradeReturnAgreementTimePtr(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
