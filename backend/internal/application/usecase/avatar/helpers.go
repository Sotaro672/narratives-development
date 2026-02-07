package avatar

import "strings"

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
