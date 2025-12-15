package middleware

import (
	"net/http"
	"strings"
)

func CORS(next http.Handler) http.Handler {
	// ★ 許可する Origin を動的に管理する
	allowedOrigins := map[string]bool{
		"https://narratives.jp":                        true, // 本番
		"https://narratives-console-dev.web.app":       true, // 管理画面フロント dev
		"https://narratives-development-26c2d.web.app": true, // 検品アプリ dev
	}

	// ★ 許可するヘッダ（preflight の Access-Control-Request-Headers と突合される）
	allowedHeaders := strings.Join([]string{
		"Authorization",
		"Content-Type",
		"Accept",
		"X-Actor-Id",
		"X-Icon-Content-Type",
		"X-Icon-File-Name",
	}, ", ")

	allowedMethods := "GET,POST,PUT,PATCH,DELETE,OPTIONS"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		_, ok := allowedOrigins[origin]

		if ok {
			// ★ requested origin をそのまま返す
			w.Header().Set("Access-Control-Allow-Origin", origin)

			// ★ Cookie を使う可能性があるため必須
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// ★ キャッシュしつつヘッダの揺れを防ぐ
			w.Header().Set(
				"Vary",
				"Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
			)

			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)

			// ★ 重要: フロントが送るカスタムヘッダを許可する
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

			// ★ あると便利: preflight の結果をキャッシュ（秒）
			w.Header().Set("Access-Control-Max-Age", "600")
		}

		// ★ プリフライトならここで終了
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
