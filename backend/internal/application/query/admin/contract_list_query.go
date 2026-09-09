// backend/internal/application/query/admin/contract_list_query.go
package query

import (
	"context"
	"errors"
	"sort"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	memberdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

var ErrContractListQueryNotConfigured = errors.New(
	"contract list query is not configured",
)

type contractListCompanyReader interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

type contractListBrandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type contractListMemberReader interface {
	GetByID(ctx context.Context, id string) (memberdom.Record, error)
	GetByUID(ctx context.Context, uid string) (memberdom.Record, error)
}

type contractListProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productblueprintdom.ProductBlueprint, error)
}

type contractListTokenBlueprintReader interface {
	GetByID(ctx context.Context, id string) (*tokenblueprintdom.TokenBlueprint, error)
}

type contractListInventoryReader interface {
	GetByID(ctx context.Context, id string) (inventorydom.Mint, error)
}

type contractListReader interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

type contractListImageReader interface {
	ListByListID(ctx context.Context, listID string) ([]listdom.ListImage, error)
}

type contractListModelReader interface {
	GetByID(ctx context.Context, variationID string) (modeldom.ModelVariation, error)
}

type ContractListQuery struct {
	companyRepo          contractListCompanyReader
	brandRepo            contractListBrandReader
	memberRepo           contractListMemberReader
	productBlueprintRepo contractListProductBlueprintReader
	tokenBlueprintRepo   contractListTokenBlueprintReader
	inventoryRepo        contractListInventoryReader
	listRepo             contractListReader
	listImageRepo        contractListImageReader
	modelRepo            contractListModelReader
}

func NewContractListQuery(
	companyRepo contractListCompanyReader,
	brandRepo contractListBrandReader,
	memberRepo contractListMemberReader,
	productBlueprintRepo contractListProductBlueprintReader,
	tokenBlueprintRepo contractListTokenBlueprintReader,
	inventoryRepo contractListInventoryReader,
	listRepo contractListReader,
	listImageRepo contractListImageReader,
	modelRepo contractListModelReader,
) *ContractListQuery {
	return &ContractListQuery{
		companyRepo:          companyRepo,
		brandRepo:            brandRepo,
		memberRepo:           memberRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		inventoryRepo:        inventoryRepo,
		listRepo:             listRepo,
		listImageRepo:        listImageRepo,
		modelRepo:            modelRepo,
	}
}

type ContractListDetailResult struct {
	Company ContractCompanyRow    `json:"company"`
	List    ContractListDetailRow `json:"list"`
}

type ContractListDetailRow struct {
	ID                 string                 `json:"id"`
	ReadableID         string                 `json:"readableId"`
	InventoryID        string                 `json:"inventoryId"`
	ProductBlueprintID string                 `json:"productBlueprintId"`
	TokenBlueprintID   string                 `json:"tokenBlueprintId"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	ImageID            string                 `json:"imageId"`
	Images             []ContractListImageRow `json:"images"`
	Prices             []ContractListPriceRow `json:"prices"`
	ProductName        string                 `json:"productName"`
	TokenName          string                 `json:"tokenName"`
	ProductBrandID     string                 `json:"productBrandId"`
	ProductBrandName   string                 `json:"productBrandName"`
	TokenBrandID       string                 `json:"tokenBrandId"`
	TokenBrandName     string                 `json:"tokenBrandName"`
	AssigneeID         string                 `json:"assigneeId"`
	AssigneeName       string                 `json:"assigneeName"`
	Status             string                 `json:"status"`
	CreatedAt          string                 `json:"createdAt"`
	UpdatedAt          string                 `json:"updatedAt"`
}

type ContractListImageRow struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	DisplayOrder int    `json:"displayOrder"`
}

type ContractListPriceRow struct {
	ModelID     string `json:"modelId"`
	Kind        string `json:"kind"`
	ModelNumber string `json:"modelNumber"`
	Size        string `json:"size,omitempty"`
	Color       string `json:"color,omitempty"`
	RGB         *int   `json:"rgb,omitempty"`
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`
	Price       int    `json:"price"`
}

