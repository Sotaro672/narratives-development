// backend/internal/adapters/in/http/middleware/user_auth.go
package middleware

import (
	"context"
	"encoding/json"
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
		// ✅ Allow CORS preflight to pass through without auth
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if m.FirebaseAuth == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "user_auth_not_initialized")
			return
		}

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing authorization header")
			return
		}

		// ✅ Case-insensitive "Bearer "
		// (keep original header for token extraction)
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized: missing bearer token")
			return
		}

		idToken := strings.TrimSpace(authHeader[len("Bearer "):])
		if idToken == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized: empty bearer token")
			return
		}

		// Firebase ID token verification
		token, err := m.FirebaseAuth.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			// NOTE: Do not leak internal details to clients
			log.Printf("[user_auth] invalid token: %v", err)
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		uid := strings.TrimSpace(token.UID)
		if uid == "" {
			writeJSONError(w, http.StatusUnauthorized, "invalid uid in token")
			return
		}

		// email (optional)
		email := ""
		if emailRaw, ok := token.Claims["email"]; ok {
			if e, ok2 := emailRaw.(string); ok2 {
				email = strings.TrimSpace(e)
			}
		}

		// fullName (optional)
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

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// CurrentUserUID returns Firebase UID for buyer/user side.
func CurrentUserUID(r *http.Request) (string, bool) {
	vUID := r.Context().Value(ctxKeyUID)
	u, ok := vUID.(string)
	if !ok {
		return "", false
	}
	u = strings.TrimSpace(u)
	if u == "" {
		return "", false
	}
	return u, true
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
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}
