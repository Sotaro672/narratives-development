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
// - GCS bucket settings are removed.
// - Stripe keys are resolved from Google Secret Manager, not env/config.
// - Use env fallbacks only for non-secret runtime values.
// - Apply defaults only for stable collection / secret-prefix values.
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

	// Used by ShareTransfer signer provider
	AvatarWalletSecretPrefix string
}

// ResolveRuntimeSettings resolves and normalizes runtime settings from cfg/env.
//
// Notes:
//   - This function is side-effect free (no logging).
//   - It returns warnings as strings so callers can decide how to surface them.
//   - cfg is still accepted so callers can keep a single config-driven initialization flow,
//     but GCS bucket and Stripe key values are intentionally not read from cfg.
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

	// Avatar wallet secret prefix (env + default)
	s.AvatarWalletSecretPrefix = getenvTrim("AVATAR_WALLET_SECRET_PREFIX")
	if s.AvatarWalletSecretPrefix == "" {
		s.AvatarWalletSecretPrefix = defaultAvatarWalletSecretPrefix
	}

	return s, warns, nil
}

func getenvTrim(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func normalizeBaseURL(u string) string {
	if u == "" {
		return ""
	}

	return strings.TrimRight(strings.TrimSpace(u), "/")
}
