// backend/internal/adapters/in/http/middleware/auth.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	fbauth "firebase.google.com/go/v4/auth"

	usecase "narratives/internal/application/usecase"
	memdom "narratives/internal/domain/member"
)

// FirebaseAuthClient は firebase auth クライアントのエイリアス。
// RouterDeps などからは *middleware.FirebaseAuthClient 型で受けられます。
type FirebaseAuthClient = fbauth.Client

// context key は string を使わず、衝突回避のため独自型を使用（SA1029 対策）
type ctxKey struct{ name string }

var (
	ctxKeyMember    = ctxKey{name: "currentMember"}
	ctxKeyCompanyID = ctxKey{name: "companyId"}
	ctxKeyUID       = ctxKey{name: "uid"}
	ctxKeyEmail     = ctxKey{name: "email"}
	ctxKeyFullName  = ctxKey{name: "fullName"} // ★ 追加: 表示名(fullName)
)

// AuthMiddleware は
//
//   - Authorization: Bearer <ID_TOKEN>
//
// を検証し、現在メンバーと companyId、uid/email/fullName を context に詰めて次のハンドラへ渡す。
type AuthMiddleware struct {
	FirebaseAuth *FirebaseAuthClient
	MemberRepo   memdom.Repository
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 依存チェック
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

		// uid → Member
		member, err := m.MemberRepo.GetByFirebaseUID(r.Context(), uid)
		if err != nil {
			http.Error(w, "member not found", http.StatusForbidden)
			return
		}

		// context に格納
		ctx := context.WithValue(r.Context(), ctxKeyMember, member)
		ctx = context.WithValue(ctx, ctxKeyUID, uid)

		// email があれば context に格納
		emailStr := ""
		if emailRaw, ok := token.Claims["email"]; ok {
			if e, ok2 := emailRaw.(string); ok2 && strings.TrimSpace(e) != "" {
				emailStr = strings.TrimSpace(e)
				ctx = context.WithValue(ctx, ctxKeyEmail, emailStr)
			}
		}

		// ★ fullName を member から組み立てて context に格納
		fullName := memdom.FormatLastFirst(member.LastName, member.FirstName)
		if strings.TrimSpace(fullName) != "" {
			ctx = context.WithValue(ctx, ctxKeyFullName, strings.TrimSpace(fullName))
		}

		// companyId が空でなければ context に格納
		if cid := strings.TrimSpace(member.CompanyID); cid != "" {
			ctx = usecase.WithCompanyID(ctx, cid)
			ctx = context.WithValue(ctx, ctxKeyCompanyID, cid)

			// ★ ここで companyId を持たせているかログに出力する
			log.Printf(
				"[AuthMiddleware] path=%s uid=%s companyId=%s email=%s",
				r.URL.Path,
				uid,
				cid,
				emailStr,
			)
		} else {
			// ★ companyId が空だった場合もわかるようにログ
			log.Printf(
				"[AuthMiddleware] path=%s uid=%s has NO companyId (email=%s)",
				r.URL.Path,
				uid,
				emailStr,
			)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentMember は現在ログイン中の Member を取得します。
func CurrentMember(r *http.Request) (*memdom.Member, bool) {
	v := r.Context().Value(ctxKeyMember)
	if v == nil {
		return nil, false
	}

	if mPtr, ok := v.(*memdom.Member); ok && mPtr != nil {
		return mPtr, true
	}

	if mVal, ok := v.(memdom.Member); ok {
		m := mVal
		return &m, true
	}

	return nil, false
}

// CompanyID は context に注入された companyId を取得します。
func CompanyID(r *http.Request) (string, bool) {
	v := r.Context().Value(ctxKeyCompanyID)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", false
	}
	return strings.TrimSpace(s), true
}

// CurrentUIDAndEmail は middleware で検証された Firebase UID と email を返します。
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

// ★ 追加: CurrentFullName
// middleware で注入された表示名(fullName)を取得します。
func CurrentFullName(r *http.Request) (string, bool) {
	v := r.Context().Value(ctxKeyFullName)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", false
	}
	return strings.TrimSpace(s), true
}
