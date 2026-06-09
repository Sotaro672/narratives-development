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
	GetByID(ctx context.Context, id string) (invdom.Mint, error)
}

type ProductBlueprintRepository interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

type TokenBlueprintPatchRepository interface {
	GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
}

// ProductBlueprintReview repository (read-only minimal for catalog)
// CatalogQuery では summary のみ利用するため、最小契約にする
type ProductBlueprintReviewRepository interface {
	GetProductSummary(
		ctx context.Context,
		productBlueprintID string,
		status productBlueprintReview.ReviewStatus,
	) (productBlueprintReview.ProductReviewSummary, error)
}

// ListImage repository (read-only minimal for catalog)
//
// Firebase Storage 移行後:
// - domain/listImage は削除済み
// - ListImage は domain/list.ListImage を使う
// - ListImage.URL は Firebase Storage downloadURL
// - backend は GCS bucket / public URL を組み立てない
type ListImageRepository interface {
	// listId 配下の画像一覧（displayOrder を含む前提）
	ListByListID(ctx context.Context, listID string) ([]ldom.ListImage, error)
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

	// product blueprint reviews
	ProductBlueprintReviewRepo ProductBlueprintReviewRepository

	// list images
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
	invRepo InventoryRepository,
	productRepo ProductBlueprintRepository,
	modelRepo modeldom.RepositoryPort,
	listImageRepo ListImageRepository,
	tokenRepo TokenBlueprintPatchRepository,
	productBlueprintReviewRepo ProductBlueprintReviewRepository,
	nameResolver *appresolver.NameResolver,
) *CatalogQuery {
	return &CatalogQuery{
		ListRepo:                   listRepo,
		InventoryRepo:              invRepo,
		ProductRepo:                productRepo,
		TokenRepo:                  tokenRepo,
		ModelRepo:                  modelRepo,
		ProductBlueprintReviewRepo: productBlueprintReviewRepo,
		ListImageRepo:              listImageRepo,
		NameResolver:               nameResolver,
	}
}

// ============================================================
// Public APIs
// ============================================================

