// backend/internal/adapters/in/http/mall/handler/avatar/avatar_dto.go
package avatarHandler

import (
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"
	avataricon "narratives/internal/domain/avatarIcon"
	avatarstate "narratives/internal/domain/avatarState"
)

type avatarResponse struct {
	AvatarID string `json:"avatarId"`
	UserID   string `json:"userId"`

	AvatarName string  `json:"avatarName"`
	AvatarIcon *string `json:"avatarIcon,omitempty"`

	// ✅ state は handler/join 側で付与できないケースがあるため optional にする
	AvatarState *avatarstate.AvatarState `json:"avatarState,omitempty"`

	WalletAddress *string    `json:"walletAddress,omitempty"`
	Profile       *string    `json:"profile,omitempty"`
	ExternalLink  *string    `json:"externalLink,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
}

func toAvatarResponse(a avatardom.Avatar) avatarResponse {
	var stPtr *avatarstate.AvatarState
	// 既存 struct に state が埋まっている場合のみ返す（ゼロ値回避）
	if strings.TrimSpace(a.AvatarState.ID) != "" {
		tmp := a.AvatarState
		stPtr = &tmp
	}

	return avatarResponse{
		AvatarID:      strings.TrimSpace(a.ID),
		UserID:        strings.TrimSpace(a.UserID),
		AvatarName:    strings.TrimSpace(a.AvatarName),
		AvatarIcon:    trimPtr(a.AvatarIcon), // ✅ trim して返す
		AvatarState:   stPtr,
		WalletAddress: trimPtr(a.WalletAddress),
		Profile:       trimPtr(a.Profile),
		ExternalLink:  trimPtr(a.ExternalLink),
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
		DeletedAt:     a.DeletedAt,
	}
}

type avatarIconResponse struct {
	ID       string  `json:"id"`
	AvatarID *string `json:"avatarId,omitempty"`
	URL      string  `json:"url"`
	FileName *string `json:"fileName,omitempty"`
	Size     *int64  `json:"size,omitempty"`
}

func toAvatarIconResponse(icon avataricon.AvatarIcon, knownAvatarID string) avatarIconResponse {
	aid := strings.TrimSpace(knownAvatarID)
	var aidPtr *string
	if aid != "" {
		aidPtr = &aid
	}
	return avatarIconResponse{
		ID:       strings.TrimSpace(icon.ID),
		AvatarID: aidPtr,
		URL:      strings.TrimSpace(icon.URL),
		FileName: trimPtr(icon.FileName),
		Size:     icon.Size,
	}
}

type avatarAggregateResponse struct {
	Avatar avatarResponse `json:"avatar"`

	// ✅ any をやめて具体型にする（フロントの型安全性・互換性向上）
	State *avatarstate.AvatarState `json:"state,omitempty"`

	Icons []avatarIconResponse `json:"icons"`
}
