// backend/internal/application/query/sns/list_query.go
package sns

import (
	"context"
	"errors"
	"math"
	"strings"

	ldom "narratives/internal/domain/list"
)

// SNSListRow is a buyer-facing minimal row.
// NOTE: purchase-side needs title/description/image/prices.
// We keep id for routing (detail), but UI can ignore it.
type SNSListRow struct {
	ID          string              `json:"id,omitempty"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // List.ImageID (URL)
	Prices      []ldom.ListPriceRow `json:"prices"`
}

// SNSListIndexDTO is buyer-facing list index response.
type SNSListIndexDTO struct {
	Items      []SNSListRow `json:"items"`
	TotalCount int          `json:"totalCount"`
	TotalPages int          `json:"totalPages"`
	Page       int          `json:"page"`
	PerPage    int          `json:"perPage"`
}

// SNSListDetailDTO is buyer-facing detail response.
type SNSListDetailDTO struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Image       string              `json:"image"` // List.ImageID (URL)
	Prices      []ldom.ListPriceRow `json:"prices"`
}

// SNSListQuery reads "public listing" lists WITHOUT company boundary.
// - No companyId is read from context.
// - Only status=listing is exposed.
// IMPORTANT: current domain repository implementation may not support server-side filtering.
// This query scans via cursor paging and filters in-memory.
// If/when you add Firestore-side where(status=="listing"), replace this scan with that.
type SNSListQuery struct {
	Repo ldom.Repository
}

func NewSNSListQuery(repo ldom.Repository) *SNSListQuery {
	return &SNSListQuery{Repo: repo}
}

// ListListing returns all lists with status=listing, paged by (page, perPage).
// It ignores company boundary completely.
func (q *SNSListQuery) ListListing(ctx context.Context, page int, perPage int) (SNSListIndexDTO, error) {
	if q == nil || q.Repo == nil {
		return SNSListIndexDTO{}, errors.New("sns list query: repo is nil")
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 200 {
		perPage = 200
	}

	start := (page - 1) * perPage
	end := start + perPage

	items := make([]SNSListRow, 0, perPage)

	// Scan all docs by cursor, filter status=listing.
	// We also compute totalCount accurately (but this is O(N) scan).
	after := ""
	limit := 200

	totalListing := 0

	for {
		res, err := q.Repo.ListByCursor(ctx, ldom.Filter{}, ldom.Sort{}, ldom.CursorPage{
			After: after,
			Limit: limit,
		})
		if err != nil {
			return SNSListIndexDTO{}, err
		}

		for _, l := range res.Items {
			if l.Status != ldom.StatusListing {
				continue
			}

			// count always
			if totalListing >= start && totalListing < end {
				items = append(items, toSNSRow(l))
			}
			totalListing++

			// (we keep scanning to compute totalListing until the end)
		}

		if res.NextCursor == nil || strings.TrimSpace(*res.NextCursor) == "" {
			break
		}
		after = strings.TrimSpace(*res.NextCursor)
	}

	totalPages := 0
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(totalListing) / float64(perPage)))
	}

	return SNSListIndexDTO{
		Items:      items,
		TotalCount: totalListing,
		TotalPages: totalPages,
		Page:       page,
		PerPage:    perPage,
	}, nil
}

// GetListingDetail returns minimal fields for a single list,
// but only if status=listing. Otherwise it behaves as not found.
func (q *SNSListQuery) GetListingDetail(ctx context.Context, id string) (SNSListDetailDTO, error) {
	if q == nil || q.Repo == nil {
		return SNSListDetailDTO{}, errors.New("sns list query: repo is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return SNSListDetailDTO{}, ldom.ErrNotFound
	}

	l, err := q.Repo.GetByID(ctx, id)
	if err != nil {
		return SNSListDetailDTO{}, err
	}
	if l.Status != ldom.StatusListing {
		return SNSListDetailDTO{}, ldom.ErrNotFound
	}

	return SNSListDetailDTO{
		Title:       strings.TrimSpace(l.Title),
		Description: strings.TrimSpace(l.Description),
		Image:       strings.TrimSpace(l.ImageID),
		Prices:      l.Prices,
	}, nil
}

func toSNSRow(l ldom.List) SNSListRow {
	return SNSListRow{
		ID:          strings.TrimSpace(l.ID),
		Title:       strings.TrimSpace(l.Title),
		Description: strings.TrimSpace(l.Description),
		Image:       strings.TrimSpace(l.ImageID),
		Prices:      l.Prices,
	}
}
