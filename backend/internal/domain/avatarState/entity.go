// backend/internal/domain/avatarState/entity.go
package avatarState

import (
	"errors"
	"time"
)

// AvatarState mirrors web-app/src/shared/types/avatarState.ts
//
//	export interface AvatarFollowRef {
//	  avatarId: string;
//	  followedAt: Date | string;
//	}
//
//	export interface AvatarState {
//	  id?: string;              // (= avatarId as docId)
//	  followerCount?: number;
//	  followingCount?: number;
//	  postCount?: number;
//	  followers?: AvatarFollowRef[]; // subcollection: followers
//	  following?: AvatarFollowRef[]; // subcollection: following
//	  lastActiveAt: Date | string;
//	  updatedAt?: Date | string;
//	}

// AvatarFollowRef represents one follow relation stored in a subcollection.
// - followers subcollection: avatars who follow this avatar
// - following subcollection: avatars this avatar follows
type AvatarFollowRef struct {
	AvatarID   string    `json:"avatarId"`
	FollowedAt time.Time `json:"followedAt"`
}

type AvatarState struct {
	// docId = avatarId
	ID string `json:"id,omitempty"`

	// Aggregated counters
	FollowerCount  *int64 `json:"followerCount,omitempty"`
	FollowingCount *int64 `json:"followingCount,omitempty"`
	PostCount      *int64 `json:"postCount,omitempty"`

	// Firestore subcollections
	// followers: which avatars are following this avatar
	Followers []AvatarFollowRef `json:"followers,omitempty"`
	// following: which avatars this avatar is following
	Following []AvatarFollowRef `json:"following,omitempty"`

	LastActiveAt time.Time  `json:"lastActiveAt"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
}

// Errors
var (
	ErrInvalidID              = errors.New("avatarState: invalid id")
	ErrInvalidLastActiveAt    = errors.New("avatarState: invalid lastActiveAt")
	ErrInvalidUpdatedAt       = errors.New("avatarState: invalid updatedAt")
	ErrNegativeFollowerCount  = errors.New("avatarState: followerCount must be >= 0")
	ErrNegativeFollowingCount = errors.New("avatarState: followingCount must be >= 0")
	ErrNegativePostCount      = errors.New("avatarState: postCount must be >= 0")

	ErrInvalidFollowerAvatarID  = errors.New("avatarState: invalid follower avatarId")
	ErrInvalidFollowingAvatarID = errors.New("avatarState: invalid following avatarId")
	ErrInvalidFollowedAt        = errors.New("avatarState: invalid followedAt")
	ErrDuplicateFollowerAvatar  = errors.New("avatarState: duplicate follower avatarId")
	ErrDuplicateFollowingAvatar = errors.New("avatarState: duplicate following avatarId")
	ErrSelfFollowerRelation     = errors.New("avatarState: self follower relation is not allowed")
	ErrSelfFollowingRelation    = errors.New("avatarState: self following relation is not allowed")
	ErrFollowerCountMismatch    = errors.New("avatarState: followerCount does not match followers subcollection")
	ErrFollowingCountMismatch   = errors.New("avatarState: followingCount does not match following subcollection")
)

/*
Constructors
*/

// New constructs AvatarState with validation.
// id is the source of truth (docId = avatarId).
//
// followers / following are treated as Firestore subcollection snapshots.
// If followerCount or followingCount are provided, they must match the slice lengths.
// If they are nil, they are automatically aggregated from the subcollections.
func New(
	id string,
	followerCount, followingCount, postCount *int64,
	followers, following []AvatarFollowRef,
	lastActiveAt time.Time,
	updatedAt *time.Time,
) (AvatarState, error) {
	aggregatedFollowerCount := int64(len(followers))
	aggregatedFollowingCount := int64(len(following))

	if followerCount == nil {
		followerCount = &aggregatedFollowerCount
	}
	if followingCount == nil {
		followingCount = &aggregatedFollowingCount
	}

	var updatedAtUTC *time.Time
	if updatedAt != nil {
		t := updatedAt.UTC()
		updatedAtUTC = &t
	}

	as := AvatarState{
		ID:             id,
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		PostCount:      postCount,
		Followers:      followers,
		Following:      following,
		LastActiveAt:   lastActiveAt.UTC(),
		UpdatedAt:      updatedAtUTC,
	}
	if err := as.validate(); err != nil {
		return AvatarState{}, err
	}
	return as, nil
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
	if s.UpdatedAt != nil {
		if s.UpdatedAt.IsZero() {
			return ErrInvalidUpdatedAt
		}
		if s.UpdatedAt.Before(s.LastActiveAt) {
			return ErrInvalidUpdatedAt
		}
	}

	followerIDs := make(map[string]struct{}, len(s.Followers))
	for _, f := range s.Followers {
		if f.AvatarID == "" {
			return ErrInvalidFollowerAvatarID
		}
		if f.AvatarID == s.ID {
			return ErrSelfFollowerRelation
		}
		if f.FollowedAt.IsZero() {
			return ErrInvalidFollowedAt
		}
		if _, exists := followerIDs[f.AvatarID]; exists {
			return ErrDuplicateFollowerAvatar
		}
		followerIDs[f.AvatarID] = struct{}{}
	}

	followingIDs := make(map[string]struct{}, len(s.Following))
	for _, f := range s.Following {
		if f.AvatarID == "" {
			return ErrInvalidFollowingAvatarID
		}
		if f.AvatarID == s.ID {
			return ErrSelfFollowingRelation
		}
		if f.FollowedAt.IsZero() {
			return ErrInvalidFollowedAt
		}
		if _, exists := followingIDs[f.AvatarID]; exists {
			return ErrDuplicateFollowingAvatar
		}
		followingIDs[f.AvatarID] = struct{}{}
	}

	if s.FollowerCount != nil && *s.FollowerCount != int64(len(s.Followers)) {
		return ErrFollowerCountMismatch
	}
	if s.FollowingCount != nil && *s.FollowingCount != int64(len(s.Following)) {
		return ErrFollowingCountMismatch
	}

	return nil
}
