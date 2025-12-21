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
		"http://localhost:5173":                        true, // ローカル dev (Vite)
		"http://127.0.0.1:5173":                        true, // ローカル dev
	}

	// ★ 許可するヘッダ（preflight の Access-Control-Request-Headers と突合される）
	// - 実運用では「フロントが送る可能性があるもの」を広めに許可しておく
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

		// もし将来使う場合もあるので許可しておく（害は少ない）
		"X-CSRF-Token",
	}, ", ")

	allowedMethods := "GET,POST,PUT,PATCH,DELETE,OPTIONS"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))

		// ブラウザ以外（Originなし）はそのまま通す
		if origin == "" {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Origin が許可されている場合のみ CORS ヘッダ付与
		if allowedOrigins[origin] {
			// ★ requested origin をそのまま返す（credentials を使うため "*" は不可）
			w.Header().Set("Access-Control-Allow-Origin", origin)

			// ★ Cookie / Authorization 等を伴う可能性があるため
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// ★ Origin によって応答が変わるので Vary が必須
			// - ここを "Set" すると他 middleware の Vary を潰す可能性があるので Add を使う
			w.Header().Add("Vary", "Origin")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")

			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

			// ★ preflight の結果をキャッシュ（秒）
			w.Header().Set("Access-Control-Max-Age", "600")
		}

		// ★ プリフライトならここで終了（許可されない origin でも 204 は返す）
		// - ブラウザ側は CORS ヘッダが無ければ結局ブロックするが、
		//   ここで 4xx を返すより問題切り分けがしやすい
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
