// backend/internal/application/query/mall/history_query.go
package mall

import (
	"context"
	"errors"

	historydto "narratives/internal/application/query/mall/dto"
	mallshared "narratives/internal/application/query/mall/shared"
)

var (
	ErrHistoryQueryNotConfigured = errors.New("mall history query: not configured")
	ErrHistoryInventoryIDEmpty   = errors.New("mall history query: inventoryID is empty")
)

// HistoryInventoryBlueprintResolver resolves blueprint IDs from inventoryId.
//
// Concrete implementation should usually be inventory.RepositoryPort:
//
//	inventory.RepositoryPort.ResolveBlueprintIDsByInventoryID(ctx, inventoryID)
//
// Expected result:
// - productBlueprintID
// - tokenBlueprintID
type HistoryInventoryBlueprintResolver interface {
	ResolveBlueprintIDsByInventoryID(
		ctx context.Context,
		inventoryID string,
	) (productBlueprintID string, tokenBlueprintID string, err error)
}

type HistoryQuery struct {
	inventoryBlueprintResolver HistoryInventoryBlueprintResolver
	displayResolver            mallshared.MallDisplayResolver
}

func NewHistoryQuery(
	inventoryBlueprintResolver HistoryInventoryBlueprintResolver,
	displayResolver mallshared.MallDisplayResolver,
) *HistoryQuery {
	return &HistoryQuery{
		inventoryBlueprintResolver: inventoryBlueprintResolver,
		displayResolver:            displayResolver,
	}
}

// EnrichOrderPage enriches an already fetched order page for Wallet history.
//
// This query does not fetch orders by itself.
// Order listing remains the responsibility of OrderUsecase / OrderHandler.
//
// Enrichment flow per order item:
//
// list item:
//  1. inventoryId -> productBlueprintId / tokenBlueprintId
//  2. productBlueprintId -> productName / brandId
//  3. tokenBlueprintId -> tokenName / tokenIcon / brandId
//  4. brandId -> brandName / brandIcon
//
// resale item:
//  1. item.productBlueprintId / item.tokenBlueprintId / item.brandId を利用
//  2. productBlueprintId -> productName / brandId
//  3. tokenBlueprintId -> tokenName / tokenIcon / brandId
//  4. brandId -> brandName / brandIcon
func (q *HistoryQuery) EnrichOrderPage(
	ctx context.Context,
	in historydto.EnrichHistoryOrderPageInput,
) (historydto.HistoryOrderPage, error) {
	if q == nil ||
		q.inventoryBlueprintResolver == nil ||
		q.displayResolver == nil {
		return historydto.HistoryOrderPage{}, ErrHistoryQueryNotConfigured
	}

	out := historydto.HistoryOrderPage{
		Items:      cloneHistoryOrders(in.Items),
		TotalCount: in.TotalCount,
		TotalPages: in.TotalPages,
		Page:       in.Page,
		PerPage:    in.PerPage,
	}

	blueprintCache := make(map[string]historyBlueprintIDs)
	productBlueprintCache := make(map[string]mallshared.ProductBlueprintDisplay)
	tokenBlueprintCache := make(map[string]mallshared.TokenBlueprintDisplay)
	brandCache := make(map[string]mallshared.BrandDisplay)

	for orderIndex := range out.Items {
		for itemIndex := range out.Items[orderIndex].Items {
			item := &out.Items[orderIndex].Items[itemIndex]

			inventoryID := item.InventoryID

			blueprintIDs := historyBlueprintIDs{
				ProductBlueprintID: item.ProductBlueprintID,
				TokenBlueprintID:   item.TokenBlueprintID,
			}

			if inventoryID != "" {
				cached, ok := blueprintCache[inventoryID]
				if ok {
					blueprintIDs = mergeHistoryBlueprintIDs(blueprintIDs, cached)
				} else {
					productBlueprintID, tokenBlueprintID, err :=
						q.inventoryBlueprintResolver.ResolveBlueprintIDsByInventoryID(ctx, inventoryID)
					if err == nil {
						resolvedFromInventory := historyBlueprintIDs{
							ProductBlueprintID: productBlueprintID,
							TokenBlueprintID:   tokenBlueprintID,
						}

						blueprintCache[inventoryID] = resolvedFromInventory
						blueprintIDs = mergeHistoryBlueprintIDs(
							blueprintIDs,
							resolvedFromInventory,
						)
					}
				}
			}

			if blueprintIDs.ProductBlueprintID != "" {
				item.ProductBlueprintID = blueprintIDs.ProductBlueprintID
			}
			if blueprintIDs.TokenBlueprintID != "" {
				item.TokenBlueprintID = blueprintIDs.TokenBlueprintID
			}

			if blueprintIDs.ProductBlueprintID != "" {
				pbInfo, ok := productBlueprintCache[blueprintIDs.ProductBlueprintID]
				if !ok {
					pbInfo = q.resolveProductBlueprintInfo(
						ctx,
						blueprintIDs.ProductBlueprintID,
					)
					productBlueprintCache[blueprintIDs.ProductBlueprintID] = pbInfo
				}

				if item.ProductName == "" && pbInfo.ProductName != "" {
					item.ProductName = pbInfo.ProductName
				}
				if item.BrandID == "" && pbInfo.BrandID != "" {
					item.BrandID = pbInfo.BrandID
				}
			}

			if blueprintIDs.TokenBlueprintID != "" {
				tbInfo, ok := tokenBlueprintCache[blueprintIDs.TokenBlueprintID]
				if !ok {
					tbInfo = q.resolveTokenBlueprintInfo(
						ctx,
						blueprintIDs.TokenBlueprintID,
					)
					tokenBlueprintCache[blueprintIDs.TokenBlueprintID] = tbInfo
				}

				if item.TokenName == "" && tbInfo.TokenName != "" {
					item.TokenName = tbInfo.TokenName
				}
				if item.TokenIcon == "" && tbInfo.TokenIcon != "" {
					item.TokenIcon = tbInfo.TokenIcon
				}
				if item.BrandID == "" && tbInfo.BrandID != "" {
					item.BrandID = tbInfo.BrandID
				}
			}

			if item.BrandID != "" {
				brandInfo, ok := brandCache[item.BrandID]
				if !ok {
					brandInfo = q.resolveBrandInfo(ctx, item.BrandID)
					brandCache[item.BrandID] = brandInfo
				}

				if item.BrandName == "" && brandInfo.BrandName != "" {
					item.BrandName = brandInfo.BrandName
				}
				if item.BrandIcon == "" && brandInfo.BrandIcon != "" {
					item.BrandIcon = brandInfo.BrandIcon
				}
			}
		}
	}

	return out, nil
}

