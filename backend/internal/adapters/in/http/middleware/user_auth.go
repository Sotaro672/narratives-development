// backend/internal/adapters/in/http/middleware/user_auth.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// UserAuthMiddleware verifies Firebase ID token (buyer/user side) and stores uid/email in context.
// - Does NOT require MemberRepo / companyId.
// - Intended for SNS onboarding endpoints (/sns/users, /sns/shipping-addresses, /sns/billing-addresses, etc.)
type UserAuthMiddleware struct {
	FirebaseAuth *FirebaseAuthClient
}

func (m *UserAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.FirebaseAuth == nil {
			http.Error(w, "user auth middleware not initialized", http.StatusServiceUnavailable)
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

		log.Printf("[user_auth] bearer token received (len=%d)", len(idToken))

		// Firebase ID token verification
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

		// email (optional)
		email := ""
		if emailRaw, ok := token.Claims["email"]; ok {
			if e, ok2 := emailRaw.(string); ok2 {
				email = strings.TrimSpace(e)
			}
		}

		// fullName (optional) - if provided by Firebase/OIDC
		fullName := ""
		if nameRaw, ok := token.Claims["name"]; ok {
			if s, ok2 := nameRaw.(string); ok2 {
				fullName = strings.TrimSpace(s)
			}
		}
		if fullName == "" {
			if nameRaw, ok := token.Claims["fullName"]; ok {
				if s, ok2 := nameRaw.(string); ok2 {
					fullName = strings.TrimSpace(s)
				}
			}
		}

		// ------------------------------------------------------------
		// put into context
		// ------------------------------------------------------------
		ctx := r.Context()

		// ✅ share the same keys as member_auth.go (so existing helpers work)
		ctx = context.WithValue(ctx, ctxKeyUID, uid)

		if email != "" {
			ctx = context.WithValue(ctx, ctxKeyEmail, email)
		}
		if fullName != "" {
			ctx = context.WithValue(ctx, ctxKeyFullName, fullName)
		}

		// ------------------------------------------------------------
		// legacy string keys for compatibility
		// ------------------------------------------------------------
		ctx = withLegacyStringKey(ctx, "uid", uid)
		ctx = withLegacyStringKey(ctx, "userId", uid)
		ctx = withLegacyStringKey(ctx, "userID", uid)
		ctx = withLegacyStringKey(ctx, "currentUserId", uid)
		ctx = withLegacyStringKey(ctx, "currentUserID", uid)

		if email != "" {
			ctx = withLegacyStringKey(ctx, "email", email)
		}
		if fullName != "" {
			ctx = withLegacyStringKey(ctx, "fullName", fullName)
			ctx = withLegacyStringKey(ctx, "name", fullName)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentUserUID returns Firebase UID for buyer/user side.
func CurrentUserUID(r *http.Request) (string, bool) {
	vUID := r.Context().Value(ctxKeyUID)
	u, ok := vUID.(string)
	if !ok || strings.TrimSpace(u) == "" {
		return "", false
	}
	return strings.TrimSpace(u), true
}

// CurrentUserUIDAndEmail returns uid/email (email can be empty).
func CurrentUserUIDAndEmail(r *http.Request) (uid string, email string, ok bool) {
	uid, ok = CurrentUserUID(r)
	if !ok {
		return "", "", false
	}

	if vEmail := r.Context().Value(ctxKeyEmail); vEmail != nil {
		if e, okEmail := vEmail.(string); okEmail {
			email = strings.TrimSpace(e)
		}
	}

	return uid, email, true
}

// CurrentUserFullName returns fullName if present.
func CurrentUserFullName(r *http.Request) (string, bool) {
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
