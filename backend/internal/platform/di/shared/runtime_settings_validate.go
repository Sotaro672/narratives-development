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
//   - GCS bucket settings are removed because GCS usage has been deprecated.
//   - Stripe keys are resolved from Google Secret Manager and are not validated here.
func (s RuntimeSettings) Validate() error {
	// Collections must never be empty once resolved because defaults exist.
	if s.BrandsCollection == "" {
		return fmt.Errorf("shared.runtime_settings: BrandsCollection is empty")
	}
	if s.AvatarsCollection == "" {
		return fmt.Errorf("shared.runtime_settings: AvatarsCollection is empty")
	}

	// Secret prefixes must never be empty once resolved because defaults exist.
	if s.BrandWalletSecretPrefix == "" {
		return fmt.Errorf("shared.runtime_settings: BrandWalletSecretPrefix is empty")
	}
	if s.AvatarWalletSecretPrefix == "" {
		return fmt.Errorf("shared.runtime_settings: AvatarWalletSecretPrefix is empty")
	}

	// SelfBaseURL is optional, but if set it must look like an HTTP(S) base URL.
	if u := s.SelfBaseURL; u != "" {
		if !(strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")) {
			return fmt.Errorf(
				"shared.runtime_settings: SelfBaseURL must start with http:// or https:// (got %q)",
				u,
			)
		}

		// Reject obvious misconfiguration with a path component to keep base URL semantics.
		// Example allowed: https://example.com
		// Example rejected: https://example.com/path
		if strings.HasPrefix(u, "http://") {
			rest := strings.TrimPrefix(u, "http://")
			if strings.Contains(rest, "/") {
				return fmt.Errorf(
					"shared.runtime_settings: SelfBaseURL must not include a path (got %q)",
					u,
				)
			}
		}

		if strings.HasPrefix(u, "https://") {
			rest := strings.TrimPrefix(u, "https://")
			if strings.Contains(rest, "/") {
				return fmt.Errorf(
					"shared.runtime_settings: SelfBaseURL must not include a path (got %q)",
					u,
				)
			}
		}
	}

	return nil
}