func (q *HistoryQuery) ResolveBlueprintIDsByInventoryID(
	ctx context.Context,
	inventoryID string,
) (productBlueprintID string, tokenBlueprintID string, err error) {
	if q == nil || q.inventoryBlueprintResolver == nil {
		return "", "", ErrHistoryQueryNotConfigured
	}

	if inventoryID == "" {
		return "", "", ErrHistoryInventoryIDEmpty
	}

	return q.inventoryBlueprintResolver.ResolveBlueprintIDsByInventoryID(
		ctx,
		inventoryID,
	)
}

func (q *HistoryQuery) ResolveProductBlueprintInfo(
	ctx context.Context,
	productBlueprintID string,
) (productName string, brandID string, err error) {
	if q == nil || q.displayResolver == nil {
		return "", "", ErrHistoryQueryNotConfigured
	}

	if productBlueprintID == "" {
		return "", "", nil
	}

	info, err := q.displayResolver.ResolveProductBlueprintInfo(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return "", "", err
	}

	return info.ProductName, info.BrandID, nil
}

func (q *HistoryQuery) ResolveTokenBlueprintInfo(
	ctx context.Context,
	tokenBlueprintID string,
) (tokenName string, tokenIcon string, brandID string, err error) {
	if q == nil || q.displayResolver == nil {
		return "", "", "", ErrHistoryQueryNotConfigured
	}

	if tokenBlueprintID == "" {
		return "", "", "", nil
	}

	info, err := q.displayResolver.ResolveTokenBlueprintInfo(
		ctx,
		tokenBlueprintID,
	)
	if err != nil {
		return "", "", "", err
	}

	return info.TokenName,
		info.TokenIcon,
		info.BrandID,
		nil
}

func (q *HistoryQuery) ResolveBrandInfo(
	ctx context.Context,
	brandID string,
) (brandName string, brandIcon string, err error) {
	if q == nil || q.displayResolver == nil {
		return "", "", ErrHistoryQueryNotConfigured
	}

	if brandID == "" {
		return "", "", nil
	}

	info, err := q.displayResolver.ResolveBrandInfo(
		ctx,
		brandID,
	)
	if err != nil {
		return "", "", err
	}

	return info.BrandName, info.BrandIcon, nil
}

func (q *HistoryQuery) resolveProductBlueprintInfo(
	ctx context.Context,
	productBlueprintID string,
) mallshared.ProductBlueprintDisplay {
	if productBlueprintID == "" ||
		q == nil ||
		q.displayResolver == nil {
		return mallshared.ProductBlueprintDisplay{}
	}

	info, err := q.displayResolver.ResolveProductBlueprintInfo(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return mallshared.ProductBlueprintDisplay{}
	}

	return info
}

func (q *HistoryQuery) resolveTokenBlueprintInfo(
	ctx context.Context,
	tokenBlueprintID string,
) mallshared.TokenBlueprintDisplay {
	if tokenBlueprintID == "" ||
		q == nil ||
		q.displayResolver == nil {
		return mallshared.TokenBlueprintDisplay{}
	}

	info, err := q.displayResolver.ResolveTokenBlueprintInfo(
		ctx,
		tokenBlueprintID,
	)
	if err != nil {
		return mallshared.TokenBlueprintDisplay{}
	}

	return info
}

func (q *HistoryQuery) resolveBrandInfo(
	ctx context.Context,
	brandID string,
) mallshared.BrandDisplay {
	if brandID == "" ||
		q == nil ||
		q.displayResolver == nil {
		return mallshared.BrandDisplay{}
	}

	info, err := q.displayResolver.ResolveBrandInfo(
		ctx,
		brandID,
	)
	if err != nil {
		return mallshared.BrandDisplay{}
	}

	return info
}

type historyBlueprintIDs struct {
	ProductBlueprintID string
	TokenBlueprintID   string
}

func cloneHistoryOrders(
	in []historydto.HistoryOrder,
) []historydto.HistoryOrder {
	out := make([]historydto.HistoryOrder, 0, len(in))

	for _, order := range in {
		next := order

		if len(order.Items) > 0 {
			next.Items = make(
				[]historydto.HistoryOrderItem,
				len(order.Items),
			)
			copy(next.Items, order.Items)
		} else {
			next.Items = []historydto.HistoryOrderItem{}
		}

		out = append(out, next)
	}

	return out
}

func mergeHistoryBlueprintIDs(
	base historyBlueprintIDs,
	next historyBlueprintIDs,
) historyBlueprintIDs {
	if base.ProductBlueprintID == "" {
		base.ProductBlueprintID = next.ProductBlueprintID
	}
	if base.TokenBlueprintID == "" {
		base.TokenBlueprintID = next.TokenBlueprintID
	}

	return base
}
