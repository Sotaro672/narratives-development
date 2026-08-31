// backend/internal/adapters/out/firestore/sales_receivable_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	salesreceivabledom "narratives/internal/domain/salesReceivable"
)

const salesReceivablesCollection = "salesReceivables"

// SalesReceivableRepositoryFS implements salesReceivable.Repository using
// Firestore.
//
// Persistence:
//
//	salesReceivables/{receivableId}
//
// The document ID is deterministic and must equal SalesReceivable.ID.
//
// Bank-account details must never be stored in this collection. The receivable
// stores only PayoutAccountID. The actual bank destination is snapshotted later
// when a BankPayout is created.
type SalesReceivableRepositoryFS struct {
	Client *firestore.Client
}

func NewSalesReceivableRepositoryFS(client *firestore.Client) *SalesReceivableRepositoryFS {
	return &SalesReceivableRepositoryFS{Client: client}
}

var _ salesreceivabledom.Repository = (*SalesReceivableRepositoryFS)(nil)

func (r *SalesReceivableRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(salesReceivablesCollection)
}

func (r *SalesReceivableRepositoryFS) doc(receivableID string) *firestore.DocumentRef {
	return r.col().Doc(receivableID)
}

// ============================================================
// Read
// ============================================================

func (r *SalesReceivableRepositoryFS) GetByID(
	ctx context.Context,
	receivableID string,
) (salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return salesreceivabledom.SalesReceivable{},
			errors.New("salesReceivable: firestore client is nil")
	}
	if receivableID == "" || strings.Contains(receivableID, "/") {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidID
	}

	snapshot, err := r.doc(receivableID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return salesreceivabledom.SalesReceivable{},
				salesreceivabledom.ErrNotFound
		}
		return salesreceivabledom.SalesReceivable{}, err
	}

	return docToSalesReceivable(snapshot)
}

func (r *SalesReceivableRepositoryFS) ListByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("salesReceivable: firestore client is nil")
	}
	if paymentID == "" {
		return nil, salesreceivabledom.ErrInvalidPaymentID
	}

	return r.listByField(ctx, "paymentId", paymentID)
}

func (r *SalesReceivableRepositoryFS) ListByOrderID(
	ctx context.Context,
	orderID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("salesReceivable: firestore client is nil")
	}
	if orderID == "" {
		return nil, salesreceivabledom.ErrInvalidOrderID
	}

	return r.listByField(ctx, "orderId", orderID)
}

func (r *SalesReceivableRepositoryFS) ListByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("salesReceivable: firestore client is nil")
	}
	if avatarID == "" {
		return nil, salesreceivabledom.ErrInvalidAvatarID
	}

	return r.listByField(ctx, "avatarId", avatarID)
}

func (r *SalesReceivableRepositoryFS) ListByUserID(
	ctx context.Context,
	userID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("salesReceivable: firestore client is nil")
	}
	if userID == "" {
		return nil, salesreceivabledom.ErrInvalidUserID
	}

	return r.listByField(ctx, "userId", userID)
}

func (r *SalesReceivableRepositoryFS) ListByPayoutAccountID(
	ctx context.Context,
	payoutAccountID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("salesReceivable: firestore client is nil")
	}
	if payoutAccountID == "" {
		return nil, salesreceivabledom.ErrInvalidPayoutAccountID
	}

	return r.listByField(ctx, "payoutAccountId", payoutAccountID)
}

func (r *SalesReceivableRepositoryFS) ListByBankPayoutID(
	ctx context.Context,
	bankPayoutID string,
) ([]salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("salesReceivable: firestore client is nil")
	}
	if bankPayoutID == "" || strings.Contains(bankPayoutID, "/") {
		return nil, salesreceivabledom.ErrInvalidBankPayoutID
	}

	return r.listByField(ctx, "bankPayoutId", bankPayoutID)
}

// ListAvailableByUserID returns receivables currently eligible for a future
// BankPayout.
//
// Firestore queries only UserID here. Status and BankPayoutID are filtered in
// memory so this operation does not require a composite index on
// userId + status.
//
// Reservation is intentionally not performed here. BankPayout creation must
// later reserve the selected receivables inside a Firestore transaction.
func (r *SalesReceivableRepositoryFS) ListAvailableByUserID(
	ctx context.Context,
	in salesreceivabledom.ListAvailableByUserIDInput,
) ([]salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("salesReceivable: firestore client is nil")
	}
	if in.UserID == "" {
		return nil, salesreceivabledom.ErrInvalidUserID
	}
	if in.Limit <= 0 {
		return nil, errors.New("salesReceivable: invalid available receivable limit")
	}

	receivables, err := r.listByField(ctx, "userId", in.UserID)
	if err != nil {
		return nil, err
	}

	result := make([]salesreceivabledom.SalesReceivable, 0)
	for _, receivable := range receivables {
		if receivable.UserID != in.UserID {
			continue
		}
		if receivable.Status != salesreceivabledom.StatusAvailable {
			continue
		}
		if receivable.BankPayoutID != "" {
			continue
		}

		result = append(result, receivable)
	}

	sort.Slice(result, func(i, j int) bool {
		left := result[i].AvailableAt
		right := result[j].AvailableAt

		if left == nil && right == nil {
			return result[i].ID < result[j].ID
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.Equal(*right) {
			return result[i].ID < result[j].ID
		}

		return left.Before(*right)
	})

	if len(result) > in.Limit {
		result = result[:in.Limit]
	}

	return result, nil
}

