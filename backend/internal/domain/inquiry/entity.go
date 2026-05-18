// backend/internal/domain/inquiry/entity.go
package inquiry

import (
	"errors"
	"time"
)

// Types (mirror TS)
type InquiryStatus string
type InquiryType string

// Entity (mirror TS Inquiry)
type Inquiry struct {
	ID                 string        `json:"id"`
	AvatarID           string        `json:"avatarId"`
	Subject            string        `json:"subject"`
	Content            string        `json:"content"`
	Status             InquiryStatus `json:"status"`
	InquiryType        InquiryType   `json:"inquiryType"`
	ProductBlueprintID *string       `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   *string       `json:"tokenBlueprintId,omitempty"`
	AssigneeID         *string       `json:"assigneeId,omitempty"`
	ImageID            *string       `json:"imageId,omitempty"`
	CreatedAt          time.Time     `json:"createdAt"`
	UpdatedAt          time.Time     `json:"updatedAt"`
	UpdatedBy          *string       `json:"updatedBy,omitempty"`
	DeletedAt          *time.Time    `json:"deletedAt,omitempty"`
	DeletedBy          *string       `json:"deletedBy,omitempty"`
}

// Errors
var (
	ErrInvalidID          = errors.New("inquiry: invalid id")
	ErrInvalidAvatarID    = errors.New("inquiry: invalid avatarId")
	ErrInvalidSubject     = errors.New("inquiry: invalid subject")
	ErrInvalidContent     = errors.New("inquiry: invalid content")
	ErrInvalidStatus      = errors.New("inquiry: invalid status")
	ErrInvalidInquiryType = errors.New("inquiry: invalid inquiryType")
	ErrInvalidCreatedAt   = errors.New("inquiry: invalid createdAt")
	ErrInvalidUpdatedAt   = errors.New("inquiry: invalid updatedAt")
	ErrInvalidUpdatedBy   = errors.New("inquiry: invalid updatedBy")
	ErrInvalidDeletedAt   = errors.New("inquiry: invalid deletedAt")
	ErrInvalidDeletedBy   = errors.New("inquiry: invalid deletedBy")

	// ImageID は inquiryImage の主キー（= inquiryId）を指す必要がある
	ErrInconsistentImageID = errors.New("inquiry: imageId must equal inquiry id (points to inquiryImage primary key)")
)

// Constructors

// New constructs a minimal Inquiry with required fields.
func New(
	id, avatarID, subject, content string,
	status InquiryStatus,
	inquiryType InquiryType,
	createdAt, updatedAt time.Time,
) (Inquiry, error) {
	in := Inquiry{
		ID:          id,
		AvatarID:    avatarID,
		Subject:     subject,
		Content:     content,
		Status:      status,
		InquiryType: inquiryType,
		CreatedAt:   createdAt.UTC(),
		UpdatedAt:   updatedAt.UTC(),
	}
	if err := in.validate(); err != nil {
		return Inquiry{}, err
	}
	return in, nil
}

// NewWithOptional constructs an Inquiry with optional fields.
func NewWithOptional(
	id, avatarID, subject, content string,
	status InquiryStatus,
	inquiryType InquiryType,
	createdAt, updatedAt time.Time,
	productBlueprintID, tokenBlueprintID, assigneeID, imageID, updatedBy, deletedBy *string,
	deletedAt *time.Time,
) (Inquiry, error) {
	in := Inquiry{
		ID:                 id,
		AvatarID:           avatarID,
		Subject:            subject,
		Content:            content,
		Status:             status,
		InquiryType:        inquiryType,
		ProductBlueprintID: productBlueprintID,
		TokenBlueprintID:   tokenBlueprintID,
		AssigneeID:         assigneeID,
		ImageID:            imageID,
		CreatedAt:          createdAt.UTC(),
		UpdatedAt:          updatedAt.UTC(),
		UpdatedBy:          updatedBy,
		DeletedAt:          normalizeTimePtr(deletedAt),
		DeletedBy:          deletedBy,
	}
	if err := in.validate(); err != nil {
		return Inquiry{}, err
	}
	return in, nil
}

// Behavior

func (i *Inquiry) Touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	i.UpdatedAt = now.UTC()
	return nil
}

// Validation

func (i Inquiry) validate() error {
	if i.ID == "" {
		return ErrInvalidID
	}
	if i.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if i.Subject == "" {
		return ErrInvalidSubject
	}
	if i.Content == "" {
		return ErrInvalidContent
	}
	if string(i.Status) == "" {
		return ErrInvalidStatus
	}
	if string(i.InquiryType) == "" {
		return ErrInvalidInquiryType
	}
	if i.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if i.UpdatedAt.IsZero() || i.UpdatedAt.Before(i.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if i.UpdatedBy != nil && *i.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}
	if i.DeletedAt != nil && i.DeletedAt.Before(i.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if i.DeletedBy != nil && *i.DeletedBy == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

// Helpers

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	t := p.UTC()
	return &t
}