func (q *CatalogQuery) GetByListID(ctx context.Context, listID string) (dto.CatalogDTO, error) {
	if q == nil || q.ListRepo == nil {
		return dto.CatalogDTO{}, errors.New("catalog query: list repo is nil")
	}
	if listID == "" {
		return dto.CatalogDTO{}, ldom.ErrNotFound
	}

	// ------------------------------------------------------------
	// List (must)
	// ------------------------------------------------------------
	l, err := q.ListRepo.GetByID(ctx, listID)
	if err != nil {
		return dto.CatalogDTO{}, err
	}
	if l.Status != ldom.StatusListing {
		return dto.CatalogDTO{}, ldom.ErrNotFound
	}

	out := dto.CatalogDTO{
		List: toCatalogListDTO(l),
	}

	// ------------------------------------------------------------
	// ListImages (must)
	// ------------------------------------------------------------
	{
		imgs, imgErr := q.loadListImages(ctx, out.List.ID)
		if imgErr != "" {
			return dto.CatalogDTO{}, fmt.Errorf("listImages failed: %s", imgErr)
		}
		out.ListImages = imgs
	}

	// ------------------------------------------------------------
	// Inventory (must; inventoryId only)
	// ------------------------------------------------------------
	if q.InventoryRepo == nil {
		return dto.CatalogDTO{}, errors.New("inventory repo is nil")
	}

	invID := out.List.InventoryID
	if invID == "" {
		return dto.CatalogDTO{}, errors.New("inventoryId is empty")
	}

	m, invErr := q.InventoryRepo.GetByID(ctx, invID)
	if invErr != nil {
		return dto.CatalogDTO{}, invErr
	}

	invDTO := toCatalogInventoryDTOFromMint(m)
	if invDTO == nil {
		return dto.CatalogDTO{}, errors.New("inventory dto is nil")
	}
	out.Inventory = invDTO

	// ============================================================
	// SOURCE OF TRUTH: inventoryId -> inventoryDTO -> (pbId/tbId)
	// list 側の ProductBlueprintID / TokenBlueprintID は一切参照しない
	// ============================================================

	// ------------------------------------------------------------
	// ProductBlueprint (must; inventory route ONLY)
	// ------------------------------------------------------------
	resolvedPBID := invDTO.ProductBlueprintID
	if resolvedPBID == "" {
		return dto.CatalogDTO{}, errors.New("productBlueprintId is empty on inventory")
	}

	if q.ProductRepo == nil {
		return dto.CatalogDTO{}, errors.New("product repo is nil")
	}

	pb, pbErr := q.ProductRepo.GetByID(ctx, resolvedPBID)
	if pbErr != nil {
		return dto.CatalogDTO{}, pbErr
	}

	pbDTO := toCatalogProductBlueprintDTO(&pb)
	if q.NameResolver != nil {
		fillProductBlueprintNames(ctx, q.NameResolver, &pbDTO)
	}
	out.ProductBlueprint = &pbDTO

	// ------------------------------------------------------------
	// ProductBlueprintReview summary (must)
	// productBlueprintId == docId
	// ------------------------------------------------------------
	if q.ProductBlueprintReviewRepo == nil {
		return dto.CatalogDTO{}, errors.New("productBlueprintReview repo is nil")
	}

	reviewStatus := productBlueprintReview.ReviewStatusPublished

	summary, sumErr := q.ProductBlueprintReviewRepo.GetProductSummary(ctx, resolvedPBID, reviewStatus)
	if sumErr != nil {
		return dto.CatalogDTO{}, sumErr
	}
	out.ProductReviewSummary = toCatalogProductReviewSummaryDTO(summary)

	// ------------------------------------------------------------
	// TokenBlueprint (must; inventory route ONLY) -> dto.CatalogTokenBlueprintDTO
	// ------------------------------------------------------------
	resolvedTBID := invDTO.TokenBlueprintID
	if resolvedTBID == "" {
		return dto.CatalogDTO{}, errors.New("tokenBlueprintId is empty on inventory")
	}

	if q.TokenRepo == nil {
		return dto.CatalogDTO{}, errors.New("tokenBlueprint repo is nil")
	}

	tokenBlueprint, tbErr := q.TokenRepo.GetByID(ctx, resolvedTBID)
	if tbErr != nil {
		return dto.CatalogDTO{}, tbErr
	}
	if tokenBlueprint == nil {
		return dto.CatalogDTO{}, tbdom.ErrNotFound
	}

	p := toTokenBlueprintPatch(tokenBlueprint)
	if q.NameResolver != nil {
		fillTokenBlueprintPatchNames(ctx, q.NameResolver, &p)
	}

	companyName := ""
	if q.NameResolver != nil {
		companyName = q.NameResolver.ResolveCompanyName(ctx, p.CompanyID)
		if companyName == "" {
			brandCompanyID := q.NameResolver.ResolveBrandCompanyID(ctx, p.BrandID)
			if brandCompanyID != "" {
				companyName = q.NameResolver.ResolveCompanyName(ctx, brandCompanyID)
			}
		}
	}

	// Firebase Storage 移行後:
	// - Patch.IconURL には Firebase Storage の downloadURL が入る
	// - GCS objectPath から URL を解決しない
	// - gcs.NewTokenIconURLResolver / TokenIconObjectPath は使わない
	resolvedIconURL := p.IconURL

	tb := dto.CatalogTokenBlueprintDTO{
		ID:          p.ID,
		TokenName:   p.TokenName,
		Symbol:      p.Symbol,
		BrandID:     p.BrandID,
		BrandName:   p.BrandName,
		CompanyName: companyName,
		Description: p.Description,
		TokenIcon:   resolvedIconURL,
	}
	out.TokenBlueprint = &tb

	// ------------------------------------------------------------
	// Models (must; ProductBlueprintID comes from inventory route ONLY)
	// ------------------------------------------------------------
	if q.ModelRepo == nil {
		return dto.CatalogDTO{}, errors.New("model repo is nil")
	}
	if q.NameResolver == nil {
		return dto.CatalogDTO{}, errors.New("name resolver is nil")
	}

	variations, mvErr := q.ModelRepo.ListByProductBlueprintID(ctx, resolvedPBID)
	if mvErr != nil {
		return dto.CatalogDTO{}, mvErr
	}

	items := make([]dto.CatalogModelVariationDTO, 0, len(variations))
	for _, mv := range variations {
		if mv == nil {
			return dto.CatalogDTO{}, errors.New("model variation is nil")
		}

		modelID := mv.GetID()
		if modelID == "" {
			return dto.CatalogDTO{}, errors.New("model variation id is empty")
		}

		resolved := q.NameResolver.ResolveModelResolved(ctx, modelID)
		if resolved.Kind == "" {
			return dto.CatalogDTO{}, fmt.Errorf("model variation resolve failed: modelId=%s", modelID)
		}

		items = append(items, toCatalogModelVariationDTOFromResolved(
			modelID,
			resolvedPBID,
			resolved,
		))
	}

	attachStockToModelVariations(&items, invDTO)
	out.ModelVariations = items

	return out, nil
}

