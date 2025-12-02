package middleware

import "net/http"

func CORS(next http.Handler) http.Handler {

	// ★ 許可する Origin を動的に管理する
	allowedOrigins := map[string]bool{
		"https://narratives.jp":                        true, // 本番
		"https://narratives-console-dev.web.app":       true, // 管理画面フロント dev
		"https://narratives-development-26c2d.web.app": true, // 検品アプリ dev
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		origin := r.Header.Get("Origin")
		_, ok := allowedOrigins[origin]

		if ok {
			// ★ requested origin をそのまま返す
			w.Header().Set("Access-Control-Allow-Origin", origin)

			// ★ Cookie を使う可能性があるため必須
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// ★ キャッシュしつつヘッダの揺れを防ぐ
			w.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")

			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		}

		// ★ プリフライトならここで終了
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
