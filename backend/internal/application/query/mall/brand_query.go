// backend/internal/application/query/mall/brand_query.go
package mall

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	productBlueprintdom "narratives/internal/domain/productBlueprint"
)

type BrandQuery struct {
	brandRepo            brand.Repository
	companyRepo          companydom.Repository
	productBlueprintRepo productBlueprintdom.Repository
	inventoryRepo        inventorydom.RepositoryPort
	listRepo             listdom.Repository
}

func NewBrandQuery(
	brandRepo brand.Repository,
	companyRepo companydom.Repository,
	productBlueprintRepo productBlueprintdom.Repository,
	inventoryRepo inventorydom.RepositoryPort,
	listRepo listdom.Repository,
) *BrandQuery {
	return &BrandQuery{
		brandRepo:            brandRepo,
		companyRepo:          companyRepo,
		productBlueprintRepo: productBlueprintRepo,
		inventoryRepo:        inventoryRepo,
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

	if q.productBlueprintRepo == nil || q.inventoryRepo == nil {
		return []string{}, nil
	}

	productBlueprintIDs, err := q.productBlueprintRepo.ListIDsByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}

	if len(productBlueprintIDs) == 0 {
		return []string{}, nil
	}

	seen := make(map[string]struct{})
	inventoryIDs := make([]string, 0)

	for _, productBlueprintID := range productBlueprintIDs {
		if productBlueprintID == "" {
			continue
		}

		inventories, err := q.inventoryRepo.ListByProductBlueprintID(ctx, productBlueprintID)
		if err != nil {
			return nil, err
		}

		for _, inv := range inventories {
			if inv.ID == "" {
				continue
			}

			if _, ok := seen[inv.ID]; ok {
				continue
			}

			seen[inv.ID] = struct{}{}
			inventoryIDs = append(inventoryIDs, inv.ID)
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

func writeMallBrandErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case brand.ErrInvalidID:
		code = http.StatusBadRequest
	case brand.ErrNotFound:
		code = http.StatusNotFound
	case brand.ErrConflict:
		code = http.StatusConflict
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
