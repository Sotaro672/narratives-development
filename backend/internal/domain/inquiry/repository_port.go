// backend/internal/domain/inquiry/repository_port.go
package inquiry

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// InquiryPatch is used for partial inquiry updates.
// nil means unchanged.
//
// Images is a full replacement field.
// Since inquiry images are now part of the Inquiry aggregate,
// image add/update/delete should be handled by reading Inquiry,
// mutating Inquiry.Images, and calling Update.
type InquiryPatch struct {
	ProductID   *string
	Subject     *string
	Content     *string
	Status      *InquiryStatus
	InquiryType *InquiryType
	IsRead      *bool
	Images      *[]ImageFile

	ResolvedAt *time.Time
	ResolvedBy *string
	ClosedAt   *time.Time
	ClosedBy   *string

	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

// Filter is used by repository implementations.
//
// Inquiry is identified in the mall context by productId + avatarId.
// Image filters are included here because inquiryImage is no longer a separate domain.
type Filter struct {
	SearchQuery string

	IDs         []string
	ProductID   *string
	AvatarID    *string
	Status      *InquiryStatus
	InquiryType *InquiryType
	UpdatedBy   *string
	DeletedBy   *string

	ResolvedBy *string
	ClosedBy   *string

	ImageFileName *string

	Deleted  *bool
	Resolved *bool
	Closed   *bool
}

// Common aliases.
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// Contract errors.
var (
	ErrNotFound = errors.New("inquiry: not found")
	ErrConflict = errors.New("inquiry: conflict")
)

// Repository is the inquiry aggregate repository.
//
// Images are part of Inquiry.
// Therefore, image add/update/delete should be expressed as Inquiry.Images mutation
// through Update, not as separate repository methods.
type Repository interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
		filter Filter,
		sort Sort,
		page Page,
	) (PageResult[Inquiry], error)

	// CountUnreadByCompanyID counts inquiries where isRead is false
	// within the given company scope.
	//
	// Inquiry itself does not store companyId. Repository implementations that
	// cannot resolve company scope directly should keep this as a compatibility
	// method until company-scoped inquiry listing/counting is moved to a query service.
	CountUnreadByCompanyID(
		ctx context.Context,
		companyID string,
		filter Filter,
	) (int, error)

	GetByID(ctx context.Context, id string) (Inquiry, error)
	Create(ctx context.Context, inq Inquiry) (Inquiry, error)
	Update(ctx context.Context, id string, patch InquiryPatch) (Inquiry, error)
	Delete(ctx context.Context, id string) error
}
