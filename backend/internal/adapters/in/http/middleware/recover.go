// backend\internal\adapters\in\http\middleware\recover.go
package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// panic の真因を Cloud Run logs に残す
				log.Printf("[recover] PANIC: %v\n%s", rec, string(debug.Stack()))

				// ここで必ずレスポンスを返す（Cloud Run に 503 を作らせない）
				// ※ CORS は外側で付ける（後述のチェーン順が重要）
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"internal server error","detail":"%v"}`, rec)))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
