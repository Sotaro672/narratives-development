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
	Subject     *string
	Content     *string
	Status      *InquiryStatus
	InquiryType *InquiryType
	IsRead      *bool

	ProductBlueprintID *string
	TokenBlueprintID   *string
	AssigneeID         *string
	Images             *[]ImageFile

	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

// Filter is used by repository implementations.
// CompanyID is supplied by ListByCompanyID / CountUnreadByCompanyID,
// so this filter represents additional conditions within the company scope.
//
// Image filters are included here because inquiryImage is no longer a separate domain.
type Filter struct {
	SearchQuery string

	IDs                []string
	AvatarID           *string
	AssigneeID         *string
	Status             *InquiryStatus
	InquiryType        *InquiryType
	ProductBlueprintID *string
	TokenBlueprintID   *string
	UpdatedBy          *string
	DeletedBy          *string

	ImageFileName *string

	Deleted *bool
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
