package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

// 共通: 405
func methodNotAllowed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
}

// 共通: 未サポート判定（メッセージベース）
func isNotSupported(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not supported")
}

// 共通: 文字列ポインタの正規化（空白のみは nil にする）
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
