package usecase

import "strings"

// 共通の「未サポート」エラー型とヘルパー
type notSupportedError struct{ op string }

func (e notSupportedError) Error() string {
	return "usecase: operation not supported: " + e.op
}

// ErrNotSupported は未サポート操作を表すエラーを返します。
func ErrNotSupported(op string) error { return notSupportedError{op: op} }

// 共通ヘルパー: 重複排除 + 空白除去
func dedupStrings(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, v := range xs {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// 共通ヘルパー: *string をトリムし、空なら nil にする
func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

// 追加: *[]string を重複排除＋トリムして返す（nil 可）
func normalizeSlicePtr(xs *[]string) *[]string {
	if xs == nil {
		return nil
	}
	clean := dedupStrings(*xs)
	return &clean
}

// 追加: 任意型のポインタを返すユーティリティ
func ptr[T any](v T) *T { return &v }
