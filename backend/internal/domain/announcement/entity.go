// backend\internal\domain\announcement\entity.go
package announcement

import (
	"errors"
	"fmt"
	annatt "narratives/internal/domain/announcementAttachment"
	"strings"
	"time"
)

// Types (mirror TS)
type AnnouncementCategory string
type TargetAudience string
type AnnouncementStatus string

// Domain errors
var (
	ErrInvalidID             = errors.New("announcement: invalid id")
	ErrInvalidTitle          = errors.New("announcement: invalid title")
	ErrInvalidContent        = errors.New("announcement: invalid content")
	ErrInvalidCategory       = errors.New("announcement: invalid category")
	ErrInvalidTargetAudience = errors.New("announcement: invalid targetAudience")
	ErrInvalidStatus         = errors.New("announcement: invalid status")
	ErrInvalidCreatedBy      = errors.New("announcement: invalid createdBy")
	ErrInvalidCreatedAt      = errors.New("announcement: invalid createdAt")
	ErrInvalidUpdatedAt      = errors.New("announcement: invalid updatedAt")
)

// Entity (aligned with web-app interface)
type Announcement struct {
	ID             string               `json:"id"`
	Title          string               `json:"title"`
	Content        string               `json:"content"`
	Category       AnnouncementCategory `json:"category"`
	TargetAudience TargetAudience       `json:"targetAudience"`
	TargetToken    *string              `json:"targetToken,omitempty"`
	TargetProducts []string             `json:"targetProducts,omitempty"`
	TargetAvatars  []string             `json:"targetAvatars,omitempty"`
	IsPublished    bool                 `json:"isPublished"`
	PublishedAt    *time.Time           `json:"publishedAt,omitempty"`
	Attachments    []string             `json:"attachments,omitempty"` // IDs of announcementAttachment
	Status         AnnouncementStatus   `json:"status"`
	CreatedAt      time.Time            `json:"createdAt"`
	CreatedBy      string               `json:"createdBy"`
	UpdatedAt      *time.Time           `json:"updatedAt,omitempty"`
	UpdatedBy      *string              `json:"updatedBy,omitempty"`
	DeletedAt      *time.Time           `json:"deletedAt,omitempty"`
	DeletedBy      *string              `json:"deletedBy,omitempty"`
}

// Constructors

func New(
	id, title, content string,
	category AnnouncementCategory,
	targetAudience TargetAudience,
	targetToken *string,
	targetProducts, targetAvatars, attachments []string,
	isPublished bool,
	status AnnouncementStatus,
	createdAt time.Time,
	createdBy string,
	publishedAt, updatedAt, deletedAt *time.Time,
	updatedBy, deletedBy *string,
) (Announcement, error) {
	a := Announcement{
		ID:             strings.TrimSpace(id),
		Title:          strings.TrimSpace(title),
		Content:        strings.TrimSpace(content),
		Category:       category,
		TargetAudience: targetAudience,
		TargetToken:    normalizePtr(targetToken),
		TargetProducts: normalizeList(targetProducts),
		TargetAvatars:  normalizeList(targetAvatars),
		IsPublished:    isPublished,
		PublishedAt:    normalizeTimePtr(publishedAt),
		Attachments:    normalizeList(attachments),
		Status:         status,
		CreatedAt:      createdAt.UTC(),
		CreatedBy:      strings.TrimSpace(createdBy),
		UpdatedAt:      normalizeTimePtr(updatedAt),
		UpdatedBy:      normalizePtr(updatedBy),
		DeletedAt:      normalizeTimePtr(deletedAt),
		DeletedBy:      normalizePtr(deletedBy),
	}
	if err := a.validate(); err != nil {
		return Announcement{}, err
	}
	return a, nil
}

func NewFromStringTimes(
	id, title, content string,
	category AnnouncementCategory,
	targetAudience TargetAudience,
	targetToken *string,
	targetProducts, targetAvatars, attachments []string,
	isPublished bool,
	status AnnouncementStatus,
	createdAtStr string,
	createdBy string,
	publishedAtStr, updatedAtStr, deletedAtStr *string,
	updatedBy, deletedBy *string,
) (Announcement, error) {
	ct, err := mustParseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return Announcement{}, err
	}
	var pt, ut, dt *time.Time
	if publishedAtStr != nil {
		if t, err := parseOptionalTime(*publishedAtStr); err != nil {
			return Announcement{}, err
		} else {
			pt = t
		}
	}
	if updatedAtStr != nil {
		if t, err := parseOptionalTime(*updatedAtStr); err != nil {
			return Announcement{}, err
		} else {
			ut = t
		}
	}
	if deletedAtStr != nil {
		if t, err := parseOptionalTime(*deletedAtStr); err != nil {
			return Announcement{}, err
		} else {
			dt = t
		}
	}
	return New(
		id, title, content,
		category, targetAudience, targetToken,
		targetProducts, targetAvatars, attachments,
		isPublished, status,
		ct, createdBy, pt, ut, dt, updatedBy, deletedBy,
	)
}

