// backend/internal/application/query/console/productBlueprint_management_query.go
package query

import (
	"context"
	"errors"

	resolver "narratives/internal/application/resolver"
	memberdom "narratives/internal/domain/member"
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
	memberRepo           memberdom.Repository
	nameResolver         *resolver.NameResolver
	companyIDFromContext ProductBlueprintCompanyIDFromContext
}

func NewProductBlueprintManagementQuery(
	repo ProductBlueprintManagementRepo,
	memberRepo memberdom.Repository,
	nameResolver *resolver.NameResolver,
	companyIDFromContext ProductBlueprintCompanyIDFromContext,
) *ProductBlueprintManagementQuery {
	return &ProductBlueprintManagementQuery{
		repo:                 repo,
		memberRepo:           memberRepo,
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
		assigneeName = q.resolveMemberNameByID(ctx, pb.AssigneeID)
	}

	createdByName := ""
	if pb.CreatedBy != nil && *pb.CreatedBy != "" {
		createdByName = q.resolveMemberNameByID(ctx, *pb.CreatedBy)
	}

	updatedByName := ""
	if pb.UpdatedBy != nil && *pb.UpdatedBy != "" {
		updatedByName = q.resolveMemberNameByID(ctx, *pb.UpdatedBy)
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

func (q *ProductBlueprintManagementQuery) resolveMemberNameByID(
	ctx context.Context,
	memberID string,
) string {
	if memberID == "" {
		return ""
	}

	if q.memberRepo == nil {
		return memberID
	}

	rec, err := q.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		if errors.Is(err, memberdom.ErrNotFound) {
			return memberID
		}
		return memberID
	}

	name := memberdom.FormatLastFirst(rec.Member.LastName, rec.Member.FirstName)
	if name == "" {
		return memberID
	}

	return name
}
