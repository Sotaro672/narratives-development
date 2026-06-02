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
func (q *MemberManagementQuery) ListByCompanyID(
	ctx context.Context,
	companyID string,
	f memdom.Filter,
	p memdom.Page,
) (MemberRecordPageResult, error) {
	if companyID == "" {
		return MemberRecordPageResult{}, errors.New("member: companyID is empty")
	}

	res, err := q.repo.ListByCompanyID(ctx, companyID, f, p)
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
