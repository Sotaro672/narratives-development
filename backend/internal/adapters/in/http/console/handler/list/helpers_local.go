// backend/internal/adapters/in/http/console/handler/list/helpers_local.go
//
// Responsibility:
// - list handler 内部で使う小さなユーティリティ（文字列正規化など）を提供する。
// - 他パッケージへ輸出しない前提の helper の置き場。
package list

import "strings"

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}
