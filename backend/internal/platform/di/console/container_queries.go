// backend/internal/platform/di/console/container_queries.go
package console

import (
	companyquery "narratives/internal/application/query/console"
)

type queries struct {
	companyProductionQueryService *companyquery.CompanyProductionQueryService
	mintRequestQueryService       *companyquery.MintRequestQueryService
	inventoryQuery                *companyquery.InventoryQuery
	listCreateQuery               *companyquery.ListCreateQuery
	listManagementQuery           *companyquery.ListManagementQuery
	listDetailQuery               *companyquery.ListDetailQuery
}

func buildQueries(r *repos, res *resolvers, u *usecases) *queries {
	pbQueryRepo := &pbQueryRepoAdapter{repo: r.productBlueprintRepo}

	companyProductionQueryService := companyquery.NewCompanyProductionQueryService(
		pbQueryRepo,
		r.productionRepo,
		res.nameResolver,
	)

	mintRequestQueryService := companyquery.NewMintRequestQueryService(
		u.mintUC,
		u.productionUC,
		res.nameResolver,
	)
	mintRequestQueryService.SetModelRepo(r.modelRepo)

	inventoryQuery := companyquery.NewInventoryQueryWithTokenBlueprintPatch(
		r.inventoryRepoForUC,
		&pbIDsByCompanyAdapter{repo: r.productBlueprintRepo},
		&pbPatchByIDAdapter{repo: r.productBlueprintRepo},
		&tbPatchByIDAdapter{repo: r.tokenBlueprintRepo},
		res.nameResolver,
	)

	listCreateQuery := companyquery.NewListCreateQueryWithInventoryAndModels(
		r.inventoryRepoForUC,
		r.modelRepo,
		&pbPatchByIDAdapter{repo: r.productBlueprintRepo},
		&tbPatchByIDAdapter{repo: r.tokenBlueprintRepo},
		res.nameResolver,
	)

	listManagementQuery := companyquery.NewListManagementQueryWithBrandInventoryAndInventoryRows(
		r.listRepo,
		res.nameResolver,
		r.productBlueprintRepo,
		&tbGetterAdapter{repo: r.tokenBlueprintRepo},
		inventoryQuery,
	)

	listDetailQuery := companyquery.NewListDetailQueryWithBrandInventoryAndInventoryRows(
		r.listRepo,
		res.nameResolver,
		r.productBlueprintRepo,
		&tbGetterAdapter{repo: r.tokenBlueprintRepo},
		inventoryQuery,
		inventoryQuery,
	)

	return &queries{
		companyProductionQueryService: companyProductionQueryService,
		mintRequestQueryService:       mintRequestQueryService,
		inventoryQuery:                inventoryQuery,
		listCreateQuery:               listCreateQuery,
		listManagementQuery:           listManagementQuery,
		listDetailQuery:               listDetailQuery,
	}
}