// Validation

func (a Announcement) validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}
	if a.Title == "" {
		return ErrInvalidTitle
	}
	if a.Content == "" {
		return ErrInvalidContent
	}
	if strings.TrimSpace(string(a.Category)) == "" {
		return ErrInvalidCategory
	}
	if strings.TrimSpace(string(a.TargetAudience)) == "" {
		return ErrInvalidTargetAudience
	}
	if strings.TrimSpace(string(a.Status)) == "" {
		return ErrInvalidStatus
	}
	if strings.TrimSpace(a.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	if a.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if a.UpdatedAt != nil && a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if a.DeletedAt != nil && a.DeletedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if a.PublishedAt != nil && a.PublishedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// Helpers

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	t := p.UTC()
	return &t
}

func normalizeList(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func mustParseTime(s string, classify error) (time.Time, error) {
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
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}

func parseOptionalTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := mustParseTime(s, ErrInvalidUpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// AnnouncementsTableDDL defines the SQL for the announcements table migration.
const AnnouncementsTableDDL = `
-- Migration: Initialize Announcement domain
-- Mirrors backend/internal/domain/annoucement/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS announcements (
  id               TEXT        PRIMARY KEY,
  title            TEXT        NOT NULL,
  content          TEXT        NOT NULL,
  category         TEXT        NOT NULL,
  target_audience  TEXT        NOT NULL,
  target_token     TEXT,
  target_products  TEXT[]      NOT NULL DEFAULT '{}',
  target_avatars   TEXT[]      NOT NULL DEFAULT '{}',
  is_published     BOOLEAN     NOT NULL DEFAULT false,
  published_at     TIMESTAMPTZ,
  attachments      TEXT[]      NOT NULL DEFAULT '{}',
  status           TEXT        NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL,
  created_by       TEXT        NOT NULL,
  updated_at       TIMESTAMPTZ,
  updated_by       TEXT,
  deleted_at       TIMESTAMPTZ,
  deleted_by       TEXT,

  -- Non-empty checks
  CONSTRAINT chk_ann_title_non_empty       CHECK (char_length(trim(title)) > 0),
  CONSTRAINT chk_ann_content_non_empty     CHECK (char_length(trim(content)) > 0),
  CONSTRAINT chk_ann_category_non_empty    CHECK (char_length(trim(category)) > 0),
  CONSTRAINT chk_ann_audience_non_empty    CHECK (char_length(trim(target_audience)) > 0),
  CONSTRAINT chk_ann_status_non_empty      CHECK (char_length(trim(status)) > 0),
  CONSTRAINT chk_ann_created_by_non_empty  CHECK (char_length(trim(created_by)) > 0),

  -- Time order
  CONSTRAINT chk_ann_time_updated   CHECK (updated_at   IS NULL OR updated_at   >= created_at),
  CONSTRAINT chk_ann_time_deleted   CHECK (deleted_at   IS NULL OR deleted_at   >= created_at),
  CONSTRAINT chk_ann_time_published CHECK (published_at IS NULL OR published_at >= created_at)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_ann_is_published ON announcements(is_published);
CREATE INDEX IF NOT EXISTS idx_ann_status       ON announcements(status);
CREATE INDEX IF NOT EXISTS idx_ann_category     ON announcements(category);
CREATE INDEX IF NOT EXISTS idx_ann_created_at   ON announcements(created_at);
CREATE INDEX IF NOT EXISTS idx_ann_published_at ON announcements(published_at);

COMMIT;
`

// 添付ファイル群から Announcement.Attachments 用の ID 配列を作るヘルパ
// 呼び出し側で announcementAttachment を import してください。
//
// 例:
//
//	a.Attachments = AttachmentIDsFromFiles(files)
func AttachmentIDsFromFiles(files []annatt.AttachmentFile) []string {
	ids := make([]string, 0, len(files))
	for _, f := range files {
		id := strings.TrimSpace(f.ID)
		if id == "" {
			id = annatt.MakeAttachmentID(f.AnnouncementID, f.FileName)
		}
		ids = append(ids, id)
	}
	return ids
}
