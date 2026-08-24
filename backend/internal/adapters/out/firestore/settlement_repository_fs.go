// backend/internal/adapters/out/firestore/settlement_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	settlementdom "narratives/internal/domain/settlement"
)

var (
	_ settlementdom.Repository = (*SettlementRepositoryFS)(nil)

	_ usecase.SettlementTransferRepository = (*SettlementRepositoryFS)(nil)
)

// ============================================================
// SettlementRepositoryFS
// ============================================================

// SettlementRepositoryFS is the Firestore-backed repository for seller-side
// Stripe Connect settlements.
//
// Firestore:
//
//	settlements/{settlementId}
//
// Settlement document ID:
//
//	{paymentId}_{accountId}
//
// Settlement is separate from:
//
// - payments: buyer -> AMOL Stripe Platform
// - Order item token transfer: Solana ownership transfer
//
// Settlement represents:
//
//	AMOL Stripe Platform -> Stripe Connected Account
type SettlementRepositoryFS struct {
	Client *firestore.Client
}

func NewSettlementRepositoryFS(
	client *firestore.Client,
) *SettlementRepositoryFS {
	return &SettlementRepositoryFS{
		Client: client,
	}
}

func (r *SettlementRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("settlements")
}

// ============================================================
// settlement.Repository
// ============================================================

func (r *SettlementRepositoryFS) GetByID(
	ctx context.Context,
	settlementID string,
) (settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return settlementdom.Settlement{},
			errors.New("settlement: firestore client is nil")
	}

	if settlementID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidID
	}

	snapshot, err := r.col().Doc(settlementID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return settlementdom.Settlement{},
				settlementdom.ErrNotFound
		}

		return settlementdom.Settlement{}, err
	}

	return docToSettlement(snapshot)
}

func (r *SettlementRepositoryFS) ListByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return nil,
			errors.New("settlement: firestore client is nil")
	}

	if paymentID == "" {
		return nil,
			settlementdom.ErrInvalidPaymentID
	}

	return r.listByField(
		ctx,
		"paymentId",
		paymentID,
	)
}

func (r *SettlementRepositoryFS) ListByOrderID(
	ctx context.Context,
	orderID string,
) ([]settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return nil,
			errors.New("settlement: firestore client is nil")
	}

	if orderID == "" {
		return nil,
			settlementdom.ErrInvalidOrderID
	}

	return r.listByField(
		ctx,
		"orderId",
		orderID,
	)
}

func (r *SettlementRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return nil,
			errors.New("settlement: firestore client is nil")
	}

	if companyID == "" {
		return nil,
			settlementdom.ErrInvalidCompanyID
	}

	return r.listByField(
		ctx,
		"companyId",
		companyID,
	)
}

func (r *SettlementRepositoryFS) ListByAccountID(
	ctx context.Context,
	accountID string,
) ([]settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return nil,
			errors.New("settlement: firestore client is nil")
	}

	if accountID == "" {
		return nil,
			settlementdom.ErrInvalidAccountID
	}

	return r.listByField(
		ctx,
		"accountId",
		accountID,
	)
}

// ListTransferCandidates returns Settlements that may require a transfer
// Cloud Task to be present or retried.
//
// Candidates:
//
//	ready
//	failed_retryable
//	transferring with UpdatedAt <= StaleBefore
//
// Firestore queries only by status here to avoid requiring an additional
// composite index. Stale filtering, deterministic ordering, deduplication,
// and Limit are applied in application memory.
func (r *SettlementRepositoryFS) ListTransferCandidates(
	ctx context.Context,
	in settlementdom.ListTransferCandidatesInput,
) ([]settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return nil,
			errors.New("settlement: firestore client is nil")
	}

	if in.StaleBefore.IsZero() {
		return nil,
			settlementdom.ErrInvalidUpdatedAt
	}

	if in.Limit <= 0 {
		return nil,
			errors.New(
				"settlement: invalid transfer candidate limit",
			)
	}

	staleBefore :=
		in.StaleBefore.UTC()

	statuses := []settlementdom.SettlementStatus{
		settlementdom.StatusReady,
		settlementdom.StatusFailedRetryable,
		settlementdom.StatusTransferring,
	}

	result := make(
		[]settlementdom.Settlement,
		0,
	)

	seen := make(
		map[string]struct{},
	)

	for _, candidateStatus := range statuses {
		settlements, err :=
			r.listByField(
				ctx,
				"status",
				string(candidateStatus),
			)
		if err != nil {
			return nil, err
		}

		for _, settlement := range settlements {
			if settlement.Status ==
				settlementdom.StatusTransferring &&
				settlement.UpdatedAt.After(
					staleBefore,
				) {
				continue
			}

			if _, exists :=
				seen[settlement.ID]; exists {
				continue
			}

			seen[settlement.ID] =
				struct{}{}

			result = append(
				result,
				settlement,
			)
		}
	}

	sort.Slice(
		result,
		func(i, j int) bool {
			if result[i].UpdatedAt.Equal(
				result[j].UpdatedAt,
			) {
				return result[i].ID <
					result[j].ID
			}

			return result[i].UpdatedAt.Before(
				result[j].UpdatedAt,
			)
		},
	)

	if len(result) >
		in.Limit {
		result =
			result[:in.Limit]
	}

	return result, nil
}