// ============================================================
// ListImages (listId -> listImage[])
// - best-effort: ListImageRepo が nil の場合はエラーにせず空で返す
// - sort: displayOrder asc (known first), then id asc
//
// Firebase Storage migration policy:
// - ListImage は domain/list.ListImage を使う
// - ListImage.URL は Firebase Storage downloadURL
// - backend は GCS bucket / public URL を組み立てない
// - backend は objectPath / fileName / size を扱わない
// ============================================================

// loadListImages returns DTO-ready list images + error string (empty means OK).
func (q *CatalogQuery) loadListImages(ctx context.Context, listID string) ([]dto.CatalogListImageDTO, string) {
	if listID == "" {
		return nil, "listId is empty"
	}

	// best-effort: repo が無ければ壊さない（catalogの必須要件にしない）
	if q == nil || q.ListImageRepo == nil {
		return nil, ""
	}

	imgs, err := q.ListImageRepo.ListByListID(ctx, listID)
	if err != nil {
		return nil, err.Error()
	}

	out := make([]dto.CatalogListImageDTO, 0, len(imgs))
	seen := map[string]struct{}{}

	for _, it := range imgs {
		id := it.ID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		out = append(out, toCatalogListImageDTO(it))
	}

	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]

		ao := a.DisplayOrder
		bo := b.DisplayOrder

		aKnown := ao > 0
		bKnown := bo > 0

		// known first
		if aKnown != bKnown {
			return aKnown
		}

		// both known: order asc
		if aKnown && bKnown && ao != bo {
			return ao < bo
		}

		// fallback: id asc
		return a.ID < b.ID
	})

	return out, ""
}

// ============================================================
// Mappers
// ============================================================

func toCatalogListDTO(l ldom.List) dto.CatalogListDTO {
	return dto.CatalogListDTO{
		ID:          l.ID,
		Title:       l.Title,
		Description: l.Description,
		Image:       l.ImageID, // primary image docID (not URL)
		Prices:      l.Prices,

		InventoryID: l.InventoryID,
	}
}

func toCatalogListImageDTO(img ldom.ListImage) dto.CatalogListImageDTO {
	return dto.CatalogListImageDTO{
		ID:     img.ID,
		ListID: img.ListID,
		URL:    img.URL,
		DisplayOrder: func() int {
			if img.DisplayOrder <= 0 {
				return 0
			}
			return img.DisplayOrder
		}(),
	}
}

func toCatalogProductBlueprintDTO(
	pb *pbdom.ProductBlueprint,
) dto.CatalogProductBlueprintDTO {
	if pb == nil {
		return dto.CatalogProductBlueprintDTO{}
	}

	category := pb.ProductBlueprintCategory

	out := dto.CatalogProductBlueprintDTO{
		ID:          pb.ID,
		ProductName: pb.ProductName,
		BrandID:     pb.BrandID,
		CompanyID:   pb.CompanyID,

		Printed:          pb.Printed,
		ProductIDTagType: pb.ProductIdTag.Type,

		ProductBlueprintCategoryID:     category.ID,
		ProductBlueprintCategoryCode:   category.Code,
		ProductBlueprintCategoryKind:   string(category.Kind),
		ProductBlueprintCategoryNameEn: category.NameEn,
		ProductBlueprintCategoryNameJa: category.NameJa,
		ProductBlueprintCategoryPath:   append([]string(nil), category.Path...),

		CategoryFields: cloneCatalogCategoryFields(pb.CategoryFields),

		ModelRefs: nil,
	}

	if len(pb.ModelRefs) > 0 {
		refs := make(
			[]dto.CatalogProductBlueprintModelRefDTO,
			0,
			len(pb.ModelRefs),
		)

		for _, r := range pb.ModelRefs {
			if r.ModelID == "" {
				continue
			}

			refs = append(refs, dto.CatalogProductBlueprintModelRefDTO{
				ModelID:      r.ModelID,
				DisplayOrder: r.DisplayOrder,
			})
		}

		if len(refs) > 0 {
			out.ModelRefs = refs
		}
	}

	return out
}

