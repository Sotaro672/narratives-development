// backend\internal\application\usecase\avatar\commands_avatar.go
package avatar

import (
	"context"
	"errors"
	"log"
	"strings"

	avatardom "narratives/internal/domain/avatar"
	avatarstate "narratives/internal/domain/avatarState"
	cartdom "narratives/internal/domain/cart"
	walletdom "narratives/internal/domain/wallet"
)

// =======================
// Commands (Avatar CRUD for handler)
// =======================

// CreateAvatarInput は avatar_create.dart の入力を正とした作成入力です。
type CreateAvatarInput struct {
	// legacy / optional (client compatibility)
	UserID string `json:"userId"`

	// ✅ auth principal uid (MUST)
	UserUID string `json:"userUid"`

	AvatarName   string  `json:"avatarName"`
	AvatarIcon   *string `json:"avatarIcon,omitempty"`
	Profile      *string `json:"profile,omitempty"`
	ExternalLink *string `json:"externalLink,omitempty"`
}

// Create は /avatars POST 用の作成コマンドです。
func (u *AvatarUsecase) Create(ctx context.Context, in CreateAvatarInput) (avatardom.Avatar, error) {
	if u.avRepo == nil {
		return avatardom.Avatar{}, errors.New("avatar repo not configured")
	}
	// ✅ avatarState は同時作成したいので必須
	if u.stRepo == nil {
		return avatardom.Avatar{}, errors.New("avatarState repo not configured")
	}
	// ✅ walletSvc は Create では必須
	if u.walletSvc == nil {
		return avatardom.Avatar{}, ErrAvatarWalletServiceMissing
	}
	// ✅ wallet table も同時起票したいので必須
	if u.walletRepo == nil {
		return avatardom.Avatar{}, errors.New("wallet repo not configured")
	}
	// ✅ cart も同時起票したい（docId=avatarId を保証する）
	if u.cartRepo == nil {
		return avatardom.Avatar{}, errors.New("cart repo not configured")
	}

	// ✅ 保存する userId は userUid（期待値: userId=userUid）
	userUID := strings.TrimSpace(in.UserUID)
	if userUID == "" {
		userUID = strings.TrimSpace(in.UserID)
	}
	if userUID == "" {
		return avatardom.Avatar{}, ErrInvalidUserUID
	}

	name := strings.TrimSpace(in.AvatarName)
	if name == "" {
		return avatardom.Avatar{}, avatardom.ErrInvalidAvatarName
	}

	now := u.now().UTC()

	// NOTE:
	// - avatarIcon は Create 時点では “固定URL方式” を採用するため、ここでは client input を保存しない。
	// - avatarId 採番後に、server-truth の gs://bucket/objectPath を avatarIcon に入れる（後段で patch）。
	a := avatardom.Avatar{
		UserID:       userUID,
		AvatarName:   name,
		AvatarIcon:   nil, // ✅ server-truth で後で入れる
		Profile:      trimPtr(in.Profile),
		ExternalLink: trimPtr(in.ExternalLink),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := u.avRepo.Create(ctx, a)
	if err != nil {
		return avatardom.Avatar{}, err
	}

	avatarID := strings.TrimSpace(created.ID)
	if avatarID == "" {
		_ = u.avRepo.Delete(ctx, created.ID)
		return avatardom.Avatar{}, avatardom.ErrInvalidID
	}

	rollback := func() {
		if u.avRepo != nil {
			_ = u.avRepo.Delete(ctx, avatarID)
		}
	}

	// ✅ AvatarState doc を同時作成 (strict)
	zero := int64(0)
	as, aerr := avatarstate.New(
		avatarID, // id (=avatarId, docId)
		&zero,
		&zero,
		&zero,
		now,
		&now,
	)
	if aerr != nil {
		rollback()
		return avatardom.Avatar{}, aerr
	}
	if _, err := u.stRepo.Upsert(ctx, as); err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	// ✅ Cart doc を同時作成 (strict): docId=avatarId
	cart, cerr := cartdom.NewCart(avatarID, nil, now)
	if cerr != nil {
		rollback()
		return avatardom.Avatar{}, cerr
	}
	log.Printf(`[avatar_uc] cart upsert start avatarId=%q`, avatarID)
	if err := u.cartRepo.Upsert(ctx, cart); err != nil {
		log.Printf(`[avatar_uc] cart upsert fail avatarId=%q err=%v`, avatarID, err)
		rollback()
		return avatardom.Avatar{}, err
	}
	log.Printf(`[avatar_uc] cart upsert ok avatarId=%q`, avatarID)

	// ✅ Wallet open (strict)
	w, werr := u.walletSvc.OpenAvatarWallet(ctx, avatarID)
	if werr != nil {
		rollback()
		return avatardom.Avatar{}, werr
	}

	addr := strings.TrimSpace(w.Address)
	if addr == "" {
		rollback()
		return avatardom.Avatar{}, ErrAvatarWalletAddressEmpty
	}

	// ✅ avatar に walletAddress + avatarIcon(固定URL) を反映 (strict)
	//
	// 固定URL方式（推奨）:
	// - avatars.avatarIcon は「一定の gs://bucket/objectPath」を持ち続ける
	// - 実体の差し替えは同一 objectPath への上書き（upload）で行う
	const iconBucket = "narratives-development_avatar_icon"
	objPath := avatarID + "/icon" // ←固定名（必要なら "icon.png" 等に変更OK）
	gs := "gs://" + iconBucket + "/" + objPath

	patch := avatardom.AvatarPatch{
		WalletAddress: &addr,
		AvatarIcon:    &gs,
	}
	updated, uerr := u.avRepo.Update(ctx, avatarID, patch)
	if uerr != nil {
		rollback()
		return avatardom.Avatar{}, uerr
	}
	created = updated

	// ✅ wallet テーブルを起票 (strict)
	walletRow, werr2 := walletdom.New(addr, nil, now)
	if werr2 != nil {
		rollback()
		return avatardom.Avatar{}, werr2
	}
	if err := u.walletRepo.Save(ctx, avatarID, walletRow); err != nil {
		rollback()
		return avatardom.Avatar{}, err
	}

	// ✅ GCS prefix を作成（best-effort）
	if u.objStore != nil {
		_ = u.objStore.EnsurePrefix(ctx, iconBucket, avatarID+"/")
	}

	return created, nil
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
