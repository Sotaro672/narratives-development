// backend/internal/application/usecase/avatar_usecase.go
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

// ----------------------------------------
// Local errors (usecase-level validation)
// ----------------------------------------

var (
	ErrInvalidUserUID             = errors.New("avatar: invalid userUid")
	ErrAvatarWalletAlreadyOpened  = errors.New("avatar: wallet already opened")
	ErrAvatarWalletServiceMissing = errors.New("avatar: wallet service not configured")
	ErrAvatarWalletAddressEmpty   = errors.New("avatar: opened wallet address is empty")
)

// AvatarRepo は Avatar 本体の永続化ポートです。
// Firestore 実装（avatar_repository_fs.go）が Create/Update/Delete を提供する前提で揃えます。
type AvatarRepo interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
	Create(ctx context.Context, a avatardom.Avatar) (avatardom.Avatar, error)
	Update(ctx context.Context, id string, patch avatardom.AvatarPatch) (avatardom.Avatar, error)
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

	// ✅ NEW: 画像が空でも avatarDocId/ の “入れ物” を作る（例: <avatarId>/.keep を作成）
	EnsurePrefix(ctx context.Context, bucket, prefix string) error
}

// AvatarWalletService は Avatar 作成時に Solana wallet を開設するためのポートです。
// - 秘密鍵は Secret Manager に保存される想定
// - 公開鍵(base58) を avatar.WalletAddress に反映する
type AvatarWalletService interface {
	OpenAvatarWallet(ctx context.Context, a avatardom.Avatar) (avatardom.SolanaAvatarWallet, error)
}

type AvatarUsecase struct {
	avRepo   AvatarRepo
	stRepo   AvatarStateRepo
	icRepo   AvatarIconRepo
	objStore AvatarIconObjectStoragePort

	walletSvc AvatarWalletService

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

// WithWalletService injects wallet opener (optional).
func (u *AvatarUsecase) WithWalletService(svc AvatarWalletService) *AvatarUsecase {
	u.walletSvc = svc
	return u
}

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
		if st, err := u.stRepo.GetByAvatarID(ctx, id); err == nil && strings.TrimSpace(st.AvatarID) != "" {
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
// Commands (Avatar CRUD for handler)
// =======================

// CreateAvatarInput は avatar_create.dart の入力を正とした作成入力です。
// ※ アイコンは「画像そのもの」ではなく、アップロード後の AvatarIcon（URL/Path統一）を受け取る想定です。
type CreateAvatarInput struct {
	UserID string `json:"userId"`

	// ✅ firebaseUid ではなく userUid へ移譲（認証主体の UID）
	// NOTE: Avatar ドメインには保持しない（必要なら上位の auth context / user domain で管理）。
	UserUID string `json:"userUid"`

	AvatarName   string  `json:"avatarName"`
	AvatarIcon   *string `json:"avatarIcon,omitempty"`
	Profile      *string `json:"profile,omitempty"`
	ExternalLink *string `json:"externalLink,omitempty"`
}

// Create は /avatars POST 用の作成コマンドです。
// ✅ 期待値:
// - avatar 作成と同時に Solana wallet を開設し、秘密鍵は Secret Manager に保存される。
// - narratives-development_avatar_icon に <avatarDocId>/ の “入れ物” を作成する（画像が空でも）。
func (u *AvatarUsecase) Create(ctx context.Context, in CreateAvatarInput) (avatardom.Avatar, error) {
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}

	userID := strings.TrimSpace(in.UserID)
	if userID == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidUserID
	}

	userUID := strings.TrimSpace(in.UserUID)
	if userUID == "" {
		return avatardom.Avatar{}, ErrInvalidUserUID
	}

	name := strings.TrimSpace(in.AvatarName)
	if name == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidAvatarName
	}

	now := u.now().UTC()

	// entity.go のフィールドに合わせる（firebaseUid は保持しない）
	a := avatardom.Avatar{
		// ID は実装側で採番可（空で渡す）
		UserID:       userID,
		AvatarName:   name,
		AvatarIcon:   trimPtr(in.AvatarIcon),
		Profile:      trimPtr(in.Profile),
		ExternalLink: trimPtr(in.ExternalLink),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := u.avRepo.Create(ctx, a)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	// ✅ 画像が空でも “入れ物” を作成（best-effort）
	// - GCS はフォルダを作れないので <avatarId>/.keep のような空オブジェクトを置く想定。
	if u.objStore != nil && strings.TrimSpace(created.ID) != "" {
		_ = u.objStore.EnsurePrefix(ctx, "narratives-development_avatar_icon", strings.TrimSpace(created.ID)+"/")
	}

	// ✅ Wallet open (walletSvc が DI 済みなら strict に実行)
	if u.walletSvc != nil {
		w, werr := u.walletSvc.OpenAvatarWallet(ctx, created)
		if werr != nil {
			return avatardom.Avatar{}, werr
		}

		addr := strings.TrimSpace(w.Address)
		if addr == "" {
			return avatardom.Avatar{}, ErrAvatarWalletAddressEmpty
		}

		patch := avatardom.AvatarPatch{WalletAddress: &addr}
		updated, uerr := u.avRepo.Update(ctx, strings.TrimSpace(created.ID), patch)
		if uerr != nil {
			return avatardom.Avatar{}, uerr
		}
		created = updated
	}

	_ = userUID // NOTE: 現状は保持しない。必要なら auth/handler 層で整合チェックに利用。

	return created, nil
}