func cloneCatalogCategoryFields(fields pbdom.CategoryFields) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	out := make(map[string]any, len(fields))

	for key, value := range fields {
		if key == "" || value == nil {
			continue
		}

		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// Mint -> CatalogInventoryDTO
// Firestore 正:
// productBlueprintId / tokenBlueprintId / modelIds / stock.*.accumulation / stock.*.reservedCount
func toCatalogInventoryDTOFromMint(m invdom.Mint) *dto.CatalogInventoryDTO {
	out := &dto.CatalogInventoryDTO{
		ID:                 m.ID,
		ProductBlueprintID: m.ProductBlueprintID,
		TokenBlueprintID:   m.TokenBlueprintID,
		ModelIDs:           append([]string{}, m.ModelIDs...),
		Stock:              map[string]dto.CatalogInventoryModelStockDTO{},
	}

	if m.Stock == nil {
		return out
	}

	for modelID, ms := range m.Stock {
		if modelID == "" {
			continue
		}

		out.Stock[modelID] = dto.CatalogInventoryModelStockDTO{
			Accumulation:  ms.Accumulation,
			ReservedCount: ms.ReservedCount,
		}
	}

	return out
}

func toTokenBlueprintPatch(tb *tbdom.TokenBlueprint) tbdom.Patch {
	if tb == nil {
		return tbdom.Patch{}
	}

	return tbdom.Patch{
		ID:          tb.ID,
		TokenName:   tb.Name,
		Symbol:      tb.Symbol,
		BrandID:     tb.BrandID,
		BrandName:   "",
		CompanyID:   tb.CompanyID,
		Description: tb.Description,
		Minted:      tb.Minted,
		MetadataURI: tb.MetadataURI,
		IconURL:     tb.IconURL,
	}
}

// ProductBlueprintReview summary -> CatalogProductReviewSummaryDTO
func toCatalogProductReviewSummaryDTO(
	s productBlueprintReview.ProductReviewSummary,
) *dto.CatalogProductReviewSummaryDTO {
	return &dto.CatalogProductReviewSummaryDTO{
		ProductBlueprintID: s.ProductBlueprintID,
		Status:             s.Status,
		TotalCount:         s.TotalCount,
		AverageRating:      s.AverageRating,
		Rating5Count:       s.Rating5Count,
		Rating4Count:       s.Rating4Count,
		Rating3Count:       s.Rating3Count,
		Rating2Count:       s.Rating2Count,
		Rating1Count:       s.Rating1Count,
	}
}

func toCatalogModelVariationDTOFromResolved(
	modelID string,
	productBlueprintID string,
	resolved appresolver.ModelResolved,
) dto.CatalogModelVariationDTO {
	out := dto.CatalogModelVariationDTO{
		ID:                 modelID,
		ProductBlueprintID: productBlueprintID,
		Kind:               resolved.Kind,
		ModelNumber:        resolved.ModelNumber,

		Size:      resolved.Size,
		ColorName: resolved.Color,

		VolumeUnit: resolved.VolumeUnit,

		Measurements: map[string]int{},
		StockKeys:    0,
	}

	if resolved.RGB != nil {
		out.ColorRGB = *resolved.RGB
	}

	if resolved.VolumeValue != nil {
		v := float64(*resolved.VolumeValue)
		out.VolumeValue = &v
	}

	return out
}

// ============================================================
// Name resolvers
// ============================================================

func fillProductBlueprintNames(ctx context.Context, r *appresolver.NameResolver, dtoPB *dto.CatalogProductBlueprintDTO) {
	if r == nil || dtoPB == nil {
		return
	}

	if dtoPB.BrandID != "" {
		if bn := r.ResolveBrandName(ctx, dtoPB.BrandID); bn != "" {
			dtoPB.BrandName = bn
		}
	}

	if dtoPB.CompanyID != "" {
		if cn := r.ResolveCompanyName(ctx, dtoPB.CompanyID); cn != "" {
			dtoPB.CompanyName = cn
		}
	}
}

// tbdom.Patch は value 型（string/bool）前提。CompanyName は存在しない。
func fillTokenBlueprintPatchNames(ctx context.Context, r *appresolver.NameResolver, p *tbdom.Patch) {
	if r == nil || p == nil {
		return
	}

	if p.BrandID != "" && p.BrandName == "" {
		if bn := r.ResolveBrandName(ctx, p.BrandID); bn != "" {
			p.BrandName = bn
		}
	}
}

// ============================================================
// Stock helpers
// ============================================================

func stockKeyCount(stock map[string]dto.CatalogInventoryModelStockDTO) int {
	return len(stock)
}

// attachStockToModelVariations sets StockKeys only.
func attachStockToModelVariations(items *[]dto.CatalogModelVariationDTO, inv *dto.CatalogInventoryDTO) {
	if items == nil || len(*items) == 0 {
		return
	}

	stockKeys := 0
	if inv != nil {
		stockKeys = stockKeyCount(inv.Stock)
	}

	for i := range *items {
		(*items)[i].StockKeys = stockKeys
	}
}