func (r *SettlementRepositoryFS) listByField(
	ctx context.Context,
	field string,
	value string,
) ([]settlementdom.Settlement, error) {
	iter := r.col().
		Where(field, "==", value).
		Documents(ctx)
	defer iter.Stop()

	result := make(
		[]settlementdom.Settlement,
		0,
	)

	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		settlement, err :=
			docToSettlement(snapshot)
		if err != nil {
			return nil, err
		}

		result = append(
			result,
			settlement,
		)
	}

	// Firestore側でOrderByを付けないことで不要なcomposite indexを
	// 増やさず、返却順だけをapplication上で決定的にする。
	sort.Slice(
		result,
		func(i, j int) bool {
			if result[i].CreatedAt.Equal(
				result[j].CreatedAt,
			) {
				return result[i].ID <
					result[j].ID
			}

			return result[i].CreatedAt.Before(
				result[j].CreatedAt,
			)
		},
	)

	return result, nil
}

func (r *SettlementRepositoryFS) Create(
	ctx context.Context,
	in settlementdom.CreateSettlementInput,
) (settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return settlementdom.Settlement{},
			errors.New("settlement: firestore client is nil")
	}

	in.Currency = strings.ToUpper(
		in.Currency,
	)

	if in.SettlementID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidID
	}

	now := time.Now().UTC()

	entity, err := settlementdom.New(
		in.SettlementID,
		in.OrderID,
		in.PaymentID,
		in.CompanyID,
		in.AccountID,
		in.StripeAccountID,
		in.StripePaymentIntentID,
		in.StripeChargeID,
		in.TransferGroup,
		in.GrossAmount,
		in.PlatformFeeAmount,
		in.TransferAmount,
		in.Currency,
		in.Status,
		now,
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	documentReference := r.col().Doc(
		entity.ID,
	)

	_, err = documentReference.Create(
		ctx,
		settlementToData(entity),
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return settlementdom.Settlement{},
				settlementdom.ErrConflict
		}

		return settlementdom.Settlement{}, err
	}

	return entity, nil
}

// UpdateByID performs a validated Settlement state transition.
//
// Financial transfer state must never be updated by blindly patching Firestore
// fields. The current Settlement is read inside a Firestore Transaction and
// the domain transition methods are used before the complete resulting entity
// is written.
//
// Transfer workers should normally use:
//
// - ClaimForTransfer
// - CompleteTransfer
// - FailTransfer
//
// directly.
func (r *SettlementRepositoryFS) UpdateByID(
	ctx context.Context,
	settlementID string,
	patch settlementdom.UpdateSettlementInput,
) (settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return settlementdom.Settlement{},
			errors.New("settlement: firestore client is nil")
	}

	if settlementID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidID
	}

	documentReference := r.col().Doc(
		settlementID,
	)

	var result settlementdom.Settlement

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			transaction *firestore.Transaction,
		) error {
			snapshot, err :=
				transaction.Get(documentReference)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return settlementdom.ErrNotFound
				}

				return err
			}

			current, err :=
				docToSettlement(snapshot)
			if err != nil {
				return err
			}

			next, changed, err :=
				applySettlementPatch(
					current,
					patch,
					time.Now().UTC(),
				)
			if err != nil {
				return err
			}

			if !changed {
				result = current
				return nil
			}

			if err := transaction.Set(
				documentReference,
				settlementToData(next),
			); err != nil {
				return err
			}

			result = next

			return nil
		},
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	return result, nil
}

