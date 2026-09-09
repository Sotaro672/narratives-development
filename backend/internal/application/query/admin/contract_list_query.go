// backend/internal/application/query/admin/contract_list_query.go
package query

import (
	"context"
	"errors"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	memberdom "narratives/internal/domain/member"
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

type ContractListQuery struct {
	companyRepo          contractListCompanyReader
	brandRepo            contractListBrandReader
	memberRepo           contractListMemberReader
	productBlueprintRepo contractListProductBlueprintReader
	tokenBlueprintRepo   contractListTokenBlueprintReader
	inventoryRepo        contractListInventoryReader
	listRepo             contractListReader
}

func NewContractListQuery(
	companyRepo contractListCompanyReader,
	brandRepo contractListBrandReader,
	memberRepo contractListMemberReader,
	productBlueprintRepo contractListProductBlueprintReader,
	tokenBlueprintRepo contractListTokenBlueprintReader,
	inventoryRepo contractListInventoryReader,
	listRepo contractListReader,
) *ContractListQuery {
	return &ContractListQuery{
		companyRepo:          companyRepo,
		brandRepo:            brandRepo,
		memberRepo:           memberRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		inventoryRepo:        inventoryRepo,
		listRepo:             listRepo,
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
	Prices             []ContractListPriceRow `json:"prices"`
	ProductName        string                 `json:"productName"`
	TokenName          string                 `json:"tokenName"`
	BrandID            string                 `json:"brandId"`
	BrandName          string                 `json:"brandName"`
	AssigneeID         string                 `json:"assigneeId"`
	AssigneeName       string                 `json:"assigneeName"`
	Status             string                 `json:"status"`
	CreatedAt          string                 `json:"createdAt"`
	UpdatedAt          string                 `json:"updatedAt"`
}

type ContractListPriceRow struct {
	ModelID string `json:"modelId"`
	Price   int    `json:"price"`
}

func (q *ContractListQuery) Get(
	ctx context.Context,
	companyID string,
	listID string,
) (ContractListDetailResult, error) {
	if q == nil || q.companyRepo == nil || q.brandRepo == nil || q.memberRepo == nil || q.productBlueprintRepo == nil || q.tokenBlueprintRepo == nil || q.inventoryRepo == nil || q.listRepo == nil {
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

	prices := make([]ContractListPriceRow, 0, len(item.Prices))
	for _, price := range item.Prices {
		prices = append(prices, ContractListPriceRow{
			ModelID: price.ModelID,
			Price:   price.Price,
		})
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
			Prices:             prices,
			ProductName:        productBlueprint.ProductName,
			TokenName:          tokenBlueprint.Name,
			BrandID:            productBlueprint.BrandID,
			BrandName:          q.resolveBrandName(ctx, productBlueprint.BrandID),
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
