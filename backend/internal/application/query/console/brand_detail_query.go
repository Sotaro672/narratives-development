// backend/internal/application/query/console/brand_detail_query.go
package query

import (
	"context"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
)

type BrandDetailQuery struct {
	brandRepo  branddom.RepositoryPort
	memberRepo memberdom.Repository
}

func NewBrandDetailQuery(
	brandRepo branddom.RepositoryPort,
	memberRepo memberdom.Repository,
) *BrandDetailQuery {
	return &BrandDetailQuery{
		brandRepo:  brandRepo,
		memberRepo: memberRepo,
	}
}

type BrandDetailResult struct {
	Brand      branddom.Brand
	MemberName string
}

func (q *BrandDetailQuery) GetByID(
	ctx context.Context,
	id string,
) (BrandDetailResult, error) {
	if id == "" {
		return BrandDetailResult{}, branddom.ErrInvalidID
	}

	b, err := q.brandRepo.GetByID(ctx, id)
	if err != nil {
		return BrandDetailResult{}, err
	}

	return BrandDetailResult{
		Brand:      b,
		MemberName: q.resolveMemberName(ctx, b.ManagerID),
	}, nil
}

func (q *BrandDetailQuery) resolveMemberName(
	ctx context.Context,
	memberID *string,
) string {
	if memberID == nil || *memberID == "" || q.memberRepo == nil {
		return ""
	}

	svc := memberdom.NewService(q.memberRepo)
	name, err := svc.GetNameLastFirstByID(ctx, *memberID)
	if err != nil {
		return ""
	}

	return name
}
