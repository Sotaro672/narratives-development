package port

import (
	"context"

	pbdom "narratives/internal/domain/productBlueprint"
)

type ProductBlueprintGetter interface {
	GetByID(
		ctx context.Context,
		id string,
	) (
		pbdom.ProductBlueprint,
		error,
	)
}

type ProductBlueprintCompanyLister interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) (
		[]pbdom.ProductBlueprint,
		error,
	)
}

type ProductBlueprintReader interface {
	ProductBlueprintGetter
	ProductBlueprintCompanyLister
}
