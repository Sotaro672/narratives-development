// backend/internal/application/query/admin/contract_detail_time.go
package query

import "time"

func formatContractDetailTime(
	value time.Time,
) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(
		time.RFC3339Nano,
	)
}

func formatOptionalContractDetailTime(
	value *time.Time,
) string {
	if value == nil || value.IsZero() {
		return ""
	}

	return formatContractDetailTime(*value)
}

func formatOptionalContractDetailTimePointer(
	value *time.Time,
) *string {
	if value == nil || value.IsZero() {
		return nil
	}

	formatted := formatContractDetailTime(*value)
	if formatted == "" {
		return nil
	}

	return &formatted
}
