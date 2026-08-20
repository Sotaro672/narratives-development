// backend/internal/domain/transportation/repository_port.go
package transportation

import (
	"context"
	"errors"
)

// RepositoryPort defines the persistence contract for TransportationFeeSetting.
//
// Identity:
//   - one company may own multiple TransportationFeeSettings;
//   - TransportationFeeSetting.ID is the aggregate identity;
//   - the Firestore document ID must be the same value as TransportationFeeSetting.ID;
//   - CompanyID identifies the owning company and is immutable after creation;
//   - ID is immutable after creation.
//
// Persistence rules:
//   - Create must never overwrite an existing document with the same ID;
//   - Update must never create a missing document;
//   - repository implementations must call setting.Validate() before writing;
//   - the complete aggregate is persisted atomically because all 47
//     PrefectureRates are required and IslandRates belong to the same
//     TransportationFeeSetting aggregate;
//   - ListByCompanyID returns only settings owned by the specified company.
//
// Ownership authorization is handled by the application layer.
// Repository methods expose persistence operations and ownership data through
// TransportationFeeSetting.CompanyID.
type RepositoryPort interface {
	// GetByID returns one TransportationFeeSetting by its aggregate ID.
	//
	// If id is invalid, ErrInvalidID is returned.
	// If no setting exists with the specified ID, ErrNotFound is returned.
	GetByID(
		ctx context.Context,
		id string,
	) (*TransportationFeeSetting, error)

	// ListByCompanyID returns all TransportationFeeSettings owned by one company.
	//
	// If companyID is invalid, ErrInvalidCompanyID is returned.
	// If the company has no settings, an empty slice is returned.
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]TransportationFeeSetting, error)

	// Create persists a new TransportationFeeSetting.
	//
	// setting.ID is used as the Firestore document ID.
	// setting.CompanyID is persisted as the owner company field.
	// The setting must satisfy all domain invariants before persistence,
	// including ID, CompanyID, Name, all 47 PrefectureRates, IslandRates,
	// and timestamps.
	//
	// If a document already exists with the same setting.ID,
	// ErrConflict is returned.
	// Multiple settings with the same CompanyID are allowed.
	// Create must not overwrite an existing document.
	Create(
		ctx context.Context,
		setting TransportationFeeSetting,
	) (*TransportationFeeSetting, error)

	// Update replaces an existing TransportationFeeSetting atomically.
	//
	// Update is not an upsert. If the setting does not exist,
	// ErrNotFound is returned.
	// ID, CompanyID, and CreatedAt must remain unchanged from the
	// persisted aggregate.
	// Name, PrefectureRates, IslandRates, and UpdatedAt may be updated.
	// UpdatedAt must not be earlier than CreatedAt.
	//
	// The repository implementation must persist Name, PrefectureRates,
	// IslandRates, and UpdatedAt as one aggregate so that readers never
	// observe a partial transportation configuration.
	Update(
		ctx context.Context,
		setting TransportationFeeSetting,
	) (*TransportationFeeSetting, error)
}

// Repository errors.
//
// Adapter implementations may wrap these errors. Callers should use errors.Is
// rather than comparing error strings.
var (
	ErrNotFound = errors.New("transportation: not found")
	ErrConflict = errors.New("transportation: conflict")
)
