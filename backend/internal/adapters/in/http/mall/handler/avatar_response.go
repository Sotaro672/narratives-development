// backend/internal/adapters/in/http/mall/handler/avatar_response.go
package mallHandler

import (
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"
)

// avatarResponse は公開Avatar APIのレスポンスです。
type avatarResponse struct {
	AvatarID      string    `json:"avatarId"`
	UserID        string    `json:"userId"`
	AvatarName    string    `json:"avatarName"`
	AvatarIcon    *string   `json:"avatarIcon,omitempty"`
	WalletAddress *string   `json:"walletAddress,omitempty"`
	Profile       *string   `json:"profile,omitempty"`
	ExternalLink  *string   `json:"externalLink,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// meAvatarResponse は本人向けAvatar APIのレスポンスです。
// 本人向けAPIではwalletAddressが必須です。
type meAvatarResponse struct {
	AvatarID      string  `json:"avatarId"`
	UserID        string  `json:"userId"`
	AvatarName    string  `json:"avatarName"`
	AvatarIcon    *string `json:"avatarIcon,omitempty"`
	WalletAddress string  `json:"walletAddress"`
	Profile       *string `json:"profile,omitempty"`
	ExternalLink  *string `json:"externalLink,omitempty"`
}

// avatarIconURL はHTTP(S) URLとして公開可能なAvatarIconのみ返します。
func avatarIconURL(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}

	if !strings.HasPrefix(*value, "http://") &&
		!strings.HasPrefix(*value, "https://") {
		return nil
	}

	return value
}

// toAvatarResponse はAvatarドメインを公開APIレスポンスへ変換します。
func toAvatarResponse(a avatardom.Avatar) avatarResponse {
	return avatarResponse{
		AvatarID:      a.ID,
		UserID:        a.UserID,
		AvatarName:    a.AvatarName,
		AvatarIcon:    avatarIconURL(a.AvatarIcon),
		WalletAddress: a.WalletAddress,
		Profile:       a.Profile,
		ExternalLink:  a.ExternalLink,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

// newMeAvatarResponse はAvatarPatchを本人向けAPIレスポンスへ変換します。
// 本人向けレスポンスで必須となる値が欠けている場合はドメインエラーを返します。
func newMeAvatarResponse(
	avatarID string,
	patch avatardom.AvatarPatch,
) (meAvatarResponse, error) {
	if avatarID == "" {
		return meAvatarResponse{}, avatardom.ErrInvalidID
	}

	if patch.UserID == "" {
		return meAvatarResponse{}, avatardom.ErrInvalidUserID
	}

	if patch.AvatarName == nil || *patch.AvatarName == "" {
		return meAvatarResponse{}, avatardom.ErrInvalidAvatarName
	}

	if patch.WalletAddress == nil || *patch.WalletAddress == "" {
		return meAvatarResponse{}, avatardom.ErrInvalidWalletAddressLink
	}

	return meAvatarResponse{
		AvatarID:      avatarID,
		UserID:        patch.UserID,
		AvatarName:    *patch.AvatarName,
		AvatarIcon:    avatarIconURL(patch.AvatarIcon),
		WalletAddress: *patch.WalletAddress,
		Profile:       patch.Profile,
		ExternalLink:  patch.ExternalLink,
	}, nil
}
