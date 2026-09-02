// backend/internal/adapters/out/firestore/bank_payout_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bankpayoutdom "narratives/internal/domain/bankPayout"
)

const bankPayoutsCollection = "bankPayouts"

// BankPayoutRepositoryFS implements bankPayout.Repository using Firestore.
//
// Persistence:
//
//	bankPayouts/{bankPayoutId}
//
// One item-level SalesReceivable creates at most one BankPayout.
//
// The document ID is deterministic and must equal:
//
//	bankPayout.NewID(SalesReceivableID)
//
// Bank destination data is snapshotted inside the BankPayout at creation time.
// AccountNumberCiphertext remains encrypted at rest. Plaintext account numbers
// must never be persisted by this repository.
type BankPayoutRepositoryFS struct {
	Client *firestore.Client
}

func NewBankPayoutRepositoryFS(
	client *firestore.Client,
) *BankPayoutRepositoryFS {
	return &BankPayoutRepositoryFS{Client: client}
}

var _ bankpayoutdom.Repository = (*BankPayoutRepositoryFS)(nil)

func (r *BankPayoutRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(bankPayoutsCollection)
}

func (r *BankPayoutRepositoryFS) doc(
	payoutID string,
) *firestore.DocumentRef {
	return r.col().Doc(payoutID)
}

// ============================================================
// Read
// ============================================================

func (r *BankPayoutRepositoryFS) GetByID(
	ctx context.Context,
	payoutID string,
) (bankpayoutdom.BankPayout, error) {
	if r == nil || r.Client == nil {
		return bankpayoutdom.BankPayout{},
			errors.New("bankPayout: firestore client is nil")
	}
	if payoutID == "" || strings.Contains(payoutID, "/") {
		return bankpayoutdom.BankPayout{},
			bankpayoutdom.ErrInvalidID
	}

	snapshot, err := r.doc(payoutID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return bankpayoutdom.BankPayout{},
				bankpayoutdom.ErrNotFound
		}

		return bankpayoutdom.BankPayout{}, err
	}

	return docToBankPayout(snapshot)
}

func (r *BankPayoutRepositoryFS) GetBySalesReceivableID(
	ctx context.Context,
	salesReceivableID string,
) (bankpayoutdom.BankPayout, error) {
	if r == nil || r.Client == nil {
		return bankpayoutdom.BankPayout{},
			errors.New("bankPayout: firestore client is nil")
	}

	payoutID, err := bankpayoutdom.NewID(salesReceivableID)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	return r.GetByID(ctx, payoutID)
}

// ============================================================
// Create
// ============================================================

// CreateIfAbsent persists a new pending BankPayout using its deterministic ID.
//
// If the document already exists and its immutable payout scope matches the
// candidate, the existing document is returned unchanged with created=false.
// This is the main idempotency boundary for repeated payout initiation.
//
// In particular, an existing processing or paid BankPayout is never reset to
// pending by a retry of the original create request.
func (r *BankPayoutRepositoryFS) CreateIfAbsent(
	ctx context.Context,
	payout bankpayoutdom.BankPayout,
) (
	bankpayoutdom.BankPayout,
	bool,
	error,
) {
	if r == nil || r.Client == nil {
		return bankpayoutdom.BankPayout{}, false,
			errors.New("bankPayout: firestore client is nil")
	}

	normalizeBankPayoutTimestamps(&payout)

	if err := validateNewBankPayout(payout); err != nil {
		return bankpayoutdom.BankPayout{}, false, err
	}

	ref := r.doc(payout.ID)

	var (
		result  bankpayoutdom.BankPayout
		created bool
	)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snapshot, err := tx.Get(ref)
			if err == nil {
				existing, err := docToBankPayout(snapshot)
				if err != nil {
					return err
				}

				if err := validateBankPayoutImmutableFields(
					existing,
					payout,
				); err != nil {
					return err
				}

				result = existing
				created = false
				return nil
			}

			if status.Code(err) != codes.NotFound {
				return err
			}

			if err := tx.Create(ref, payout); err != nil {
				if status.Code(err) == codes.AlreadyExists {
					return bankpayoutdom.ErrConflict
				}
				return err
			}

			result = payout
			created = true
			return nil
		},
	)
	if err != nil {
		return bankpayoutdom.BankPayout{}, false, err
	}

	return result, created, nil
}