func (r *SalesReceivableRepositoryFS) listByField(
	ctx context.Context,
	field string,
	value string,
) ([]salesreceivabledom.SalesReceivable, error) {
	iter := r.col().
		Where(field, "==", value).
		Documents(ctx)
	defer iter.Stop()

	result := make([]salesreceivabledom.SalesReceivable, 0)

	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		receivable, err := docToSalesReceivable(snapshot)
		if err != nil {
			return nil, err
		}

		result = append(result, receivable)
	}

	// Avoid additional composite indexes. The query uses only one equality
	// filter and deterministic ordering is applied in application memory.
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

// ============================================================
// Create
// ============================================================

// Create persists a new pending SalesReceivable.
//
// The deterministic ID is used directly as the Firestore document ID.
// Existing documents are never overwritten.
func (r *SalesReceivableRepositoryFS) Create(
	ctx context.Context,
	receivable salesreceivabledom.SalesReceivable,
) (salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return salesreceivabledom.SalesReceivable{},
			errors.New("salesReceivable: firestore client is nil")
	}

	normalizeSalesReceivableTimestamps(&receivable)

	if err := receivable.Validate(); err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}
	if receivable.Status != salesreceivabledom.StatusPending {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidStatus
	}
	if receivable.BankPayoutID != "" ||
		receivable.AvailableAt != nil ||
		receivable.ReservedAt != nil ||
		receivable.PaidAt != nil ||
		receivable.CanceledAt != nil {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidStatus
	}

	expectedID, err := salesreceivabledom.NewID(
		receivable.PaymentID,
		receivable.PayoutAccountID,
	)
	if err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}
	if receivable.ID != expectedID {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidID
	}

	_, err = r.doc(receivable.ID).Create(ctx, receivable)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return salesreceivabledom.SalesReceivable{},
				salesreceivabledom.ErrConflict
		}
		return salesreceivabledom.SalesReceivable{}, err
	}

	return receivable, nil
}

// ============================================================
// Update
// ============================================================

// Update replaces one SalesReceivable after validating both immutable fields and
// the requested domain state transition.
//
// The current entity is read inside the transaction so concurrent payout,
// fulfillment, or cancellation operations cannot blindly overwrite each other.
func (r *SalesReceivableRepositoryFS) Update(
	ctx context.Context,
	receivable salesreceivabledom.SalesReceivable,
) (salesreceivabledom.SalesReceivable, error) {
	if r == nil || r.Client == nil {
		return salesreceivabledom.SalesReceivable{},
			errors.New("salesReceivable: firestore client is nil")
	}

	normalizeSalesReceivableTimestamps(&receivable)

	if err := receivable.Validate(); err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}
	if receivable.ID == "" || strings.Contains(receivable.ID, "/") {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidID
	}

	expectedID, err := salesreceivabledom.NewID(
		receivable.PaymentID,
		receivable.PayoutAccountID,
	)
	if err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}
	if receivable.ID != expectedID {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidID
	}

	ref := r.doc(receivable.ID)
	var result salesreceivabledom.SalesReceivable

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snapshot, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return salesreceivabledom.ErrNotFound
				}
				return err
			}

			current, err := docToSalesReceivable(snapshot)
			if err != nil {
				return err
			}

			if err := validateSalesReceivableImmutableFields(
				current,
				receivable,
			); err != nil {
				return err
			}

			if salesReceivableStateEqual(current, receivable) {
				result = current
				return nil
			}

			if err := validateSalesReceivableStatusTransition(
				current.Status,
				receivable.Status,
			); err != nil {
				return err
			}

			if receivable.UpdatedAt.Before(current.UpdatedAt) {
				return salesreceivabledom.ErrInvalidUpdatedAt
			}

			if err := receivable.Validate(); err != nil {
				return err
			}

			if err := tx.Set(ref, receivable); err != nil {
				return err
			}

			result = receivable
			return nil
		},
	)
	if err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}

	return result, nil
}

