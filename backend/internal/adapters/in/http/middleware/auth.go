// backend/internal/adapters/in/http/middleware/auth.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	fbauth "firebase.google.com/go/v4/auth"

	memdom "narratives/internal/domain/member"
)

// FirebaseAuthClient は firebase auth クライアントのエイリアス。
// RouterDeps などからは *middleware.FirebaseAuthClient 型で受けられます。
type FirebaseAuthClient = fbauth.Client

// ─────────────────────────────────────────────────────────────
// context key は string を使わず、衝突回避のため独自型を用いる（SA1029 対策）
// ─────────────────────────────────────────────────────────────
type ctxKey struct{ name string }

var (
	ctxKeyMember    = ctxKey{name: "currentMember"}
	ctxKeyCompanyID = ctxKey{name: "companyId"}
	ctxKeyUID       = ctxKey{name: "uid"}
	ctxKeyEmail     = ctxKey{name: "email"}
)

// AuthMiddleware は
//   - Authorization: Bearer <ID_TOKEN>
//
// を検証し、現在メンバーと companyId、uid/email を context に詰めて次のハンドラへ渡します。
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
			log.Printf("[auth] uid=%s member lookup error: %v", uid, err)
			http.Error(w, "member not found", http.StatusForbidden)
			return
		}

		// 追加ログ
		log.Printf("[auth] uid=%s member.ID=%s companyId=%q", uid, member.ID, member.CompanyID)

		// context に格納
		ctx := context.WithValue(r.Context(), ctxKeyMember, member)
		ctx = context.WithValue(ctx, ctxKeyUID, uid)

		// email クレームがあれば context にも入れておく
		if emailRaw, ok := token.Claims["email"]; ok {
			if emailStr, ok2 := emailRaw.(string); ok2 {
				emailStr = strings.TrimSpace(emailStr)
				if emailStr != "" {
					ctx = context.WithValue(ctx, ctxKeyEmail, emailStr)
				}
			}
		}

		// companyId が空でなければ同時に注入
		if cid := strings.TrimSpace(member.CompanyID); cid != "" {
			ctx = context.WithValue(ctx, ctxKeyCompanyID, cid)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentMember は現在ログイン中の Member を取得します。
// 取得できない場合は (nil, false)。
func CurrentMember(r *http.Request) (*memdom.Member, bool) {
	v := r.Context().Value(ctxKeyMember)
	if v == nil {
		return nil, false
	}
	m, ok := v.(*memdom.Member)
	if !ok || m == nil {
		return nil, false
	}
	return m, true
}

// CompanyID は context に注入された companyId を取得します。
// 取得できない場合は ("", false)。
func CompanyID(r *http.Request) (string, bool) {
	v := r.Context().Value(ctxKeyCompanyID)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}

// CurrentUIDAndEmail は middleware で検証された Firebase UID と email を返します。
// email はトークンに含まれない場合、空文字になりえます。
// どちらかが取得できない場合は ok=false。
func CurrentUIDAndEmail(r *http.Request) (uid string, email string, ok bool) {
	vUID := r.Context().Value(ctxKeyUID)
	u, okUID := vUID.(string)
	if !okUID || strings.TrimSpace(u) == "" {
		return "", "", false
	}

	uid = strings.TrimSpace(u)

	vEmail := r.Context().Value(ctxKeyEmail)
	if vEmail != nil {
		if e, okEmail := vEmail.(string); okEmail {
			email = strings.TrimSpace(e)
		}
	}
	return uid, email, true
}
