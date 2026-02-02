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

	// ✅ 追加: ProductionUsecase に listQuery を注入（List / ListWithAssigneeName の 500 回避）
	u.productionUC.SetListQuery(companyProductionQueryService)

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

	// ✅ modelRepo(variations) を廃止したため、WithInventory のみを使用
	listCreateQuery := companyquery.NewListCreateQueryWithInventory(
		r.inventoryRepoForUC,
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

	// ✅ FIX: ListDetailQuery に (1) listImage と (2) productBlueprintPatch を注入する
	// - displayOrder を priceRows に載せるには pbPatchRepo の注入が必須
	// - listImage bucket の imageUrls を返すには imgLister の注入が必須
	listDetailQuery := companyquery.NewListDetailQueryWithBrandInventoryRowsImagesAndPBPatch(
		r.listRepo,
		res.nameResolver,
		r.productBlueprintRepo,
		&tbGetterAdapter{repo: r.tokenBlueprintRepo},
		inventoryQuery,
		inventoryQuery,
		r.listImageRepo,
		&pbPatchByIDAdapter{repo: r.productBlueprintRepo},
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
