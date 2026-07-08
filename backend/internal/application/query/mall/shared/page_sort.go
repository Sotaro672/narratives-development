// backend/internal/application/query/mall/shared/page_sort.go
package shared

import (
	"strings"

	common "narratives/internal/domain/common"
)

func NormalizeIntPage(
	number int,
	perPage int,
	fallbackNumber int,
	fallbackPerPage int,
	maxPerPage int,
) (int, int) {
	if fallbackNumber <= 0 {
		fallbackNumber = 1
	}

	if number <= 0 {
		number = fallbackNumber
	}
	if number <= 0 {
		number = 1
	}

	if perPage <= 0 {
		perPage = fallbackPerPage
	}

	if maxPerPage > 0 && perPage > maxPerPage {
		perPage = maxPerPage
	}

	if perPage < 0 {
		perPage = 0
	}

	return number, perPage
}

func NormalizeLimit(
	limit int,
	fallbackLimit int,
	maxLimit int,
) int {
	if limit <= 0 {
		limit = fallbackLimit
	}

	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}

	if limit < 0 {
		return 0
	}

	return limit
}

func NormalizeCommonPage(
	page common.Page,
	fallbackNumber int,
	fallbackPerPage int,
	maxPerPage int,
) common.Page {
	number, perPage := NormalizeIntPage(
		page.Number,
		page.PerPage,
		fallbackNumber,
		fallbackPerPage,
		maxPerPage,
	)

	return common.Page{
		Number:  number,
		PerPage: perPage,
	}
}

// NormalizeSortParts normalizes a sort column/order pair.
//
// allowedColumns:
// - key: accepted incoming column
// - value: canonical output column
//
// If allowedColumns is nil or empty, any non-empty column is accepted.
// This is useful for existing query services where repository-side validation
// already owns the allowed column policy.
func NormalizeSortParts(
	column string,
	order string,
	allowedColumns map[string]string,
	fallbackColumn string,
	fallbackOrder string,
) (string, string) {
	normalizedColumn := strings.TrimSpace(column)
	if normalizedColumn == "" {
		normalizedColumn = fallbackColumn
	}

	if len(allowedColumns) > 0 {
		if canonical, ok := allowedColumns[normalizedColumn]; ok {
			normalizedColumn = canonical
		} else {
			normalizedColumn = fallbackColumn
		}
	}

	normalizedOrder := strings.ToLower(strings.TrimSpace(order))
	if normalizedOrder != "asc" && normalizedOrder != "desc" {
		normalizedOrder = strings.ToLower(strings.TrimSpace(fallbackOrder))
	}

	if normalizedOrder != "asc" && normalizedOrder != "desc" {
		normalizedOrder = "desc"
	}

	return normalizedColumn, normalizedOrder
}

func NormalizeCommonSort(
	sort common.Sort,
	allowedColumns map[string]string,
	fallbackColumn string,
	fallbackOrder common.SortOrder,
) common.Sort {
	column, order := NormalizeSortParts(
		sort.Column,
		string(sort.Order),
		allowedColumns,
		fallbackColumn,
		string(fallbackOrder),
	)

	return common.Sort{
		Column: column,
		Order:  common.SortOrder(order),
	}
}
