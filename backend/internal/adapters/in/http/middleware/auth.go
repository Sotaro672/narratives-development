// backend/internal/adapters/in/http/middleware/auth.go
package middleware

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"

	fbauth "firebase.google.com/go/v4/auth"

	usecase "narratives/internal/application/usecase"
	memdom "narratives/internal/domain/member"
)

// FirebaseAuthClient は firebase auth クライアントのエイリアス。
type FirebaseAuthClient = fbauth.Client

// context key は string を使わず、衝突回避のため独自型を使用（SA1029 対策）
type ctxKey struct{ name string }

var (
	ctxKeyMember    = ctxKey{name: "currentMember"}
	ctxKeyCompanyID = ctxKey{name: "companyId"}
	ctxKeyUID       = ctxKey{name: "uid"}
	ctxKeyEmail     = ctxKey{name: "email"}
	ctxKeyFullName  = ctxKey{name: "fullName"}
)

// AuthMiddleware は Bearer <ID_TOKEN> を検証し、member と各情報を context に詰める。
type AuthMiddleware struct {
	FirebaseAuth *FirebaseAuthClient
	MemberRepo   memdom.Repository
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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

		// ============================================================
		// ★ ID TOKEN をログに完全表示 + Base64 化してファイル保存
		// ============================================================
		log.Printf("[auth] RAW ID TOKEN (first 50 chars): %.50s...", idToken)

		// Base64 encode the token for safe logging
		encoded := base64.StdEncoding.EncodeToString([]byte(idToken))

		// Write full encoded token to file
		f, err := os.OpenFile("debug-idtoken.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			defer f.Close()
			f.WriteString("========================================\n")
			f.WriteString("RAW ID TOKEN:\n")
			f.WriteString(idToken + "\n\n")
			f.WriteString("BASE64 ENCODED:\n")
			f.WriteString(encoded + "\n")
			f.WriteString("========================================\n\n")
		} else {
			log.Printf("[auth] ERROR writing debug-idtoken.log: %v", err)
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

		// ★★★ usecase に memberID をセット（今回の 500 の原因修正ポイント）★★★
		if strings.TrimSpace(member.ID) != "" {
			ctx = usecase.WithMemberID(ctx, member.ID)
		}

		// email 格納
		if emailRaw, ok := token.Claims["email"]; ok {
			if e, ok2 := emailRaw.(string); ok2 && strings.TrimSpace(e) != "" {
				ctx = context.WithValue(ctx, ctxKeyEmail, strings.TrimSpace(e))
			}
		}

		// fullName 格納
		fullName := memdom.FormatLastFirst(member.LastName, member.FirstName)
		if strings.TrimSpace(fullName) != "" {
			ctx = context.WithValue(ctx, ctxKeyFullName, strings.TrimSpace(fullName))
		}

		// companyId 格納
		if cid := strings.TrimSpace(member.CompanyID); cid != "" {
			ctx = usecase.WithCompanyID(ctx, cid)
			ctx = context.WithValue(ctx, ctxKeyCompanyID, cid)
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

// CompanyID を取得
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

// CurrentUIDAndEmail を返す
func CurrentUIDAndEmail(r *http.Request) (uid string, email string, ok bool) {
	vUID := r.Context().Value(ctxKeyUID)
	u, okUID := vUID.(string)
	if !okUID || strings.TrimSpace(u) == "" {
		return "", "", false
	}
	uid = strings.TrimSpace(u)

	if vEmail := r.Context().Value(ctxKeyEmail); vEmail != nil {
		if e, okEmail := vEmail.(string); okEmail {
			email = strings.TrimSpace(e)
		}
	}

	return uid, email, true
}

// CurrentFullName を返す
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
