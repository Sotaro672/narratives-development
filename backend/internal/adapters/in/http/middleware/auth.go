// backend/internal/adapters/in/http/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	fbAuth "firebase.google.com/go/v4/auth"

	memdom "narratives/internal/domain/member"
)

type ctxKey string

const ctxKeyMember ctxKey = "currentMember"

// AuthMiddleware は Firebase ID トークンを検証し、
// 対応する Member をコンテキストに積むためのミドルウェアです。
type AuthMiddleware struct {
	FirebaseAuth *fbAuth.Client
	MemberRepo   memdom.Repository // Firestore 実装（MemberRepositoryFS）を渡す
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		idToken := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := m.FirebaseAuth.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		uid := token.UID

		// Firebase UID から Member を取得（MemberRepositoryFS.GetByFirebaseUID）
		member, err := m.MemberRepo.GetByFirebaseUID(r.Context(), uid)
		if err != nil {
			http.Error(w, "member not found", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyMember, member)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentMember はコンテキストからログイン中 Member を取り出すためのヘルパーです。
func CurrentMember(r *http.Request) (*memdom.Member, bool) {
	m, ok := r.Context().Value(ctxKeyMember).(memdom.Member)
	if !ok {
		return nil, false
	}
	return &m, true
}
