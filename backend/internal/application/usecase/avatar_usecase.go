package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"
	avataricon "narratives/internal/domain/avatarIcon"
	avatarstate "narratives/internal/domain/avatarState"
)

type AvatarRepo interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
	// Update は内製Repoの署名と乖離があるため未使用。必要時にパッチ方式へ拡張してください。
	Delete(ctx context.Context, id string) error
}

type AvatarStateRepo interface {
	GetByAvatarID(ctx context.Context, avatarID string) (avatarstate.AvatarState, error)
	// Upsert がない実装もあるため、必要時はアダプタ側でエラー返却可
	Upsert(ctx context.Context, s avatarstate.AvatarState) (avatarstate.AvatarState, error)
}

type AvatarIconRepo interface {
	GetByAvatarID(ctx context.Context, avatarID string) ([]avataricon.AvatarIcon, error)
	// Repo 実装が Save(ctx, icon) 以外（例: Save(ctx, icon, opts)）の場合は
	// アダプタ側で opts=nil などに委譲してください。
	Save(ctx context.Context, ic avataricon.AvatarIcon, opts *avataricon.SaveOptions) (avataricon.AvatarIcon, error)
}

type AvatarIconObjectStoragePort interface {
	DeleteObjects(ctx context.Context, ops []avataricon.GCSDeleteOp) error
}

type AvatarUsecase struct {
	avRepo   AvatarRepo
	stRepo   AvatarStateRepo
	icRepo   AvatarIconRepo
	objStore AvatarIconObjectStoragePort

	now func() time.Time
}

func NewAvatarUsecase(
	avRepo AvatarRepo,
	stRepo AvatarStateRepo,
	icRepo AvatarIconRepo,
	objStore AvatarIconObjectStoragePort,
) *AvatarUsecase {
	return &AvatarUsecase{
		avRepo:   avRepo,
		stRepo:   stRepo,
		icRepo:   icRepo,
		objStore: objStore,
		now:      time.Now,
	}
}

func (u *AvatarUsecase) WithNow(now func() time.Time) *AvatarUsecase {
	u.now = now
	return u
}

// =======================
// Queries
// =======================
func (u *AvatarUsecase) GetByID(ctx context.Context, id string) (avatardom.Avatar, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return avatardom.Avatar{}, errors.New("avatar: invalid id")
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
		if st, err := u.stRepo.GetByAvatarID(ctx, id); err == nil && st.AvatarID != "" {
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

// =======================
// Commands
// =======================

type ReplaceIconInput struct {
	Bucket     string
	ObjectPath string
	FileName   *string
	Size       *int64
}

func (u *AvatarUsecase) ReplaceAvatarIcon(ctx context.Context, avatarID string, in ReplaceIconInput) (avataricon.AvatarIcon, error) {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return avataricon.AvatarIcon{}, errors.New("avatar: invalid id")
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

func (u *AvatarUsecase) TouchLastActive(ctx context.Context, avatarID string) (avatarstate.AvatarState, error) {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return avatarstate.AvatarState{}, errors.New("avatar: invalid id")
	}
	if u.stRepo == nil {
		return avatarstate.AvatarState{}, errors.New("avatarState repo not configured")
	}
	now := u.now().UTC()
	state := avatarstate.AvatarState{
		AvatarID:     avatarID,
		LastActiveAt: now,
		UpdatedAt:    &now,
	}
	return u.stRepo.Upsert(ctx, state)
}

func (u *AvatarUsecase) DeleteAvatarCascade(ctx context.Context, avatarID string) error {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return errors.New("avatar: invalid id")
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

	if u.avRepo == nil {
		return errors.New("avatar repo not configured")
	}
	return u.avRepo.Delete(ctx, avatarID)
}

// --- helpers ---
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