// ============================================================
// usecase.SettlementTransferRepository
// ============================================================

// ClaimForTransfer atomically claims one Settlement for Stripe Transfer.
//
// The following states can be claimed:
//
//	ready
//	failed_retryable
//
// A transferring Settlement can also be reclaimed when its UpdatedAt is not
// after staleBefore. This recovers a worker crash after:
//
//	ready -> transferring
//
// but before the Stripe result was persisted.
//
// Stripe Transfer uses a deterministic idempotency key, so reclaiming a stale
// transfer does not intentionally create a second transfer.
//
// If another worker still owns a non-stale transferring claim, or the
// Settlement is already completed/terminal, Claimed=false is returned and
// Stripe must not be called by this invocation.
func (r *SettlementRepositoryFS) ClaimForTransfer(
	ctx context.Context,
	settlementID string,
	now time.Time,
	staleBefore time.Time,
) (usecase.ClaimSettlementTransferResult, error) {
	if r == nil || r.Client == nil {
		return usecase.ClaimSettlementTransferResult{},
			errors.New("settlement: firestore client is nil")
	}

	if settlementID == "" {
		return usecase.ClaimSettlementTransferResult{},
			settlementdom.ErrInvalidID
	}

	if now.IsZero() ||
		staleBefore.IsZero() {
		return usecase.ClaimSettlementTransferResult{},
			settlementdom.ErrInvalidUpdatedAt
	}

	now = now.UTC()
	staleBefore = staleBefore.UTC()

	if !staleBefore.Before(
		now,
	) {
		return usecase.ClaimSettlementTransferResult{},
			settlementdom.ErrInvalidUpdatedAt
	}

	documentReference := r.col().Doc(
		settlementID,
	)

	result :=
		usecase.ClaimSettlementTransferResult{}

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			transaction *firestore.Transaction,
		) error {
			snapshot, err :=
				transaction.Get(documentReference)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return settlementdom.ErrNotFound
				}

				return err
			}

			current, err :=
				docToSettlement(snapshot)
			if err != nil {
				return err
			}

			next := current

			switch current.Status {
			case settlementdom.StatusReady,
				settlementdom.StatusFailedRetryable:
				if err := next.StartTransfer(
					now,
				); err != nil {
					return err
				}

			case settlementdom.StatusTransferring:
				if current.UpdatedAt.After(
					staleBefore,
				) {
					result =
						usecase.ClaimSettlementTransferResult{
							Settlement: current,
							Claimed:    false,
						}

					return nil
				}

				if err := next.ReclaimTransfer(
					now,
				); err != nil {
					return err
				}

			default:
				result =
					usecase.ClaimSettlementTransferResult{
						Settlement: current,
						Claimed:    false,
					}

				return nil
			}

			if err := transaction.Set(
				documentReference,
				settlementToData(next),
			); err != nil {
				return err
			}

			result =
				usecase.ClaimSettlementTransferResult{
					Settlement: next,
					Claimed:    true,
				}

			return nil
		},
	)
	if err != nil {
		return usecase.ClaimSettlementTransferResult{},
			err
	}

	return result, nil
}

