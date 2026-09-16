// backend/internal/domain/refund/stripe_id.go
package refund

import "strings"

func isStripeAccountID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"acct_",
	) &&
		len(value) > len("acct_")
}

func isStripeRefundID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"re_",
	) &&
		len(value) > len("re_")
}

func isStripeTransferReversalID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"trr_",
	) &&
		len(value) > len("trr_")
}