func validateNewBankPayout(
	payout bankpayoutdom.BankPayout,
) error {
	if err := payout.Validate(); err != nil {
		return err
	}

	expectedID, err := bankpayoutdom.NewID(
		payout.SalesReceivableID,
	)
	if err != nil {
		return err
	}
	if payout.ID != expectedID {
		return bankpayoutdom.ErrInvalidID
	}

	if payout.Status != bankpayoutdom.StatusPending ||
		payout.ProviderPayoutID != "" ||
		payout.ProcessingAt != nil ||
		payout.PaidAt != nil ||
		payout.ErrorType != nil ||
		payout.ErrorCode != nil ||
		payout.ErrorMsg != nil {
		return bankpayoutdom.ErrInvalidStatus
	}

	return nil
}

// ============================================================
// Update
// ============================================================

// Update replaces one BankPayout after validating immutable fields and the
// requested lifecycle transition.
//
// The current entity is read inside the Firestore transaction so concurrent
// workers cannot blindly overwrite a newer payout result.
//
// This repository intentionally does not mutate SalesReceivable. Coordination
// between BankPayout and SalesReceivable must be performed through a separate
// shared transactional boundary.
func (r *BankPayoutRepositoryFS) Update(
	ctx context.Context,
	payout bankpayoutdom.BankPayout,
) (bankpayoutdom.BankPayout, error) {
	if r == nil || r.Client == nil {
		return bankpayoutdom.BankPayout{},
			errors.New("bankPayout: firestore client is nil")
	}

	normalizeBankPayoutTimestamps(&payout)

	if err := payout.Validate(); err != nil {
		return bankpayoutdom.BankPayout{}, err
	}
	if payout.ID == "" || strings.Contains(payout.ID, "/") {
		return bankpayoutdom.BankPayout{},
			bankpayoutdom.ErrInvalidID
	}

	expectedID, err := bankpayoutdom.NewID(
		payout.SalesReceivableID,
	)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}
	if payout.ID != expectedID {
		return bankpayoutdom.BankPayout{},
			bankpayoutdom.ErrInvalidID
	}

	ref := r.doc(payout.ID)
	var result bankpayoutdom.BankPayout

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snapshot, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return bankpayoutdom.ErrNotFound
				}
				return err
			}

			current, err := docToBankPayout(snapshot)
			if err != nil {
				return err
			}

			if err := validateBankPayoutImmutableFields(
				current,
				payout,
			); err != nil {
				return err
			}

			if bankPayoutStateEqual(current, payout) {
				result = current
				return nil
			}

			if err := validateBankPayoutStatusTransition(
				current.Status,
				payout.Status,
			); err != nil {
				return err
			}

			if payout.UpdatedAt.Before(current.UpdatedAt) {
				return bankpayoutdom.ErrInvalidUpdatedAt
			}

			if err := payout.Validate(); err != nil {
				return err
			}

			if err := tx.Set(ref, payout); err != nil {
				return err
			}

			result = payout
			return nil
		},
	)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	return result, nil
}

func validateBankPayoutImmutableFields(
	current bankpayoutdom.BankPayout,
	next bankpayoutdom.BankPayout,
) error {
	if current.ID != next.ID ||
		current.SalesReceivableID != next.SalesReceivableID ||
		current.OrderID != next.OrderID ||
		current.PaymentID != next.PaymentID ||
		current.OrderItemIndex != next.OrderItemIndex ||
		current.ResaleID != next.ResaleID ||
		current.SellerUserID != next.SellerUserID ||
		current.PayoutAccountID != next.PayoutAccountID ||
		!bankDestinationEqual(
			current.BankDestination,
			next.BankDestination,
		) ||
		current.Amount != next.Amount ||
		current.Currency != next.Currency ||
		!current.CreatedAt.Equal(next.CreatedAt) {
		return bankpayoutdom.ErrConflict
	}

	return nil
}

func validateBankPayoutStatusTransition(
	current bankpayoutdom.Status,
	next bankpayoutdom.Status,
) error {
	switch current {
	case bankpayoutdom.StatusPending:
		if next == bankpayoutdom.StatusProcessing {
			return nil
		}

	case bankpayoutdom.StatusProcessing:
		switch next {
		case bankpayoutdom.StatusProcessing,
			bankpayoutdom.StatusPaid,
			bankpayoutdom.StatusFailedRetryable,
			bankpayoutdom.StatusFailed:
			return nil
		}

	case bankpayoutdom.StatusFailedRetryable:
		if next == bankpayoutdom.StatusProcessing {
			return nil
		}

	case bankpayoutdom.StatusPaid,
		bankpayoutdom.StatusFailed:
		return bankpayoutdom.ErrInvalidStatusTransition
	}

	return bankpayoutdom.ErrInvalidStatusTransition
}

