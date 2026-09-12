// backend\internal\application\query\admin\brand.go
package query

import (
	"context"

	branddom "narratives/internal/domain/brand"
)

func (q *ContractDetailQuery) listBrandsByCompanyID(
	ctx context.Context,
	companyID string,
) ([]branddom.Brand, error) {
	items := make([]branddom.Brand, 0)

	for pageNumber := 1; ; pageNumber++ {
		result, err := q.brandRepo.ListByCompanyID(
			ctx,
			companyID,
			branddom.Page{
				Number:  pageNumber,
				PerPage: contractDetailPageSize,
			},
		)
		if err != nil {
			return nil, err
		}

		items = append(items, result.Items...)

		if len(result.Items) == 0 ||
			result.TotalPages <= pageNumber {
			break
		}
	}

	return items, nil
}

func (q *ContractDetailQuery) buildBrandRows(
	ctx context.Context,
	companyID string,
	brands []branddom.Brand,
	brandNameCache map[string]string,
	memberNameCache map[string]string,
) []ContractBrandRow {
	rows := make(
		[]ContractBrandRow,
		0,
		len(brands),
	)

	for _, brand := range brands {
		if brand.ID == "" ||
			brand.CompanyID != companyID {
			continue
		}

		brandNameCache[brand.ID] = brand.Name

		managerName := ""
		if brand.ManagerID != nil {
			managerName = q.resolveMemberName(
				ctx,
				*brand.ManagerID,
				memberNameCache,
			)
		}

		rows = append(
			rows,
			ContractBrandRow{
				ID:                   brand.ID,
				Name:                 brand.Name,
				ManagerName:          managerName,
				WebsiteURL:           brand.URL,
				BrandIcon:            brand.BrandIcon,
				BrandBackgroundImage: brand.BrandBackgroundImage,
				IsActive:             brand.IsActive,
				CreatedAt: formatContractDetailTime(
					brand.CreatedAt,
				),
				UpdatedAt: formatOptionalContractDetailTime(
					brand.UpdatedAt,
				),
			},
		)
	}

	return rows
}
