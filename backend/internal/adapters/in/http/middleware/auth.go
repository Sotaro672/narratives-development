// backend/internal/adapters/in/http/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	fbauth "firebase.google.com/go/v4/auth"

	memdom "narratives/internal/domain/member"
)

// FirebaseAuthClient は firebase auth クライアントのエイリアス。
// RouterDeps などからは *middleware.FirebaseAuthClient 型で受けられます。
type FirebaseAuthClient = fbauth.Client

type ctxKey string

const ctxKeyMember ctxKey = "currentMember"

// AuthMiddleware は
//   - Authorization: Bearer <ID_TOKEN>
//
// で送られてきた Firebase ID トークンを検証し、
//   - currentMember（memdom.Member）
//   - companyId / auth.companyId（string）
//
// を context に詰めて下流ハンドラへ渡します。
type AuthMiddleware struct {
	FirebaseAuth *FirebaseAuthClient
	MemberRepo   memdom.Repository
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 依存が nil の場合は 503 を返して早期終了
		if m.FirebaseAuth == nil || m.MemberRepo == nil {
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

		// uid → Member 解決（現在は「id = FirebaseUID」前提のラッパ）
		member, err := m.MemberRepo.GetByFirebaseUID(r.Context(), uid)
		if err != nil {
			http.Error(w, "member not found", http.StatusForbidden)
			return
		}

		// ★ currentMember と companyId を context に詰める
		ctx := context.WithValue(r.Context(), ctxKeyMember, member)

		cid := strings.TrimSpace(member.CompanyID)
		if cid != "" {
			// MemberUsecase.companyIDFromContext が読むキー
			ctx = context.WithValue(ctx, "companyId", cid)
			ctx = context.WithValue(ctx, "auth.companyId", cid) // 互換キー
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentMember はミドルウェアで注入した現在ログイン中の Member を取得します。
func CurrentMember(r *http.Request) (*memdom.Member, bool) {
	m, ok := r.Context().Value(ctxKeyMember).(memdom.Member)
	if !ok {
		return nil, false
	}
	return &m, true
}
