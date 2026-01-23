// backend/internal/adapters/in/http/middleware/member_auth.go
package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
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

// MemberCompanyIDReader は「member entity を作らずに companyId だけ」取得するためのオプショナル拡張。
// Firestore 側の repo が実装していれば利用する（validate 失敗による companyId 欠落を回避）。
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

		// ✅ debug: token length / request path
		log.Printf("[auth] bearer token received (len=%d) path=%s", len(idToken), r.URL.Path)

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

		// ------------------------------------------------------------
		// uid → Member（失敗しても companyId だけは回収できるようにする）
		// ------------------------------------------------------------
		var member memdom.Member
		memberOK := false

		memberVal, mErr := m.MemberRepo.GetByFirebaseUID(r.Context(), uid)
		if mErr == nil {
			member = memberVal
			memberOK = true
		}

		// companyId は「member.CompanyID」優先。ただし空なら repo 拡張から回収を試みる。
		companyID := strings.TrimSpace(member.CompanyID)

		// ✅ debug: primary resolve result
		log.Printf(
			"[auth] resolved (primary) uid=%s memberOK=%v memberID=%s companyId=%q",
			uid,
			memberOK,
			strings.TrimSpace(member.ID),
			companyID,
		)

		if companyID == "" {
			// ✅ debug: try optional extension
			if r2, ok := any(m.MemberRepo).(MemberCompanyIDReader); ok {
				if cid, e := r2.GetCompanyIDByFirebaseUID(r.Context(), uid); e == nil {
					companyID = strings.TrimSpace(cid)

					log.Printf(
						"[auth] resolved (repo-ext) uid=%s memberOK(beforePlaceholder)=%v companyId=%q",
						uid,
						memberOK,
						companyID,
					)

					// member が取れていない場合でも、最低限 ctx に積めるように placeholder を作る
					if !memberOK && companyID != "" {
						member = memdom.Member{
							ID:        uid,
							CompanyID: companyID,
						}
						memberOK = true

						log.Printf("[auth] placeholder member created uid=%s companyId=%q", uid, companyID)
					}
				} else {
					log.Printf("[auth] repo-ext GetCompanyIDByFirebaseUID failed uid=%s err=%v", uid, e)
				}
			} else {
				log.Printf("[auth] repo-ext not implemented uid=%s", uid)
			}
		}

		// ✅ company 境界必須：companyId が取れないなら downstream で 500 にせず、ここで止める
		if strings.TrimSpace(companyID) == "" {
			// member が取れていないなら "member not found" に寄せる
			if !memberOK {
				log.Printf("[auth] forbidden: member not found uid=%s", uid)
				http.Error(w, "member not found", http.StatusForbidden)
				return
			}
			log.Printf("[auth] forbidden: companyId not resolved uid=%s memberID=%s", uid, strings.TrimSpace(member.ID))
			http.Error(w, "companyId not resolved for current user", http.StatusForbidden)
			return
		}

		// ------------------------------------------------------------
		// context に格納
		// ------------------------------------------------------------
		ctx := r.Context()

		if memberOK {
			ctx = context.WithValue(ctx, ctxKeyMember, member)

			// usecase に memberID をセット
			if strings.TrimSpace(member.ID) != "" {
				ctx = usecase.WithMemberID(ctx, member.ID)
			} else {
				// placeholder 等で member.ID が空の場合
				ctx = usecase.WithMemberID(ctx, uid)
			}

			// fullName（member が正常に取れている時だけ）
			fullName := memdom.FormatLastFirst(member.LastName, member.FirstName)
			if strings.TrimSpace(fullName) != "" {
				ctx = context.WithValue(ctx, ctxKeyFullName, strings.TrimSpace(fullName))
			}
		} else {
			log.Printf("[auth] internal error: member not found after resolve uid=%s", uid)
			returnErr(w, errors.New("member not found"))
			return
		}

		ctx = context.WithValue(ctx, ctxKeyUID, uid)

		// email 格納
		if emailRaw, ok := token.Claims["email"]; ok {
			if e, ok2 := emailRaw.(string); ok2 && strings.TrimSpace(e) != "" {
				ctx = context.WithValue(ctx, ctxKeyEmail, strings.TrimSpace(e))
			}
		}

		// companyId 格納（必須）
		ctx = usecase.WithCompanyID(ctx, companyID)
		ctx = context.WithValue(ctx, ctxKeyCompanyID, companyID)

		// ✅ debug: final resolved values
		log.Printf(
			"[auth] context set uid=%s memberID=%s companyId=%q path=%s",
			uid,
			strings.TrimSpace(member.ID),
			companyID,
			r.URL.Path,
		)

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
