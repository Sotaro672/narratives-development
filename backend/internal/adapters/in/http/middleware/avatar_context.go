// backend/internal/adapters/in/http/middleware/avatar_context.go
package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// context keys (typed)
type ctxKeyAvatarIDType struct{}
type ctxKeyWalletAddressType struct{}

var (
	ctxKeyAvatarID      = ctxKeyAvatarIDType{}
	ctxKeyWalletAddress = ctxKeyWalletAddressType{}
)

// AvatarResolver resolves avatarId + walletAddress by Firebase uid.
type AvatarResolver interface {
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarId string, walletAddress string, err error)
}

// AvatarContextMiddleware resolves uid -> (avatarId, walletAddress) and stores them into request context.
// Intended to run AFTER UserAuthMiddleware.
type AvatarContextMiddleware struct {
	Resolver AvatarResolver
}

func (m *AvatarContextMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ✅ Allow CORS preflight to pass through
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// ✅ fail-closed if middleware wiring is broken
		if m == nil || m.Resolver == nil {
			writeJSONErrorAvatar(w, http.StatusServiceUnavailable, "avatar_resolver_not_initialized")
			return
		}

		uid, ok := CurrentUserUID(r)
		if !ok || strings.TrimSpace(uid) == "" {
			writeJSONErrorAvatar(w, http.StatusUnauthorized, "unauthorized: missing uid")
			return
		}
		uid = strings.TrimSpace(uid)

		avatarId, walletAddress, err := m.Resolver.ResolveAvatarByUID(r.Context(), uid)
		if err != nil {
			// Prefer 404 for “no avatar for uid”, but do not hide real infra errors.
			if isNotFoundLike(err) {
				writeJSONErrorAvatar(w, http.StatusNotFound, "avatar_not_found_for_uid")
				return
			}
			log.Printf("[avatar_context] ResolveAvatarByUID failed uid=%q err=%v", maskID(uid), err)
			writeJSONErrorAvatar(w, http.StatusInternalServerError, "avatar_resolve_failed")
			return
		}

		avatarId = strings.TrimSpace(avatarId)
		walletAddress = strings.TrimSpace(walletAddress)

		if avatarId == "" {
			writeJSONErrorAvatar(w, http.StatusNotFound, "avatar_not_found_for_uid")
			return
		}

		ctx := withAvatarContext(r.Context(), avatarId, walletAddress)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withAvatarContext(ctx context.Context, avatarId string, walletAddress string) context.Context {
	avatarId = strings.TrimSpace(avatarId)
	walletAddress = strings.TrimSpace(walletAddress)

	ctx = context.WithValue(ctx, ctxKeyAvatarID, avatarId)
	ctx = context.WithValue(ctx, ctxKeyWalletAddress, walletAddress)
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

// CurrentWalletAddress returns walletAddress stored by AvatarContextMiddleware.
func CurrentWalletAddress(r *http.Request) (string, bool) {
	v := r.Context().Value(ctxKeyWalletAddress)
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	// walletAddress は未設定の avatar もあり得るので、空なら false を返す
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

// Heuristic to classify “not found” errors without importing domain-specific packages.
func isNotFoundLike(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(err.Error()))
	if s == "" {
		return false
	}
	return strings.Contains(s, "not_found") ||
		strings.Contains(s, "not found") ||
		strings.Contains(s, "no such document") ||
		strings.Contains(s, "document not found") ||
		strings.Contains(s, "no documents") ||
		strings.Contains(s, "does not exist")
}

func maskID(s string) string {
	t := strings.TrimSpace(s)
	if len(t) <= 8 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
