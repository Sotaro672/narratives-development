// backend/internal/domain/payoutAccount/repository_port.go

package payoutAccount

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("payoutAccount: not found")
	ErrConflict = errors.New("payoutAccount: conflict")
)

// Repository defines persistence operations for a Mall user's payout account.
//
// Persistence policy:
//   - Firestore collection: payoutAccounts
//   - document ID: UserID
//   - one User has at most one PayoutAccount
//
// Repository implementations must not generate a separate PayoutAccount ID.
// UserID is the aggregate identity and persistence document ID.
type Repository interface {
	// GetByUserID returns the payout account belonging to userID.
	//
	// If no payout account exists, ErrNotFound must be returned.
	GetByUserID(
		ctx context.Context,
		userID string,
	) (PayoutAccount, error)

	// Create persists a new payout account.
	//
	// Implementations must reject creation when payoutAccounts/{UserID}
	// already exists and return ErrConflict.
	Create(
		ctx context.Context,
		account PayoutAccount,
	) (PayoutAccount, error)

	// Update replaces the persisted payout account state for the same UserID.
	//
	// Implementations must return ErrNotFound when payoutAccounts/{UserID}
	// does not exist.
	//
	// UserID must not be changed by an update.
	Update(
		ctx context.Context,
		account PayoutAccount,
	) (PayoutAccount, error)
}