// CompleteTransfer atomically records a successful Stripe Connect Transfer.
//
// Idempotency:
//
// If the Settlement is already transferred with the same StripeTransferID,
// the current Settlement is returned as a successful no-op.
func (r *SettlementRepositoryFS) CompleteTransfer(
	ctx context.Context,
	settlementID string,
	stripeTransferID string,
	now time.Time,
) (settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return settlementdom.Settlement{},
			errors.New("settlement: firestore client is nil")
	}

	if settlementID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidID
	}

	if stripeTransferID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidStripeTransferID
	}

	if now.IsZero() {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidTransferredAt
	}

	now = now.UTC()

	documentReference := r.col().Doc(
		settlementID,
	)

	var result settlementdom.Settlement

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			transaction *firestore.Transaction,
		) error {
			snapshot, err :=
				transaction.Get(documentReference)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return settlementdom.ErrNotFound
				}

				return err
			}

			current, err :=
				docToSettlement(snapshot)
			if err != nil {
				return err
			}

			if current.Status ==
				settlementdom.StatusTransferred {
				if current.StripeTransferID ==
					stripeTransferID {
					result = current
					return nil
				}

				return settlementdom.
					ErrInvalidStripeTransferID
			}

			if current.Status !=
				settlementdom.StatusTransferring {
				return settlementdom.
					ErrInvalidStatusTransition
			}

			next := current

			if err := next.MarkTransferred(
				stripeTransferID,
				now,
			); err != nil {
				return err
			}

			if err := transaction.Set(
				documentReference,
				settlementToData(next),
			); err != nil {
				return err
			}

			result = next

			return nil
		},
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	return result, nil
}

// FailTransfer atomically records a Stripe Transfer failure.
//
// status must be:
//
//	failed_retryable
//	failed
//
// Only a Settlement currently in transferring state may enter either state.
//
// Repeated persistence of an already-recorded identical failure state is
// treated as a successful no-op.
func (r *SettlementRepositoryFS) FailTransfer(
	ctx context.Context,
	settlementID string,
	nextStatus settlementdom.SettlementStatus,
	errorType *string,
	errorCode *string,
	errorMsg *string,
	now time.Time,
) (settlementdom.Settlement, error) {
	if r == nil || r.Client == nil {
		return settlementdom.Settlement{},
			errors.New("settlement: firestore client is nil")
	}

	if settlementID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidID
	}

	if nextStatus !=
		settlementdom.StatusFailedRetryable &&
		nextStatus !=
			settlementdom.StatusFailed {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidStatusTransition
	}

	errorType = normalizeSettlementOptionalString(
		errorType,
	)
	errorCode = normalizeSettlementOptionalString(
		errorCode,
	)
	errorMsg = normalizeSettlementOptionalString(
		errorMsg,
	)

	if errorType == nil &&
		errorCode == nil &&
		errorMsg == nil {
		return settlementdom.Settlement{},
			settlementdom.ErrFailureReasonRequired
	}

	if now.IsZero() {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidUpdatedAt
	}

	now = now.UTC()

	documentReference := r.col().Doc(
		settlementID,
	)

	var result settlementdom.Settlement

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			transaction *firestore.Transaction,
		) error {
			snapshot, err :=
				transaction.Get(documentReference)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return settlementdom.ErrNotFound
				}

				return err
			}

			current, err :=
				docToSettlement(snapshot)
			if err != nil {
				return err
			}

			if current.Status == nextStatus &&
				optionalStringEqual(
					current.ErrorType,
					errorType,
				) &&
				optionalStringEqual(
					current.ErrorCode,
					errorCode,
				) &&
				optionalStringEqual(
					current.ErrorMsg,
					errorMsg,
				) {
				result = current
				return nil
			}

			if current.Status !=
				settlementdom.StatusTransferring {
				return settlementdom.
					ErrInvalidStatusTransition
			}

			next := current

			switch nextStatus {
			case settlementdom.StatusFailedRetryable:
				err = next.MarkFailedRetryable(
					errorType,
					errorCode,
					errorMsg,
					now,
				)

			case settlementdom.StatusFailed:
				err = next.MarkFailed(
					errorType,
					errorCode,
					errorMsg,
					now,
				)
			}

			if err != nil {
				return err
			}

			if err := transaction.Set(
				documentReference,
				settlementToData(next),
			); err != nil {
				return err
			}

			result = next

			return nil
		},
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	return result, nil
}

// ============================================================
// Update patch
// ============================================================

