// backend/internal/domain/systemconfig/repository.go
package systemconfig

import (
	"context"
	"errors"
)

var (
	// Firestore 上に設定がまだ無い場合に返す
	ErrMintAuthorityNotConfigured = errors.New("systemconfig: mint authority pubkey is not configured")
)

type Repository interface {
	// システムのミント権限ウォレットの pubkey を取得
	GetMintAuthorityPubkey(ctx context.Context) (string, error)
}
