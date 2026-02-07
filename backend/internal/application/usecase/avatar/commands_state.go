// backend/internal/application/usecase/avatar/commands_state.go
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

	// 固定上書き方式（{avatarId}/icon）を採用しているため、
	// アイコンの「履歴列挙→全削除」は基本不要＆誤削除リスクがある。
	//
	// 「アバター削除時にアイコンも消したい」場合のみ、
	// avatars.avatarIcon に保存されている参照（https://storage.googleapis.com/...）から
	// bucket/objectPath を復元して 1件だけ削除する。
	if u.objStore != nil && u.avRepo != nil {
		av, err := u.avRepo.GetByID(ctx, avatarID)
		if err == nil {
			// avatardom.Avatar.AvatarIcon は *string
			var iconRef string
			if av.AvatarIcon != nil {
				iconRef = strings.TrimSpace(*av.AvatarIcon)
			}

			if iconRef != "" {
				// ✅ domain の ParseGCSURL は https://storage.googleapis.com/{bucket}/{objectPath} を解釈できる
				if b, obj, ok := avataricon.ParseGCSURL(iconRef); ok {
					_ = u.objStore.DeleteObjects(ctx, []avataricon.GCSDeleteOp{
						{Bucket: b, ObjectPath: obj},
					})
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
