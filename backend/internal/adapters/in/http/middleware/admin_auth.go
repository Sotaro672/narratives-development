// backend/internal/adapters/in/http/middleware/admin_auth.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
)

type adminContextKey struct {
	name string
}

var (
	ctxKeyAdminUID   = adminContextKey{name: "adminUID"}
	ctxKeyAdminEmail = adminContextKey{name: "adminEmail"}
)

// AdminAuthMiddleware verifies the Firebase ID token and restricts access
// to the configured AMOL administrator.
//
// Authorization conditions:
//   - valid Firebase ID token
//   - token has not been revoked
//   - Firebase UID exactly matches AllowedUID
//   - email exactly matches AllowedEmail, ignoring case only
//   - email is verified
//
// Admin authentication is independent from Console member/company
// authentication.
type AdminAuthMiddleware struct {
	FirebaseAuth *FirebaseAuthClient
	AllowedUID   string
	AllowedEmail string
}

func (m *AdminAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS preflight is allowed without authentication.
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if m.FirebaseAuth == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "admin_auth_not_initialized")
			return
		}

		allowedUID := strings.TrimSpace(m.AllowedUID)
		allowedEmail := strings.TrimSpace(m.AllowedEmail)

		if allowedUID == "" || allowedEmail == "" {
			log.Print("[admin_auth] allowed admin identity is not configured")
			writeJSONError(w, http.StatusServiceUnavailable, "admin_auth_not_configured")
			return
		}

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing authorization header")
			return
		}

		const bearerPrefix = "bearer "

		if !strings.HasPrefix(strings.ToLower(authHeader), bearerPrefix) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing bearer token")
			return
		}

		idToken := strings.TrimSpace(authHeader[len(bearerPrefix):])
		if idToken == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized: empty bearer token")
			return
		}

		token, err := m.FirebaseAuth.VerifyIDTokenAndCheckRevoked(r.Context(), idToken)
		if err != nil {
			log.Printf("[admin_auth] invalid or revoked token: %v", err)
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		uid := strings.TrimSpace(token.UID)
		if uid == "" {
			writeJSONError(w, http.StatusUnauthorized, "invalid uid in token")
			return
		}

		if uid != allowedUID {
			log.Printf("[admin_auth] forbidden uid: %s", uid)
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		emailRaw, ok := token.Claims["email"]
		if !ok {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		email, ok := emailRaw.(string)
		if !ok {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		email = strings.TrimSpace(email)
		if !strings.EqualFold(email, allowedEmail) {
			log.Printf("[admin_auth] forbidden email for allowed uid")
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		emailVerifiedRaw, ok := token.Claims["email_verified"]
		if !ok {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		emailVerified, ok := emailVerifiedRaw.(bool)
		if !ok || !emailVerified {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyAdminUID, uid)
		ctx = context.WithValue(ctx, ctxKeyAdminEmail, email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentAdminUID returns the Firebase UID of the authenticated administrator.
func CurrentAdminUID(r *http.Request) (string, bool) {
	value := r.Context().Value(ctxKeyAdminUID)

	uid, ok := value.(string)
	if !ok || strings.TrimSpace(uid) == "" {
		return "", false
	}

	return uid, true
}

// CurrentAdminEmail returns the Firebase email address of the authenticated
// administrator.
func CurrentAdminEmail(r *http.Request) (string, bool) {
	value := r.Context().Value(ctxKeyAdminEmail)

	email, ok := value.(string)
	if !ok || strings.TrimSpace(email) == "" {
		return "", false
	}

	return email, true
}

// CurrentAdminUIDAndEmail returns the authenticated administrator identity.
func CurrentAdminUIDAndEmail(r *http.Request) (uid string, email string, ok bool) {
	uid, ok = CurrentAdminUID(r)
	if !ok {
		return "", "", false
	}

	email, ok = CurrentAdminEmail(r)
	if !ok {
		return "", "", false
	}

	return uid, email, true
}
