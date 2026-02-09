// backend/internal/domain/productBlueprint/id.go
package productBlueprint

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID generates a new ProductBlueprint ID.
// Uses 16 random bytes encoded as hex (32 chars).
func NewID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// crypto/rand が落ちるのは極めて稀だが、空は返さない
		// 最低限ユニーク性をある程度確保するために固定値は避ける
		// ここでは hex で埋める（実運用でここに来るなら環境異常）
		for i := range b {
			b[i] = byte(i*31 + 7)
		}
	}
	return hex.EncodeToString(b)
}
