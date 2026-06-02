// backend/internal/application/query/console/brand_management_query.go
package query

import (
	"context"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
)

type BrandManagementQuery struct {
	brandRepo  branddom.RepositoryPort
	memberRepo memberdom.Repository
}

func NewBrandManagementQuery(
	brandRepo branddom.RepositoryPort,
	memberRepo memberdom.Repository,
) *BrandManagementQuery {
	return &BrandManagementQuery{
		brandRepo:  brandRepo,
		memberRepo: memberRepo,
	}
}

type BrandManagementItem struct {
	Brand      branddom.Brand
	MemberName string
}

type BrandManagementPageResult struct {
	Items      []BrandManagementItem
	TotalCount int
	Page       int
	PerPage    int
	TotalPages int
}

func (q *BrandManagementQuery) ListCurrentCompanyBrands(
	ctx context.Context,
	page branddom.Page,
) (BrandManagementPageResult, error) {
	cid := usecase.CompanyIDFromContext(ctx)
	if cid == "" {
		return BrandManagementPageResult{
			Items:      []BrandManagementItem{},
			TotalCount: 0,
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalPages: 0,
		}, nil
	}

	result, err := q.brandRepo.ListByCompanyID(ctx, cid, page)
	if err != nil {
		return BrandManagementPageResult{}, err
	}

	items := make([]BrandManagementItem, 0, len(result.Items))
	for _, b := range result.Items {
		items = append(items, BrandManagementItem{
			Brand:      b,
			MemberName: q.resolveMemberName(ctx, b.ManagerID),
		})
	}

	return BrandManagementPageResult{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	}, nil
}

func (q *BrandManagementQuery) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page branddom.Page,
) (BrandManagementPageResult, error) {
	if companyID == "" {
		return BrandManagementPageResult{}, branddom.ErrInvalidID
	}

	result, err := q.brandRepo.ListByCompanyID(ctx, companyID, page)
	if err != nil {
		return BrandManagementPageResult{}, err
	}

	items := make([]BrandManagementItem, 0, len(result.Items))
	for _, b := range result.Items {
		items = append(items, BrandManagementItem{
			Brand:      b,
			MemberName: q.resolveMemberName(ctx, b.ManagerID),
		})
	}

	return BrandManagementPageResult{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	}, nil
}

func (q *BrandManagementQuery) resolveMemberName(
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
