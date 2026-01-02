// backend/internal/adapters/in/http/middleware/avatar_context.go
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// context key
type ctxKeyAvatarIDType struct{}

var ctxKeyAvatarID = ctxKeyAvatarIDType{}

// AvatarIDResolver resolves avatarId by Firebase uid.
type AvatarIDResolver interface {
	ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error)
}

// AvatarContextMiddleware resolves uid -> avatarId and stores it into request context.
// Intended to run AFTER UserAuthMiddleware.
type AvatarContextMiddleware struct {
	Resolver      AvatarIDResolver
	AllowExplicit bool // if true, allow X-Avatar-Id / ?avatarId to override
}

func (m *AvatarContextMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ✅ Allow CORS preflight
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		uid, ok := CurrentUserUID(r)
		if !ok {
			writeJSONErrorAvatar(w, http.StatusUnauthorized, "unauthorized: missing uid")
			return
		}

		// optional explicit avatarId (header/query)
		if m.AllowExplicit {
			if v := strings.TrimSpace(r.Header.Get("X-Avatar-Id")); v != "" {
				ctx := withAvatarID(r.Context(), v)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if v := strings.TrimSpace(r.URL.Query().Get("avatarId")); v != "" {
				ctx := withAvatarID(r.Context(), v)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if m == nil || m.Resolver == nil {
			writeJSONErrorAvatar(w, http.StatusServiceUnavailable, "avatar_resolver_not_initialized")
			return
		}

		avatarId, err := m.Resolver.ResolveAvatarIDByUID(r.Context(), uid)
		if err != nil {
			// ✅ ここは「uid に紐づく avatar が無い」ケースが最も多いので 404 扱いに寄せる
			writeJSONErrorAvatar(w, http.StatusNotFound, "avatar_not_found_for_uid")
			return
		}
		avatarId = strings.TrimSpace(avatarId)
		if avatarId == "" {
			writeJSONErrorAvatar(w, http.StatusNotFound, "avatar_not_found_for_uid")
			return
		}

		ctx := withAvatarID(r.Context(), avatarId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withAvatarID(ctx context.Context, avatarId string) context.Context {
	avatarId = strings.TrimSpace(avatarId)
	ctx = context.WithValue(ctx, ctxKeyAvatarID, avatarId)

	// legacy string keys（互換用）
	ctx = withLegacyStringKey(ctx, "avatarId", avatarId)
	ctx = withLegacyStringKey(ctx, "avatarID", avatarId)
	ctx = withLegacyStringKey(ctx, "currentAvatarId", avatarId)
	ctx = withLegacyStringKey(ctx, "currentAvatarID", avatarId)

	return ctx
}

// CurrentAvatarID returns avatarId stored by AvatarContextMiddleware.
func CurrentAvatarID(r *http.Request) (string, bool) {
	v := r.Context().Value(ctxKeyAvatarID)
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

func writeJSONErrorAvatar(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
