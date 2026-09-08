// backend/internal/application/query/admin/contract_detail_query.go
package query

import (
	"context"
	"errors"
	"sort"
	"time"

	branddom "narratives/internal/domain/brand"
	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	memberdom "narratives/internal/domain/member"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

const contractDetailPageSize = 200

var ErrContractDetailQueryNotConfigured = errors.New(
	"contract detail query is not configured",
)

// ============================================================
// Read-only ports
// ============================================================

type contractDetailCompanyReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (companydom.Company, error)
}

type contractDetailBrandReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (branddom.Brand, error)
}

type contractDetailMemberReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (memberdom.Record, error)

	GetByUID(
		ctx context.Context,
		uid string,
	) (memberdom.Record, error)
}

type contractDetailProductBlueprintReader interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]productblueprintdom.ProductBlueprint, error)
}

type contractDetailTokenBlueprintReader interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
		page common.Page,
	) (common.PageResult[tokenblueprintdom.TokenBlueprint], error)
}

type contractDetailInventoryReader interface {
	ListByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
	) ([]inventorydom.Mint, error)
}

type contractDetailListReader interface {
	ListByInventoryID(
		ctx context.Context,
		inventoryID string,
	) ([]listdom.List, error)
}

// ============================================================
// Query
// ============================================================

type ContractDetailQuery struct {
	companyRepo          contractDetailCompanyReader
	brandRepo            contractDetailBrandReader
	memberRepo           contractDetailMemberReader
	productBlueprintRepo contractDetailProductBlueprintReader
	tokenBlueprintRepo   contractDetailTokenBlueprintReader
	inventoryRepo        contractDetailInventoryReader
	listRepo             contractDetailListReader
}

func NewContractDetailQuery(
	companyRepo contractDetailCompanyReader,
	brandRepo contractDetailBrandReader,
	memberRepo contractDetailMemberReader,
	productBlueprintRepo contractDetailProductBlueprintReader,
	tokenBlueprintRepo contractDetailTokenBlueprintReader,
	inventoryRepo contractDetailInventoryReader,
	listRepo contractDetailListReader,
) *ContractDetailQuery {
	return &ContractDetailQuery{
		companyRepo:          companyRepo,
		brandRepo:            brandRepo,
		memberRepo:           memberRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		inventoryRepo:        inventoryRepo,
		listRepo:             listRepo,
	}
}

// ============================================================
// Result
// ============================================================

type ContractDetailResult struct {
	Company           ContractCompanyRow            `json:"company"`
	Lists             []ContractListRow             `json:"lists"`
	TokenBlueprints   []ContractTokenBlueprintRow   `json:"tokenBlueprints"`
	ProductBlueprints []ContractProductBlueprintRow `json:"productBlueprints"`
}

