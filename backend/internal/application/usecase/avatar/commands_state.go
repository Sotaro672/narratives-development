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
// Commands (State / Cascade)
// =======================

func (u *AvatarUsecase) TouchLastActive(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return avatarstate.AvatarState{}, avatardom.ErrInvalidID
	}
	if u.stRepo == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState repo not configured")
	}
	now := u.now().UTC()

	// ✅ docId=avatarId
	state := avatarstate.AvatarState{
		ID:           avatarID,
		LastActiveAt: now,
		UpdatedAt:    &now,
	}
	return u.stRepo.Upsert(ctx, state)
}

func (u *AvatarUsecase) DeleteAvatarCascade(ctx context.Context, avatarID string) error {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return avatardom.ErrInvalidID
	}

	// icons: best-effort GCS delete（メタデータ削除はRepo機能がない場合スキップ）
	if u.icRepo != nil {
		if list, err := u.icRepo.GetByAvatarID(ctx, avatarID); err == nil && len(list) > 0 && u.objStore != nil {
			var ops []avataricon.GCSDeleteOp
			for _, ic := range list {
				ops = append(ops, toGCSDeleteOp(ic))
			}
			if len(ops) > 0 {
				if err := u.objStore.DeleteObjects(ctx, ops); err != nil {
					return err
				}
			}
		}
	}

	// cart: best-effort delete (optional)
	if u.cartRepo != nil {
		_ = u.cartRepo.DeleteByAvatarID(ctx, avatarID)
	}

	if u.avRepo == nil {
		return errors.New("avatar repo not configured")
	}
	return u.avRepo.Delete(ctx, avatarID)
}
