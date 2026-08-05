// backend/internal/adapters/out/firestore/common/sqlutil.go
package common

// NormalizePage はページ番号・件数を正規化し、limit・offset を返します。
func NormalizePage(
	number int,
	perPage int,
	defaultPerPage int,
	maxPerPage int,
) (
	page int,
	limit int,
	offset int,
) {
	page = number
	if page <= 0 {
		page = 1
	}

	limit = perPage
	if limit <= 0 {
		limit = defaultPerPage
	}

	if maxPerPage > 0 && limit > maxPerPage {
		limit = maxPerPage
	}

	offset = (page - 1) * limit

	return page, limit, offset
}

// ComputeTotalPages は合計件数と1ページあたり件数から総ページ数を計算します。
func ComputeTotalPages(total int, perPage int) int {
	if perPage <= 0 {
		return 0
	}

	return (total + perPage - 1) / perPage
}
