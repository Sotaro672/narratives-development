// backend/internal/adapters/in/http/middleware/cors.go
package middleware

import (
	"net/http"
	"strings"
)

func CORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"https://amol.jp":                              true,
		"https://narratives-console-dev.web.app":       true,
		"https://narratives-development-26c2d.web.app": true,

		// Admin
		"https://amol-admin.web.app":         true,
		"https://amol-admin.firebaseapp.com": true,

		// Inspector
		"https://amol-inspector.web.app":         true,
		"https://amol-inspector.firebaseapp.com": true,

		// Local dev
		"http://localhost:5173": true,
		"http://127.0.0.1:5173": true,

		// Introduction
		"https://narratives-introduction.web.app":         true,
		"https://narratives-introduction.firebaseapp.com": true,

		// Mall
		"https://narratives-development-mall.web.app":         true,
		"https://narratives-development-mall.firebaseapp.com": true,
	}

	allowedHeaders := strings.Join([]string{
		"Authorization",
		"Content-Type",
		"Accept",
		"Origin",
		"X-Requested-With",
		"X-Actor-Id",
		"X-Icon-Content-Type",
		"X-Icon-File-Name",
		"Idempotency-Key",
		"X-CSRF-Token",
	}, ", ")

	allowedMethods := "GET,POST,PUT,PATCH,DELETE,OPTIONS"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Browser以外のリクエスト
		if origin == "" {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
			return
		}

		// 許可されていないOrigin
		if !allowedOrigins[origin] {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			// CORSヘッダを付与せず通し、ブラウザ側で拒否させる
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		w.Header().Set("Access-Control-Max-Age", "600")

		appendVary(w, "Origin")
		appendVary(w, "Access-Control-Request-Method")
		appendVary(w, "Access-Control-Request-Headers")

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

	parts := strings.Split(cur, ",")
	for _, part := range parts {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return
		}
	}

	w.Header().Set(key, cur+", "+value)
}
