// backend/internal/adapters/in/http/middleware/cors.go
package middleware

import "net/http"

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 許可するフロントのオリジンに置き換え（例：Firebase Hosting のドメイン）
		// 開発中は "*" でも可だが、本番は厳密に！
		w.Header().Set("Access-Control-Allow-Origin", "https://narratives-console-dev.web.app")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
