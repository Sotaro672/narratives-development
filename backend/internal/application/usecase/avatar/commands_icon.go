// backend\internal\application\usecase\avatar\commands_icon.go
package avatar

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	avatardom "narratives/internal/domain/avatar"
	avataricon "narratives/internal/domain/avatarIcon"
)

// =======================
// Commands (Icon)
// =======================

type ReplaceIconInput struct {
	Bucket     string
	ObjectPath string
	FileName   *string
	Size       *int64
}

// ✅ 推奨（最小変更）:
// - avatars.avatarIcon に gs://bucket/objectPath を保存（= objectPath を含めて保存）
// - アイコンメタデータ（icons 側）はこれまで通り保存
func (u *AvatarUsecase) ReplaceAvatarIcon(ctx context.Context, avatarID string, in ReplaceIconInput) (avataricon.AvatarIcon, error) {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return avataricon.AvatarIcon{}, avatardom.ErrInvalidID
	}

	var oldIcons []avataricon.AvatarIcon
	if u.icRepo != nil {
		if list, err := u.icRepo.GetByAvatarID(ctx, avatarID); err == nil {
			oldIcons = list
		}
	}

	now := u.now().UTC()
	newIcon, err := avataricon.NewFromBucketObject(
		avatarID+"-"+now.Format("20060102T150405Z0700"),
		in.Bucket,
		in.ObjectPath,
		in.FileName,
		in.Size,
	)
	if err != nil {
		return avataricon.AvatarIcon{}, err
	}
	if newIcon.AvatarID == nil || strings.TrimSpace(*newIcon.AvatarID) == "" {
		aid := avatarID
		newIcon.AvatarID = &aid
	}

	if u.icRepo == nil {
		return avataricon.AvatarIcon{}, errors.New("avatarIcon repo not configured")
	}
	saved, err := u.icRepo.Save(ctx, newIcon, nil)
	if err != nil {
		return avataricon.AvatarIcon{}, err
	}

	// ✅ avatars.avatarIcon を更新（推奨: usecase で保証）
	// - gs://bucket/objectPath を保存（objectPath を含めて保存）
	// - ここは best-effort（icons 保存は成功しているため）とする
	if u.avRepo != nil {
		b := strings.TrimSpace(in.Bucket)
		obj := strings.TrimLeft(strings.TrimSpace(in.ObjectPath), "/")
		if b != "" && obj != "" {
			gs := fmt.Sprintf("gs://%s/%s", b, obj)
			patch := avatardom.AvatarPatch{AvatarIcon: &gs}
			if _, e := u.avRepo.Update(ctx, avatarID, patch); e != nil {
				log.Printf("[avatar_uc] avatarIcon patch failed avatarId=%q gs=%q err=%v", avatarID, gs, e)
			} else {
				log.Printf("[avatar_uc] avatarIcon patched avatarId=%q gs=%q", avatarID, gs)
			}
		} else {
			log.Printf("[avatar_uc] avatarIcon patch skipped (empty bucket/objectPath) avatarId=%q bucket=%q objectPath=%q", avatarID, b, obj)
		}
	}

	// best-effort: GCS から古いオブジェクトのみ削除（メタデータ削除はRepo機能に依存）
	if len(oldIcons) > 0 && u.objStore != nil {
		var ops []avataricon.GCSDeleteOp
		for _, ic := range oldIcons {
			ops = append(ops, toGCSDeleteOp(ic))
		}
		if len(ops) > 0 {
			_ = u.objStore.DeleteObjects(ctx, ops)
		}
	}
	return saved, nil
}

func toGCSDeleteOp(ic avataricon.AvatarIcon) avataricon.GCSDeleteOp {
	if b, obj, ok := avataricon.ParseGCSURL(ic.URL); ok {
		return avataricon.GCSDeleteOp{Bucket: b, ObjectPath: obj}
	}
	if ic.AvatarID != nil && strings.TrimSpace(*ic.AvatarID) != "" &&
		ic.FileName != nil && strings.TrimSpace(*ic.FileName) != "" {
		return avataricon.GCSDeleteOp{
			Bucket:     avataricon.DefaultBucket,
			ObjectPath: strings.TrimSpace(*ic.AvatarID) + "/" + strings.TrimSpace(*ic.FileName),
		}
	}
	return avataricon.GCSDeleteOp{
		Bucket:     avataricon.DefaultBucket,
		ObjectPath: "avatar_icons/" + strings.TrimSpace(ic.ID),
	}
}
