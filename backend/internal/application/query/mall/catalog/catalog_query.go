// backend/internal/application/query/mall/catalog/catalog_query.go
package catalogQuery

import (
	"context"
	"errors"
	"fmt"
	"log"

	dto "narratives/internal/application/query/mall/dto"

	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	productBlueprintReview "narratives/internal/domain/productBlueprintReview"
)

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
		log.Printf("[catalog][error] list getById failed listId=%q err=%q", listID, err.Error())
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
			log.Printf("[catalog][error] listImages failed listId=%q err=%q", listID, imgErr)
			return dto.CatalogDTO{}, fmt.Errorf("listImages failed: %s", imgErr)
		}
		out.ListImages = imgs
	}

	// ------------------------------------------------------------
	// Inventory (must; inventoryId only; fallback removed)
	// ------------------------------------------------------------
	if q.InventoryRepo == nil {
		log.Printf("[catalog][error] inventory repo is nil listId=%q", listID)
		return dto.CatalogDTO{}, errors.New("inventory repo is nil")
	}

	invID := out.List.InventoryID
	if invID == "" {
		log.Printf("[catalog][error] inventoryId is empty listId=%q", listID)
		return dto.CatalogDTO{}, errors.New("inventoryId is empty")
	}

	m, invErr := q.InventoryRepo.GetByID(ctx, invID)
	if invErr != nil {
		log.Printf("[catalog][error] inventory getById failed listId=%q invId=%q err=%q", listID, invID, invErr.Error())
		return dto.CatalogDTO{}, invErr
	}

	invDTO := toCatalogInventoryDTOFromMint(m)
	if invDTO == nil {
		log.Printf("[catalog][error] inventory dto is nil listId=%q invId=%q", listID, invID)
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
		log.Printf("[catalog][error] productBlueprintId is empty on inventory listId=%q invId=%q", listID, invID)
		return dto.CatalogDTO{}, errors.New("productBlueprintId is empty on inventory")
	}

	if q.ProductRepo == nil {
		log.Printf("[catalog][error] product repo is nil listId=%q invId=%q pbId=%q", listID, invID, resolvedPBID)
		return dto.CatalogDTO{}, errors.New("product repo is nil")
	}

	pb, pbErr := q.ProductRepo.GetByID(ctx, resolvedPBID)
	if pbErr != nil {
		log.Printf("[catalog][error] product getById failed listId=%q invId=%q pbId=%q err=%q", listID, invID, resolvedPBID, pbErr.Error())
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
		log.Printf("[catalog][error] productBlueprintReview repo is nil listId=%q pbId=%q", listID, resolvedPBID)
		return dto.CatalogDTO{}, errors.New("productBlueprintReview repo is nil")
	}

	reviewStatus := productBlueprintReview.ReviewStatusPublished

	summary, sumErr := q.ProductBlueprintReviewRepo.GetProductSummary(ctx, resolvedPBID, reviewStatus)
	if sumErr != nil {
		log.Printf("[catalog][error] product review summary failed listId=%q pbId=%q err=%q", listID, resolvedPBID, sumErr.Error())
		return dto.CatalogDTO{}, sumErr
	}
	out.ProductReviewSummary = toCatalogProductReviewSummaryDTO(summary)

	// ------------------------------------------------------------
	// TokenBlueprint patch (must; inventory route ONLY) -> dto.CatalogTokenBlueprintDTO
	// ------------------------------------------------------------
	resolvedTBID := invDTO.TokenBlueprintID
	if resolvedTBID == "" {
		log.Printf("[catalog][error] tokenBlueprintId is empty on inventory listId=%q invId=%q", listID, invID)
		return dto.CatalogDTO{}, errors.New("tokenBlueprintId is empty on inventory")
	}

	if q.TokenRepo == nil {
		log.Printf("[catalog][error] tokenBlueprint repo is nil listId=%q invId=%q tbId=%q", listID, invID, resolvedTBID)
		return dto.CatalogDTO{}, errors.New("tokenBlueprint repo is nil")
	}

	patch, tbErr := q.TokenRepo.GetPatchByID(ctx, resolvedTBID)
	if tbErr != nil {
		log.Printf("[catalog][error] tokenBlueprint getPatchById failed listId=%q invId=%q tbId=%q err=%q", listID, invID, resolvedTBID, tbErr.Error())
		return dto.CatalogDTO{}, tbErr
	}

	p := patch
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
		log.Printf("[catalog][error] model repo is nil listId=%q pbId=%q", listID, resolvedPBID)
		return dto.CatalogDTO{}, errors.New("model repo is nil")
	}

	deletedFalse := false
	res, mvErr := q.ModelRepo.ListVariations(
		ctx,
		modeldom.VariationFilter{
			ProductBlueprintID: resolvedPBID,
			Deleted:            &deletedFalse,
		},
		modeldom.Page{
			Number:  1,
			PerPage: 200,
		},
	)
	if mvErr != nil {
		log.Printf("[catalog][error] model listVariations failed listId=%q pbId=%q err=%q", listID, resolvedPBID, mvErr.Error())
		return dto.CatalogDTO{}, mvErr
	}

	items := make([]dto.CatalogModelVariationDTO, 0, len(res.Items))
	for _, it := range res.Items {
		if it == nil {
			log.Printf("[catalog][error] model variation is nil listId=%q pbId=%q", listID, resolvedPBID)
			return dto.CatalogDTO{}, errors.New("model variation is nil")
		}

		modelID := it.GetID()
		if modelID == "" {
			log.Printf("[catalog][error] model variation id is empty listId=%q pbId=%q", listID, resolvedPBID)
			return dto.CatalogDTO{}, errors.New("model variation id is empty")
		}

		mv, ge := q.ModelRepo.GetModelVariationByID(ctx, modelID)
		if ge != nil {
			log.Printf("[catalog][error] model getById failed listId=%q modelId=%q err=%q", listID, modelID, ge.Error())
			return dto.CatalogDTO{}, ge
		}

		mvDTO, ok := toCatalogModelVariationDTOAny(mv)
		if !ok {
			log.Printf("[catalog][error] model variation dto convert failed listId=%q modelId=%q", listID, modelID)
			return dto.CatalogDTO{}, fmt.Errorf("model variation dto convert failed: modelId=%s", modelID)
		}
		if mvDTO.Measurements == nil {
			mvDTO.Measurements = map[string]int{}
		}

		items = append(items, mvDTO)
	}

	attachStockToModelVariations(&items, invDTO)
	out.ModelVariations = items

	return out, nil
}
