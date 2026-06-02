// backend/internal/application/query/mall/follow_query.go
package mall

import (
	"context"
	"errors"
	"time"

	avatardom "narratives/internal/domain/avatar"
	avatarstate "narratives/internal/domain/avatarState"
)

var ErrAvatarStateQueryNotConfigured = errors.New("avatar state query not configured")

type AvatarStateQuery struct {
	avatarRepo      AvatarQueryAvatarRepository
	avatarStateRepo AvatarQueryAvatarStateRepository
}

type AvatarQueryAvatarRepository interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

type AvatarQueryAvatarStateRepository interface {
	GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error)
}

func NewAvatarStateQuery(
	avatarRepo AvatarQueryAvatarRepository,
	avatarStateRepo AvatarQueryAvatarStateRepository,
) *AvatarStateQuery {
	return &AvatarStateQuery{
		avatarRepo:      avatarRepo,
		avatarStateRepo: avatarStateRepo,
	}
}

type AvatarResolvedFollowRef struct {
	AvatarID   string     `json:"avatarId"`
	AvatarName string     `json:"avatarName,omitempty"`
	AvatarIcon string     `json:"avatarIcon,omitempty"`
	FollowedAt *time.Time `json:"followedAt,omitempty"`
}

type AvatarStateResolvedView struct {
	AvatarID       string                    `json:"avatarId"`
	FollowerCount  *int64                    `json:"followerCount,omitempty"`
	FollowingCount *int64                    `json:"followingCount,omitempty"`
	PostCount      *int64                    `json:"postCount,omitempty"`
	Followers      []AvatarResolvedFollowRef `json:"followers"`
	Following      []AvatarResolvedFollowRef `json:"following"`
	LastActiveAt   time.Time                 `json:"lastActiveAt"`
	UpdatedAt      *time.Time                `json:"updatedAt,omitempty"`
}

func (q *AvatarStateQuery) GetResolvedByAvatarID(
	ctx context.Context,
	avatarID string,
) (AvatarStateResolvedView, error) {
	if avatarID == "" {
		return AvatarStateResolvedView{}, avatardom.ErrInvalidID
	}
	if q == nil || q.avatarStateRepo == nil || q.avatarRepo == nil {
		return AvatarStateResolvedView{}, ErrAvatarStateQueryNotConfigured
	}

	st, err := q.avatarStateRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		return AvatarStateResolvedView{}, err
	}

	return q.BuildResolvedView(ctx, avatarID, &st), nil
}

func (q *AvatarStateQuery) BuildResolvedView(
	ctx context.Context,
	avatarID string,
	st *avatarstate.AvatarState,
) AvatarStateResolvedView {
	if st == nil {
		return AvatarStateResolvedView{
			AvatarID:  avatarID,
			Followers: []AvatarResolvedFollowRef{},
			Following: []AvatarResolvedFollowRef{},
		}
	}

	return AvatarStateResolvedView{
		AvatarID:       avatarID,
		FollowerCount:  st.FollowerCount,
		FollowingCount: st.FollowingCount,
		PostCount:      st.PostCount,
		Followers:      q.ResolveFollowRefs(ctx, st.Followers),
		Following:      q.ResolveFollowRefs(ctx, st.Following),
		LastActiveAt:   st.LastActiveAt,
		UpdatedAt:      st.UpdatedAt,
	}
}

func (q *AvatarStateQuery) ResolveFollowRefs(
	ctx context.Context,
	refs []avatarstate.AvatarFollowRef,
) []AvatarResolvedFollowRef {
	if len(refs) == 0 {
		return []AvatarResolvedFollowRef{}
	}

	out := make([]AvatarResolvedFollowRef, 0, len(refs))
	for _, ref := range refs {
		out = append(out, q.ResolveFollowRef(ctx, ref))
	}

	return out
}

func (q *AvatarStateQuery) ResolveFollowRef(
	ctx context.Context,
	ref avatarstate.AvatarFollowRef,
) AvatarResolvedFollowRef {
	out := AvatarResolvedFollowRef{
		AvatarID: ref.AvatarID,
	}

	if !ref.FollowedAt.IsZero() {
		t := ref.FollowedAt.UTC()
		out.FollowedAt = &t
	}

	if ref.AvatarID == "" || q == nil || q.avatarRepo == nil {
		return out
	}

	avatar, err := q.avatarRepo.GetByID(ctx, ref.AvatarID)
	if err != nil {
		return out
	}

	out.AvatarName = avatar.AvatarName
	if avatar.AvatarIcon != nil {
		out.AvatarIcon = *avatar.AvatarIcon
	}

	return out
}
