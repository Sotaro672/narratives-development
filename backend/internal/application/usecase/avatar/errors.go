// backend\internal\application\usecase\avatar\errors.go
package avatar

import "errors"

// ----------------------------------------
// Local errors (usecase-level validation)
// ----------------------------------------

var (
	// ✅ userUid を avatar.UserID に保存する前提のため、userUid は必須
	ErrInvalidUserUID             = errors.New("avatar: invalid userUid")
	ErrAvatarWalletAlreadyOpened  = errors.New("avatar: wallet already opened")
	ErrAvatarWalletServiceMissing = errors.New("avatar: wallet service not configured")
	ErrAvatarWalletAddressEmpty   = errors.New("avatar: opened wallet address is empty")
)
