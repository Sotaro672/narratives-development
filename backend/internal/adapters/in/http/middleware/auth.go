// backend/internal/adapters/in/http/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"

	memdom "narratives/internal/domain/member"
)

type ctxKey string

const ctxKeyMember ctxKey = "currentMember"

type AuthMiddleware struct {
	FirebaseAuth *auth.Client
	MemberRepo   memdom.Repository // Firestore 実装を渡す
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

		// ここで uid から Member を引く（実装はプロジェクト次第）
		member, err := m.MemberRepo.GetByFirebaseUID(r.Context(), uid)
		if err != nil {
			http.Error(w, "member not found", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyMember, member)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// どこからでも current member を取り出せる helper
func CurrentMember(r *http.Request) (*memdom.Member, bool) {
	m, ok := r.Context().Value(ctxKeyMember).(memdom.Member)
	if !ok {
		return nil, false
	}
	return &m, true
}