// ✅ NEW: OpenWallet は既存 Avatar に対して Solana wallet を開設し、walletAddress を反映します。
// - handler の POST /avatars/{id}/wallet から呼ばれる想定
func (u *AvatarUsecase) OpenWallet(ctx context.Context, avatarID string) (avatardom.Avatar, error) {
	avatarID = strings.TrimSpace(avatarID)
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

	if a.WalletAddress != nil && strings.TrimSpace(*a.WalletAddress) != "" {
		return avatardom.Avatar{}, ErrAvatarWalletAlreadyOpened
	}

	w, err := u.walletSvc.OpenAvatarWallet(ctx, a)
	if err != nil {
		return avatardom.Avatar{}, err
	}
	addr := strings.TrimSpace(w.Address)
	if addr == "" {
		return avatardom.Avatar{}, ErrAvatarWalletAddressEmpty
	}

	patch := avatardom.AvatarPatch{WalletAddress: &addr}
	return u.avRepo.Update(ctx, avatarID, patch)
}

// Update は /avatars/{id} PATCH/PUT 用の部分更新コマンドです。
func (u *AvatarUsecase) Update(ctx context.Context, id string, patch avatardom.AvatarPatch) (avatardom.Avatar, error) {
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}

	// 正規化（nil は「更新しない」契約）
	if patch.AvatarName != nil {
		v := strings.TrimSpace(*patch.AvatarName)
		patch.AvatarName = &v
	}
	if patch.AvatarIcon != nil {
		patch.AvatarIcon = trimPtr(patch.AvatarIcon)
	}
	if patch.Profile != nil {
		patch.Profile = trimPtr(patch.Profile)
	}
	if patch.ExternalLink != nil {
		patch.ExternalLink = trimPtr(patch.ExternalLink)
	}
	if patch.WalletAddress != nil {
		v := strings.TrimSpace(*patch.WalletAddress)
		if v == "" {
			patch.WalletAddress = nil
		} else {
			patch.WalletAddress = &v
		}
	}

	return u.avRepo.Update(ctx, id, patch)
}

// Delete は /avatars/{id} DELETE 用です（既存の cascade delete を利用）。
func (u *AvatarUsecase) Delete(ctx context.Context, avatarID string) error {
	return u.DeleteAvatarCascade(ctx, avatarID)
}

// =======================
// Commands (existing)
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
		return avatarstate.AvatarState{}, avatardom.ErrInvalidID
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

	if u.avRepo == nil {
		return errors.New("avatar repo not configured")
	}
	return u.avRepo.Delete(ctx, avatarID)
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
