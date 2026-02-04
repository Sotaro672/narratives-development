// backend\internal\application\usecase\helper_usecase.go
package usecase

import "strings"

// 共通の「未サポート」エラー型とヘルパー
type notSupportedError struct{ op string }

func (e notSupportedError) Error() string {
	return "usecase: operation not supported: " + e.op
}

// ErrNotSupported は未サポート操作を表すエラーを返します。
func ErrNotSupported(op string) error { return notSupportedError{op: op} }

// ✅ 追加: 不正リクエスト/引数エラー（usecase パッケージ内で完結させる）
type invalidRequestError struct{ msg string }

func (e invalidRequestError) Error() string {
	m := strings.TrimSpace(e.msg)
	if m == "" {
		return "usecase: invalid request"
	}
	return "usecase: invalid request: " + m
}

// ErrInvalidRequest は「入力が不正」を表すエラーを返します。
func ErrInvalidRequest(msg string) error { return invalidRequestError{msg: msg} }

// ErrInvalidArgument は ErrInvalidRequest のエイリアスとして扱います（互換用）。
func ErrInvalidArgument(msg string) error { return ErrInvalidRequest(msg) }

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

// 追加: ログ用マスク（usecase 内で共通利用）
// NOTE: 既に transfer_usecase.go にある _mask と同一実装にしてください。
func _mask(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
