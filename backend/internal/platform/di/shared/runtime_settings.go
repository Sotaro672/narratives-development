// backend/internal/platform/di/shared/runtime_settings.go
package shared

import (
	"errors"
	"os"
	"strings"

	appcfg "narratives/internal/infra/config"
)

// RuntimeSettings is env/config-resolved runtime settings (normalized once).
// It intentionally contains only "values" (no external clients).
//
// Policy:
// - Prefer config (cfg) where available.
// - Use env fallbacks where historically used.
// - Apply defaults to keep legacy behavior.
// - Keep normalization (trim, trailing slash removal) here.
// - Keep hard validation in runtime_settings_validate.go.
type RuntimeSettings struct {
	// Used by PaymentFlow (self webhook trigger)
	SelfBaseURL string

	// Used by OwnerResolve query
	BrandsCollection  string
	AvatarsCollection string

	// Used by Transfer signer provider (Design B)
	BrandWalletSecretPrefix string

	// Buckets
	TokenIconBucket     string
	TokenContentsBucket string
	ListImageBucket     string
	AvatarIconBucket    string
	PostImageBucket     string
}

// ResolveRuntimeSettings resolves and normalizes runtime settings from cfg/env.
//
// Notes:
// - This function is side-effect free (no logging).
// - It returns warnings as strings so callers can decide how to surface them.
func ResolveRuntimeSettings(cfg *appcfg.Config) (RuntimeSettings, []string, error) {
	if cfg == nil {
		return RuntimeSettings{}, nil, errors.New("shared.runtime_settings: cfg is nil")
	}

	var warns []string
	var s RuntimeSettings

	// Self base URL (env only; normalize trailing slash)
	s.SelfBaseURL = normalizeBaseURL(getenvTrim("SELF_BASE_URL"))

	// Owner-resolve collections (env + defaults)
	s.BrandsCollection = getenvTrim("BRANDS_COLLECTION")
	if s.BrandsCollection == "" {
		s.BrandsCollection = defaultBrandsCollection
	}
	s.AvatarsCollection = getenvTrim("AVATARS_COLLECTION")
	if s.AvatarsCollection == "" {
		s.AvatarsCollection = defaultAvatarsCollection
	}

	// Brand wallet secret prefix (env + default)
	s.BrandWalletSecretPrefix = getenvTrim("BRAND_WALLET_SECRET_PREFIX")
	if s.BrandWalletSecretPrefix == "" {
		s.BrandWalletSecretPrefix = defaultBrandWalletSecretPrefix
	}

	// Token icon bucket (config first; warn if empty)
	s.TokenIconBucket = strings.TrimSpace(cfg.TokenIconBucket)
	if s.TokenIconBucket == "" {
		warns = append(warns, "TOKEN_ICON_BUCKET is empty (token icon features may fail)")
	}

	// Token contents bucket (config -> env fallback -> default)
	s.TokenContentsBucket = strings.TrimSpace(cfg.TokenContentsBucket)
	if s.TokenContentsBucket == "" {
		// Backward compatibility: env fallback + default
		// (keeps current behavior in shared/infra.go)
		s.TokenContentsBucket = getenvOrDefault("TOKEN_CONTENTS_BUCKET", "narratives-development-token")
	}
	if s.TokenContentsBucket == "" {
		warns = append(warns, "TOKEN_CONTENTS_BUCKET is empty (token contents features may fail)")
	}

	// List images bucket:
	// deploy-backend.ps1 passes LIST_BUCKET, so check LIST_BUCKET first.
	// Also accept legacy LIST_IMAGE_BUCKET.
	s.ListImageBucket = getenvTrim("LIST_BUCKET")
	if s.ListImageBucket == "" {
		s.ListImageBucket = getenvTrim("LIST_IMAGE_BUCKET")
	}
	if s.ListImageBucket == "" {
		warns = append(warns, "LIST_BUCKET/LIST_IMAGE_BUCKET is empty (list image features may fail)")
	}

	// Avatar/Post buckets (env + defaults)
	s.AvatarIconBucket = getenvOrDefault("AVATAR_ICON_BUCKET", "narratives-development_avatar_icon")
	s.PostImageBucket = getenvOrDefault("POST_IMAGE_BUCKET", "narratives-development-posts")

	return s, warns, nil
}

func getenvTrim(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func getenvOrDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func normalizeBaseURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	return strings.TrimRight(u, "/")
}
