package avatar

import (
	"context"
	"errors"
	"strings"

	avatardom "narratives/internal/domain/avatar"
	avataricon "narratives/internal/domain/avatarIcon"
	avatarstate "narratives/internal/domain/avatarState"
)

// =======================
// Queries
// =======================

func (u *AvatarUsecase) GetByID(ctx context.Context, id string) (avatardom.Avatar, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}
	return u.avRepo.GetByID(ctx, id)
}

type AvatarAggregate struct {
	Avatar avatardom.Avatar
	State  *avatarstate.AvatarState
	Icons  []avataricon.AvatarIcon
}

func (u *AvatarUsecase) GetAggregate(ctx context.Context, id string) (AvatarAggregate, error) {
	a, err := u.GetByID(ctx, id)
	if err != nil {
		return AvatarAggregate{}, err
	}

	var stPtr *avatarstate.AvatarState
	if u.stRepo != nil {
		// ✅ avatarState は docId=avatarId
		if st, err := u.stRepo.GetByAvatarID(ctx, id); err == nil && strings.TrimSpace(st.ID) != "" {
			tmp := st
			stPtr = &tmp
		}
	}

	var icons []avataricon.AvatarIcon
	if u.icRepo != nil {
		if list, err := u.icRepo.GetByAvatarID(ctx, id); err == nil {
			icons = list
		}
	}

	return AvatarAggregate{Avatar: a, State: stPtr, Icons: icons}, nil
}
