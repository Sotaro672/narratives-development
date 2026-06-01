// backend\internal\application\query\console\productBlueprint_management_query.go
package query

import (
	"context"

	resolver "narratives/internal/application/resolver"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintManagementRepo defines the read port needed by the management list screen.
type ProductBlueprintManagementRepo interface {
	ListByCompanyID(ctx context.Context, companyID string) ([]productbpdom.ProductBlueprint, error)
}

// ProductBlueprintCompanyIDFromContext resolves companyId from request context.
//
// NOTE:
// query package should not import usecase package just to call CompanyIDFromContext.
// Pass usecase.CompanyIDFromContext from DI if you want to reuse the existing function.
type ProductBlueprintCompanyIDFromContext func(ctx context.Context) string

type ProductBlueprintManagementQuery struct {
	repo                 ProductBlueprintManagementRepo
	nameResolver         *resolver.NameResolver
	companyIDFromContext ProductBlueprintCompanyIDFromContext
}

func NewProductBlueprintManagementQuery(
	repo ProductBlueprintManagementRepo,
	nameResolver *resolver.NameResolver,
	companyIDFromContext ProductBlueprintCompanyIDFromContext,
) *ProductBlueprintManagementQuery {
	return &ProductBlueprintManagementQuery{
		repo:                 repo,
		nameResolver:         nameResolver,
		companyIDFromContext: companyIDFromContext,
	}
}

type ProductBlueprintResolvedNames struct {
	BrandName     string
	AssigneeName  string
	CreatedByName string
	UpdatedByName string
}

type ProductBlueprintResolved struct {
	ProductBlueprint productbpdom.ProductBlueprint
	Names            ProductBlueprintResolvedNames
}

// ListByCompanyID builds the read model used by the product blueprint management list screen.
func (q *ProductBlueprintManagementQuery) ListByCompanyID(
	ctx context.Context,
) ([]ProductBlueprintResolved, error) {
	cid := q.resolveCompanyID(ctx)
	if cid == "" {
		return nil, productbpdom.ErrInvalidCompanyID
	}

	rows, err := q.repo.ListByCompanyID(ctx, cid)
	if err != nil {
		return nil, err
	}

	out := make([]ProductBlueprintResolved, 0, len(rows))
	for _, pb := range rows {
		out = append(out, q.resolveProductBlueprint(ctx, pb))
	}

	return out, nil
}

func (q *ProductBlueprintManagementQuery) resolveCompanyID(ctx context.Context) string {
	if q.companyIDFromContext == nil {
		return ""
	}
	return q.companyIDFromContext(ctx)
}

func (q *ProductBlueprintManagementQuery) resolveProductBlueprint(
	ctx context.Context,
	pb productbpdom.ProductBlueprint,
) ProductBlueprintResolved {
	brandName := q.resolveBrandName(ctx, pb.BrandID)

	assigneeName := "-"
	if pb.AssigneeID != "" {
		assigneeName = q.resolveAssigneeName(ctx, pb.AssigneeID)
	}

	createdByName := ""
	if pb.CreatedBy != nil && *pb.CreatedBy != "" {
		createdByName = q.resolveCreatedByName(ctx, pb.CreatedBy)
	}

	updatedByName := ""
	if pb.UpdatedBy != nil && *pb.UpdatedBy != "" {
		updatedByName = q.resolveUpdatedByName(ctx, pb.UpdatedBy)
	}

	return ProductBlueprintResolved{
		ProductBlueprint: pb,
		Names: ProductBlueprintResolvedNames{
			BrandName:     brandName,
			AssigneeName:  assigneeName,
			CreatedByName: createdByName,
			UpdatedByName: updatedByName,
		},
	}
}

func (q *ProductBlueprintManagementQuery) resolveBrandName(
	ctx context.Context,
	brandID string,
) string {
	if brandID == "" {
		return ""
	}

	if q.nameResolver == nil {
		return brandID
	}

	name := q.nameResolver.ResolveBrandName(ctx, brandID)
	if name == "" {
		return brandID
	}

	return name
}

func (q *ProductBlueprintManagementQuery) resolveAssigneeName(
	ctx context.Context,
	assigneeID string,
) string {
	if assigneeID == "" {
		return ""
	}

	if q.nameResolver == nil {
		return assigneeID
	}

	name := q.nameResolver.ResolveProductBlueprintAssigneeName(ctx, assigneeID)
	if name == "" {
		return assigneeID
	}

	return name
}

func (q *ProductBlueprintManagementQuery) resolveCreatedByName(
	ctx context.Context,
	createdBy *string,
) string {
	if createdBy == nil || *createdBy == "" {
		return ""
	}

	if q.nameResolver == nil {
		return *createdBy
	}

	name := q.nameResolver.ResolveProductBlueprintCreatedByName(ctx, createdBy)
	if name == "" {
		return *createdBy
	}

	return name
}

func (q *ProductBlueprintManagementQuery) resolveUpdatedByName(
	ctx context.Context,
	updatedBy *string,
) string {
	if updatedBy == nil || *updatedBy == "" {
		return ""
	}

	if q.nameResolver == nil {
		return *updatedBy
	}

	name := q.nameResolver.ResolveProductBlueprintUpdatedByName(ctx, updatedBy)
	if name == "" {
		return *updatedBy
	}

	return name
}
