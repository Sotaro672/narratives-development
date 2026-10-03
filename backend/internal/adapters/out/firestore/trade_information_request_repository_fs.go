// backend/internal/adapters/out/firestore/trade_information_request_repository_fs.go

package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tradedom "narratives/internal/domain/trade"
)

const tradeInformationRequestsCollection = "tradeInformationRequests"

var (
	ErrTradeInformationRequestRepositoryNotConfigured = errors.New(
		"trade_information_request_repository_fs: not configured",
	)
	ErrInvalidTradeInformationRequestDocumentData = errors.New(
		"trade_information_request_repository_fs: invalid document data",
	)
)

// TradeInformationRequestRepositoryFS implements
// tradedom.InformationRequestRepository using Firestore.
//
// Firestore:
//
//	tradeInformationRequests/{tradeId}
//
// One Trade has at most one InformationRequest. The Firestore document ID,
// InformationRequest.ID and InformationRequest.TradeID are identical to tradeId.
//
// InformationRequest does not own Mall Trade-message unread state. The
// application layer is responsible for creating the corresponding system
// message and the Mall Trade query uses the pending request state to keep the
// message action counter active until the target Avatar submits a response.
type TradeInformationRequestRepositoryFS struct {
	Client *firestore.Client
}

var _ tradedom.InformationRequestRepository = (*TradeInformationRequestRepositoryFS)(nil)

func NewTradeInformationRequestRepositoryFS(
	client *firestore.Client,
) *TradeInformationRequestRepositoryFS {
	return &TradeInformationRequestRepositoryFS{
		Client: client,
	}
}

func (r *TradeInformationRequestRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(tradeInformationRequestsCollection)
}

// ============================================================
// Read
// ============================================================

func (r *TradeInformationRequestRepositoryFS) GetByTradeID(
	ctx context.Context,
	tradeID string,
) (tradedom.InformationRequest, error) {
	if r == nil || r.Client == nil {
		return tradedom.InformationRequest{},
			ErrTradeInformationRequestRepositoryNotConfigured
	}
	if tradeID == "" {
		return tradedom.InformationRequest{},
			tradedom.ErrInformationRequestNotFound
	}
	if strings.ContainsAny(tradeID, " \t\r\n") ||
		strings.Contains(tradeID, "/") ||
		strings.Contains(tradeID, "://") {
		return tradedom.InformationRequest{},
			tradedom.ErrInvalidInformationRequestTradeID
	}

	snap, err := r.col().Doc(tradeID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tradedom.InformationRequest{},
				tradedom.ErrInformationRequestNotFound
		}

		return tradedom.InformationRequest{}, err
	}

	request, err := docToTradeInformationRequest(snap)
	if err != nil {
		return tradedom.InformationRequest{}, err
	}
	if request.ID != tradeID ||
		request.TradeID != tradeID {
		return tradedom.InformationRequest{},
			ErrInvalidTradeInformationRequestDocumentData
	}

	return request, nil
}

// ============================================================
// Create
// ============================================================

func (r *TradeInformationRequestRepositoryFS) Create(
	ctx context.Context,
	request tradedom.InformationRequest,
) (tradedom.InformationRequest, error) {
	if r == nil || r.Client == nil {
		return tradedom.InformationRequest{},
			ErrTradeInformationRequestRepositoryNotConfigured
	}
	if request.TradeID == "" {
		return tradedom.InformationRequest{},
			tradedom.ErrInvalidInformationRequestTradeID
	}
	if strings.ContainsAny(request.TradeID, " \t\r\n") ||
		strings.Contains(request.TradeID, "/") ||
		strings.Contains(request.TradeID, "://") {
		return tradedom.InformationRequest{},
			tradedom.ErrInvalidInformationRequestTradeID
	}

	if request.ID == "" {
		request.ID = request.TradeID
	}
	if request.ID != request.TradeID {
		return tradedom.InformationRequest{},
			tradedom.ErrInformationRequestConflict
	}

	now := time.Now().UTC()

	if request.CreatedAt.IsZero() {
		request.CreatedAt = now
	} else {
		request.CreatedAt = request.CreatedAt.UTC()
	}

	if request.UpdatedAt.IsZero() {
		request.UpdatedAt = request.CreatedAt
	} else {
		request.UpdatedAt = request.UpdatedAt.UTC()
	}

	request.SubmittedAt = cloneTradeInformationRequestTimePtr(
		request.SubmittedAt,
	)

	if request.Status != tradedom.InformationRequestStatusPending {
		return tradedom.InformationRequest{},
			tradedom.ErrInvalidInformationRequestStatus
	}
	if err := request.ValidateForPersist(); err != nil {
		return tradedom.InformationRequest{}, err
	}

	ref := r.col().Doc(request.TradeID)

	_, err := ref.Create(
		ctx,
		tradeInformationRequestToDoc(request),
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return tradedom.InformationRequest{},
				tradedom.ErrInformationRequestAlreadyExists
		}

		return tradedom.InformationRequest{}, err
	}

	return r.GetByTradeID(ctx, request.TradeID)
}

// ============================================================
// Update
// ============================================================

