// backend/internal/adapters/in/http/mall/handler/resale_pagination.go
package mallHandler

import (
	"net/http"

	common "narratives/internal/domain/common"
	resaledom "narratives/internal/domain/resale"
)

type resalePageResponse struct {
	Items []resaledom.Resale `json:"items"`

	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`

	Page    int `json:"page"`
	PerPage int `json:"perPage"`
}

func buildResalePageResponse(
	items []resaledom.Resale,
	page resaledom.Page,
) resalePageResponse {
	pageNum := page.Number
	if pageNum <= 0 {
		pageNum = 1
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}

	totalCount := len(items)

	totalPages := 0
	if totalCount > 0 {
		totalPages = (totalCount + perPage - 1) / perPage
	}

	offset := (pageNum - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	pagedItems := []resaledom.Resale{}

	if offset < totalCount {
		end := offset + perPage
		if end > totalCount {
			end = totalCount
		}

		pagedItems = items[offset:end]
	}

	return resalePageResponse{
		Items: pagedItems,

		TotalCount: totalCount,
		TotalPages: totalPages,

		Page:    pageNum,
		PerPage: perPage,
	}
}

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
