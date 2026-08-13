// backend/internal/platform/di/console/container_queries.go
package console

import (
	"context"
	"log"

	fsrepo "narratives/internal/adapters/out/firestore"
	companyquery "narratives/internal/application/query/console"
	inspectorquery "narratives/internal/application/query/inspector"
	"narratives/internal/application/usecase"
	solanainfra "narratives/internal/infra/solana"
	shared "narratives/internal/platform/di/shared"
)

type queries struct {
	companyProductionQueryService *companyquery.CompanyProductionQueryService
	mintRequestQueryService       *companyquery.MintRequestQueryService
	mintFundingEstimateQuery      *companyquery.MintFundingEstimateQuery

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

func buildQueries(
	infra *shared.Infra,
	r *repos,
	res *resolvers,
	u *usecases,
	s *services,
) *queries {
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
		r.modelRepo,
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
		mintTaskProgressQuery = fsrepo.NewMintTaskProgressQueryFS(
			r.mintRepo.Client,
		)
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

	var mintFundingEstimateQuery *companyquery.MintFundingEstimateQuery
	if u != nil && u.solanaMintClient != nil {
		estimateExecutor := companyquery.MintFundingEstimateExecutor(
			func(
				ctx context.Context,
				params companyquery.MintFundingEstimateParams,
			) (*companyquery.MintFundingEstimateResult, error) {
				result, err := u.solanaMintClient.EstimateMintFunding(
					ctx,
					solanainfra.MintFundingEstimateParams{
						TokenBlueprintID: params.TokenBlueprintID,
						MintQuantity:     params.MintQuantity,
						ToAddress:        params.ToAddress,
						Name:             params.Name,
						Symbol:           params.Symbol,
					},
				)
				if err != nil {
					return nil, err
				}

				if result == nil {
					return nil, nil
				}

				return &companyquery.MintFundingEstimateResult{
					Cluster:      result.Cluster,
					MintQuantity: result.MintQuantity,
					Reserve: companyquery.MintFundingEstimateReserve{
						Address:         result.Reserve.Address,
						BalanceLamports: result.Reserve.BalanceLamports,
						BalanceSOL:      result.Reserve.BalanceSOL,
						MinimumLamports: result.Reserve.MinimumLamports,
						MinimumSOL:      result.Reserve.MinimumSOL,
					},
					FeePayer: companyquery.MintFundingEstimateFeePayer{
						Address:         result.FeePayer.Address,
						BalanceLamports: result.FeePayer.BalanceLamports,
						BalanceSOL:      result.FeePayer.BalanceSOL,
						TargetLamports:  result.FeePayer.TargetLamports,
						TargetSOL:       result.FeePayer.TargetSOL,
					},
					Resources: companyquery.MintFundingEstimateResources{
						SharedMerkleTreeExists:  result.Resources.SharedMerkleTreeExists,
						SharedMerkleTreeAddress: result.Resources.SharedMerkleTreeAddress,
						CoreCollectionExists:    result.Resources.CoreCollectionExists,
						CoreCollectionAddress:   result.Resources.CoreCollectionAddress,
					},
					Estimate: companyquery.MintFundingEstimateCosts{
						MintTransactionFeePerItemLamports:            result.Estimate.MintTransactionFeePerItemLamports,
						MintTransactionFeePerItemSOL:                 result.Estimate.MintTransactionFeePerItemSOL,
						MintTransactionFeeTotalLamports:              result.Estimate.MintTransactionFeeTotalLamports,
						MintTransactionFeeTotalSOL:                   result.Estimate.MintTransactionFeeTotalSOL,
						MerkleTreeCreationTransactionFeeLamports:     result.Estimate.MerkleTreeCreationTransactionFeeLamports,
						MerkleTreeCreationTransactionFeeSOL:          result.Estimate.MerkleTreeCreationTransactionFeeSOL,
						MerkleTreeCreationRentLamports:               result.Estimate.MerkleTreeCreationRentLamports,
						MerkleTreeCreationRentSOL:                    result.Estimate.MerkleTreeCreationRentSOL,
						MerkleTreeCreationCostLamports:               result.Estimate.MerkleTreeCreationCostLamports,
						MerkleTreeCreationCostSOL:                    result.Estimate.MerkleTreeCreationCostSOL,
						CoreCollectionCreationTransactionFeeLamports: result.Estimate.CoreCollectionCreationTransactionFeeLamports,
						CoreCollectionCreationTransactionFeeSOL:      result.Estimate.CoreCollectionCreationTransactionFeeSOL,
						CoreCollectionCreationRentLamports:           result.Estimate.CoreCollectionCreationRentLamports,
						CoreCollectionCreationRentSOL:                result.Estimate.CoreCollectionCreationRentSOL,
						CoreCollectionCreationCostLamports:           result.Estimate.CoreCollectionCreationCostLamports,
						CoreCollectionCreationCostSOL:                result.Estimate.CoreCollectionCreationCostSOL,
						ProvisioningCostLamports:                     result.Estimate.ProvisioningCostLamports,
						ProvisioningCostSOL:                          result.Estimate.ProvisioningCostSOL,
						EstimatedNetworkCostLamports:                 result.Estimate.EstimatedNetworkCostLamports,
						EstimatedNetworkCostSOL:                      result.Estimate.EstimatedNetworkCostSOL,
						RequiredFeePayerBalanceLamports:              result.Estimate.RequiredFeePayerBalanceLamports,
						RequiredFeePayerBalanceSOL:                   result.Estimate.RequiredFeePayerBalanceSOL,
						EstimatedReserveTopUpLamports:                result.Estimate.EstimatedReserveTopUpLamports,
						EstimatedReserveTopUpSOL:                     result.Estimate.EstimatedReserveTopUpSOL,
						ReserveTransferFeeBufferLamports:             result.Estimate.ReserveTransferFeeBufferLamports,
						ReserveTransferFeeBufferSOL:                  result.Estimate.ReserveTransferFeeBufferSOL,
						RequiredReserveForTopUpLamports:              result.Estimate.RequiredReserveForTopUpLamports,
						RequiredReserveForTopUpSOL:                   result.Estimate.RequiredReserveForTopUpSOL,
						Sufficient:                                   result.Estimate.Sufficient,
					},
				}, nil
			},
		)

		mintFundingEstimateQuery = companyquery.NewMintFundingEstimateQuery(
			r.inspectionRepo,
			r.tokenBlueprintRepo,
			r.brandRepo,
			estimateExecutor,
		)
	}

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
		res.mintProductBlueprintResolver,
	)

	printQueryService := companyquery.NewPrintQueryService(
		r.productRepo,
		r.printLogRepo,
		res.nameResolver,
	)

	inspectorQuery := inspectorquery.NewQueryService(
		inspectorquery.NewQueryServiceParams{
			InspectionRepo: r.inspectionRepo,

			ProductRepo:          r.productRepo,
			ModelRepo:            r.modelRepo,
			ProductBlueprintRepo: r.productBlueprintRepo,

			BrandRepo:   r.brandRepo,
			CompanyRepo: r.companyRepo,
		},
	)

	// =========================================================
	// ListManagementQuery
	// SINGLE ENTRYPOINT: NewListManagementQuery(params) だけ
	// - company boundary は InvRows(ListByCurrentCompany) が必須
	// =========================================================
	listManagementQuery := companyquery.NewListManagementQuery(
		companyquery.NewListManagementQueryParams{
			Lister:       r.listRepoFS,
			NameResolver: res.nameResolver,
			PBGetter:     r.productBlueprintRepo,
			TBGetter:     r.tokenBlueprintRepo,
			InvRows:      inventoryManagementQuery,
		},
	)

	// =========================================================
	// ListDetailQuery
	// SINGLE ENTRYPOINT: NewListDetailQuery(params) だけ
	// - listID 確定後に GetByID して detail DTO を組み立てる
	// - imageUrls を返すには Firestore subcollection reader 注入
	// - displayOrder は ProductBlueprintGetter.GetByID の ModelRefs から解決する
	// =========================================================
	listDetailQuery := companyquery.NewListDetailQuery(
		companyquery.NewListDetailQueryParams{
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
		},
	)

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
	if u != nil && u.orderUC != nil {
		orderDetailQuery = companyquery.NewOrderDetailQuery(
			companyquery.NewOrderDetailQueryParams{
				OrderGetter: u.orderUC,

				InvBlueprint: invBlueprint,
				PBName:       pbName,
				TBName:       tbName,

				AvatarName: avatarName,
				UserName:   userName,

				ModelResolver: modelResolver,
			},
		)
	}

	_ = infra
	_ = s

	return &queries{
		companyProductionQueryService: companyProductionQueryService,
		mintRequestQueryService:       mintRequestQueryService,
		mintFundingEstimateQuery:      mintFundingEstimateQuery,

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