func (q *ContractListQuery) Get(
	ctx context.Context,
	companyID string,
	listID string,
) (ContractListDetailResult, error) {
	if q == nil || q.companyRepo == nil || q.brandRepo == nil || q.memberRepo == nil || q.productBlueprintRepo == nil || q.tokenBlueprintRepo == nil || q.inventoryRepo == nil || q.listRepo == nil || q.listImageRepo == nil || q.modelRepo == nil {
		return ContractListDetailResult{}, ErrContractListQueryNotConfigured
	}
	if companyID == "" {
		return ContractListDetailResult{}, companydom.ErrInvalidID
	}
	if listID == "" {
		return ContractListDetailResult{}, listdom.ErrInvalidID
	}

	company, err := q.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return ContractListDetailResult{}, err
	}

	item, err := q.listRepo.GetByID(ctx, listID)
	if err != nil {
		return ContractListDetailResult{}, err
	}
	if item.ID == "" || item.InventoryID == "" {
		return ContractListDetailResult{}, listdom.ErrNotFound
	}

	inventory, err := q.inventoryRepo.GetByID(ctx, item.InventoryID)
	if err != nil {
		return ContractListDetailResult{}, err
	}
	if inventory.ID != "" && inventory.ID != item.InventoryID {
		return ContractListDetailResult{}, listdom.ErrNotFound
	}
	if inventory.ProductBlueprintID == "" || inventory.TokenBlueprintID == "" {
		return ContractListDetailResult{}, listdom.ErrNotFound
	}

	productBlueprint, err := q.productBlueprintRepo.GetByID(ctx, inventory.ProductBlueprintID)
	if err != nil {
		if errors.Is(err, productblueprintdom.ErrNotFound) {
			return ContractListDetailResult{}, listdom.ErrNotFound
		}
		return ContractListDetailResult{}, err
	}
	if productBlueprint.ID != inventory.ProductBlueprintID || productBlueprint.CompanyID != companyID {
		return ContractListDetailResult{}, listdom.ErrNotFound
	}

	tokenBlueprint, err := q.tokenBlueprintRepo.GetByID(ctx, inventory.TokenBlueprintID)
	if err != nil {
		if errors.Is(err, tokenblueprintdom.ErrNotFound) {
			return ContractListDetailResult{}, listdom.ErrNotFound
		}
		return ContractListDetailResult{}, err
	}
	if tokenBlueprint == nil || tokenBlueprint.ID != inventory.TokenBlueprintID || tokenBlueprint.CompanyID != companyID {
		return ContractListDetailResult{}, listdom.ErrNotFound
	}

	listImages, err := q.listImageRepo.ListByListID(ctx, item.ID)
	if err != nil {
		return ContractListDetailResult{}, err
	}

	images := make([]ContractListImageRow, 0, len(listImages))
	seenImageIDs := make(map[string]struct{}, len(listImages))
	for _, image := range listImages {
		if image.ID == "" || image.URL == "" {
			continue
		}
		if _, exists := seenImageIDs[image.ID]; exists {
			continue
		}
		seenImageIDs[image.ID] = struct{}{}
		images = append(images, ContractListImageRow{
			ID:           image.ID,
			URL:          image.URL,
			DisplayOrder: image.DisplayOrder,
		})
	}

	sort.SliceStable(images, func(i, j int) bool {
		if images[i].DisplayOrder != images[j].DisplayOrder {
			return images[i].DisplayOrder < images[j].DisplayOrder
		}
		return images[i].ID < images[j].ID
	})

	prices := make([]ContractListPriceRow, 0, len(item.Prices))
	for _, price := range item.Prices {
		row := ContractListPriceRow{
			ModelID: price.ModelID,
			Price:   price.Price,
		}

		if price.ModelID != "" {
			variation, err := q.modelRepo.GetByID(ctx, price.ModelID)
			if err == nil && variation != nil {
				switch model := variation.(type) {
				case modeldom.ApparelModelVariation:
					rgb := model.Color.RGB
					row.Kind = string(modeldom.ModelVariationKindApparel)
					row.ModelNumber = model.ModelNumber
					row.Size = model.Size
					row.Color = model.Color.Name
					row.RGB = &rgb
				case modeldom.AlcoholModelVariation:
					volumeValue := model.Volume.Value
					row.Kind = string(modeldom.ModelVariationKindAlcohol)
					row.ModelNumber = model.ModelNumber
					row.VolumeValue = &volumeValue
					row.VolumeUnit = model.Volume.Unit
				}
			}
		}

		prices = append(prices, row)
	}

	updatedAt := ""
	if item.UpdatedAt != nil {
		updatedAt = formatContractDetailTime(*item.UpdatedAt)
	}

	representativeName := q.resolveMemberName(ctx, company.Admin)
	if representativeName == "" || representativeName == company.Admin {
		representativeName = "-"
	}

	return ContractListDetailResult{
		Company: ContractCompanyRow{
			ID:                 company.ID,
			Name:               company.Name,
			RepresentativeName: representativeName,
			IsActive:           company.IsActive,
			CreatedAt:          formatContractDetailTime(company.CreatedAt),
			UpdatedAt:          formatContractDetailTime(company.UpdatedAt),
		},
		List: ContractListDetailRow{
			ID:                 item.ID,
			ReadableID:         item.ReadableID,
			InventoryID:        item.InventoryID,
			ProductBlueprintID: productBlueprint.ID,
			TokenBlueprintID:   tokenBlueprint.ID,
			Title:              item.Title,
			Description:        item.Description,
			ImageID:            item.ImageID,
			Images:             images,
			Prices:             prices,
			ProductName:        productBlueprint.ProductName,
			TokenName:          tokenBlueprint.Name,
			ProductBrandID:     productBlueprint.BrandID,
			ProductBrandName:   q.resolveBrandName(ctx, productBlueprint.BrandID),
			TokenBrandID:       tokenBlueprint.BrandID,
			TokenBrandName:     q.resolveBrandName(ctx, tokenBlueprint.BrandID),
			AssigneeID:         item.AssigneeID,
			AssigneeName:       q.resolveMemberName(ctx, item.AssigneeID),
			Status:             string(item.Status),
			CreatedAt:          formatContractDetailTime(item.CreatedAt),
			UpdatedAt:          updatedAt,
		},
	}, nil
}

func (q *ContractListQuery) resolveBrandName(ctx context.Context, brandID string) string {
	if brandID == "" {
		return "-"
	}

	brand, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil || brand.Name == "" {
		return brandID
	}
	return brand.Name
}

func (q *ContractListQuery) resolveMemberName(ctx context.Context, memberID string) string {
	if memberID == "" {
		return "-"
	}

	record, err := q.memberRepo.GetByID(ctx, memberID)
	if err == nil {
		name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName)
		if name != "" {
			return name
		}
	}

	record, err = q.memberRepo.GetByUID(ctx, memberID)
	if err == nil {
		name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName)
		if name != "" {
			return name
		}
	}

	return memberID
}
