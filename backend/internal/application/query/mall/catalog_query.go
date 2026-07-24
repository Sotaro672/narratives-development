// backend/internal/application/query/mall/catalog_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"sort"

	dto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"

	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
	productBlueprintReview "narratives/internal/domain/productBlueprintReview"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Ports (minimal contracts for this query)
// ============================================================

type InventoryRepository interface {
	GetByID(
		ctx context.Context,
		id string,
	) (invdom.Mint, error)
}

type ProductBlueprintRepository interface {
	GetByID(
		ctx context.Context,
		id string,
	) (pbdom.ProductBlueprint, error)
}

type TokenBlueprintPatchRepository interface {
	GetByID(
		ctx context.Context,
		id string,
	) (*tbdom.TokenBlueprint, error)
}

// ProductBlueprintReview repository (read-only minimal for catalog)
//
// CatalogQueryではsummaryのみ利用するため、最小契約にする。
type ProductBlueprintReviewRepository interface {
	GetProductSummary(
		ctx context.Context,
		productBlueprintID string,
		status productBlueprintReview.ReviewStatus,
	) (
		productBlueprintReview.ProductReviewSummary,
		error,
	)
}

// ListImage repository (read-only minimal for catalog)
//
// Firebase Storage移行後:
//   - domain/listImageは削除済み
//   - ListImageはdomain/list.ListImageを使う
//   - ListImage.URLはFirebase Storage downloadURL
//   - backendはGCS bucket / public URLを組み立てない
type ListImageRepository interface {
	// listId配下の画像一覧を取得する。
	ListByListID(
		ctx context.Context,
		listID string,
	) ([]ldom.ListImage, error)
}

// ============================================================
// Query
// ============================================================

type CatalogQuery struct {
	ListRepo ldom.Repository

	InventoryRepo InventoryRepository
	ProductRepo   ProductBlueprintRepository
	TokenRepo     TokenBlueprintPatchRepository

	ModelRepo modeldom.RepositoryPort

	ProductBlueprintReviewRepo ProductBlueprintReviewRepository

	ListImageRepo ListImageRepository

	NameResolver *appresolver.NameResolver
}

// ============================================================
// Constructor
// ============================================================

// NewCatalogQuery is the only wiring entrypoint.
// All dependencies must be routed through this constructor.
func NewCatalogQuery(
	listRepo ldom.Repository,
	inventoryRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	modelRepo modeldom.RepositoryPort,
	listImageRepo ListImageRepository,
	tokenRepo TokenBlueprintPatchRepository,
	productBlueprintReviewRepo ProductBlueprintReviewRepository,
	nameResolver *appresolver.NameResolver,
) *CatalogQuery {
	return &CatalogQuery{
		ListRepo: listRepo,

		InventoryRepo: inventoryRepo,
		ProductRepo:   productRepo,
		TokenRepo:     tokenRepo,

		ModelRepo: modelRepo,

		ProductBlueprintReviewRepo: productBlueprintReviewRepo,

		ListImageRepo: listImageRepo,

		NameResolver: nameResolver,
	}
}

// ============================================================
// Public APIs
// ============================================================

