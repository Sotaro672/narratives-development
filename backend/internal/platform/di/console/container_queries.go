// backend/internal/platform/di/console/container_queries.go
package console

import (
	"log"

	companyquery "narratives/internal/application/query/console"

	// ✅ Shared infra (Firestore/GCS clients, bucket names)
	shared "narratives/internal/platform/di/shared"

	// ✅ ListImage uploader interface (used by console list handler wiring)
	listHandler "narratives/internal/adapters/in/http/console/handler/list"
)

type queries struct {
	companyProductionQueryService *companyquery.CompanyProductionQueryService
	mintRequestQueryService       *companyquery.MintRequestQueryService
	inventoryQuery                *companyquery.InventoryQuery
	listCreateQuery               *companyquery.ListCreateQuery
	listManagementQuery           *companyquery.ListManagementQuery
	listDetailQuery               *companyquery.ListDetailQuery

	// ✅ ListImage wiring (for /lists/{id}/images endpoints in console)
	// NOTE: DELETE API is abolished, so deleter wiring is removed.
	listImageUploader listHandler.ListImageUploader
}

func buildQueries(infra *shared.Infra, r *repos, res *resolvers, u *usecases) *queries {
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

	// ✅ FIX: ListDetailQuery に (1) listImageRecords(Firestore) と (2) productBlueprintPatch を注入する
	// - displayOrder を priceRows に載せるには pbPatchRepo の注入が必須
	// - imageUrls を返すには Firestore subcollection (/lists/{listId}/images) の reader 注入が必須
	//
	// NOTE:
	// - r.listImageRepo (GCS) ではなく r.listImageRecordRepo (Firestore) を注入する
	listDetailQuery := companyquery.NewListDetailQueryWithBrandInventoryRowsImagesAndPBPatch(
		r.listRepo,
		res.nameResolver,
		r.productBlueprintRepo,
		&tbGetterAdapter{repo: r.tokenBlueprintRepo},
		inventoryQuery,
		inventoryQuery,
		r.listImageRecordRepo, // ✅ Firestore records
		&pbPatchByIDAdapter{repo: r.productBlueprintRepo},
	)

	// =========================================================
	// ✅ ListImageUploader wiring
	//
	// NOTE:
	// - DELETE API is abolished, so ListImageDeleter wiring is removed.
	// - signed-url PUT + SaveImageFromGCS 方式なら uploader は不要（nil でもOK）
	// - 将来 "bytes upload endpoint" を作るならここで inject
	// =========================================================
	var uploader listHandler.ListImageUploader

	// 期待通りに配線されているかを確認しやすいようにログ（運用デバッグ用）
	log.Printf(
		"[di.console] list image ports wired (uploader=%t recordRepo=%t)",
		uploader != nil,
		r != nil && r.listImageRecordRepo != nil,
	)

	_ = infra // reserved for future wiring; keeps signature stable

	return &queries{
		companyProductionQueryService: companyProductionQueryService,
		mintRequestQueryService:       mintRequestQueryService,
		inventoryQuery:                inventoryQuery,
		listCreateQuery:               listCreateQuery,
		listManagementQuery:           listManagementQuery,
		listDetailQuery:               listDetailQuery,

		listImageUploader: uploader,
	}
}
