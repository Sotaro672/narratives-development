// backend/internal/adapters/in/http/mall/handler/resale_pagination.go
package mallHandler

import (
	"net/http"

	common "narratives/internal/domain/common"
	resaledom "narratives/internal/domain/resale"
)

func buildResalePageFromQuery(r *http.Request) resaledom.Page {
	query := r.URL.Query()

	pageNum := parsePositiveIntDefault(query.Get("page"), 1)
	perPage := parsePositiveIntDefault(query.Get("perPage"), 50)

	if perPage > 100 {
		perPage = 100
	}

	return resaledom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}
}

func buildResaleReviewPageFromQuery(r *http.Request) common.Page {
	query := r.URL.Query()

	pageNum := parsePositiveIntDefault(query.Get("page"), 1)
	perPage := parsePositiveIntDefault(query.Get("perPage"), 20)

	if perPage > 100 {
		perPage = 100
	}

	return common.Page{
		Number:  pageNum,
		PerPage: perPage,
	}
}
