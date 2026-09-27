// backend/internal/application/usecase/transfer_display.go
package usecase

import "context"

func (u *TransferUsecase) resolveBrandDisplayName(
	ctx context.Context,
	brandID string,
) string {
	if u == nil ||
		u.brandDisplay == nil ||
		brandID == "" {
		return ""
	}

	brand, err := u.brandDisplay.GetByID(
		ctx,
		brandID,
	)
	if err != nil {
		return ""
	}

	return brand.Name
}

func (u *TransferUsecase) resolveAvatarDisplayName(
	ctx context.Context,
	avatarID string,
) string {
	if u == nil ||
		u.avatarDisplay == nil ||
		avatarID == "" {
		return ""
	}

	avatar, err := u.avatarDisplay.GetByID(
		ctx,
		avatarID,
	)
	if err != nil {
		return ""
	}

	return avatar.AvatarName
}