func (
	q *CatalogQuery,
) GetByListID(
	ctx context.Context,
	listID string,
) (dto.CatalogDTO, error) {
	if q == nil || q.ListRepo == nil {
		return dto.CatalogDTO{},
			errors.New(
				"catalog query: list repo is nil",
			)
	}

	if listID == "" {
		return dto.CatalogDTO{},
			ldom.ErrNotFound
	}

	// ------------------------------------------------------------
	// List
	// ------------------------------------------------------------

	listItem, err :=
		q.ListRepo.GetByID(
			ctx,
			listID,
		)
	if err != nil {
		return dto.CatalogDTO{}, err
	}

	if listItem.Status != ldom.StatusListing {
		return dto.CatalogDTO{},
			ldom.ErrNotFound
	}

	output := dto.CatalogDTO{
		List: toCatalogListDTO(listItem),
	}

	// ------------------------------------------------------------
	// List images
	// ------------------------------------------------------------

	listImages, listImagesError :=
		q.loadListImages(
			ctx,
			output.List.ID,
		)
	if listImagesError != "" {
		return dto.CatalogDTO{},
			fmt.Errorf(
				"listImages failed: %s",
				listImagesError,
			)
	}

	output.ListImages = listImages

	// ------------------------------------------------------------
	// Inventory
	// ------------------------------------------------------------

	if q.InventoryRepo == nil {
		return dto.CatalogDTO{},
			errors.New(
				"inventory repo is nil",
			)
	}

	inventoryID := output.List.InventoryID

	if inventoryID == "" {
		return dto.CatalogDTO{},
			errors.New(
				"inventoryId is empty",
			)
	}

	inventory, err :=
		q.InventoryRepo.GetByID(
			ctx,
			inventoryID,
		)
	if err != nil {
		return dto.CatalogDTO{}, err
	}

	inventoryDTO :=
		toCatalogInventoryDTOFromMint(
			inventory,
		)

	if inventoryDTO == nil {
		return dto.CatalogDTO{},
			errors.New(
				"inventory dto is nil",
			)
	}

	output.Inventory = inventoryDTO

	// ============================================================
	// SOURCE OF TRUTH:
	// inventoryId -> inventoryDTO -> productBlueprintId/tokenBlueprintId
	//
	// List側のProductBlueprintIDとTokenBlueprintIDは参照しない。
	// ============================================================

	// ------------------------------------------------------------
	// ProductBlueprint
	// ------------------------------------------------------------

	resolvedProductBlueprintID :=
		inventoryDTO.ProductBlueprintID

	if resolvedProductBlueprintID == "" {
		return dto.CatalogDTO{},
			errors.New(
				"productBlueprintId is empty on inventory",
			)
	}

	if q.ProductRepo == nil {
		return dto.CatalogDTO{},
			errors.New(
				"product repo is nil",
			)
	}

	productBlueprint, err :=
		q.ProductRepo.GetByID(
			ctx,
			resolvedProductBlueprintID,
		)
	if err != nil {
		return dto.CatalogDTO{}, err
	}

	productBlueprintDTO :=
		toCatalogProductBlueprintDTO(
			&productBlueprint,
		)

	if q.NameResolver != nil {
		fillProductBlueprintNames(
			ctx,
			q.NameResolver,
			&productBlueprintDTO,
		)
	}

	output.ProductBlueprint =
		&productBlueprintDTO

	// ------------------------------------------------------------
	// ProductBlueprintReview summary
	// ------------------------------------------------------------

	if q.ProductBlueprintReviewRepo == nil {
		return dto.CatalogDTO{},
			errors.New(
				"productBlueprintReview repo is nil",
			)
	}

	reviewStatus :=
		productBlueprintReview.
			ReviewStatusPublished

	reviewSummary, err :=
		q.ProductBlueprintReviewRepo.
			GetProductSummary(
				ctx,
				resolvedProductBlueprintID,
				reviewStatus,
			)
	if err != nil {
		return dto.CatalogDTO{}, err
	}

	output.ProductReviewSummary =
		toCatalogProductReviewSummaryDTO(
			reviewSummary,
		)

	// ------------------------------------------------------------
	// TokenBlueprint
	// ------------------------------------------------------------

	resolvedTokenBlueprintID :=
		inventoryDTO.TokenBlueprintID

	if resolvedTokenBlueprintID == "" {
		return dto.CatalogDTO{},
			errors.New(
				"tokenBlueprintId is empty on inventory",
			)
	}

	if q.TokenRepo == nil {
		return dto.CatalogDTO{},
			errors.New(
				"tokenBlueprint repo is nil",
			)
	}

	tokenBlueprint, err :=
		q.TokenRepo.GetByID(
			ctx,
			resolvedTokenBlueprintID,
		)
	if err != nil {
		return dto.CatalogDTO{}, err
	}

	if tokenBlueprint == nil {
		return dto.CatalogDTO{},
			tbdom.ErrNotFound
	}

	tokenBlueprintPatch :=
		toTokenBlueprintPatch(
			tokenBlueprint,
		)

	if q.NameResolver != nil {
		fillTokenBlueprintPatchNames(
			ctx,
			q.NameResolver,
			&tokenBlueprintPatch,
		)
	}

	companyName := ""

	if q.NameResolver != nil {
		companyName =
			q.NameResolver.ResolveCompanyName(
				ctx,
				tokenBlueprintPatch.CompanyID,
			)

		if companyName == "" {
			brandCompanyID :=
				q.NameResolver.
					ResolveBrandCompanyID(
						ctx,
						tokenBlueprintPatch.BrandID,
					)

			if brandCompanyID != "" {
				companyName =
					q.NameResolver.
						ResolveCompanyName(
							ctx,
							brandCompanyID,
						)
			}
		}
	}

	// Firebase Storage移行後:
	//   - Patch.IconURLにはFirebase StorageのdownloadURLが入る
	//   - GCS objectPathからURLを解決しない
	//   - TokenIconObjectPathは使わない
	resolvedIconURL :=
		tokenBlueprintPatch.IconURL

	tokenBlueprintDTO :=
		dto.CatalogTokenBlueprintDTO{
			ID: tokenBlueprintPatch.ID,

			TokenName: tokenBlueprintPatch.TokenName,

			Symbol: tokenBlueprintPatch.Symbol,

			BrandID: tokenBlueprintPatch.BrandID,

			BrandName: tokenBlueprintPatch.BrandName,

			CompanyName: companyName,

			Description: tokenBlueprintPatch.Description,

			TokenIcon: resolvedIconURL,
		}

	output.TokenBlueprint =
		&tokenBlueprintDTO

	// ------------------------------------------------------------
	// Models
	// ------------------------------------------------------------

	if q.ModelRepo == nil {
		return dto.CatalogDTO{},
			errors.New(
				"model repo is nil",
			)
	}

	if q.NameResolver == nil {
		return dto.CatalogDTO{},
			errors.New(
				"name resolver is nil",
			)
	}

	variations, err :=
		q.ModelRepo.
			ListByProductBlueprintID(
				ctx,
				resolvedProductBlueprintID,
			)
	if err != nil {
		return dto.CatalogDTO{}, err
	}

	modelVariationItems := make(
		[]dto.CatalogModelVariationDTO,
		0,
		len(variations),
	)

	for _, variation := range variations {
		if variation == nil {
			return dto.CatalogDTO{},
				errors.New(
					"model variation is nil",
				)
		}

		modelID := variation.GetID()

		if modelID == "" {
			return dto.CatalogDTO{},
				errors.New(
					"model variation id is empty",
				)
		}

		resolved :=
			q.NameResolver.
				ResolveModelResolved(
					ctx,
					modelID,
				)

		if resolved.Kind == "" {
			return dto.CatalogDTO{},
				fmt.Errorf(
					"model variation resolve failed: modelId=%s",
					modelID,
				)
		}

		modelVariationItems = append(
			modelVariationItems,
			toCatalogModelVariationDTOFromResolved(
				modelID,
				resolvedProductBlueprintID,
				resolved,
			),
		)
	}

	attachStockToModelVariations(
		&modelVariationItems,
		inventoryDTO,
	)

	output.ModelVariations =
		modelVariationItems

	return output, nil
}

