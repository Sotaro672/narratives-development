// backend/internal/domain/avatarState/entity.go
package avatarState

import (
	"errors"
	"fmt"
	"time"

	domcommon "narratives/internal/domain/common"
)

// AvatarState mirrors web-app/src/shared/types/avatarState.ts
//
//	export interface AvatarState {
//	  id?: string;              // (= avatarId as docId)
//	  followerCount?: number;
//	  followingCount?: number;
//	  postCount?: number;
//	  lastActiveAt: Date | string;
//	  updatedAt?: Date | string;
//	}
type AvatarState struct {
	// ✅ docId = avatarId
	ID             string     `json:"id,omitempty"`
	FollowerCount  *int64     `json:"followerCount,omitempty"`
	FollowingCount *int64     `json:"followingCount,omitempty"`
	PostCount      *int64     `json:"postCount,omitempty"`
	LastActiveAt   time.Time  `json:"lastActiveAt"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty"`
}

// Errors
var (
	ErrInvalidID              = errors.New("avatarState: invalid id")
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
// ✅ id is the source of truth (docId = avatarId).
func New(
	id string,
	followerCount, followingCount, postCount *int64,
	lastActiveAt time.Time,
	updatedAt *time.Time,
) (AvatarState, error) {
	as := AvatarState{
		ID:             id,
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
	id string,
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
		// NormalizeStringPtr により空白のみ/空文字は nil になる
		if v := domcommon.NormalizeStringPtr(updatedAtStr); v != nil {
			t, err := parseTimeRequired(*v, ErrInvalidUpdatedAt)
			if err != nil {
				return AvatarState{}, err
			}
			ua = &t
		}
	}

	return New(id, followerCount, followingCount, postCount, la, ua)
}

/*
Validation
*/

func (s AvatarState) validate() error {
	if s.ID == "" {
		return ErrInvalidID
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
	t, err := domcommon.ParseTime(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", classify, err)
	}
	if t.IsZero() {
		return time.Time{}, classify
	}
	return t.UTC(), nil
}
