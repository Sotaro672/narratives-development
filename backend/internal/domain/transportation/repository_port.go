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
//   - Delete must return ErrNotFound when the specified document does not exist;
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
	GetByID(
		ctx context.Context,
		id string,
	) (*TransportationFeeSetting, error)

	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]TransportationFeeSetting, error)

	Create(
		ctx context.Context,
		setting TransportationFeeSetting,
	) (*TransportationFeeSetting, error)

	Update(
		ctx context.Context,
		setting TransportationFeeSetting,
	) (*TransportationFeeSetting, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

var (
	ErrNotFound = errors.New("transportation: not found")
	ErrConflict = errors.New("transportation: conflict")
)