// ============================================================
// ListImages
// ============================================================

// loadListImages returns DTO-ready list images and an error string.
// Empty error string means success.
func (
	q *CatalogQuery,
) loadListImages(
	ctx context.Context,
	listID string,
) ([]dto.CatalogListImageDTO, string) {
	if listID == "" {
		return nil, "listId is empty"
	}

	if q == nil ||
		q.ListImageRepo == nil {
		return nil, ""
	}

	listImages, err :=
		q.ListImageRepo.ListByListID(
			ctx,
			listID,
		)
	if err != nil {
		return nil, err.Error()
	}

	output := make(
		[]dto.CatalogListImageDTO,
		0,
		len(listImages),
	)

	seen := make(map[string]struct{})

	for _, listImage := range listImages {
		id := listImage.ID

		if id == "" {
			continue
		}

		if _, exists := seen[id]; exists {
			continue
		}

		seen[id] = struct{}{}

		output = append(
			output,
			toCatalogListImageDTO(
				listImage,
			),
		)
	}

	sort.Slice(
		output,
		func(i, j int) bool {
			left := output[i]
			right := output[j]

			leftOrder := left.DisplayOrder
			rightOrder := right.DisplayOrder

			leftKnown := leftOrder > 0
			rightKnown := rightOrder > 0

			if leftKnown != rightKnown {
				return leftKnown
			}

			if leftKnown &&
				rightKnown &&
				leftOrder != rightOrder {
				return leftOrder <
					rightOrder
			}

			return left.ID < right.ID
		},
	)

	return output, ""
}

