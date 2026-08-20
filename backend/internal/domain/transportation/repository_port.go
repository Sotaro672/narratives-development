// backend/internal/domain/transportation/repository_port.go
package transportation

import (
	"context"
	"errors"
)

// RepositoryPort defines the persistence contract for TransportationFeeSetting.
//
// Identity:
//   - one company has at most one TransportationFeeSetting;
//   - TransportationFeeSetting.CompanyID is the aggregate identity;
//   - the Firestore document ID must be the same value as CompanyID;
//   - CompanyID is immutable after creation.
//
// This repository intentionally does not embed the common generic CRUD/List
// repository contracts because this aggregate is company-scoped configuration:
// it has no independent ID, does not require list operations, and should not
// expose deletion as a normal domain operation.
//
// Persistence rules:
//   - Create must never overwrite an existing document;
//   - Update must never create a missing document;
//   - repository implementations must call setting.Validate() before writing;
//   - the complete aggregate is persisted atomically because all 47
//     PrefectureRates are required and IslandRates are overrides belonging to
//     the same configuration aggregate.
type RepositoryPort interface {
	// GetByCompanyID returns the shipping fee setting for one company.
	//
	// If companyID is invalid, ErrInvalidCompanyID is returned.
	// If no setting exists for the company, ErrNotFound is returned.
	GetByCompanyID(
		ctx context.Context,
		companyID string,
	) (*TransportationFeeSetting, error)

	// ExistsByCompanyID reports whether a shipping fee setting exists for the company.
	//
	// If companyID is invalid, false and ErrInvalidCompanyID are returned.
	ExistsByCompanyID(
		ctx context.Context,
		companyID string,
	) (bool, error)

	// Create persists a new TransportationFeeSetting.
	//
	// setting.CompanyID is used as the document ID.
	// The setting must satisfy all domain invariants before persistence,
	// including the requirement that all 47 PrefectureRates are present.
	//
	// If a setting already exists for the same CompanyID, ErrConflict is returned.
	// Create must not overwrite an existing document.
	Create(
		ctx context.Context,
		setting TransportationFeeSetting,
	) (*TransportationFeeSetting, error)

	// Update replaces an existing TransportationFeeSetting atomically.
	//
	// Update is not an upsert. If the setting does not exist, ErrNotFound is returned.
	// CompanyID and CreatedAt must remain unchanged from the persisted aggregate.
	// UpdatedAt must not be earlier than CreatedAt.
	//
	// The repository implementation must persist PrefectureRates and IslandRates
	// as one aggregate so that readers never observe a partial rate configuration.
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
