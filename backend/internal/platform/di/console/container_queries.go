// backend/internal/platform/di/console/container_queries.go
package console

import (
	"log"

	companyquery "narratives/internal/application/query/console"

	// moved queries
	listdetail "narratives/internal/application/query/console/list/detail"
	listmgmt "narratives/internal/application/query/console/list/management"

	// Shared infra
	shared "narratives/internal/platform/di/shared"
)

type queries struct {
	companyProductionQueryService *companyquery.CompanyProductionQueryService
	mintRequestQueryService       *companyquery.MintRequestQueryService
	inventoryQuery                *companyquery.InventoryQuery
	listCreateQuery               *companyquery.ListCreateQuery
	salesQuery                    *companyquery.SalesQuery

	listManagementQuery *listmgmt.ListManagementQuery
	listDetailQuery     *listdetail.ListDetailQuery
}

func buildQueries(infra *shared.Infra, r *repos, res *resolvers, u *usecases) *queries {
	companyProductionQueryService := companyquery.NewCompanyProductionQueryService(
		r.productBlueprintRepo,
		r.productionRepo,
		res.nameResolver,
	)

	// 追加: ProductionUsecase に listQuery を注入（List / ListWithAssigneeName の 500 回避）
	u.productionUC.SetListQuery(companyProductionQueryService)

	mintRequestQueryService := companyquery.NewMintRequestQueryService(
		u.mintUC,
		u.productionUC,
		res.nameResolver,
	)
	mintRequestQueryService.SetModelRepo(r.modelRepo)
	inventoryQuery := companyquery.NewInventoryQueryWithTokenBlueprintPatch(
		r.inventoryRepo,
		r.productBlueprintRepo,
		r.productBlueprintRepo,
		r.tokenBlueprintRepo,
		res.nameResolver,
	)

	// modelRepo(variations) を廃止したため、WithInventory のみを使用
	listCreateQuery := companyquery.NewListCreateQueryWithInventory(
		r.inventoryRepo,
		r.productBlueprintRepo,
		r.tokenBlueprintRepo,
		res.nameResolver,
	)

	// salesQuery は mintAddress -> productId -> modelId -> productBlueprintId 解決のため
	// productRepo / productBlueprintRepo を追加
	salesQuery := companyquery.NewSalesQuery(
		r.tokenBlueprintRepo,
		r.brandRepo,
		r.tokenReaderRepo,
		r.tokenReaderRepo,
		r.productRepo,
		r.productBlueprintRepo,
		r.walletRepo,
		res.ownerResolveQuery,
		r.avatarRepo,
		r.avatarStateRepo,
	)

	// =========================================================
	// moved: ListManagementQuery
	// SINGLE ENTRYPOINT: NewListManagementQuery(params) だけ
	// - company boundary は InvRows(ListByCurrentCompany) が必須
	// =========================================================
	listManagementQuery := listmgmt.NewListManagementQuery(listmgmt.NewListManagementQueryParams{
		Lister:       r.listRepoFS,
		NameResolver: res.nameResolver,
		PBGetter:     r.productBlueprintRepo,
		TBGetter:     r.tokenBlueprintRepo,
		InvRows:      inventoryQuery, // company boundary
	})

	// =========================================================
	// moved: ListDetailQuery
	// SINGLE ENTRYPOINT: NewListDetailQuery(params) だけ
	// - displayOrder を priceRows に載せるには pbPatchRepo 注入
	// - imageUrls を返すには Firestore subcollection reader 注入
	//
	// ProductBlueprintPatchReader を typed(Patch) に寄せたため、
	// =========================================================
	listDetailQuery := listdetail.NewListDetailQuery(listdetail.NewListDetailQueryParams{
		Getter:       r.listRepoFS,
		NameResolver: res.nameResolver,

		PBGetter: r.productBlueprintRepo,
		TBGetter: r.tokenBlueprintRepo,

		InvGetter: inventoryQuery,
		InvRows:   inventoryQuery,

		// Firebase Storage 移行後:
		// - frontend が Firebase Storage へ直接 upload
		// - backend は Firestore の /lists/{listId}/images/{imageId} record を読む
		// - ImageURLs は ListImage.URL(Firebase Storage downloadURL) から組み立てる
		ImgLister: r.listImageRecordRepo,

		PBPatchRepo: r.productBlueprintRepo, // adapter廃止: repo直渡し（GetPatchByID）
	})

	log.Printf(
		"[di.console] list image record repo wired (recordRepo=%t)",
		r != nil && r.listImageRecordRepo != nil,
	)

	_ = infra // reserved for future wiring; keeps signature stable

	return &queries{
		companyProductionQueryService: companyProductionQueryService,
		mintRequestQueryService:       mintRequestQueryService,
		inventoryQuery:                inventoryQuery,
		listCreateQuery:               listCreateQuery,
		salesQuery:                    salesQuery,
		listManagementQuery:           listManagementQuery,
		listDetailQuery:               listDetailQuery,
	}
}
