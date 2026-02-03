// backend/internal/platform/di/console/container_queries.go
package console

import (
	"context"
	"log"

	companyquery "narratives/internal/application/query/console"

	// ✅ moved queries
	listdetail "narratives/internal/application/query/console/list/detail"
	listmgmt "narratives/internal/application/query/console/list/management"

	// ✅ Shared infra (Firestore/GCS clients, bucket names)
	shared "narratives/internal/platform/di/shared"

	// ✅ ListImage uploader interface (used by console list handler wiring)
	listHandler "narratives/internal/adapters/in/http/console/handler/list"
)

// =========================================================
// ✅ Adapter: pbPatchByIDAdapter(Patch) -> ProductBlueprintPatchReader(any)
// - list.ProductBlueprintPatchReader は GetPatchByID(...) (any, error)
// - pbPatchByIDAdapter は GetPatchByID(...) (productBlueprint.Patch, error)
// - method signature を合わせるために薄いラッパを挟む
// =========================================================

type pbPatchAnyAdapter struct {
	inner *pbPatchByIDAdapter
}

func (a *pbPatchAnyAdapter) GetPatchByID(ctx context.Context, id string) (any, error) {
	if a == nil || a.inner == nil {
		return nil, nil
	}
	patch, err := a.inner.GetPatchByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return patch, nil
}

type queries struct {
	companyProductionQueryService *companyquery.CompanyProductionQueryService
	mintRequestQueryService       *companyquery.MintRequestQueryService
	inventoryQuery                *companyquery.InventoryQuery
	listCreateQuery               *companyquery.ListCreateQuery

	// ✅ moved
	listManagementQuery *listmgmt.ListManagementQuery
	listDetailQuery     *listdetail.ListDetailQuery

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

	// =========================================================
	// ✅ moved: ListManagementQuery
	// ✅ SINGLE ENTRYPOINT: NewListManagementQuery(params) だけ
	// - company boundary は InvRows(ListByCurrentCompany) が必須
	// =========================================================
	listManagementQuery := listmgmt.NewListManagementQuery(listmgmt.NewListManagementQueryParams{
		Lister:       r.listRepo,
		NameResolver: res.nameResolver,
		PBGetter:     r.productBlueprintRepo,
		TBGetter:     &tbGetterAdapter{repo: r.tokenBlueprintRepo},
		InvRows:      inventoryQuery, // ✅ company boundary
	})

	// =========================================================
	// ✅ moved: ListDetailQuery
	// ✅ SINGLE ENTRYPOINT: NewListDetailQuery(params) だけ
	// - displayOrder を priceRows に載せるには pbPatchRepo 注入
	// - imageUrls を返すには Firestore subcollection reader 注入
	//
	// ✅ FIX: pbPatchByIDAdapter を any-return に合わせるため pbPatchAnyAdapter を挟む
	// =========================================================
	listDetailQuery := listdetail.NewListDetailQuery(listdetail.NewListDetailQueryParams{
		Getter:       r.listRepo,
		NameResolver: res.nameResolver,

		PBGetter: r.productBlueprintRepo,
		TBGetter: &tbGetterAdapter{repo: r.tokenBlueprintRepo},

		InvGetter: inventoryQuery,
		InvRows:   inventoryQuery,

		ImgLister: r.listImageRecordRepo, // ✅ Firestore records

		PBPatchRepo: &pbPatchAnyAdapter{
			inner: &pbPatchByIDAdapter{repo: r.productBlueprintRepo},
		},
	})

	// =========================================================
	// ✅ ListImageUploader wiring
	//
	// NOTE:
	// - DELETE API is abolished, so ListImageDeleter wiring is removed.
	// - signed-url PUT + SaveImageFromGCS 方式なら uploader は不要（nil でもOK）
	// =========================================================
	var uploader listHandler.ListImageUploader

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
		listImageUploader:             uploader,
	}
}
