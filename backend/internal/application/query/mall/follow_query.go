// backend\internal\application\query\mall\follow_query.go
package mall

import (
	"context"
	"time"

	avatardom "narratives/internal/domain/avatar"
	avatarstate "narratives/internal/domain/avatarState"
)

type AvatarStateQuery struct {
	avatarRepo      AvatarQueryAvatarRepository
	avatarStateRepo AvatarQueryAvatarStateRepository
}

type AvatarQueryAvatarRepository interface {
	GetNameAndIconByID(ctx context.Context, id string) (name string, icon string, err error)
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
	AvatarName string     `json:"avatarName"`
	AvatarIcon string     `json:"avatarIcon"`
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
	if q == nil || q.avatarStateRepo == nil {
		return AvatarStateResolvedView{}, avatardom.ErrInvalidID
	}
	if q.avatarRepo == nil {
		return AvatarStateResolvedView{}, avatardom.ErrInvalidID
	}

	st, err := q.avatarStateRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		return AvatarStateResolvedView{}, err
	}

	followers := make([]AvatarResolvedFollowRef, 0, len(st.Followers))
	for _, ref := range st.Followers {
		followers = append(followers, q.resolveFollowRef(ctx, ref))
	}

	following := make([]AvatarResolvedFollowRef, 0, len(st.Following))
	for _, ref := range st.Following {
		following = append(following, q.resolveFollowRef(ctx, ref))
	}

	return AvatarStateResolvedView{
		AvatarID:       avatarID,
		FollowerCount:  st.FollowerCount,
		FollowingCount: st.FollowingCount,
		PostCount:      st.PostCount,
		Followers:      followers,
		Following:      following,
		LastActiveAt:   st.LastActiveAt,
		UpdatedAt:      st.UpdatedAt,
	}, nil
}

func (q *AvatarStateQuery) resolveFollowRef(
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

	name, icon, err := q.avatarRepo.GetNameAndIconByID(ctx, ref.AvatarID)
	if err != nil {
		return out
	}

	out.AvatarName = name
	out.AvatarIcon = icon
	return out
}