// ============================================================
// Mappers
// ============================================================

func toCatalogListDTO(
	listItem ldom.List,
) dto.CatalogListDTO {
	return dto.CatalogListDTO{
		ID: listItem.ID,

		Title: listItem.Title,

		Description: listItem.Description,

		Image: listItem.ImageID,

		Prices: listItem.Prices,

		InventoryID: listItem.InventoryID,
	}
}

func toCatalogListImageDTO(
	listImage ldom.ListImage,
) dto.CatalogListImageDTO {
	return dto.CatalogListImageDTO{
		ID: listImage.ID,

		ListID: listImage.ListID,

		URL: listImage.URL,

		DisplayOrder: func() int {
			if listImage.DisplayOrder <= 0 {
				return 0
			}

			return listImage.DisplayOrder
		}(),
	}
}

func toCatalogProductBlueprintDTO(
	productBlueprint *pbdom.ProductBlueprint,
) dto.CatalogProductBlueprintDTO {
	if productBlueprint == nil {
		return dto.CatalogProductBlueprintDTO{}
	}

	category :=
		productBlueprint.
			ProductBlueprintCategory

	output :=
		dto.CatalogProductBlueprintDTO{
			ID: productBlueprint.ID,

			ProductName: productBlueprint.ProductName,

			BrandID: productBlueprint.BrandID,

			CompanyID: productBlueprint.CompanyID,

			Printed: productBlueprint.Printed,

			ProductIDTagType: string(
				productBlueprint.
					ProductIdTag.
					Type,
			),

			ProductBlueprintCategoryID: category.ID,

			ProductBlueprintCategoryCode: category.Code,

			ProductBlueprintCategoryKind: string(category.Kind),

			ProductBlueprintCategoryNameEn: category.NameEn,

			ProductBlueprintCategoryNameJa: category.NameJa,

			ProductBlueprintCategoryPath: append(
				[]string(nil),
				category.Path...,
			),

			CategoryFields: cloneCatalogCategoryFields(
				productBlueprint.
					CategoryFields,
			),

			ModelRefs: nil,
		}

	if len(productBlueprint.ModelRefs) > 0 {
		modelRefs := make(
			[]dto.CatalogProductBlueprintModelRefDTO,
			0,
			len(productBlueprint.ModelRefs),
		)

		for _, modelRef := range productBlueprint.ModelRefs {
			if modelRef.ModelID == "" {
				continue
			}

			modelRefs = append(
				modelRefs,
				dto.CatalogProductBlueprintModelRefDTO{
					ModelID: modelRef.ModelID,

					DisplayOrder: modelRef.DisplayOrder,
				},
			)
		}

		if len(modelRefs) > 0 {
			output.ModelRefs = modelRefs
		}
	}

	return output
}

func cloneCatalogCategoryFields(
	fields pbdom.CategoryFields,
) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	output := make(
		map[string]any,
		len(fields),
	)

	for key, value := range fields {
		if key == "" ||
			value == nil {
			continue
		}

		output[key] = value
	}

	if len(output) == 0 {
		return nil
	}

	return output
}

// Mint -> CatalogInventoryDTO
//
// Firestore source of truth:
// productBlueprintId / tokenBlueprintId / modelIds /
// stock.*.accumulation / stock.*.reservedCount
func toCatalogInventoryDTOFromMint(
	mint invdom.Mint,
) *dto.CatalogInventoryDTO {
	output := &dto.CatalogInventoryDTO{
		ID: mint.ID,

		ProductBlueprintID: mint.ProductBlueprintID,

		TokenBlueprintID: mint.TokenBlueprintID,

		ModelIDs: append(
			[]string{},
			mint.ModelIDs...,
		),

		Stock: map[string]dto.
			CatalogInventoryModelStockDTO{},
	}

	if mint.Stock == nil {
		return output
	}

	for modelID, modelStock := range mint.Stock {
		if modelID == "" {
			continue
		}

		output.Stock[modelID] =
			dto.CatalogInventoryModelStockDTO{
				Accumulation: modelStock.Accumulation,

				ReservedCount: modelStock.ReservedCount,
			}
	}

	return output
}

