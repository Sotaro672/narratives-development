// backend/internal/domain/refund/opened_return_policy.go
package refund

import "errors"

// ============================================================
// Opened Return Refund Policy
// ============================================================

// OpenedReturnRefundPolicy represents the refund policy selected by a company
// member when accepting a return after the product has been opened.
//
// The frontend may select only one of these policy values.
// Actual refund amounts must never be accepted from the frontend.
//
// The application / Order domain must calculate authoritative amounts from the
// persisted Order snapshot.
type OpenedReturnRefundPolicy string

const (
	// OpenedReturnRefundHalfMerchandise refunds 50% of the merchandise amount
	// together with the corresponding proportional merchandise consumption tax.
	//
	// Shipping and shipping consumption tax are not refunded.
	OpenedReturnRefundHalfMerchandise OpenedReturnRefundPolicy = "half_merchandise"

	// OpenedReturnRefundMerchandiseOnly refunds the full merchandise amount
	// together with the consumption tax attributable to that merchandise.
	//
	// Shipping and shipping consumption tax are not refunded.
	OpenedReturnRefundMerchandiseOnly OpenedReturnRefundPolicy = "merchandise_only"

	// OpenedReturnRefundMerchandiseRoundTripShipping represents the policy where
	// the company bears:
	//
	// - full merchandise amount
	// - merchandise consumption tax
	// - outbound shipping
	// - outbound shipping consumption tax
	// - return shipping
	// - return shipping consumption tax
	//
	// Return shipping is not necessarily part of the original Stripe Charge.
	// Therefore the application layer must distinguish the purchaser-side
	// Stripe Refund amount from the total company burden.
	OpenedReturnRefundMerchandiseRoundTripShipping OpenedReturnRefundPolicy = "merchandise_round_trip_shipping"
)

var AllowedOpenedReturnRefundPolicies = map[OpenedReturnRefundPolicy]struct{}{
	OpenedReturnRefundHalfMerchandise:              {},
	OpenedReturnRefundMerchandiseOnly:              {},
	OpenedReturnRefundMerchandiseRoundTripShipping: {},
}

// ============================================================
// Errors
// ============================================================

var (
	ErrInvalidOpenedReturnRefundPolicy = errors.New(
		"refund: invalid opened return refund policy",
	)
)

// ============================================================
// Validation
// ============================================================

// IsValidOpenedReturnRefundPolicy reports whether policy is one of the supported
// opened-return refund policies.
//
// Empty policy is intentionally invalid. Opened returns require an explicit
// policy selection and must not silently fall back to a default refund amount.
func IsValidOpenedReturnRefundPolicy(
	policy OpenedReturnRefundPolicy,
) bool {
	if policy == "" {
		return false
	}

	_, ok := AllowedOpenedReturnRefundPolicies[policy]
	return ok
}

// ValidateOpenedReturnRefundPolicy validates one opened-return refund policy.
func ValidateOpenedReturnRefundPolicy(
	policy OpenedReturnRefundPolicy,
) error {
	if !IsValidOpenedReturnRefundPolicy(policy) {
		return ErrInvalidOpenedReturnRefundPolicy
	}

	return nil
}
