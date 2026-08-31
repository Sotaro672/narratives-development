// backend/internal/adapters/out/firestore/payoutAccount_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	payoutdom "narratives/internal/domain/payoutAccount"
)

const payoutAccountsCollection = "payoutAccounts"

// PayoutAccountRepositoryFS implements payoutAccount.Repository using Firestore.
//
// Persistence:
//   - collection: payoutAccounts
//   - document ID: UserID
//   - one User has at most one PayoutAccount
//   - plaintext bank account numbers are never persisted
//   - AccountNumberCiphertext contains only the encrypted account number
//   - BankLast4 is persisted separately for display
type PayoutAccountRepositoryFS struct {
	Client *firestore.Client
}

func NewPayoutAccountRepositoryFS(client *firestore.Client) *PayoutAccountRepositoryFS {
	return &PayoutAccountRepositoryFS{
		Client: client,
	}
}

// Compile-time interface check.
var _ payoutdom.Repository = (*PayoutAccountRepositoryFS)(nil)

func (r *PayoutAccountRepositoryFS) doc(userID string) *firestore.DocumentRef {
	return r.Client.
		Collection(payoutAccountsCollection).
		Doc(userID)
}

// GetByUserID returns the payout account belonging to userID.
func (r *PayoutAccountRepositoryFS) GetByUserID(
	ctx context.Context,
	userID string,
) (payoutdom.PayoutAccount, error) {
	if r == nil || r.Client == nil {
		return payoutdom.PayoutAccount{},
			errors.New("payoutAccount: repository is nil")
	}
	if userID == "" {
		return payoutdom.PayoutAccount{},
			payoutdom.ErrInvalidUserID
	}

	doc, err := r.doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return payoutdom.PayoutAccount{},
				payoutdom.ErrNotFound
		}

		return payoutdom.PayoutAccount{}, err
	}

	var account payoutdom.PayoutAccount
	if err := doc.DataTo(&account); err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	normalizePayoutAccountTimestamps(&account)

	// Document ID and persisted UserID must always identify the same owner.
	if account.UserID != userID {
		return payoutdom.PayoutAccount{},
			payoutdom.ErrInvalidUserID
	}
	if err := account.Validate(); err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	return account, nil
}

// Create persists a new payout account.
//
// payoutAccounts/{UserID} is created atomically. If the document already
// exists, ErrConflict is returned.
//
// AccountNumberCiphertext must already contain the encrypted bank account
// number. This repository must never receive or persist a plaintext account
// number.
func (r *PayoutAccountRepositoryFS) Create(
	ctx context.Context,
	account payoutdom.PayoutAccount,
) (payoutdom.PayoutAccount, error) {
	if r == nil || r.Client == nil {
		return payoutdom.PayoutAccount{},
			errors.New("payoutAccount: repository is nil")
	}

	// Firestore Timestamp is persisted at microsecond precision.
	// Normalize before validation and persistence so the value returned from
	// Create is identical to the value that will be read back from Firestore.
	normalizePayoutAccountTimestamps(&account)

	if err := account.Validate(); err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	ref := r.doc(account.UserID)

	_, err := ref.Create(ctx, account)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return payoutdom.PayoutAccount{},
				payoutdom.ErrConflict
		}

		return payoutdom.PayoutAccount{}, err
	}

	return account, nil
}

// Update replaces the persisted payout account state.
//
// Update is executed in a Firestore transaction so that:
//   - the document must already exist
//   - UserID cannot change
//   - CreatedAt cannot change
//
// Bank-account destination fields, including the encrypted account number, may
// change when the user replaces the registered payout destination.
func (r *PayoutAccountRepositoryFS) Update(
	ctx context.Context,
	account payoutdom.PayoutAccount,
) (payoutdom.PayoutAccount, error) {
	if r == nil || r.Client == nil {
		return payoutdom.PayoutAccount{},
			errors.New("payoutAccount: repository is nil")
	}

	normalizePayoutAccountTimestamps(&account)

	if err := account.Validate(); err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	ref := r.doc(account.UserID)

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			doc, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return payoutdom.ErrNotFound
				}

				return err
			}

			var current payoutdom.PayoutAccount
			if err := doc.DataTo(&current); err != nil {
				return err
			}

			// Normalize both persisted and incoming timestamps to the same
			// Firestore-compatible precision before comparison.
			normalizePayoutAccountTimestamps(&current)

			if current.UserID != account.UserID {
				return payoutdom.ErrInvalidUserID
			}

			// CreatedAt is immutable after initial creation.
			if !current.CreatedAt.Equal(account.CreatedAt) {
				return payoutdom.ErrInvalidCreatedAt
			}

			return tx.Set(ref, account)
		},
	)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	return account, nil
}

func normalizePayoutAccountTimestamps(
	account *payoutdom.PayoutAccount,
) {
	if account == nil {
		return
	}

	account.CreatedAt = normalizePayoutAccountTimestamp(account.CreatedAt)
	account.UpdatedAt = normalizePayoutAccountTimestamp(account.UpdatedAt)
}

func normalizePayoutAccountTimestamp(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return value
	}

	return value.UTC().Truncate(time.Microsecond)
}
