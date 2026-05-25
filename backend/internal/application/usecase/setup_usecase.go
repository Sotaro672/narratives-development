// backend/internal/application/usecase/setup_usecase.go
package usecase

import (
	"context"
	"errors"
)

// SetupAvatarRepository is the minimum repository contract required by setup status.
//
// Avatar document id is avatarId, not userId.
// Therefore setup status checks avatar existence by userId field.
type SetupAvatarRepository interface {
	ExistsByUserID(ctx context.Context, userID string) (bool, error)
}

type SetupUsecase struct {
	avatarRepo SetupAvatarRepository
}

func NewSetupUsecase(avatarRepo SetupAvatarRepository) *SetupUsecase {
	return &SetupUsecase{
		avatarRepo: avatarRepo,
	}
}

type SetupStatusOutput struct {
	HasAvatar      bool
	SetupCompleted bool
	Required       SetupRequiredOutput
}

type SetupRequiredOutput struct {
	Avatar bool
}

var (
	ErrSetupInvalidUserUID     = errors.New("setup: invalid user uid")
	ErrSetupAvatarRepoNotReady = errors.New("setup: avatar repo not configured")
)

func (u *SetupUsecase) GetSetupStatus(ctx context.Context, uid string) (SetupStatusOutput, error) {
	if uid == "" {
		return SetupStatusOutput{}, ErrSetupInvalidUserUID
	}
	if u == nil || u.avatarRepo == nil {
		return SetupStatusOutput{}, ErrSetupAvatarRepoNotReady
	}

	hasAvatar, err := u.avatarRepo.ExistsByUserID(ctx, uid)
	if err != nil {
		return SetupStatusOutput{}, err
	}

	return SetupStatusOutput{
		HasAvatar:      hasAvatar,
		SetupCompleted: hasAvatar,
		Required: SetupRequiredOutput{
			Avatar: true,
		},
	}, nil
}
