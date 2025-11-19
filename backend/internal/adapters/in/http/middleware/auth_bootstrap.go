// backend/internal/adapters/in/http/middleware/auth_bootstrap.go
package middleware

import (
	"context"
	"net/http"
	"strings"
)

// BootstrapAuthMiddleware は /auth/bootstrap 専用の簡易ミドルウェア。
// - Authorization: Bearer <ID_TOKEN> を検証
// - UID と email を context に詰める
// - MemberRepo での member lookup は行わない（まだ存在しない可能性があるため）
type BootstrapAuthMiddleware struct {
	FirebaseAuth *FirebaseAuthClient
}

func (m *BootstrapAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.FirebaseAuth == nil {
			http.Error(w, "auth middleware not initialized", http.StatusServiceUnavailable)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized: missing bearer token", http.StatusUnauthorized)
			return
		}

		idToken := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if idToken == "" {
			http.Error(w, "unauthorized: empty bearer token", http.StatusUnauthorized)
			return
		}

		// Firebase ID トークン検証
		token, err := m.FirebaseAuth.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		uid := strings.TrimSpace(token.UID)
		if uid == "" {
			http.Error(w, "invalid uid in token", http.StatusUnauthorized)
			return
		}

		// UID を context に格納（既存の ctxKeyUID / ctxKeyEmail をそのまま利用）
		ctx := context.WithValue(r.Context(), ctxKeyUID, uid)

		// email クレームがあれば context にも入れておく
		if emailRaw, ok := token.Claims["email"]; ok {
			if emailStr, ok2 := emailRaw.(string); ok2 {
				emailStr = strings.TrimSpace(emailStr)
				if emailStr != "" {
					ctx = context.WithValue(ctx, ctxKeyEmail, emailStr)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
