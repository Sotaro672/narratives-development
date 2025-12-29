// backend/internal/adapters/in/http/middleware/cors.go
package middleware

import (
	"net/http"
	"strings"
)

func CORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"https://narratives.jp":                        true, // 本番
		"https://narratives-console-dev.web.app":       true, // 管理画面フロント dev
		"https://narratives-development-26c2d.web.app": true, // 検品アプリ dev
		"http://localhost:5173":                        true, // ローカル dev (Vite)
		"http://127.0.0.1:5173":                        true, // ローカル dev

		// ✅ SNS (buyer-facing)
		"https://narratives-development-sns.web.app":         true,
		"https://narratives-development-sns.firebaseapp.com": true,
	}

	allowedHeaders := strings.Join([]string{
		"Authorization",
		"Content-Type",
		"Accept",
		"Origin",
		"X-Requested-With",

		// custom headers
		"X-Actor-Id",
		"X-Icon-Content-Type",
		"X-Icon-File-Name",

		"X-CSRF-Token",
	}, ", ")

	allowedMethods := "GET,POST,PUT,PATCH,DELETE,OPTIONS"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))

		// ブラウザ以外（Originなし）
		if origin == "" {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Origin が許可されていない場合
		if !allowedOrigins[origin] {
			// preflight はここで明示的に拒否（デバッグしやすくする）
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			// 非preflightはCORSヘッダ無しで通す（ブラウザ側でブロックされる）
			next.ServeHTTP(w, r)
			return
		}

		// ✅ 許可 origin の場合のみ CORS ヘッダ付与
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		appendVary(w, "Origin")
		appendVary(w, "Access-Control-Request-Method")
		appendVary(w, "Access-Control-Request-Headers")

		w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		w.Header().Set("Access-Control-Max-Age", "600")

		// preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func appendVary(w http.ResponseWriter, value string) {
	const key = "Vary"
	cur := w.Header().Get(key)
	if cur == "" {
		w.Header().Set(key, value)
		return
	}

	// すでに含まれていれば何もしない
	parts := strings.Split(cur, ",")
	for _, p := range parts {
		if strings.EqualFold(strings.TrimSpace(p), value) {
			return
		}
	}
	w.Header().Set(key, cur+", "+value)
}
