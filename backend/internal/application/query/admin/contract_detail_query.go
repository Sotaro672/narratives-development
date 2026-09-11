// backend/internal/application/query/admin/contract_detail_query.go
package query

import (
	"context"
	"errors"
	"sort"

	companydom "narratives/internal/domain/company"
)

const contractDetailPageSize = 200

var ErrContractDetailQueryNotConfigured = errors.New(
	"contract detail query is not configured",
)

type ContractDetailQuery struct {
	companyRepo                contractDetailCompanyReader
	brandRepo                  contractDetailBrandReader
	announcementRepo           contractDetailAnnouncementReader
	memberRepo                 contractDetailMemberReader
	productBlueprintRepo       contractDetailProductBlueprintReader
	productBlueprintReviewRepo contractDetailProductBlueprintReviewReader
	tokenBlueprintRepo         contractDetailTokenBlueprintReader
	inventoryRepo              contractDetailInventoryReader
	listRepo                   contractDetailListReader
	reportCaseRepo             contractDetailReportCaseReader
}

func NewContractDetailQuery(
	companyRepo contractDetailCompanyReader,
	brandRepo contractDetailBrandReader,
	announcementRepo contractDetailAnnouncementReader,
	memberRepo contractDetailMemberReader,
	productBlueprintRepo contractDetailProductBlueprintReader,
	productBlueprintReviewRepo contractDetailProductBlueprintReviewReader,
	tokenBlueprintRepo contractDetailTokenBlueprintReader,
	inventoryRepo contractDetailInventoryReader,
	listRepo contractDetailListReader,
	reportCaseRepo contractDetailReportCaseReader,
) *ContractDetailQuery {
	return &ContractDetailQuery{
		companyRepo:                companyRepo,
		brandRepo:                  brandRepo,
		announcementRepo:           announcementRepo,
		memberRepo:                 memberRepo,
		productBlueprintRepo:       productBlueprintRepo,
		productBlueprintReviewRepo: productBlueprintReviewRepo,
		tokenBlueprintRepo:         tokenBlueprintRepo,
		inventoryRepo:              inventoryRepo,
		listRepo:                   listRepo,
		reportCaseRepo:             reportCaseRepo,
	}
}

func (q *ContractDetailQuery) Get(
	ctx context.Context,
	companyID string,
) (ContractDetailResult, error) {
	if q == nil ||
		q.companyRepo == nil ||
		q.brandRepo == nil ||
		q.announcementRepo == nil ||
		q.memberRepo == nil ||
		q.productBlueprintRepo == nil ||
		q.productBlueprintReviewRepo == nil ||
		q.tokenBlueprintRepo == nil ||
		q.inventoryRepo == nil ||
		q.listRepo == nil ||
		q.reportCaseRepo == nil {
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

	brands, err := q.listBrandsByCompanyID(
		ctx,
		companyID,
	)
	if err != nil {
		return ContractDetailResult{}, err
	}

	brandNameCache := make(map[string]string)
	memberNameCache := make(map[string]string)

	brandRows := q.buildBrandRows(
		ctx,
		companyID,
		brands,
		brandNameCache,
		memberNameCache,
	)

	productRows, err := q.buildProductBlueprintRows(
		ctx,
		companyID,
		productBlueprints,
		brandNameCache,
		memberNameCache,
	)
	if err != nil {
		return ContractDetailResult{}, err
	}

	tokenRows, tokenByID, err := q.buildTokenBlueprintRows(
		ctx,
		companyID,
		tokenBlueprints,
		brandNameCache,
		memberNameCache,
	)
	if err != nil {
		return ContractDetailResult{}, err
	}

	announcementRows, err := q.buildAnnouncementRows(
		ctx,
		companyID,
		tokenBlueprints,
	)
	if err != nil {
		return ContractDetailResult{}, err
	}

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
		brandRows,
		func(i, j int) bool {
			return brandRows[i].CreatedAt > brandRows[j].CreatedAt
		},
	)
	sort.SliceStable(
		announcementRows,
		func(i, j int) bool {
			return announcementRows[i].CreatedAt > announcementRows[j].CreatedAt
		},
	)
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
		Brands:            brandRows,
		Announcements:     announcementRows,
		Lists:             listRows,
		TokenBlueprints:   tokenRows,
		ProductBlueprints: productRows,
	}, nil
}
