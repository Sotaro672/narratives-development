package inquiry

import (
	"errors"
	"strings"
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
		ID:          strings.TrimSpace(id),
		AvatarID:    strings.TrimSpace(avatarID),
		Subject:     strings.TrimSpace(subject),
		Content:     strings.TrimSpace(content),
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
		ID:                 strings.TrimSpace(id),
		AvatarID:           strings.TrimSpace(avatarID),
		Subject:            strings.TrimSpace(subject),
		Content:            strings.TrimSpace(content),
		Status:             status,
		InquiryType:        inquiryType,
		ProductBlueprintID: normalizeStrPtr(productBlueprintID),
		TokenBlueprintID:   normalizeStrPtr(tokenBlueprintID),
		AssigneeID:         normalizeStrPtr(assigneeID),
		ImageID:            normalizeStrPtr(imageID),
		CreatedAt:          createdAt.UTC(),
		UpdatedAt:          updatedAt.UTC(),
		UpdatedBy:          normalizeStrPtr(updatedBy),
		DeletedAt:          normalizeTimePtr(deletedAt),
		DeletedBy:          normalizeStrPtr(deletedBy),
	}
	if err := in.validate(); err != nil {
		return Inquiry{}, err
	}
	return in, nil
}

// NewFromStringTimes builds with string times for createdAt/updatedAt (RFC3339 preferred).
func NewFromStringTimes(
	id, avatarID, subject, content string,
	status InquiryStatus,
	inquiryType InquiryType,
	createdAtStr, updatedAtStr string,
) (Inquiry, error) {
	ct, err := parseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return Inquiry{}, err
	}
	ut, err := parseTime(updatedAtStr, ErrInvalidUpdatedAt)
	if err != nil {
		return Inquiry{}, err
	}
	return New(id, avatarID, subject, content, status, inquiryType, ct, ut)
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
	if strings.TrimSpace(i.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(i.AvatarID) == "" {
		return ErrInvalidAvatarID
	}
	if strings.TrimSpace(i.Subject) == "" {
		return ErrInvalidSubject
	}
	if strings.TrimSpace(i.Content) == "" {
		return ErrInvalidContent
	}
	if strings.TrimSpace(string(i.Status)) == "" {
		return ErrInvalidStatus
	}
	if strings.TrimSpace(string(i.InquiryType)) == "" {
		return ErrInvalidInquiryType
	}
	if i.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if i.UpdatedAt.IsZero() || i.UpdatedAt.Before(i.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if i.UpdatedBy != nil && strings.TrimSpace(*i.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if i.DeletedAt != nil && i.DeletedAt.Before(i.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	if i.DeletedBy != nil && strings.TrimSpace(*i.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

// Helpers

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	t := p.UTC()
	return &t
}

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, classify
}

// ========================================
// SQL DDL
// ========================================

const InquiriesTableDDL = `
CREATE TABLE IF NOT EXISTS inquiries (
  id                   UUID        PRIMARY KEY,
  avatar_id            TEXT        NOT NULL,
  subject              TEXT        NOT NULL,
  content              TEXT        NOT NULL,
  status               TEXT        NOT NULL,
  inquiry_type         TEXT        NOT NULL,
  product_blueprint_id TEXT        NULL,
  token_blueprint_id   TEXT        NULL,
  assignee_id          TEXT        NULL,
  image                TEXT        NULL,
  created_at           TIMESTAMPTZ NOT NULL,
  updated_at           TIMESTAMPTZ NOT NULL,
  updated_by           TEXT        NULL,
  deleted_at           TIMESTAMPTZ NULL,
  deleted_by           TEXT        NULL,

  -- Non-empty checks
  CONSTRAINT chk_inquiries_non_empty CHECK (
    char_length(trim(subject)) > 0 AND
    char_length(trim(content)) > 0 AND
    char_length(trim(status)) > 0 AND
    char_length(trim(inquiry_type)) > 0
  ),

  -- time order
  CONSTRAINT chk_inquiries_time_order CHECK (updated_at >= created_at),
  CONSTRAINT chk_inquiries_deleted_time CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_inquiries_avatar_id       ON inquiries(avatar_id);
CREATE INDEX IF NOT EXISTS idx_inquiries_assignee_id     ON inquiries(assignee_id);
CREATE INDEX IF NOT EXISTS idx_inquiries_status          ON inquiries(status);
CREATE INDEX IF NOT EXISTS idx_inquiries_inquiry_type    ON inquiries(inquiry_type);
CREATE INDEX IF NOT EXISTS idx_inquiries_created_at      ON inquiries(created_at);
CREATE INDEX IF NOT EXISTS idx_inquiries_updated_at      ON inquiries(updated_at);
CREATE INDEX IF NOT EXISTS idx_inquiries_deleted_at      ON inquiries(deleted_at);
`
