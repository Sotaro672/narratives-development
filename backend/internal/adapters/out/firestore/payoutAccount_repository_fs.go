// backend/internal/adapters/out/firestore/payoutAccount_repository_fs.go

package firestore

import (
	"context"
	"errors"
	"strings"

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
type PayoutAccountRepositoryFS struct {
	Client *firestore.Client
}

func NewPayoutAccountRepositoryFS(
	client *firestore.Client,
) *PayoutAccountRepositoryFS {
	return &PayoutAccountRepositoryFS{
		Client: client,
	}
}

// Compile-time interface check.
var _ payoutdom.Repository = (*PayoutAccountRepositoryFS)(nil)

func (r *PayoutAccountRepositoryFS) doc(
	userID string,
) *firestore.DocumentRef {
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

	userID = strings.TrimSpace(userID)
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

	// document ID and persisted UserID must always identify the same owner.
	if strings.TrimSpace(account.UserID) != userID {
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
// exists, ErrConflict is returned so a second Stripe Connected Account is not
// silently associated with the same User.
func (r *PayoutAccountRepositoryFS) Create(
	ctx context.Context,
	account payoutdom.PayoutAccount,
) (payoutdom.PayoutAccount, error) {
	if r == nil || r.Client == nil {
		return payoutdom.PayoutAccount{},
			errors.New("payoutAccount: repository is nil")
	}

	account.UserID = strings.TrimSpace(account.UserID)
	account.StripeAccountID = strings.TrimSpace(account.StripeAccountID)
	account.BankName = strings.TrimSpace(account.BankName)
	account.BankLast4 = strings.TrimSpace(account.BankLast4)

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
// StripeAccountID may only change through the domain's controlled
// SetStripeAccountID path before this repository is called.
func (r *PayoutAccountRepositoryFS) Update(
	ctx context.Context,
	account payoutdom.PayoutAccount,
) (payoutdom.PayoutAccount, error) {
	if r == nil || r.Client == nil {
		return payoutdom.PayoutAccount{},
			errors.New("payoutAccount: repository is nil")
	}

	account.UserID = strings.TrimSpace(account.UserID)
	account.StripeAccountID = strings.TrimSpace(account.StripeAccountID)
	account.BankName = strings.TrimSpace(account.BankName)
	account.BankLast4 = strings.TrimSpace(account.BankLast4)

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

			current.UserID = strings.TrimSpace(current.UserID)

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