func applySettlementPatch(
	current settlementdom.Settlement,
	patch settlementdom.UpdateSettlementInput,
	now time.Time,
) (
	settlementdom.Settlement,
	bool,
	error,
) {
	if patch.Status == nil {
		if patch.StripeTransferID != nil ||
			patch.StripeTransferReversalID != nil ||
			patch.ErrorType != nil ||
			patch.ErrorCode != nil ||
			patch.ErrorMsg != nil {
			return settlementdom.Settlement{},
				false,
				settlementdom.ErrInvalidStatusTransition
		}

		return current, false, nil
	}

	if now.IsZero() {
		return settlementdom.Settlement{},
			false,
			settlementdom.ErrInvalidUpdatedAt
	}

	now = now.UTC()

	nextStatus := *patch.Status

	// Idempotent no-op when no state-specific fields are changing.
	if current.Status == nextStatus &&
		patch.StripeTransferID == nil &&
		patch.StripeTransferReversalID == nil &&
		patch.ErrorType == nil &&
		patch.ErrorCode == nil &&
		patch.ErrorMsg == nil {
		return current, false, nil
	}

	next := current

	switch nextStatus {
	case settlementdom.StatusPending:
		if current.Status !=
			settlementdom.StatusPending {
			return settlementdom.Settlement{},
				false,
				settlementdom.ErrInvalidStatusTransition
		}

		return current, false, nil

	case settlementdom.StatusReady:
		if err := next.MarkReady(
			now,
		); err != nil {
			return settlementdom.Settlement{},
				false,
				err
		}

	case settlementdom.StatusTransferring:
		if err := next.StartTransfer(
			now,
		); err != nil {
			return settlementdom.Settlement{},
				false,
				err
		}

	case settlementdom.StatusTransferred:
		if patch.StripeTransferID == nil {
			return settlementdom.Settlement{},
				false,
				settlementdom.ErrInvalidStripeTransferID
		}

		stripeTransferID :=
			*patch.StripeTransferID

		if current.Status ==
			settlementdom.StatusTransferred {
			if current.StripeTransferID ==
				stripeTransferID {
				return current, false, nil
			}

			return settlementdom.Settlement{},
				false,
				settlementdom.ErrInvalidStripeTransferID
		}

		if err := next.MarkTransferred(
			stripeTransferID,
			now,
		); err != nil {
			return settlementdom.Settlement{},
				false,
				err
		}

	case settlementdom.StatusFailedRetryable:
		if err := next.MarkFailedRetryable(
			normalizeSettlementOptionalString(
				patch.ErrorType,
			),
			normalizeSettlementOptionalString(
				patch.ErrorCode,
			),
			normalizeSettlementOptionalString(
				patch.ErrorMsg,
			),
			now,
		); err != nil {
			return settlementdom.Settlement{},
				false,
				err
		}

	case settlementdom.StatusFailed:
		if err := next.MarkFailed(
			normalizeSettlementOptionalString(
				patch.ErrorType,
			),
			normalizeSettlementOptionalString(
				patch.ErrorCode,
			),
			normalizeSettlementOptionalString(
				patch.ErrorMsg,
			),
			now,
		); err != nil {
			return settlementdom.Settlement{},
				false,
				err
		}

	case settlementdom.StatusCanceled:
		if err := next.Cancel(
			now,
		); err != nil {
			return settlementdom.Settlement{},
				false,
				err
		}

	case settlementdom.StatusReversed:
		if patch.StripeTransferReversalID == nil {
			return settlementdom.Settlement{},
				false,
				settlementdom.
					ErrInvalidStripeTransferReversalID
		}

		reversalID :=
			*patch.StripeTransferReversalID

		if current.Status ==
			settlementdom.StatusReversed {
			if current.StripeTransferReversalID ==
				reversalID {
				return current, false, nil
			}

			return settlementdom.Settlement{},
				false,
				settlementdom.
					ErrInvalidStripeTransferReversalID
		}

		if err := next.MarkReversed(
			reversalID,
			now,
		); err != nil {
			return settlementdom.Settlement{},
				false,
				err
		}

	default:
		return settlementdom.Settlement{},
			false,
			settlementdom.ErrInvalidStatus
	}

	if patch.StripeTransferID != nil &&
		nextStatus !=
			settlementdom.StatusTransferred {
		value :=
			*patch.StripeTransferID

		if value != next.StripeTransferID {
			return settlementdom.Settlement{},
				false,
				settlementdom.ErrInvalidStripeTransferID
		}
	}

	if patch.StripeTransferReversalID != nil &&
		nextStatus !=
			settlementdom.StatusReversed {
		value :=
			*patch.StripeTransferReversalID

		if value !=
			next.StripeTransferReversalID {
			return settlementdom.Settlement{},
				false,
				settlementdom.
					ErrInvalidStripeTransferReversalID
		}
	}

	if err := next.Validate(); err != nil {
		return settlementdom.Settlement{},
			false,
			err
	}

	return next, true, nil
}