func toTokenBlueprintPatch(
	tokenBlueprint *tbdom.TokenBlueprint,
) tbdom.Patch {
	if tokenBlueprint == nil {
		return tbdom.Patch{}
	}

	return tbdom.Patch{
		ID: tokenBlueprint.ID,

		TokenName: tokenBlueprint.Name,

		Symbol: tokenBlueprint.Symbol,

		BrandID: tokenBlueprint.BrandID,

		BrandName: "",

		CompanyID: tokenBlueprint.CompanyID,

		Description: tokenBlueprint.Description,

		Minted: tokenBlueprint.Minted,

		MetadataURI: tokenBlueprint.MetadataURI,

		IconURL: tokenBlueprint.IconURL,
	}
}

func toCatalogProductReviewSummaryDTO(
	summary productBlueprintReview.
		ProductReviewSummary,
) *dto.CatalogProductReviewSummaryDTO {
	return &dto.CatalogProductReviewSummaryDTO{
		ProductBlueprintID: summary.ProductBlueprintID,

		Status: summary.Status,

		TotalCount: summary.TotalCount,

		AverageRating: summary.AverageRating,

		Rating5Count: summary.Rating5Count,

		Rating4Count: summary.Rating4Count,

		Rating3Count: summary.Rating3Count,

		Rating2Count: summary.Rating2Count,

		Rating1Count: summary.Rating1Count,
	}
}

func toCatalogModelVariationDTOFromResolved(
	modelID string,
	productBlueprintID string,
	resolved appresolver.ModelResolved,
) dto.CatalogModelVariationDTO {
	output :=
		dto.CatalogModelVariationDTO{
			ID: modelID,

			ProductBlueprintID: productBlueprintID,

			Kind: resolved.Kind,

			ModelNumber: resolved.ModelNumber,

			Size: resolved.Size,

			ColorName: resolved.Color,

			VolumeUnit: resolved.VolumeUnit,

			Measurements: map[string]int{},

			StockKeys: 0,
		}

	if resolved.RGB != nil {
		output.ColorRGB =
			*resolved.RGB
	}

	if resolved.VolumeValue != nil {
		value :=
			float64(
				*resolved.VolumeValue,
			)

		output.VolumeValue = &value
	}

	return output
}

// ============================================================
// Name resolvers
// ============================================================

func fillProductBlueprintNames(
	ctx context.Context,
	resolver *appresolver.NameResolver,
	productBlueprintDTO *dto.CatalogProductBlueprintDTO,
) {
	if resolver == nil ||
		productBlueprintDTO == nil {
		return
	}

	if productBlueprintDTO.BrandID != "" {
		brandName :=
			resolver.ResolveBrandName(
				ctx,
				productBlueprintDTO.BrandID,
			)

		if brandName != "" {
			productBlueprintDTO.BrandName =
				brandName
		}
	}

	if productBlueprintDTO.CompanyID != "" {
		companyName :=
			resolver.ResolveCompanyName(
				ctx,
				productBlueprintDTO.CompanyID,
			)

		if companyName != "" {
			productBlueprintDTO.CompanyName =
				companyName
		}
	}
}

// tbdom.Patchはvalue型を前提とする。
// CompanyNameは存在しない。
func fillTokenBlueprintPatchNames(
	ctx context.Context,
	resolver *appresolver.NameResolver,
	patch *tbdom.Patch,
) {
	if resolver == nil ||
		patch == nil {
		return
	}

	if patch.BrandID != "" &&
		patch.BrandName == "" {
		brandName :=
			resolver.ResolveBrandName(
				ctx,
				patch.BrandID,
			)

		if brandName != "" {
			patch.BrandName = brandName
		}
	}
}

// ============================================================
// Stock helpers
// ============================================================

func stockKeyCount(
	stock map[string]dto.
		CatalogInventoryModelStockDTO,
) int {
	return len(stock)
}

// attachStockToModelVariations sets StockKeys only.
func attachStockToModelVariations(
	items *[]dto.CatalogModelVariationDTO,
	inventory *dto.CatalogInventoryDTO,
) {
	if items == nil ||
		len(*items) == 0 {
		return
	}

	stockKeys := 0

	if inventory != nil {
		stockKeys =
			stockKeyCount(
				inventory.Stock,
			)
	}

	for index := range *items {
		(*items)[index].StockKeys =
			stockKeys
	}
}
