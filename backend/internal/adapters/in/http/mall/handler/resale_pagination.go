// backend/internal/adapters/in/http/mall/handler/resale_pagination.go
package mallHandler

import (
	"net/http"

	common "narratives/internal/domain/common"
	resaledom "narratives/internal/domain/resale"
)

func buildResalePageFromQuery(r *http.Request) resaledom.Page {
	page := parsePageFromQuery(r, 50, 100)

	return resaledom.Page{
		Number:  page.Number,
		PerPage: page.PerPage,
	}
}

func buildResaleReviewPageFromQuery(r *http.Request) common.Page {
	return parsePageFromQuery(r, 20, 100)
}
