// backend/internal/platform/di/shared/runtime_settings_validate.go
package shared

import (
	"fmt"
	"strings"
)

// Validate performs hard validation for RuntimeSettings.
//
// Policy:
//   - This should be stricter than Normalize.
//   - It should fail fast for values that would cause undefined behavior,
//     while allowing optional features to remain disabled when settings are empty.
func (s RuntimeSettings) Validate() error {
	// Collections must never be empty once resolved (defaults exist).
	if strings.TrimSpace(s.BrandsCollection) == "" {
		return fmt.Errorf("shared.runtime_settings: BrandsCollection is empty")
	}
	if strings.TrimSpace(s.AvatarsCollection) == "" {
		return fmt.Errorf("shared.runtime_settings: AvatarsCollection is empty")
	}

	// Secret prefix must never be empty once resolved (default exists).
	if strings.TrimSpace(s.BrandWalletSecretPrefix) == "" {
		return fmt.Errorf("shared.runtime_settings: BrandWalletSecretPrefix is empty")
	}

	// SelfBaseURL is optional, but if set it must look like an HTTP(S) base URL.
	if u := strings.TrimSpace(s.SelfBaseURL); u != "" {
		if !(strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")) {
			return fmt.Errorf("shared.runtime_settings: SelfBaseURL must start with http:// or https:// (got %q)", u)
		}
		// Reject obvious misconfiguration (path component) to keep "base URL" semantics.
		// This is intentionally conservative; adjust if you need to allow paths.
		if strings.Contains(u[len("http://"):], "/") && strings.HasPrefix(u, "http://") {
			// allow only scheme://host[:port]
			rest := strings.TrimPrefix(u, "http://")
			if strings.Contains(rest, "/") {
				return fmt.Errorf("shared.runtime_settings: SelfBaseURL must not include a path (got %q)", u)
			}
		}
		if strings.Contains(u[len("https://"):], "/") && strings.HasPrefix(u, "https://") {
			rest := strings.TrimPrefix(u, "https://")
			if strings.Contains(rest, "/") {
				return fmt.Errorf("shared.runtime_settings: SelfBaseURL must not include a path (got %q)", u)
			}
		}
	}

	// Buckets are mostly optional by feature; validate only what is structurally invalid.
	// (GCS bucket names cannot contain spaces.)
	for name, v := range map[string]string{
		"TokenIconBucket":     s.TokenIconBucket,
		"TokenContentsBucket": s.TokenContentsBucket,
		"ListImageBucket":     s.ListImageBucket,
		"AvatarIconBucket":    s.AvatarIconBucket,
		"PostImageBucket":     s.PostImageBucket,
	} {
		if strings.ContainsAny(v, " \t\r\n") {
			return fmt.Errorf("shared.runtime_settings: %s contains whitespace (got %q)", name, v)
		}
	}

	return nil
}
