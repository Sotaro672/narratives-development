// backend/internal/application/query/mall/catalog/catalog_query.go
package catalogQuery

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	dto "narratives/internal/application/query/mall/dto"

	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
)

func (q *CatalogQuery) GetByListID(ctx context.Context, listID string) (dto.CatalogDTO, error) {
	if q == nil || q.ListRepo == nil {
		return dto.CatalogDTO{}, errors.New("catalog query: list repo is nil")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return dto.CatalogDTO{}, ldom.ErrNotFound
	}

	log.Printf("[catalog] GetByListID start listId=%q", listID)

	l, err := q.ListRepo.GetByID(ctx, listID)
	if err != nil {
		log.Printf("[catalog] list getById error listId=%q err=%q", listID, err.Error())
		return dto.CatalogDTO{}, err
	}
	if l.Status != ldom.StatusListing {
		log.Printf("[catalog] list not listing listId=%q status=%q", listID, fmt.Sprint(l.Status))
		return dto.CatalogDTO{}, ldom.ErrNotFound
	}

	out := dto.CatalogDTO{
		List: toCatalogListDTO(l),
	}

	// ------------------------------------------------------------
	// ListImages (best-effort; nil lister is allowed)
	// ------------------------------------------------------------
	// NOTE: catalog_query_list_images.go の loadListImages をここで必ず呼ぶ
	//       q.ListImagesError は best-effort で埋める（hard fail しない）
	{
		imgs, imgErr := q.loadListImages(ctx, strings.TrimSpace(out.List.ID))
		if imgErr != "" {
			out.ListImagesError = imgErr
			log.Printf("[catalog] listImages error listId=%q err=%q", listID, imgErr)
		} else if len(imgs) > 0 {
			out.ListImages = imgs
			log.Printf("[catalog] listImages ok listId=%q count=%d", listID, len(imgs))
		} else {
			log.Printf("[catalog] listImages empty listId=%q", listID)
		}
	}

	// ------------------------------------------------------------
	// Inventory (inventoryId only; fallback removed)
	// ------------------------------------------------------------
	var invDTO *dto.CatalogInventoryDTO

	if q.InventoryRepo == nil {
		out.InventoryError = "inventory repo is nil"
		log.Printf("[catalog] inventory repo is nil listId=%q", listID)
	} else {
		invID := strings.TrimSpace(out.List.InventoryID)

		log.Printf(
			"[catalog] inventory linkage listId=%q inventoryId=%q",
			listID, invID,
		)

		if invID == "" {
			// ✅ inventoryId が無い場合の fallback 機能は廃止
			out.InventoryError = "inventoryId is empty (fallback disabled)"
			log.Printf("[catalog] inventory skip (inventoryId empty) listId=%q", listID)
		} else {
			m, e := q.InventoryRepo.GetByID(ctx, invID)
			if e != nil {
				out.InventoryError = e.Error()
				log.Printf("[catalog] inventory getById error listId=%q invId=%q err=%q", listID, invID, e.Error())
			} else {
				v := toCatalogInventoryDTOFromMint(m)
				normalizeInventoryStock(v)
				invDTO = v
				out.Inventory = v
				log.Printf("[catalog] inventory getById ok listId=%q invId=%q stockKeys=%d", listID, invID, stockKeyCount(v.Stock))
			}
		}
	}

	// ------------------------------------------------------------
	// ProductBlueprint (inventory side wins)
	// ------------------------------------------------------------
	resolvedPBID := strings.TrimSpace(out.List.ProductBlueprintID)
	if invDTO != nil {
		if s := strings.TrimSpace(invDTO.ProductBlueprintID); s != "" {
			resolvedPBID = s
		}
	}

	if q.ProductRepo == nil {
		out.ProductBlueprintError = "product repo is nil"
		log.Printf("[catalog] product repo is nil listId=%q", listID)
	} else if resolvedPBID == "" {
		out.ProductBlueprintError = "productBlueprintId is empty"
		log.Printf("[catalog] productBlueprintId is empty listId=%q", listID)
	} else {
		pb, e := q.ProductRepo.GetByID(ctx, resolvedPBID)
		if e != nil {
			out.ProductBlueprintError = e.Error()
			log.Printf("[catalog] product getById error listId=%q pbId=%q err=%q", listID, resolvedPBID, e.Error())
		} else {
			pbDTO := toCatalogProductBlueprintDTO(&pb)

			if q.NameResolver != nil {
				fillProductBlueprintNames(ctx, q.NameResolver, &pbDTO)
			}

			out.ProductBlueprint = &pbDTO
			log.Printf(
				"[catalog] product getById ok listId=%q pbId=%q brandId=%q companyId=%q brandName=%q companyName=%q",
				listID,
				resolvedPBID,
				strings.TrimSpace(pbDTO.BrandID),
				strings.TrimSpace(pbDTO.CompanyID),
				getStringFieldBestEffort(pbDTO, "BrandName"),
				getStringFieldBestEffort(pbDTO, "CompanyName"),
			)
		}
	}

	// ------------------------------------------------------------
	// TokenBlueprint patch (inventory side wins)
	// ------------------------------------------------------------
	resolvedTBID := strings.TrimSpace(out.List.TokenBlueprintID)
	if invDTO != nil {
		if s := strings.TrimSpace(invDTO.TokenBlueprintID); s != "" {
			resolvedTBID = s
		}
	}

	log.Printf("[catalog] tokenBlueprint resolve listId=%q resolvedTbId=%q (list.tbId=%q inv.tbId=%q)",
		listID,
		resolvedTBID,
		strings.TrimSpace(out.List.TokenBlueprintID),
		func() string {
			if invDTO == nil {
				return ""
			}
			return strings.TrimSpace(invDTO.TokenBlueprintID)
		}(),
	)

	// best-effort: TokenRepo が nil なら “エラーを立てない”
	if q.TokenRepo == nil {
		if resolvedTBID != "" {
			log.Printf("[catalog] tokenBlueprint repo is nil (best-effort) listId=%q tbId=%q", listID, resolvedTBID)
		} else {
			log.Printf("[catalog] tokenBlueprint skip (tbId empty & repo nil) listId=%q", listID)
		}
	} else if resolvedTBID == "" {
		out.TokenBlueprintError = "tokenBlueprintId is empty"
		log.Printf("[catalog] tokenBlueprintId is empty listId=%q", listID)
	} else {
		log.Printf("[catalog] tokenBlueprint getPatchById start listId=%q tbId=%q", listID, resolvedTBID)

		patch, e := q.TokenRepo.GetPatchByID(ctx, resolvedTBID)
		if e != nil {
			out.TokenBlueprintError = e.Error()
			log.Printf("[catalog] tokenBlueprint getPatchById error listId=%q tbId=%q err=%q", listID, resolvedTBID, e.Error())
		} else {
			p := patch

			if q.NameResolver != nil {
				fillTokenBlueprintPatchNames(ctx, q.NameResolver, &p)
			}

			out.TokenBlueprint = &p
			log.Printf(
				"[catalog] tokenBlueprint getPatchById ok listId=%q tbId=%q name=%q symbol=%q brandId=%q brandName=%q companyId=%q minted=%s hasIconUrl=%t",
				listID,
				resolvedTBID,
				strings.TrimSpace(p.TokenName),
				strings.TrimSpace(p.Symbol),
				strings.TrimSpace(p.BrandID),
				strings.TrimSpace(p.BrandName),
				strings.TrimSpace(p.CompanyID),
				boolStr(p.Minted),
				strings.TrimSpace(p.IconURL) != "",
			)
		}
	}

	// ------------------------------------------------------------
	// Models (UNIFIED)
	// ------------------------------------------------------------
	if q.ModelRepo == nil {
		out.ModelVariationsError = "model repo is nil"
		log.Printf("[catalog] model repo is nil listId=%q", listID)
	} else if resolvedPBID == "" {
		out.ModelVariationsError = "productBlueprintId is empty (skip model fetch)"
		log.Printf("[catalog] model skip (pbId empty) listId=%q", listID)
	} else {
		deletedFalse := false

		res, e := q.ModelRepo.ListVariations(
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
		if e != nil {
			out.ModelVariationsError = e.Error()
			log.Printf("[catalog] model listVariations error listId=%q pbId=%q err=%q", listID, resolvedPBID, e.Error())
		} else {
			items := make([]dto.CatalogModelVariationDTO, 0, len(res.Items))

			for _, it := range res.Items {
				modelID := extractID(it)
				if modelID == "" {
					continue
				}

				mv, ge := q.ModelRepo.GetModelVariationByID(ctx, modelID)
				if ge != nil {
					if strings.TrimSpace(out.ModelVariationsError) == "" {
						out.ModelVariationsError = ge.Error()
					}
					continue
				}

				mvDTO, ok := toCatalogModelVariationDTOAny(mv)
				if !ok {
					mvDTO = dto.CatalogModelVariationDTO{
						ID:           strings.TrimSpace(modelID),
						Measurements: map[string]int{},
					}
				}
				if mvDTO.Measurements == nil {
					mvDTO.Measurements = map[string]int{}
				}

				items = append(items, mvDTO)
			}

			attachStockToModelVariations(&items, invDTO)

			out.ModelVariations = items
			log.Printf(
				"[catalog] model variations ok(list unified) listId=%q pbId=%q items=%d stockKeys=%d",
				listID,
				resolvedPBID,
				len(items),
				func() int {
					if invDTO == nil {
						return 0
					}
					return stockKeyCount(invDTO.Stock)
				}(),
			)
		}
	}

	log.Printf("[catalog] GetByListID done listId=%q listImgErr=%q invErr=%q pbErr=%q tbErr=%q modelErr=%q",
		listID,
		strings.TrimSpace(out.ListImagesError),
		strings.TrimSpace(out.InventoryError),
		strings.TrimSpace(out.ProductBlueprintError),
		strings.TrimSpace(out.TokenBlueprintError),
		strings.TrimSpace(out.ModelVariationsError),
	)

	return out, nil
}