// ============================================================
// Firestore conversion
// ============================================================

func settlementToData(
	settlement settlementdom.Settlement,
) map[string]any {
	data := map[string]any{
		"orderId":               settlement.OrderID,
		"paymentId":             settlement.PaymentID,
		"companyId":             settlement.CompanyID,
		"accountId":             settlement.AccountID,
		"stripeAccountId":       settlement.StripeAccountID,
		"stripePaymentIntentId": settlement.StripePaymentIntentID,
		"stripeChargeId":        settlement.StripeChargeID,
		"transferGroup":         settlement.TransferGroup,
		"grossAmount":           settlement.GrossAmount,
		"platformFeeAmount":     settlement.PlatformFeeAmount,
		"transferAmount":        settlement.TransferAmount,
		"currency":              settlement.Currency,
		"status":                string(settlement.Status),
		"createdAt":             settlement.CreatedAt,
		"updatedAt":             settlement.UpdatedAt,
	}

	if settlement.StripeTransferID != "" {
		data["stripeTransferId"] =
			settlement.StripeTransferID
	}

	if settlement.StripeTransferReversalID != "" {
		data["stripeTransferReversalId"] =
			settlement.StripeTransferReversalID
	}

	if settlement.ErrorType != nil {
		data["errorType"] =
			*settlement.ErrorType
	}

	if settlement.ErrorCode != nil {
		data["errorCode"] =
			*settlement.ErrorCode
	}

	if settlement.ErrorMsg != nil {
		data["errorMsg"] =
			*settlement.ErrorMsg
	}

	if settlement.TransferredAt != nil {
		data["transferredAt"] =
			settlement.TransferredAt.UTC()
	}

	if settlement.ReversedAt != nil {
		data["reversedAt"] =
			settlement.ReversedAt.UTC()
	}

	return data
}

