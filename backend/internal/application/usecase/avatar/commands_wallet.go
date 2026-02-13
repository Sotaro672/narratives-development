// backend/internal/application/usecase/avatar/commands_wallet.go
package avatar

import (
	"context"
	"errors"

	avatardom "narratives/internal/domain/avatar"
	walletdom "narratives/internal/domain/wallet"
)

// ✅ OpenWallet は既存 Avatar に対して Solana wallet を開設し、walletAddress を反映します。
func (u *AvatarUsecase) OpenWallet(ctx context.Context, avatarID string) (avatardom.Avatar, error) {
	if avatarID == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}
	if u.walletSvc == nil {
		return avatardom.Avatar{}, ErrAvatarWalletServiceMissing
	}

	a, err := u.avRepo.GetByID(ctx, avatarID)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	if a.WalletAddress != nil && *a.WalletAddress != "" {
		return avatardom.Avatar{}, ErrAvatarWalletAlreadyOpened
	}

	w, err := u.walletSvc.OpenAvatarWallet(ctx, avatarID)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	addr := w.Address
	if addr == "" {
		return avatardom.Avatar{}, ErrAvatarWalletAddressEmpty
	}

	patch := avatardom.AvatarPatch{WalletAddress: &addr}
	updated, err := u.avRepo.Update(ctx, avatarID, patch)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	// best-effort: wallet テーブルも整合させたい場合
	if u.walletRepo != nil {
		now := u.now().UTC()
		if wrow, e := walletdom.New(addr, nil, now); e == nil {
			_ = u.walletRepo.Save(ctx, avatarID, wrow)
		}
	}

	return updated, nil
}
