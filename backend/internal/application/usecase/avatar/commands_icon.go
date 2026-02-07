// backend/internal/application/usecase/avatar/commands_icon.go
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

// 方針（固定URL / 毎回上書き）
//   - GCS objectPath は常に "{avatarId}/icon"（フロントは署名付きPUTでここに上書き）
//   - Firestore(avatars.avatarIcon) には公開URLを保存する
//     例: https://storage.googleapis.com/<bucket>/<avatarId>/icon
//   - アイコンメタデータ（avatar_icons 側）は最小限保存（ID = objectPath に統一）
//   - 固定上書き方式のため、過去オブジェクト列挙・削除は行わない（不要 & 誤削除リスク）
func (u *AvatarUsecase) ReplaceAvatarIcon(
	ctx context.Context,
	avatarID string,
	in ReplaceIconInput,
) (avataricon.AvatarIcon, error) {

	if u == nil {
		return avataricon.AvatarIcon{}, errors.New("avatar usecase is nil")
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return avataricon.AvatarIcon{}, avatardom.ErrInvalidID
	}

	// 1) 入力の正規化
	bucket := strings.TrimSpace(in.Bucket)
	objectPath := strings.TrimLeft(strings.TrimSpace(in.ObjectPath), "/")

	if bucket == "" || objectPath == "" {
		return avataricon.AvatarIcon{}, fmt.Errorf("bucket/objectPath is empty: bucket=%q objectPath=%q", bucket, objectPath)
	}

	// 2) avatar の存在確認（avatars を更新するので必須）
	if u.avRepo == nil {
		return avataricon.AvatarIcon{}, errors.New("avatar repo not configured")
	}
	if _, err := u.avRepo.GetByID(ctx, avatarID); err != nil {
		return avataricon.AvatarIcon{}, err
	}

	// 3) icon メタデータ作成（ID = objectPath に統一）
	//    固定上書き方式なので timestamp 等は不要（むしろ不整合の原因）
	newIcon, err := avataricon.NewFromBucketObject(
		objectPath, // ✅ ID = objectPath（方針に統一）
		bucket,
		objectPath,
		in.FileName,
		in.Size,
	)
	if err != nil {
		return avataricon.AvatarIcon{}, err
	}

	// 念のため avatarId を埋める（domain が nil を許容している前提の保険）
	if newIcon.AvatarID == nil || strings.TrimSpace(*newIcon.AvatarID) == "" {
		aid := avatarID
		newIcon.AvatarID = &aid
	}

	// 4) icon repo へ Save（GCS メタ更新 / URL 更新など）
	if u.icRepo == nil {
		return avataricon.AvatarIcon{}, errors.New("avatarIcon repo not configured")
	}

	saved, err := u.icRepo.Save(ctx, newIcon, nil)
	if err != nil {
		return avataricon.AvatarIcon{}, err
	}

	// 5) avatars.avatarIcon を公開URLで更新（期待値）
	//    例: https://storage.googleapis.com/narratives-development_avatar_icon/<avatarId>/icon
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectPath)

	patch := avatardom.AvatarPatch{AvatarIcon: &publicURL}
	if _, err := u.avRepo.Update(ctx, avatarID, patch); err != nil {
		// icons 側は保存済みなので best-effort とする
		log.Printf("[avatar_uc] avatarIcon patch failed avatarId=%q publicURL=%q err=%v", avatarID, publicURL, err)
	} else {
		log.Printf("[avatar_uc] avatarIcon patched avatarId=%q publicURL=%q", avatarID, publicURL)
	}

	// 6) 固定上書き方式のため削除処理は不要
	// - objectPath が固定で毎回上書きされるため GCS 上に増えない
	// - 誤削除リスク（旧方式のオブジェクト混在）を避けるため、ここでは実施しない

	return saved, nil
}
