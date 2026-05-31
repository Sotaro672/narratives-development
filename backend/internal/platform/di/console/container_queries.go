// backend/internal/platform/di/console/container_queries.go
package console

import (
	"log"

	companyquery "narratives/internal/application/query/console"

	inspectorquery "narratives/internal/application/query/inspector"

	// Shared infra
	shared "narratives/internal/platform/di/shared"
)

type queries struct {
	companyProductionQueryService *companyquery.CompanyProductionQueryService
	mintRequestQueryService       *companyquery.MintRequestQueryService

	inventoryManagementQuery *companyquery.InventoryManagementQuery
	inventoryDetailQuery     *companyquery.InventoryDetailQuery

	listCreateQuery *companyquery.ListCreateQuery
	salesQuery      *companyquery.SalesQuery

	printQueryService *companyquery.PrintQueryService

	listManagementQuery *companyquery.ListManagementQuery
	listDetailQuery     *companyquery.ListDetailQuery

	orderDetailQuery *companyquery.OrderDetailQuery

	inspectorQuery *inspectorquery.QueryService
}

func buildQueries(infra *shared.Infra, r *repos, res *resolvers, u *usecases, s *services) *queries {
	companyProductionQueryService := companyquery.NewCompanyProductionQueryService(
		r.productBlueprintRepo,
		r.productionRepo,
		res.nameResolver,
	)

	mintRequestQueryService := companyquery.NewMintRequestQueryService(
		companyProductionQueryService,
		r.mintRepo,
		r.inspectionRepo,
		r.productBlueprintRepo,
		r.tokenBlueprintRepo,
		s.brandSvc,
		res.nameResolver,
	)

	inventoryManagementQuery := companyquery.NewInventoryManagementQuery(
		r.inventoryRepo,
		r.productBlueprintRepo,
		res.nameResolver,
	)

	inventoryDetailQuery := companyquery.NewInventoryDetailQuery(
		r.inventoryRepo,
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

	printQueryService := companyquery.NewPrintQueryService(
		r.productRepo,
		r.printLogRepo,
		res.nameResolver,
	)

	inspectorQuery := inspectorquery.NewQueryService(inspectorquery.NewQueryServiceParams{
		InspectionRepo: r.inspectionRepo,

		ProductRepo:          r.productRepo,
		ModelRepo:            r.modelRepo,
		ProductBlueprintRepo: r.productBlueprintRepo,

		BrandService:   s.brandSvc,
		CompanyService: s.companySvc,
	})

	// =========================================================
	// ListManagementQuery
	// SINGLE ENTRYPOINT: NewListManagementQuery(params) だけ
	// - company boundary は InvRows(ListByCurrentCompany) が必須
	// =========================================================
	listManagementQuery := companyquery.NewListManagementQuery(companyquery.NewListManagementQueryParams{
		Lister:       r.listRepoFS,
		NameResolver: res.nameResolver,
		PBGetter:     r.productBlueprintRepo,
		TBGetter:     r.tokenBlueprintRepo,
		InvRows:      inventoryManagementQuery, // company boundary
	})

	// =========================================================
	// ListDetailQuery
	// SINGLE ENTRYPOINT: NewListDetailQuery(params) だけ
	// - imageUrls を返すには Firestore subcollection reader 注入
	// - displayOrder は ProductBlueprintGetter.GetByID の ModelRefs から解決する
	// =========================================================
	listDetailQuery := companyquery.NewListDetailQuery(companyquery.NewListDetailQueryParams{
		Getter:       r.listRepoFS,
		NameResolver: res.nameResolver,

		PBGetter: r.productBlueprintRepo,
		TBGetter: r.tokenBlueprintRepo,

		InvGetter: inventoryDetailQuery,
		InvRows:   inventoryManagementQuery,

		// Firebase Storage 移行後:
		// - frontend が Firebase Storage へ直接 upload
		// - backend は Firestore の /lists/{listId}/images/{imageId} record を読む
		// - ImageURLs は ListImage.URL(Firebase Storage downloadURL) から組み立てる
		ImgLister: r.listImageRecordRepo,
	})

	log.Printf(
		"[di.console] list image record repo wired (recordRepo=%t)",
		r != nil && r.listImageRecordRepo != nil,
	)

	// =========================================================
	// OrderDetailQuery
	// - GET /orders/{id} 用
	// - handler から detail enrichment を application/query 層へ分離
	// =========================================================
	var invBlueprint companyquery.OrderDetailInventoryBlueprintResolver
	if r.inventoryRepo != nil {
		if v, ok := any(r.inventoryRepo).(companyquery.OrderDetailInventoryBlueprintResolver); ok {
			invBlueprint = v
		}
	}

	var pbName companyquery.OrderDetailProductBlueprintNameResolver
	if r.productBlueprintRepo != nil {
		if v, ok := any(r.productBlueprintRepo).(companyquery.OrderDetailProductBlueprintNameResolver); ok {
			pbName = v
		}
	}

	var tbName companyquery.OrderDetailTokenBlueprintNameResolver
	if r.tokenBlueprintRepo != nil {
		if v, ok := any(r.tokenBlueprintRepo).(companyquery.OrderDetailTokenBlueprintNameResolver); ok {
			tbName = v
		}
	}

	var avatarName companyquery.OrderDetailAvatarNameResolver
	if r.avatarRepo != nil {
		if v, ok := any(r.avatarRepo).(companyquery.OrderDetailAvatarNameResolver); ok {
			avatarName = v
		}
	}

	var userName companyquery.OrderDetailUserNameResolver
	if u.userUC != nil {
		if v, ok := any(u.userUC).(companyquery.OrderDetailUserNameResolver); ok {
			userName = v
		}
	}

	var modelResolver companyquery.OrderDetailModelResolver
	if res.nameResolver != nil {
		if v, ok := any(res.nameResolver).(companyquery.OrderDetailModelResolver); ok {
			modelResolver = v
		}
	}

	var orderDetailQuery *companyquery.OrderDetailQuery
	if u.orderUC != nil {
		orderDetailQuery = companyquery.NewOrderDetailQuery(companyquery.NewOrderDetailQueryParams{
			OrderGetter: u.orderUC,

			InvBlueprint: invBlueprint,
			PBName:       pbName,
			TBName:       tbName,

			AvatarName: avatarName,
			UserName:   userName,

			ModelResolver: modelResolver,
		})
	}

	_ = infra // reserved for future wiring; keeps signature stable

	return &queries{
		companyProductionQueryService: companyProductionQueryService,
		mintRequestQueryService:       mintRequestQueryService,

		inventoryManagementQuery: inventoryManagementQuery,
		inventoryDetailQuery:     inventoryDetailQuery,

		listCreateQuery: listCreateQuery,
		salesQuery:      salesQuery,

		printQueryService: printQueryService,

		listManagementQuery: listManagementQuery,
		listDetailQuery:     listDetailQuery,

		orderDetailQuery: orderDetailQuery,

		inspectorQuery: inspectorQuery,
	}
}