func (r *TradeInformationRequestRepositoryFS) Update(
	ctx context.Context,
	tradeID string,
	request tradedom.InformationRequest,
) (tradedom.InformationRequest, error) {
	if r == nil || r.Client == nil {
		return tradedom.InformationRequest{},
			ErrTradeInformationRequestRepositoryNotConfigured
	}
	if tradeID == "" {
		return tradedom.InformationRequest{},
			tradedom.ErrInformationRequestNotFound
	}
	if strings.ContainsAny(tradeID, " \t\r\n") ||
		strings.Contains(tradeID, "/") ||
		strings.Contains(tradeID, "://") {
		return tradedom.InformationRequest{},
			tradedom.ErrInvalidInformationRequestTradeID
	}
	if request.ID != tradeID ||
		request.TradeID != tradeID {
		return tradedom.InformationRequest{},
			tradedom.ErrInformationRequestConflict
	}

	if !request.CreatedAt.IsZero() {
		request.CreatedAt = request.CreatedAt.UTC()
	}
	if !request.UpdatedAt.IsZero() {
		request.UpdatedAt = request.UpdatedAt.UTC()
	}
	request.SubmittedAt = cloneTradeInformationRequestTimePtr(
		request.SubmittedAt,
	)

	if err := request.ValidateForPersist(); err != nil {
		return tradedom.InformationRequest{}, err
	}

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
					return tradedom.ErrInformationRequestNotFound
				}

				return err
			}

			current, err := docToTradeInformationRequest(snap)
			if err != nil {
				return err
			}

			if !sameTradeInformationRequestIdentity(
				current,
				request,
				tradeID,
			) {
				return tradedom.ErrInformationRequestConflict
			}

			if current.Status !=
				tradedom.InformationRequestStatusPending {
				return tradedom.ErrInformationRequestConflict
			}
			if request.Status !=
				tradedom.InformationRequestStatusSubmitted {
				return tradedom.ErrInformationRequestConflict
			}
			if request.UpdatedAt.Before(current.UpdatedAt) {
				return tradedom.ErrInformationRequestConflict
			}

			updated := current
			updated.Response = request.Response
			updated.Status = request.Status
			updated.SubmittedAt = cloneTradeInformationRequestTimePtr(
				request.SubmittedAt,
			)
			updated.UpdatedAt = request.UpdatedAt.UTC()

			if err := updated.ValidateForPersist(); err != nil {
				return err
			}

			return tx.Set(
				ref,
				tradeInformationRequestToDoc(updated),
			)
		},
	)
	if err != nil {
		return tradedom.InformationRequest{}, err
	}

	return r.GetByTradeID(ctx, tradeID)
}

// ============================================================
// Firestore document
// ============================================================

type tradeInformationRequestDoc struct {
	ID      string `firestore:"id"`
	TradeID string `firestore:"tradeId"`

	TargetSide     string `firestore:"targetSide"`
	TargetAvatarID string `firestore:"targetAvatarId"`

	Reason   string `firestore:"reason"`
	Response string `firestore:"response,omitempty"`

	Status string `firestore:"status"`

	CreatedAt   time.Time  `firestore:"createdAt"`
	SubmittedAt *time.Time `firestore:"submittedAt,omitempty"`
	UpdatedAt   time.Time  `firestore:"updatedAt"`
}

func tradeInformationRequestToDoc(
	request tradedom.InformationRequest,
) tradeInformationRequestDoc {
	return tradeInformationRequestDoc{
		ID:             request.ID,
		TradeID:        request.TradeID,
		TargetSide:     string(request.TargetSide),
		TargetAvatarID: request.TargetAvatarID,
		Reason:         request.Reason,
		Response:       request.Response,
		Status:         string(request.Status),
		CreatedAt:      request.CreatedAt.UTC(),
		SubmittedAt: cloneTradeInformationRequestTimePtr(
			request.SubmittedAt,
		),
		UpdatedAt: request.UpdatedAt.UTC(),
	}
}

func docToTradeInformationRequest(
	snap *firestore.DocumentSnapshot,
) (tradedom.InformationRequest, error) {
	if snap == nil ||
		snap.Ref == nil ||
		!snap.Exists() {
		return tradedom.InformationRequest{},
			tradedom.ErrInformationRequestNotFound
	}

	var doc tradeInformationRequestDoc
	if err := snap.DataTo(&doc); err != nil {
		return tradedom.InformationRequest{}, err
	}

	request := tradedom.InformationRequest{
		ID:             doc.ID,
		TradeID:        doc.TradeID,
		TargetSide:     tradedom.InformationRequestTargetSide(doc.TargetSide),
		TargetAvatarID: doc.TargetAvatarID,
		Reason:         doc.Reason,
		Response:       doc.Response,
		Status:         tradedom.InformationRequestStatus(doc.Status),
		CreatedAt:      doc.CreatedAt.UTC(),
		SubmittedAt: cloneTradeInformationRequestTimePtr(
			doc.SubmittedAt,
		),
		UpdatedAt: doc.UpdatedAt.UTC(),
	}

	if request.ID != snap.Ref.ID ||
		request.TradeID != snap.Ref.ID {
		return tradedom.InformationRequest{},
			fmt.Errorf(
				"trade information request %s: %w: document id mismatch",
				snap.Ref.ID,
				ErrInvalidTradeInformationRequestDocumentData,
			)
	}

	if err := request.ValidateForPersist(); err != nil {
		return tradedom.InformationRequest{},
			fmt.Errorf(
				"trade information request %s: %w: %v",
				snap.Ref.ID,
				ErrInvalidTradeInformationRequestDocumentData,
				err,
			)
	}

	return request, nil
}

// ============================================================
// Invariants
// ============================================================

func sameTradeInformationRequestIdentity(
	current tradedom.InformationRequest,
	incoming tradedom.InformationRequest,
	tradeID string,
) bool {
	return current.ID == tradeID &&
		current.TradeID == tradeID &&
		incoming.ID == tradeID &&
		incoming.TradeID == tradeID &&
		current.TargetSide == incoming.TargetSide &&
		current.TargetAvatarID == incoming.TargetAvatarID &&
		current.Reason == incoming.Reason &&
		current.CreatedAt.Equal(incoming.CreatedAt)
}

func cloneTradeInformationRequestTimePtr(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
