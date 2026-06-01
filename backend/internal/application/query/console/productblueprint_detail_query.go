// backend\internal\application\query\console\productblueprint_detail_query.go
package query

import (
	"context"

	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintDetailRepo defines the read port needed by the product blueprint detail screen.
type ProductBlueprintDetailRepo interface {
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
}

type ProductBlueprintDetailQuery struct {
	repo                 ProductBlueprintDetailRepo
	managementQuery      *ProductBlueprintManagementQuery
	companyIDFromContext ProductBlueprintCompanyIDFromContext
}

func NewProductBlueprintDetailQuery(
	repo ProductBlueprintDetailRepo,
	managementQuery *ProductBlueprintManagementQuery,
	companyIDFromContext ProductBlueprintCompanyIDFromContext,
) *ProductBlueprintDetailQuery {
	return &ProductBlueprintDetailQuery{
		repo:                 repo,
		managementQuery:      managementQuery,
		companyIDFromContext: companyIDFromContext,
	}
}

// GetByID builds the read model used by the product blueprint detail screen.
func (q *ProductBlueprintDetailQuery) GetByID(
	ctx context.Context,
	id string,
) (ProductBlueprintResolved, error) {
	if id == "" {
		return ProductBlueprintResolved{}, productbpdom.ErrInvalidID
	}

	cid := q.resolveCompanyID(ctx)
	if cid == "" {
		return ProductBlueprintResolved{}, productbpdom.ErrInvalidCompanyID
	}

	pb, err := q.repo.GetByID(ctx, id)
	if err != nil {
		return ProductBlueprintResolved{}, err
	}

	if pb.CompanyID == "" || pb.CompanyID != cid {
		return ProductBlueprintResolved{}, productbpdom.ErrForbidden
	}

	if q.managementQuery == nil {
		return ProductBlueprintResolved{
			ProductBlueprint: pb,
		}, nil
	}

	return q.managementQuery.resolveProductBlueprint(ctx, pb), nil
}

func (q *ProductBlueprintDetailQuery) resolveCompanyID(ctx context.Context) string {
	if q.companyIDFromContext == nil {
		return ""
	}
	return q.companyIDFromContext(ctx)
}