// ============================================================
// Mapping / normalization
// ============================================================

func docToBankPayout(
	snapshot *firestore.DocumentSnapshot,
) (bankpayoutdom.BankPayout, error) {
	if snapshot == nil || snapshot.Ref == nil {
		return bankpayoutdom.BankPayout{},
			bankpayoutdom.ErrInvalidID
	}

	var payout bankpayoutdom.BankPayout
	if err := snapshot.DataTo(&payout); err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	normalizeBankPayoutTimestamps(&payout)

	if payout.ID == "" ||
		payout.ID != snapshot.Ref.ID {
		return bankpayoutdom.BankPayout{},
			bankpayoutdom.ErrInvalidID
	}

	expectedID, err := bankpayoutdom.NewID(
		payout.SalesReceivableID,
	)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}
	if payout.ID != expectedID {
		return bankpayoutdom.BankPayout{},
			bankpayoutdom.ErrInvalidID
	}

	if err := payout.Validate(); err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	return payout, nil
}

// Firestore Timestamp values are persisted at microsecond precision.
//
// This only normalizes persistence precision. Payout identity, bank
// destination, amount, provider result, status, and failure information are
// never normalized by the repository.
func normalizeBankPayoutTimestamps(
	payout *bankpayoutdom.BankPayout,
) {
	if payout == nil {
		return
	}

	payout.CreatedAt =
		normalizeBankPayoutTimestamp(payout.CreatedAt)
	payout.UpdatedAt =
		normalizeBankPayoutTimestamp(payout.UpdatedAt)

	payout.ProcessingAt =
		normalizeBankPayoutTimestampPointer(payout.ProcessingAt)
	payout.PaidAt =
		normalizeBankPayoutTimestampPointer(payout.PaidAt)
}

func normalizeBankPayoutTimestamp(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return value
	}

	return value.UTC().Truncate(time.Microsecond)
}

func normalizeBankPayoutTimestampPointer(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	normalized := normalizeBankPayoutTimestamp(*value)
	return &normalized
}

func bankPayoutStateEqual(
	left bankpayoutdom.BankPayout,
	right bankpayoutdom.BankPayout,
) bool {
	return left.ID == right.ID &&
		left.SalesReceivableID == right.SalesReceivableID &&
		left.OrderID == right.OrderID &&
		left.PaymentID == right.PaymentID &&
		left.OrderItemIndex == right.OrderItemIndex &&
		left.ResaleID == right.ResaleID &&
		left.SellerUserID == right.SellerUserID &&
		left.PayoutAccountID == right.PayoutAccountID &&
		bankDestinationEqual(
			left.BankDestination,
			right.BankDestination,
		) &&
		left.Amount == right.Amount &&
		left.Currency == right.Currency &&
		left.Status == right.Status &&
		left.ProviderPayoutID == right.ProviderPayoutID &&
		optionalStringPointerEqual(
			left.ErrorType,
			right.ErrorType,
		) &&
		optionalStringPointerEqual(
			left.ErrorCode,
			right.ErrorCode,
		) &&
		optionalStringPointerEqual(
			left.ErrorMsg,
			right.ErrorMsg,
		) &&
		left.CreatedAt.Equal(right.CreatedAt) &&
		left.UpdatedAt.Equal(right.UpdatedAt) &&
		bankPayoutTimePointerEqual(
			left.ProcessingAt,
			right.ProcessingAt,
		) &&
		bankPayoutTimePointerEqual(
			left.PaidAt,
			right.PaidAt,
		)
}

func bankDestinationEqual(
	left bankpayoutdom.BankDestinationSnapshot,
	right bankpayoutdom.BankDestinationSnapshot,
) bool {
	return left.BankCode == right.BankCode &&
		left.BankName == right.BankName &&
		left.BranchCode == right.BranchCode &&
		left.BranchName == right.BranchName &&
		left.AccountType == right.AccountType &&
		left.AccountNumberCiphertext ==
			right.AccountNumberCiphertext &&
		left.BankLast4 == right.BankLast4 &&
		left.AccountHolderName == right.AccountHolderName
}

func optionalStringPointerEqual(
	left *string,
	right *string,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func bankPayoutTimePointerEqual(
	left *time.Time,
	right *time.Time,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return left.Equal(*right)
}
