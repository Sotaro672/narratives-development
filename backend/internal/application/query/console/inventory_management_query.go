// backend/internal/application/query/console/inventory_management_query.go
package query

import (
	"context"
	"errors"
	"sort"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
)

type InventoryManagementQuery struct {
	invRepo      inventoryReader
	pbRepo       inventoryProductBlueprintReader
	nameResolver *resolver.NameResolver
}

func NewInventoryManagementQuery(
	invRepo inventoryReader,
	pbRepo inventoryProductBlueprintReader,
	nameResolver *resolver.NameResolver,
) *InventoryManagementQuery {
	return &InventoryManagementQuery{
		invRepo:      invRepo,
		pbRepo:       pbRepo,
		nameResolver: nameResolver,
	}
}

// ============================================================
// currentMember.companyId -> productBlueprints -> inventories list
// ============================================================

func (q *InventoryManagementQuery) ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error) {
	if q == nil || q.invRepo == nil || q.pbRepo == nil {
		return nil, errors.New("inventory management query repositories are not configured")
	}

	companyID := usecase.CompanyIDFromContext(ctx)
	if companyID == "" {
		return nil, errors.New("companyId is missing in context")
	}

	productBlueprints, err := q.pbRepo.ListByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if len(productBlueprints) == 0 {
		return []querydto.InventoryManagementRowDTO{}, nil
	}

	type key struct {
		pbID string
		tbID string
	}

	type agg struct {
		available int
		reserved  int
	}

	group := map[key]agg{}

	productNameCache := map[string]string{}
	tokenNameCache := map[string]string{}

	for _, pb := range productBlueprints {
		pbID := pb.ID
		if pbID == "" {
			continue
		}

		if _, ok := productNameCache[pbID]; !ok {
			name := pb.ProductName
			if name == "" {
				name = pbID
			}
			productNameCache[pbID] = name
		}

		invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err != nil {
			return nil, err
		}
		if len(invs) == 0 {
			continue
		}

		for _, inv := range invs {
			tbID := inv.TokenBlueprintID

			if _, ok := tokenNameCache[tbID]; !ok {
				name := ""
				if q.nameResolver != nil {
					name = q.nameResolver.ResolveTokenName(ctx, tbID)
				}
				if name == "" {
					name = tbID
				}
				if name == "" {
					name = "-"
				}
				tokenNameCache[tbID] = name
			}

			k := key{
				pbID: pbID,
				tbID: tbID,
			}

			if len(inv.Stock) == 0 {
				if _, ok := group[k]; !ok {
					group[k] = agg{
						available: 0,
						reserved:  0,
					}
				}
				continue
			}

			for modelID, ms := range inv.Stock {
				if modelID == "" {
					continue
				}

				reserved := ms.ReservedCount
				available := ms.Accumulation - reserved
				if available < 0 {
					available = 0
				}

				a := group[k]
				a.available += available
				a.reserved += reserved
				group[k] = a
			}
		}
	}

	rows := make([]querydto.InventoryManagementRowDTO, 0, len(group))
	for k, a := range group {
		rows = append(rows, querydto.InventoryManagementRowDTO{
			ProductBlueprintID: k.pbID,
			ProductName:        productNameCache[k.pbID],
			TokenBlueprintID:   k.tbID,
			TokenName:          tokenNameCache[k.tbID],
			AvailableStock:     a.available,
			ReservedCount:      a.reserved,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ProductName != rows[j].ProductName {
			return rows[i].ProductName < rows[j].ProductName
		}
		if rows[i].TokenName != rows[j].TokenName {
			return rows[i].TokenName < rows[j].TokenName
		}
		if rows[i].ProductBlueprintID != rows[j].ProductBlueprintID {
			return rows[i].ProductBlueprintID < rows[j].ProductBlueprintID
		}
		if rows[i].TokenBlueprintID != rows[j].TokenBlueprintID {
			return rows[i].TokenBlueprintID < rows[j].TokenBlueprintID
		}
		return rows[i].AvailableStock < rows[j].AvailableStock
	})

	return rows, nil
}
