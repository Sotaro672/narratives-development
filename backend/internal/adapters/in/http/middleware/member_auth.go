// backend/internal/adapters/in/http/middleware/member_auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	fbauth "firebase.google.com/go/v4/auth"

	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
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

// MemberCompanyIDReader は「member entity を作らずに companyId だけ」取得するためのオプショナル拡張。
// repository port 本体は GetByID / ListByCompanyID に寄せているため、
// Firebase UID から companyID を解決する処理だけ adapter 拡張として扱う。
type MemberCompanyIDReader interface {
	GetCompanyIDByFirebaseUID(ctx context.Context, uid string) (string, error)
}

// AuthMiddleware は Bearer <ID_TOKEN> を検証し、member と各情報を context に詰める。
type AuthMiddleware struct {
	FirebaseAuth *FirebaseAuthClient
	MemberRepo   memdom.Repository
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS preflight は認証なしで通す
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if m.FirebaseAuth == nil || m.MemberRepo == nil {
			http.Error(w, "auth middleware not initialized", http.StatusServiceUnavailable)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized: missing bearer token", http.StatusUnauthorized)
			return
		}

		idToken := strings.TrimPrefix(authHeader, "Bearer ")
		idToken = strings.TrimSpace(idToken)
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

		uid := token.UID
		if uid == "" {
			http.Error(w, "invalid uid in token", http.StatusUnauthorized)
			return
		}

		// ------------------------------------------------------------
		// Firebase UID -> companyID -> Member
		// ------------------------------------------------------------
		// そのため、まず adapter 拡張で companyID を解決し、
		// その companyID scope 内で ListByCompanyID + Filter.UID により member を取得する。
		var member memdom.Member
		memberDocID := ""
		memberOK := false
		companyID := ""

		if r2, ok := any(m.MemberRepo).(MemberCompanyIDReader); ok {
			cid, e := r2.GetCompanyIDByFirebaseUID(r.Context(), uid)
			if e == nil {
				companyID = strings.TrimSpace(cid)
			}
		}

		if companyID != "" {
			res, e := m.MemberRepo.ListByCompanyID(
				r.Context(),
				companyID,
				memdom.Filter{
					UID: uid,
				},
				common.Page{
					Number:  1,
					PerPage: 1,
				},
			)
			if e == nil && len(res.Items) > 0 {
				rec := res.Items[0]
				memberDocID = rec.DocID
				member = rec.Member
				memberOK = true

				if member.CompanyID == "" {
					member.CompanyID = companyID
				}
			}
		}

		// company 境界必須
		if companyID == "" {
			http.Error(w, "companyId not resolved for current user", http.StatusForbidden)
			return
		}

		if !memberOK {
			http.Error(w, "member not found", http.StatusForbidden)
			return
		}

		if member.CompanyID != "" && member.CompanyID != companyID {
			http.Error(w, "companyId mismatch for current user", http.StatusForbidden)
			return
		}

		// ------------------------------------------------------------
		// context に格納
		// ------------------------------------------------------------
		ctx := r.Context()

		ctx = context.WithValue(ctx, ctxKeyMember, member)

		// member の識別子は Firestore docId を使う。
		// docId が空になることは通常ないが、念のため uid fallback を置く。
		if memberDocID != "" {
			ctx = usecase.WithMemberID(ctx, memberDocID)
		} else {
			ctx = usecase.WithMemberID(ctx, uid)
		}

		fullName := memdom.FormatLastFirst(member.LastName, member.FirstName)
		if fullName != "" {
			ctx = context.WithValue(ctx, ctxKeyFullName, fullName)
		}

		ctx = context.WithValue(ctx, ctxKeyUID, uid)

		// email 格納
		if emailRaw, ok := token.Claims["email"]; ok {
			if e, ok2 := emailRaw.(string); ok2 && e != "" {
				ctx = context.WithValue(ctx, ctxKeyEmail, e)
			}
		}

		// companyId 格納（必須）
		ctx = usecase.WithCompanyID(ctx, companyID)
		ctx = context.WithValue(ctx, ctxKeyCompanyID, companyID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func returnErr(w http.ResponseWriter, err error) {
	if err == nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
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
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// CurrentUIDAndEmail を返す
func CurrentUIDAndEmail(r *http.Request) (uid string, email string, ok bool) {
	vUID := r.Context().Value(ctxKeyUID)
	u, okUID := vUID.(string)
	if !okUID || u == "" {
		return "", "", false
	}
	uid = u

	if vEmail := r.Context().Value(ctxKeyEmail); vEmail != nil {
		if e, okEmail := vEmail.(string); okEmail {
			email = e
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
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// CurrentMemberID は現在ログイン中の member docId を取得します。
// usecase.WithMemberID で context に格納される値を参照したい場合は、
// application/usecase 側の getter を使ってください。
func CurrentMemberID(r *http.Request) (string, bool) {
	member, ok := CurrentMember(r)
	if !ok || member == nil {
		return "", false
	}
	return "", false
}
