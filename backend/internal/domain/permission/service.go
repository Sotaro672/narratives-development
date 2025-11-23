// backend\internal\domain\permission\service.go
package permission

import (
	"strings"
)

// ------------------------------------------------------------
// 追加: 権限名から日本語名を取得するヘルパ
// ------------------------------------------------------------

// DisplayNameJaFromPermissionName は、権限名から日本語表示名を返します。
// - name は "wallet.view" などの Permission.Name
// - allPermissions カタログに存在する場合、その Description（日本語名）を返す
// - 見つからない場合は ("", false) を返す
func DisplayNameJaFromPermissionName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	for _, p := range allPermissions {
		if p.Name == name {
			// Permission の第3引数（Description）を日本語表示名として扱う
			return strings.TrimSpace(p.Description), true
		}
	}
	return "", false
}