func validateSalesReceivableImmutableFields(
	current salesreceivabledom.SalesReceivable,
	next salesreceivabledom.SalesReceivable,
) error {
	if current.ID != next.ID ||
		current.OrderID != next.OrderID ||
		current.PaymentID != next.PaymentID ||
		current.AvatarID != next.AvatarID ||
		current.UserID != next.UserID ||
		current.PayoutAccountID != next.PayoutAccountID ||
		current.GrossAmount != next.GrossAmount ||
		current.PlatformFeeAmount != next.PlatformFeeAmount ||
		current.ReceivableAmount != next.ReceivableAmount ||
		current.Currency != next.Currency ||
		!current.CreatedAt.Equal(next.CreatedAt) {
		return salesreceivabledom.ErrConflict
	}

	return nil
}

func validateSalesReceivableStatusTransition(
	current salesreceivabledom.Status,
	next salesreceivabledom.Status,
) error {
	switch current {
	case salesreceivabledom.StatusPending:
		switch next {
		case salesreceivabledom.StatusAvailable,
			salesreceivabledom.StatusCanceled:
			return nil
		}

	case salesreceivabledom.StatusAvailable:
		switch next {
		case salesreceivabledom.StatusReserved,
			salesreceivabledom.StatusCanceled:
			return nil
		}

	case salesreceivabledom.StatusReserved:
		switch next {
		case salesreceivabledom.StatusAvailable,
			salesreceivabledom.StatusPaid:
			return nil
		}

	case salesreceivabledom.StatusPaid,
		salesreceivabledom.StatusCanceled:
		return salesreceivabledom.ErrInvalidStatusTransition
	}

	return salesreceivabledom.ErrInvalidStatusTransition
}

// ============================================================
// Mapping / normalization
// ============================================================

func docToSalesReceivable(
	snapshot *firestore.DocumentSnapshot,
) (salesreceivabledom.SalesReceivable, error) {
	if snapshot == nil || snapshot.Ref == nil {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidID
	}

	var receivable salesreceivabledom.SalesReceivable
	if err := snapshot.DataTo(&receivable); err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}

	normalizeSalesReceivableTimestamps(&receivable)

	if receivable.ID == "" ||
		receivable.ID != snapshot.Ref.ID {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidID
	}

	expectedID, err := salesreceivabledom.NewID(
		receivable.PaymentID,
		receivable.PayoutAccountID,
	)
	if err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}
	if receivable.ID != expectedID {
		return salesreceivabledom.SalesReceivable{},
			salesreceivabledom.ErrInvalidID
	}

	if err := receivable.Validate(); err != nil {
		return salesreceivabledom.SalesReceivable{}, err
	}

	return receivable, nil
}

// Firestore Timestamp values are persisted at microsecond precision.
//
// This is persistence-level precision normalization only. Seller identity,
// amounts, payout identity, status, and other business values are never
// normalized by the repository.
func normalizeSalesReceivableTimestamps(
	receivable *salesreceivabledom.SalesReceivable,
) {
	if receivable == nil {
		return
	}

	receivable.CreatedAt =
		normalizeSalesReceivableTimestamp(receivable.CreatedAt)
	receivable.UpdatedAt =
		normalizeSalesReceivableTimestamp(receivable.UpdatedAt)

	receivable.AvailableAt =
		normalizeSalesReceivableTimestampPointer(receivable.AvailableAt)
	receivable.ReservedAt =
		normalizeSalesReceivableTimestampPointer(receivable.ReservedAt)
	receivable.PaidAt =
		normalizeSalesReceivableTimestampPointer(receivable.PaidAt)
	receivable.CanceledAt =
		normalizeSalesReceivableTimestampPointer(receivable.CanceledAt)
}

func normalizeSalesReceivableTimestamp(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return value
	}

	return value.UTC().Truncate(time.Microsecond)
}

func normalizeSalesReceivableTimestampPointer(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	normalized := normalizeSalesReceivableTimestamp(*value)
	return &normalized
}

func salesReceivableStateEqual(
	left salesreceivabledom.SalesReceivable,
	right salesreceivabledom.SalesReceivable,
) bool {
	return left.ID == right.ID &&
		left.OrderID == right.OrderID &&
		left.PaymentID == right.PaymentID &&
		left.AvatarID == right.AvatarID &&
		left.UserID == right.UserID &&
		left.PayoutAccountID == right.PayoutAccountID &&
		left.GrossAmount == right.GrossAmount &&
		left.PlatformFeeAmount == right.PlatformFeeAmount &&
		left.ReceivableAmount == right.ReceivableAmount &&
		left.Currency == right.Currency &&
		left.Status == right.Status &&
		left.BankPayoutID == right.BankPayoutID &&
		left.CreatedAt.Equal(right.CreatedAt) &&
		left.UpdatedAt.Equal(right.UpdatedAt) &&
		timePointerEqual(left.AvailableAt, right.AvailableAt) &&
		timePointerEqual(left.ReservedAt, right.ReservedAt) &&
		timePointerEqual(left.PaidAt, right.PaidAt) &&
		timePointerEqual(left.CanceledAt, right.CanceledAt)
}

func timePointerEqual(
	left *time.Time,
	right *time.Time,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return left.Equal(*right)
}