type ContractCompanyRow struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	RepresentativeName string `json:"representativeName"`
	IsActive           bool   `json:"isActive"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type ContractListRow struct {
	ID           string `json:"id"`
	ReadableID   string `json:"readableId"`
	InventoryID  string `json:"inventoryId"`
	Title        string `json:"title"`
	ProductName  string `json:"productName"`
	TokenName    string `json:"tokenName"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type ContractTokenBlueprintRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	Minted       bool   `json:"minted"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type ContractProductBlueprintRow struct {
	ID           string `json:"id"`
	ProductName  string `json:"productName"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	Printed      bool   `json:"printed"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ============================================================
// Get
// ============================================================

func (q *ContractDetailQuery) Get(
	ctx context.Context,
	companyID string,
) (ContractDetailResult, error) {
	if q == nil ||
		q.companyRepo == nil ||
		q.brandRepo == nil ||
		q.memberRepo == nil ||
		q.productBlueprintRepo == nil ||
		q.tokenBlueprintRepo == nil ||
		q.inventoryRepo == nil ||
		q.listRepo == nil {
		return ContractDetailResult{}, ErrContractDetailQueryNotConfigured
	}

	if companyID == "" {
		return ContractDetailResult{}, companydom.ErrInvalidID
	}

	company, err := q.companyRepo.GetByID(
		ctx,
		companyID,
	)
	if err != nil {
		return ContractDetailResult{}, err
	}

	productBlueprints, err := q.productBlueprintRepo.ListByCompanyID(
		ctx,
		companyID,
	)
	if err != nil {
		return ContractDetailResult{}, err
	}

	tokenBlueprints, err := q.listTokenBlueprintsByCompanyID(
		ctx,
		companyID,
	)
	if err != nil {
		return ContractDetailResult{}, err
	}

	brandNameCache := make(map[string]string)
	memberNameCache := make(map[string]string)

	productRows := q.buildProductBlueprintRows(
		ctx,
		companyID,
		productBlueprints,
		brandNameCache,
		memberNameCache,
	)

	tokenRows, tokenByID := q.buildTokenBlueprintRows(
		ctx,
		companyID,
		tokenBlueprints,
		brandNameCache,
		memberNameCache,
	)

	listRows, err := q.buildListRows(
		ctx,
		companyID,
		productBlueprints,
		tokenByID,
		brandNameCache,
		memberNameCache,
	)
	if err != nil {
		return ContractDetailResult{}, err
	}

	sort.SliceStable(
		productRows,
		func(i, j int) bool {
			return productRows[i].CreatedAt > productRows[j].CreatedAt
		},
	)
	sort.SliceStable(
		tokenRows,
		func(i, j int) bool {
			return tokenRows[i].CreatedAt > tokenRows[j].CreatedAt
		},
	)
	sort.SliceStable(
		listRows,
		func(i, j int) bool {
			return listRows[i].CreatedAt > listRows[j].CreatedAt
		},
	)

	representativeName := q.resolveMemberName(
		ctx,
		company.Admin,
		memberNameCache,
	)
	if representativeName == "" || representativeName == company.Admin {
		representativeName = "-"
	}

	return ContractDetailResult{
		Company: ContractCompanyRow{
			ID:                 company.ID,
			Name:               company.Name,
			RepresentativeName: representativeName,
			IsActive:           company.IsActive,
			CreatedAt:          formatContractDetailTime(company.CreatedAt),
			UpdatedAt:          formatContractDetailTime(company.UpdatedAt),
		},
		Lists:             listRows,
		TokenBlueprints:   tokenRows,
		ProductBlueprints: productRows,
	}, nil
}

// ============================================================
// ProductBlueprint
// ============================================================

func (q *ContractDetailQuery) buildProductBlueprintRows(
	ctx context.Context,
	companyID string,
	productBlueprints []productblueprintdom.ProductBlueprint,
	brandNameCache map[string]string,
	memberNameCache map[string]string,
) []ContractProductBlueprintRow {
	rows := make(
		[]ContractProductBlueprintRow,
		0,
		len(productBlueprints),
	)

	for _, productBlueprint := range productBlueprints {
		if productBlueprint.ID == "" ||
			productBlueprint.CompanyID != companyID {
			continue
		}

		rows = append(
			rows,
			ContractProductBlueprintRow{
				ID:          productBlueprint.ID,
				ProductName: productBlueprint.ProductName,
				BrandName: q.resolveBrandName(
					ctx,
					productBlueprint.BrandID,
					brandNameCache,
				),
				AssigneeName: q.resolveMemberName(
					ctx,
					productBlueprint.AssigneeID,
					memberNameCache,
				),
				Printed: productBlueprint.Printed,
				CreatedAt: formatContractDetailTime(
					productBlueprint.CreatedAt,
				),
				UpdatedAt: formatContractDetailTime(
					productBlueprint.UpdatedAt,
				),
			},
		)
	}

	return rows
}

// ============================================================
// TokenBlueprint
// ============================================================

func (q *ContractDetailQuery) listTokenBlueprintsByCompanyID(
	ctx context.Context,
	companyID string,
) ([]tokenblueprintdom.TokenBlueprint, error) {
	items := make(
		[]tokenblueprintdom.TokenBlueprint,
		0,
	)

	for pageNumber := 1; ; pageNumber++ {
		result, err := q.tokenBlueprintRepo.ListByCompanyID(
			ctx,
			companyID,
			common.Page{
				Number:  pageNumber,
				PerPage: contractDetailPageSize,
			},
		)
		if err != nil {
			return nil, err
		}

		items = append(
			items,
			result.Items...,
		)

		if len(result.Items) == 0 ||
			result.TotalPages <= pageNumber {
			break
		}
	}

	return items, nil
}

func (q *ContractDetailQuery) buildTokenBlueprintRows(
	ctx context.Context,
	companyID string,
	tokenBlueprints []tokenblueprintdom.TokenBlueprint,
	brandNameCache map[string]string,
	memberNameCache map[string]string,
) (
	[]ContractTokenBlueprintRow,
	map[string]tokenblueprintdom.TokenBlueprint,
) {
	rows := make(
		[]ContractTokenBlueprintRow,
		0,
		len(tokenBlueprints),
	)
	tokenByID := make(
		map[string]tokenblueprintdom.TokenBlueprint,
		len(tokenBlueprints),
	)

	for _, tokenBlueprint := range tokenBlueprints {
		if tokenBlueprint.ID == "" ||
			tokenBlueprint.CompanyID != companyID {
			continue
		}

		tokenByID[tokenBlueprint.ID] = tokenBlueprint

		rows = append(
			rows,
			ContractTokenBlueprintRow{
				ID:     tokenBlueprint.ID,
				Name:   tokenBlueprint.Name,
				Symbol: tokenBlueprint.Symbol,
				BrandName: q.resolveBrandName(
					ctx,
					tokenBlueprint.BrandID,
					brandNameCache,
				),
				AssigneeName: q.resolveMemberName(
					ctx,
					tokenBlueprint.AssigneeID,
					memberNameCache,
				),
				Minted: tokenBlueprint.Minted,
				CreatedAt: formatContractDetailTime(
					tokenBlueprint.CreatedAt,
				),
				UpdatedAt: formatContractDetailTime(
					tokenBlueprint.UpdatedAt,
				),
			},
		)
	}

	return rows, tokenByID
}

// ============================================================
// List
// ============================================================

func (q *ContractDetailQuery) buildListRows(
	ctx context.Context,
	companyID string,
	productBlueprints []productblueprintdom.ProductBlueprint,
	tokenByID map[string]tokenblueprintdom.TokenBlueprint,
	brandNameCache map[string]string,
	memberNameCache map[string]string,
) ([]ContractListRow, error) {
	rows := make(
		[]ContractListRow,
		0,
	)

	seenInventoryID := make(
		map[string]struct{},
	)
	seenListID := make(
		map[string]struct{},
	)

	for _, productBlueprint := range productBlueprints {
		if productBlueprint.ID == "" ||
			productBlueprint.CompanyID != companyID {
			continue
		}

		inventories, err := q.inventoryRepo.ListByProductBlueprintID(
			ctx,
			productBlueprint.ID,
		)
		if err != nil {
			return nil, err
		}

		for _, inventory := range inventories {
			if inventory.ID == "" ||
				inventory.ProductBlueprintID != productBlueprint.ID {
				continue
			}

			tokenBlueprint, ok := tokenByID[inventory.TokenBlueprintID]
			if !ok {
				continue
			}

			if _, exists := seenInventoryID[inventory.ID]; exists {
				continue
			}
			seenInventoryID[inventory.ID] = struct{}{}

			lists, err := q.listRepo.ListByInventoryID(
				ctx,
				inventory.ID,
			)
			if err != nil {
				return nil, err
			}

			for _, item := range lists {
				if item.ID == "" ||
					item.InventoryID != inventory.ID {
					continue
				}

				if _, exists := seenListID[item.ID]; exists {
					continue
				}
				seenListID[item.ID] = struct{}{}

				updatedAt := ""
				if item.UpdatedAt != nil {
					updatedAt = formatContractDetailTime(
						*item.UpdatedAt,
					)
				}

				rows = append(
					rows,
					ContractListRow{
						ID:          item.ID,
						ReadableID:  item.ReadableID,
						InventoryID: item.InventoryID,
						Title:       item.Title,
						ProductName: productBlueprint.ProductName,
						TokenName:   tokenBlueprint.Name,
						BrandName: q.resolveBrandName(
							ctx,
							productBlueprint.BrandID,
							brandNameCache,
						),
						AssigneeName: q.resolveMemberName(
							ctx,
							item.AssigneeID,
							memberNameCache,
						),
						Status: string(item.Status),
						CreatedAt: formatContractDetailTime(
							item.CreatedAt,
						),
						UpdatedAt: updatedAt,
					},
				)
			}
		}
	}

	return rows, nil
}

// ============================================================
// Name resolvers
// ============================================================

func (q *ContractDetailQuery) resolveBrandName(
	ctx context.Context,
	brandID string,
	cache map[string]string,
) string {
	if brandID == "" {
		return ""
	}

	if cached, ok := cache[brandID]; ok {
		return cached
	}

	name := brandID

	brand, err := q.brandRepo.GetByID(
		ctx,
		brandID,
	)
	if err == nil && brand.Name != "" {
		name = brand.Name
	}

	cache[brandID] = name
	return name
}

func (q *ContractDetailQuery) resolveMemberName(
	ctx context.Context,
	memberID string,
	cache map[string]string,
) string {
	if memberID == "" {
		return ""
	}

	if cached, ok := cache[memberID]; ok {
		return cached
	}

	name := memberID

	if record, err := q.memberRepo.GetByID(
		ctx,
		memberID,
	); err == nil {
		if resolved := memberdom.FormatLastFirst(
			record.Member.LastName,
			record.Member.FirstName,
		); resolved != "" {
			name = resolved
		}
	} else if record, uidErr := q.memberRepo.GetByUID(
		ctx,
		memberID,
	); uidErr == nil {
		if resolved := memberdom.FormatLastFirst(
			record.Member.LastName,
			record.Member.FirstName,
		); resolved != "" {
			name = resolved
		}
	}

	cache[memberID] = name
	return name
}

// ============================================================
// Helpers
// ============================================================

func formatContractDetailTime(
	value time.Time,
) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(
		time.RFC3339Nano,
	)
}
