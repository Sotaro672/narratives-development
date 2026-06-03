// backend/internal/application/query/console/member_management_query.go
package query

import (
	"context"
	"errors"

	memdom "narratives/internal/domain/member"
)

// -----------------------------------------------------------------------------
// DTOs
// -----------------------------------------------------------------------------

type MemberRecord struct {
	DocID  string
	Member memdom.Member
}

type MemberRecordPageResult struct {
	Items      []MemberRecord
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

type MemberListInput struct {
	CompanyID string

	SearchQuery string
	UID         string
	Status      string
	BrandIDs    []string

	Page    int
	PerPage int
}

// -----------------------------------------------------------------------------
// Query
// -----------------------------------------------------------------------------

type MemberManagementQuery struct {
	repo memdom.Repository
}

func NewMemberManagementQuery(repo memdom.Repository) *MemberManagementQuery {
	return &MemberManagementQuery{
		repo: repo,
	}
}

// ListByCompanyID は companyID scope の member 一覧を取得します。
//
// handler からは HTTP query parameter をそのまま input として受け取り、
// domain の Filter / Page への変換は query 層で行います。
func (q *MemberManagementQuery) ListByCompanyID(
	ctx context.Context,
	in MemberListInput,
) (MemberRecordPageResult, error) {
	if in.CompanyID == "" {
		return MemberRecordPageResult{}, errors.New("member: companyID is empty")
	}

	f := memdom.Filter{
		SearchQuery: in.SearchQuery,
		UID:         in.UID,
		Status:      in.Status,
		BrandIDs:    in.BrandIDs,
	}

	p := memdom.Page{
		Number:  clampInt(in.Page, 1, 1_000_000),
		PerPage: clampInt(in.PerPage, 1, 200),
	}

	res, err := q.repo.ListByCompanyID(ctx, in.CompanyID, f, p)
	if err != nil {
		return MemberRecordPageResult{}, err
	}

	items := make([]MemberRecord, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, MemberRecord{
			DocID:  item.DocID,
			Member: item.Member,
		})
	}

	return MemberRecordPageResult{
		Items:      items,
		TotalCount: res.TotalCount,
		TotalPages: res.TotalPages,
		Page:       res.Page,
		PerPage:    res.PerPage,
	}, nil
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
