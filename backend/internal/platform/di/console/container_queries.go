// backend/internal/platform/di/console/container_queries.go
package console

import (
	"log"

	fsrepo "narratives/internal/adapters/out/firestore"
	companyquery "narratives/internal/application/query/console"

	inspectorquery "narratives/internal/application/query/inspector"
	"narratives/internal/application/usecase"

	// Shared infra
	shared "narratives/internal/platform/di/shared"
)

type queries struct {
	companyProductionQueryService *companyquery.CompanyProductionQueryService
	mintRequestQueryService       *companyquery.MintRequestQueryService

	brandManagementQuery *companyquery.BrandManagementQuery
	brandDetailQuery     *companyquery.BrandDetailQuery

	productBlueprintManagementQuery *companyquery.ProductBlueprintManagementQuery
	productBlueprintDetailQuery     *companyquery.ProductBlueprintDetailQuery

	tokenBlueprintManagementQuery *companyquery.TokenBlueprintManagementQuery
	tokenBlueprintDetailQuery     *companyquery.TokenBlueprintDetailQuery

	inquiryManagementQuery *companyquery.InquiryManagementQuery
	inquiryDetailQuery     *companyquery.InquiryDetailQuery

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
	brandManagementQuery := companyquery.NewBrandManagementQuery(
		r.brandRepo,
		r.memberRepo,
	)

	brandDetailQuery := companyquery.NewBrandDetailQuery(
		r.brandRepo,
		r.memberRepo,
	)

	productBlueprintManagementQuery := companyquery.NewProductBlueprintManagementQuery(
		r.productBlueprintRepo,
		r.memberRepo,
		res.nameResolver,
		usecase.CompanyIDFromContext,
	)

	productBlueprintDetailQuery := companyquery.NewProductBlueprintDetailQuery(
		r.productBlueprintRepo,
		productBlueprintManagementQuery,
		usecase.CompanyIDFromContext,
	)

	tokenBlueprintManagementQuery := companyquery.NewTokenBlueprintManagementQuery(
		r.tokenBlueprintRepo,
		r.memberRepo,
		r.brandRepo,
	)

	tokenBlueprintDetailQuery := companyquery.NewTokenBlueprintDetailQuery(
		r.tokenBlueprintRepo,
		r.memberRepo,
		r.brandRepo,
	)

	inquiryManagementQuery := companyquery.NewInquiryManagementQuery(
		r.inquiryRepo,
		r.productRepo,
		r.modelRepo,
		r.productBlueprintRepo,
		r.brandRepo,
		r.avatarRepo,
		r.userRepo,
	)

	inquiryDetailQuery := companyquery.NewInquiryDetailQuery(
		r.inquiryRepo,
		r.productRepo,
		r.modelRepo,
		r.productBlueprintRepo,
		r.tokenBlueprintRepo,
		r.tokenReaderRepo,
		r.transferRepo,
		r.brandRepo,
		r.avatarRepo,
		r.userRepo,
		r.shippingAddressRepo,
		r.orderRepo,
	)

	companyProductionQueryService := companyquery.NewCompanyProductionQueryService(
		r.productBlueprintRepo,
		r.productionRepo,
		res.nameResolver,
	)

	var mintTaskProgressQuery companyquery.MintTaskProgressQuery
	if r.mintRepo != nil && r.mintRepo.Client != nil {
		mintTaskProgressQuery = fsrepo.NewMintTaskProgressQueryFS(r.mintRepo.Client)
	}

	mintRequestQueryService := companyquery.NewMintRequestQueryService(
		companyProductionQueryService,
		r.mintRepo,
		r.inspectionRepo,
		r.productBlueprintRepo,
		r.tokenBlueprintRepo,
		r.brandRepo,
		r.memberRepo,
		mintTaskProgressQuery,
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

	// salesQuery は mintAddress -> productName 解決を
	// application/resolver.MintProductBlueprintResolver に委譲する
	salesQuery := companyquery.NewSalesQuery(
		r.tokenBlueprintRepo,
		r.brandRepo,
		r.tokenReaderRepo,
		r.walletRepo,
		res.ownerResolveQuery,
		r.avatarRepo,
		res.mintProductBlueprintResolver,
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

		BrandRepo:   r.brandRepo,
		CompanyRepo: r.companyRepo,
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
	// - listID 確定後に GetByID して detail DTO を組み立てる
	// - imageUrls を返すには Firestore subcollection reader 注入
	// - displayOrder は ProductBlueprintGetter.GetByID の ModelRefs から解決する
	// =========================================================
	listDetailQuery := companyquery.NewListDetailQuery(companyquery.NewListDetailQueryParams{
		Getter:       r.listRepoFS,
		NameResolver: res.nameResolver,

		PBGetter: r.productBlueprintRepo,
		TBGetter: r.tokenBlueprintRepo,

		InvGetter: inventoryDetailQuery,

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
	if res.nameResolver != nil {
		if v, ok := any(res.nameResolver).(companyquery.OrderDetailUserNameResolver); ok {
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
	_ = s     // reserved for future wiring; keeps signature stable

	return &queries{
		companyProductionQueryService: companyProductionQueryService,
		mintRequestQueryService:       mintRequestQueryService,

		brandManagementQuery: brandManagementQuery,
		brandDetailQuery:     brandDetailQuery,

		productBlueprintManagementQuery: productBlueprintManagementQuery,
		productBlueprintDetailQuery:     productBlueprintDetailQuery,

		tokenBlueprintManagementQuery: tokenBlueprintManagementQuery,
		tokenBlueprintDetailQuery:     tokenBlueprintDetailQuery,

		inquiryManagementQuery: inquiryManagementQuery,
		inquiryDetailQuery:     inquiryDetailQuery,

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
