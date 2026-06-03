// backend/internal/application/query/mall/brand_query.go
package mall

import (
	"context"
	"errors"
	"strings"

	"narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	listdom "narratives/internal/domain/list"
	productBlueprintdom "narratives/internal/domain/productBlueprint"
	tokenBlueprintdom "narratives/internal/domain/tokenBlueprint"
)

type BrandQuery struct {
	brandRepo            brand.Repository
	companyRepo          companydom.Repository
	productBlueprintRepo productBlueprintdom.Repository
	tokenBlueprintRepo   tokenBlueprintdom.RepositoryPort
	listRepo             listdom.Repository
}

func NewBrandQuery(
	brandRepo brand.Repository,
	companyRepo companydom.Repository,
	productBlueprintRepo productBlueprintdom.Repository,
	tokenBlueprintRepo tokenBlueprintdom.RepositoryPort,
	listRepo listdom.Repository,
) *BrandQuery {
	return &BrandQuery{
		brandRepo:            brandRepo,
		companyRepo:          companyRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		listRepo:             listRepo,
	}
}

type BrandDetailDTO struct {
	BrandID              string   `json:"brandId"`
	BrandName            string   `json:"brandName"`
	URL                  string   `json:"websiteUrl"`
	BrandIcon            string   `json:"brandIcon"`
	BrandBackgroundImage string   `json:"brandBackgroundImage"`
	Description          string   `json:"description"`
	CompanyID            string   `json:"companyId"`
	CompanyName          string   `json:"companyName"`
	InventoryIDs         []string `json:"inventoryIds"`
	ListIDs              []string `json:"listIds"`
}

func (q *BrandQuery) GetBrandDetailByID(ctx context.Context, brandID string) (BrandDetailDTO, error) {
	if brandID == "" {
		return BrandDetailDTO{}, brand.ErrInvalidID
	}

	b, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return BrandDetailDTO{}, err
	}

	companyName := ""
	if q.companyRepo != nil && b.CompanyID != "" {
		companyEntity, err := q.companyRepo.GetByID(ctx, b.CompanyID)
		if err != nil && !errors.Is(err, companydom.ErrNotFound) {
			return BrandDetailDTO{}, err
		}
		if err == nil {
			companyName = companyEntity.Name
		}
	}

	inventoryIDs, err := q.listInventoryIDsByBrandID(ctx, brandID)
	if err != nil {
		return BrandDetailDTO{}, err
	}

	listIDs, err := q.listListingListIDsByInventoryIDs(ctx, inventoryIDs)
	if err != nil {
		return BrandDetailDTO{}, err
	}

	return BrandDetailDTO{
		BrandID:              b.ID,
		BrandName:            b.Name,
		URL:                  b.URL,
		BrandIcon:            b.BrandIcon,
		BrandBackgroundImage: b.BrandBackgroundImage,
		Description:          b.Description,
		CompanyID:            b.CompanyID,
		CompanyName:          companyName,
		InventoryIDs:         inventoryIDs,
		ListIDs:              listIDs,
	}, nil
}

func (q *BrandQuery) listInventoryIDsByBrandID(ctx context.Context, brandID string) ([]string, error) {
	if brandID == "" {
		return []string{}, brand.ErrInvalidID
	}

	if q.productBlueprintRepo == nil || q.tokenBlueprintRepo == nil {
		return []string{}, nil
	}

	productBlueprintIDs, err := q.productBlueprintRepo.ListIDsByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	tokenBlueprints, err := q.listAllTokenBlueprintsByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	if len(productBlueprintIDs) == 0 || len(tokenBlueprints) == 0 {
		return []string{}, nil
	}

	seen := make(map[string]struct{}, len(productBlueprintIDs)*len(tokenBlueprints))
	inventoryIDs := make([]string, 0, len(productBlueprintIDs)*len(tokenBlueprints))

	for _, pbID := range productBlueprintIDs {
		if pbID == "" {
			continue
		}

		for _, tb := range tokenBlueprints {
			if tb.ID == "" {
				continue
			}

			inventoryID := buildInventoryID(pbID, tb.ID)
			if inventoryID == "" {
				continue
			}

			if _, ok := seen[inventoryID]; ok {
				continue
			}

			seen[inventoryID] = struct{}{}
			inventoryIDs = append(inventoryIDs, inventoryID)
		}
	}

	return inventoryIDs, nil
}

func (q *BrandQuery) listListingListIDsByInventoryIDs(ctx context.Context, inventoryIDs []string) ([]string, error) {
	if q.listRepo == nil || len(inventoryIDs) == 0 {
		return []string{}, nil
	}

	const perPage = 200

	listing := listdom.StatusListing

	seen := make(map[string]struct{})
	listIDs := make([]string, 0)

	for _, inventoryID := range inventoryIDs {
		if inventoryID == "" {
			continue
		}

		pageNumber := 1

		for {
			result, err := q.listRepo.List(
				ctx,
				listdom.Filter{
					InventoryIDs: []string{inventoryID},
					Status:       &listing,
				},
				listdom.Sort{},
				listdom.Page{
					Number:  pageNumber,
					PerPage: perPage,
				},
			)
			if err != nil {
				return nil, err
			}

			if len(result.Items) == 0 {
				break
			}

			for _, l := range result.Items {
				if l.ID == "" {
					continue
				}

				// Repository 実装差分への防御。
				// 本来は filter.Status で status=listing のみ返る想定。
				if l.Status != listdom.StatusListing {
					continue
				}

				// Repository 実装差分への防御。
				// 本来は filter.InventoryIDs で該当 inventoryId のみ返る想定。
				if l.InventoryID != inventoryID {
					continue
				}

				if _, ok := seen[l.ID]; ok {
					continue
				}

				seen[l.ID] = struct{}{}
				listIDs = append(listIDs, l.ID)
			}

			if len(result.Items) < perPage {
				break
			}

			pageNumber++
		}
	}

	return listIDs, nil
}

func (q *BrandQuery) listAllTokenBlueprintsByBrandID(
	ctx context.Context,
	brandID string,
) ([]tokenBlueprintdom.TokenBlueprint, error) {
	if q.tokenBlueprintRepo == nil {
		return []tokenBlueprintdom.TokenBlueprint{}, nil
	}

	const perPage = 200

	all := make([]tokenBlueprintdom.TokenBlueprint, 0)
	pageNumber := 1

	for {
		result, err := q.tokenBlueprintRepo.ListByBrandID(ctx, brandID, domcommon.Page{
			Number:  pageNumber,
			PerPage: perPage,
		})
		if err != nil {
			return nil, err
		}

		if len(result.Items) == 0 {
			break
		}

		all = append(all, result.Items...)

		if len(result.Items) < perPage {
			break
		}

		pageNumber++
	}

	return all, nil
}

func buildInventoryID(productBlueprintID, tokenBlueprintID string) string {
	if productBlueprintID == "" || tokenBlueprintID == "" {
		return ""
	}

	sanitize := func(s string) string {
		return strings.ReplaceAll(s, "/", "_")
	}

	pb := sanitize(productBlueprintID)
	tb := sanitize(tokenBlueprintID)

	return pb + "__" + tb
}
