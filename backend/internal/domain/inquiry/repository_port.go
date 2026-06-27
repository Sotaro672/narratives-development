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

	// ListByAvatarID lists inquiries created by / associated with the given avatar.
	//
	// This is used by avatar-side features where companyId is not available.
	// Repository implementations should apply avatarId as the primary scope.
	ListByAvatarID(
		ctx context.Context,
		avatarID string,
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

// ReplyRepository is the repository port for inquiry replies.
//
// Replies are stored outside the Inquiry aggregate body:
//
//	inquiries/{inquiryId}/replies/{replyId}
//
// Reply is intentionally separated from Inquiry.Content.
// Inquiry.Content must remain the first inquiry body only.
type ReplyRepository interface {
	Create(ctx context.Context, reply Reply) (Reply, error)

	ListByInquiryID(
		ctx context.Context,
		inquiryID string,
	) ([]Reply, error)

	// CountUnreadByAvatarID counts unread replies for the given avatar.
	//
	// Expected flow:
	// - avatar creates an Inquiry for a product
	// - member replies to that Inquiry
	// - avatar receives +1 unread reply count
	//
	// Repository implementations should count replies under inquiries associated
	// with avatarID where:
	// - reply.isRead == false
	// - reply.senderType != avatar OR reply.senderId != avatarID
	//
	// Inquiry.IsRead should not be counted here because the Inquiry body is
	// created by the avatar itself.
	//
	// If Reply does not denormalize avatarId, repository implementations should
	// first resolve inquiries by avatarId, then count unread replies under those
	// inquiry documents.
	CountUnreadByAvatarID(
		ctx context.Context,
		avatarID string,
		filter Filter,
	) (int, error)

	// MarkAsReadByInquiryID marks replies under the given inquiry as read.
	//
	// Repository implementations must not mark the reader's own replies as read.
	// A reply should be skipped when:
	//
	//	reply.SenderType == readerSenderType && reply.SenderID == readerSenderID
	//
	// Repository implementations should update:
	// - isRead = true
	// - updatedAt = readAt
	MarkAsReadByInquiryID(
		ctx context.Context,
		inquiryID string,
		readerSenderType ReplySenderType,
		readerSenderID string,
		readAt time.Time,
	) error
}