func docToSettlement(
	document *firestore.DocumentSnapshot,
) (settlementdom.Settlement, error) {
	if document == nil {
		return settlementdom.Settlement{},
			errors.New(
				"settlement: document snapshot is nil",
			)
	}

	data := document.Data()
	if data == nil {
		return settlementdom.Settlement{},
			fmt.Errorf(
				"settlement: empty document %s",
				document.Ref.ID,
			)
	}

	id := document.Ref.ID

	orderID, err :=
		settlementRequiredString(
			data,
			"orderId",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	paymentID, err :=
		settlementRequiredString(
			data,
			"paymentId",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	companyID, err :=
		settlementRequiredString(
			data,
			"companyId",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	accountID, err :=
		settlementRequiredString(
			data,
			"accountId",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	stripeAccountID, err :=
		settlementRequiredString(
			data,
			"stripeAccountId",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	stripePaymentIntentID, err :=
		settlementRequiredString(
			data,
			"stripePaymentIntentId",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	stripeChargeID, err :=
		settlementRequiredString(
			data,
			"stripeChargeId",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	transferGroup, err :=
		settlementRequiredString(
			data,
			"transferGroup",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	grossAmount, err :=
		settlementRequiredInt(
			data,
			"grossAmount",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	platformFeeAmount, err :=
		settlementRequiredInt(
			data,
			"platformFeeAmount",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	transferAmount, err :=
		settlementRequiredInt(
			data,
			"transferAmount",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	currency, err :=
		settlementRequiredString(
			data,
			"currency",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	statusText, err :=
		settlementRequiredString(
			data,
			"status",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	createdAt, err :=
		settlementRequiredTime(
			data,
			"createdAt",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	updatedAt, err :=
		settlementRequiredTime(
			data,
			"updatedAt",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	stripeTransferID :=
		settlementOptionalStringValue(
			data,
			"stripeTransferId",
		)

	stripeTransferReversalID :=
		settlementOptionalStringValue(
			data,
			"stripeTransferReversalId",
		)

	errorType :=
		settlementOptionalString(
			data,
			"errorType",
		)

	errorCode :=
		settlementOptionalString(
			data,
			"errorCode",
		)

	errorMsg :=
		settlementOptionalString(
			data,
			"errorMsg",
		)

	transferredAt, err :=
		settlementOptionalTime(
			data,
			"transferredAt",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	reversedAt, err :=
		settlementOptionalTime(
			data,
			"reversedAt",
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	entity := settlementdom.Settlement{
		ID: id,

		OrderID:   orderID,
		PaymentID: paymentID,

		CompanyID: companyID,
		AccountID: accountID,

		StripeAccountID: stripeAccountID,

		StripePaymentIntentID: stripePaymentIntentID,
		StripeChargeID:        stripeChargeID,
		StripeTransferID:      stripeTransferID,

		StripeTransferReversalID: stripeTransferReversalID,

		TransferGroup: transferGroup,

		GrossAmount:       grossAmount,
		PlatformFeeAmount: platformFeeAmount,
		TransferAmount:    transferAmount,

		Currency: currency,

		Status: settlementdom.SettlementStatus(
			statusText,
		),

		ErrorType: errorType,
		ErrorCode: errorCode,
		ErrorMsg:  errorMsg,

		CreatedAt: createdAt,
		UpdatedAt: updatedAt,

		TransferredAt: transferredAt,
		ReversedAt:    reversedAt,
	}

	if err := entity.Validate(); err != nil {
		return settlementdom.Settlement{}, err
	}

	return entity, nil
}

// ============================================================
// Firestore field helpers
// ============================================================

func settlementRequiredString(
	values map[string]any,
	key string,
) (string, error) {
	value, exists := values[key]
	if !exists || value == nil {
		return "", fmt.Errorf(
			"settlement: missing %s",
			key,
		)
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf(
			"settlement: invalid %s",
			key,
		)
	}

	if text == "" {
		return "", fmt.Errorf(
			"settlement: invalid %s",
			key,
		)
	}

	return text, nil
}

func settlementOptionalString(
	values map[string]any,
	key string,
) *string {
	value, exists := values[key]
	if !exists || value == nil {
		return nil
	}

	text, ok := value.(string)
	if !ok {
		return nil
	}

	if text == "" {
		return nil
	}

	return &text
}

func settlementOptionalStringValue(
	values map[string]any,
	key string,
) string {
	value :=
		settlementOptionalString(
			values,
			key,
		)

	if value == nil {
		return ""
	}

	return *value
}

func normalizeSettlementOptionalString(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	if *value == "" {
		return nil
	}

	return value
}

func optionalStringEqual(
	left *string,
	right *string,
) bool {
	left =
		normalizeSettlementOptionalString(
			left,
		)

	right =
		normalizeSettlementOptionalString(
			right,
		)

	if left == nil || right == nil {
		return left == nil &&
			right == nil
	}

	return *left == *right
}

func settlementRequiredInt(
	values map[string]any,
	key string,
) (int, error) {
	value, exists := values[key]
	if !exists || value == nil {
		return 0, fmt.Errorf(
			"settlement: missing %s",
			key,
		)
	}

	switch number := value.(type) {
	case int64:
		maxInt := int64(
			int(^uint(0) >> 1),
		)

		if number >
			maxInt {
			return 0, fmt.Errorf(
				"settlement: invalid %s",
				key,
			)
		}

		return int(number), nil

	case int:
		return number, nil

	default:
		return 0, fmt.Errorf(
			"settlement: invalid %s",
			key,
		)
	}
}

func settlementRequiredTime(
	values map[string]any,
	key string,
) (time.Time, error) {
	value, exists := values[key]
	if !exists || value == nil {
		return time.Time{}, fmt.Errorf(
			"settlement: missing %s",
			key,
		)
	}

	timestamp, ok := value.(time.Time)
	if !ok || timestamp.IsZero() {
		return time.Time{}, fmt.Errorf(
			"settlement: invalid %s",
			key,
		)
	}

	return timestamp.UTC(), nil
}

func settlementOptionalTime(
	values map[string]any,
	key string,
) (*time.Time, error) {
	value, exists := values[key]
	if !exists || value == nil {
		return nil, nil
	}

	timestamp, ok := value.(time.Time)
	if !ok || timestamp.IsZero() {
		return nil, fmt.Errorf(
			"settlement: invalid %s",
			key,
		)
	}

	timestamp = timestamp.UTC()

	return &timestamp, nil
}
