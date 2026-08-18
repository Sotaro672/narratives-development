// backend\internal\application\query\console\company_query.go
package query

import (
	"context"
	"fmt"

	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
)

type CompanyMemberNames struct {
	CreatedByName string `json:"createdByName"`
	UpdatedByName string `json:"updatedByName"`
}

type CompanyQuery struct {
	companyRepo companydom.Repository
	memberRepo  memberdom.Repository
}

func NewCompanyQuery(
	companyRepo companydom.Repository,
	memberRepo memberdom.Repository,
) *CompanyQuery {
	return &CompanyQuery{
		companyRepo: companyRepo,
		memberRepo:  memberRepo,
	}
}

func (q *CompanyQuery) GetByID(
	ctx context.Context,
	id string,
) (companydom.Company, CompanyMemberNames, error) {
	if q == nil || q.companyRepo == nil {
		return companydom.Company{}, CompanyMemberNames{}, fmt.Errorf("company query/repo is nil")
	}

	if id == "" {
		return companydom.Company{}, CompanyMemberNames{}, companydom.ErrInvalidID
	}

	company, err := q.companyRepo.GetByID(ctx, id)
	if err != nil {
		return companydom.Company{}, CompanyMemberNames{}, err
	}

	return company, q.ResolveMemberNames(ctx, company), nil
}

func (q *CompanyQuery) ResolveMemberNames(
	ctx context.Context,
	company companydom.Company,
) CompanyMemberNames {
	memberNameByID := resolveMemberNamesByID(
		ctx,
		q.memberRepo,
		[]string{
			company.CreatedBy,
			company.UpdatedBy,
		},
	)

	return CompanyMemberNames{
		CreatedByName: memberNameByID[company.CreatedBy],
		UpdatedByName: memberNameByID[company.UpdatedBy],
	}
}
