// backend/internal/domain/avatarState/entity.go
package avatarState

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// AvatarState mirrors web-app/src/shared/types/avatarState.ts
//
//	export interface AvatarState {
//	  id?: string;
//	  avatarId: string;
//	  followerCount?: number;
//	  followingCount?: number;
//	  postCount?: number;
//	  lastActiveAt: Date | string;
//	  updatedAt?: Date | string;
//	}
type AvatarState struct {
	ID             string     `json:"id,omitempty"`
	AvatarID       string     `json:"avatarId"`
	FollowerCount  *int64     `json:"followerCount,omitempty"`
	FollowingCount *int64     `json:"followingCount,omitempty"`
	PostCount      *int64     `json:"postCount,omitempty"`
	LastActiveAt   time.Time  `json:"lastActiveAt"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
}

// Errors
var (
	ErrInvalidAvatarID        = errors.New("avatarState: invalid avatarId")
	ErrInvalidLastActiveAt    = errors.New("avatarState: invalid lastActiveAt")
	ErrInvalidUpdatedAt       = errors.New("avatarState: invalid updatedAt")
	ErrNegativeFollowerCount  = errors.New("avatarState: followerCount must be >= 0")
	ErrNegativeFollowingCount = errors.New("avatarState: followingCount must be >= 0")
	ErrNegativePostCount      = errors.New("avatarState: postCount must be >= 0")
)

/*
Constructors
*/

// New constructs AvatarState with validation.
func New(
	id, avatarID string,
	followerCount, followingCount, postCount *int64,
	lastActiveAt time.Time,
	updatedAt *time.Time,
) (AvatarState, error) {
	as := AvatarState{
		ID:             strings.TrimSpace(id),
		AvatarID:       strings.TrimSpace(avatarID),
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		PostCount:      postCount,
		LastActiveAt:   lastActiveAt.UTC(),
		UpdatedAt:      normalizeTimePtr(updatedAt),
	}
	if err := as.validate(); err != nil {
		return AvatarState{}, err
	}
	return as, nil
}

// NewFromStringTimes parses lastActiveAt/updatedAt from strings (RFC3339 and common variants).
func NewFromStringTimes(
	id, avatarID string,
	followerCount, followingCount, postCount *int64,
	lastActiveAtStr string,
	updatedAtStr *string,
) (AvatarState, error) {
	la, err := parseTimeRequired(lastActiveAtStr, ErrInvalidLastActiveAt)
	if err != nil {
		return AvatarState{}, err
	}
	var ua *time.Time
	if updatedAtStr != nil {
		if strings.TrimSpace(*updatedAtStr) != "" {
			t, err := parseTimeRequired(*updatedAtStr, ErrInvalidUpdatedAt)
			if err != nil {
				return AvatarState{}, err
			}
			ua = &t
		}
	}
	return New(id, avatarID, followerCount, followingCount, postCount, la, ua)
}

/*
Validation
*/

func (s AvatarState) validate() error {
	if s.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if s.FollowerCount != nil && *s.FollowerCount < 0 {
		return ErrNegativeFollowerCount
	}
	if s.FollowingCount != nil && *s.FollowingCount < 0 {
		return ErrNegativeFollowingCount
	}
	if s.PostCount != nil && *s.PostCount < 0 {
		return ErrNegativePostCount
	}
	if s.LastActiveAt.IsZero() {
		return ErrInvalidLastActiveAt
	}
	if s.UpdatedAt != nil && s.UpdatedAt.Before(s.LastActiveAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

/*
Helpers
*/

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil || p.IsZero() {
		return nil
	}
	t := p.UTC()
	return &t
}

func parseTimeRequired(s string, classify error) (time.Time, error) {
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
