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
	ErrHistoryModelIDEmpty       = errors.New("mall history query: modelID is empty")
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
//  5. modelId -> size / color / modelNumber / volume
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
	modelCache := make(map[string]historydto.HistoryResolvedModel)

	for orderIndex := range out.Items {
		for itemIndex := range out.Items[orderIndex].Items {
			item := &out.Items[orderIndex].Items[itemIndex]

			inventoryID := item.InventoryID
			modelID := item.ModelID

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

			if modelID == "" {
				continue
			}

			resolved, ok := modelCache[modelID]
			if !ok {
				nextResolved, err := q.resolveHistoryModelByID(
					ctx,
					historydto.HistoryResolveModelInput{
						ItemType: item.ItemType,

						ModelID:     modelID,
						InventoryID: inventoryID,
						ListID:      item.ListID,

						ResaleID: item.ResaleID,

						ProductID:          item.ProductID,
						ProductBlueprintID: blueprintIDs.ProductBlueprintID,
						TokenBlueprintID:   blueprintIDs.TokenBlueprintID,
						BrandID:            item.BrandID,
					},
				)
				if err != nil {
					continue
				}

				resolved = nextResolved
				modelCache[modelID] = nextResolved
			}

			applyResolvedModelToItem(item, resolved)

			if blueprintIDs.ProductBlueprintID != "" {
				pbInfo := productBlueprintCache[blueprintIDs.ProductBlueprintID]

				if item.ProductName == "" {
					item.ProductName = pbInfo.ProductName
				}
				if item.BrandID == "" {
					item.BrandID = pbInfo.BrandID
				}
			}

			if blueprintIDs.TokenBlueprintID != "" {
				tbInfo := tokenBlueprintCache[blueprintIDs.TokenBlueprintID]

				if item.TokenName == "" {
					item.TokenName = tbInfo.TokenName
				}
				if item.TokenIcon == "" {
					item.TokenIcon = tbInfo.TokenIcon
				}
				if item.BrandID == "" {
					item.BrandID = tbInfo.BrandID
				}
			}

			if item.BrandID != "" {
				brandInfo, ok := brandCache[item.BrandID]
				if !ok {
					brandInfo = q.resolveBrandInfo(ctx, item.BrandID)
					brandCache[item.BrandID] = brandInfo
				}

				if item.BrandName == "" {
					item.BrandName = brandInfo.BrandName
				}
				if item.BrandIcon == "" {
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

func (q *HistoryQuery) ResolveModel(
	ctx context.Context,
	in historydto.HistoryResolveModelInput,
) (historydto.HistoryResolvedModel, error) {
	if q == nil || q.displayResolver == nil {
		return historydto.HistoryResolvedModel{}, ErrHistoryQueryNotConfigured
	}

	nextInput := historydto.HistoryResolveModelInput{
		ItemType: in.ItemType,

		ModelID:     in.ModelID,
		InventoryID: in.InventoryID,
		ListID:      in.ListID,

		ResaleID: in.ResaleID,

		ProductID:          in.ProductID,
		ProductBlueprintID: in.ProductBlueprintID,
		TokenBlueprintID:   in.TokenBlueprintID,
		BrandID:            in.BrandID,
	}

	if nextInput.ModelID == "" {
		return historydto.HistoryResolvedModel{}, ErrHistoryModelIDEmpty
	}

	if nextInput.InventoryID != "" &&
		(nextInput.ProductBlueprintID == "" || nextInput.TokenBlueprintID == "") &&
		q.inventoryBlueprintResolver != nil {
		productBlueprintID, tokenBlueprintID, err :=
			q.inventoryBlueprintResolver.ResolveBlueprintIDsByInventoryID(
				ctx,
				nextInput.InventoryID,
			)
		if err == nil {
			if nextInput.ProductBlueprintID == "" {
				nextInput.ProductBlueprintID = productBlueprintID
			}
			if nextInput.TokenBlueprintID == "" {
				nextInput.TokenBlueprintID = tokenBlueprintID
			}
		}
	}

	resolved, err := q.resolveHistoryModelByID(ctx, nextInput)
	if err != nil {
		return historydto.HistoryResolvedModel{}, err
	}

	if resolved.ItemType == "" {
		resolved.ItemType = nextInput.ItemType
	}
	if resolved.ListID == "" {
		resolved.ListID = nextInput.ListID
	}
	if resolved.ResaleID == "" {
		resolved.ResaleID = nextInput.ResaleID
	}
	if resolved.ProductID == "" {
		resolved.ProductID = nextInput.ProductID
	}
	if resolved.ProductBlueprintID == "" {
		resolved.ProductBlueprintID = nextInput.ProductBlueprintID
	}
	if resolved.TokenBlueprintID == "" {
		resolved.TokenBlueprintID = nextInput.TokenBlueprintID
	}
	if resolved.BrandID == "" {
		resolved.BrandID = nextInput.BrandID
	}

	if resolved.ProductBlueprintID != "" {
		pbInfo := q.resolveProductBlueprintInfo(
			ctx,
			resolved.ProductBlueprintID,
		)

		if resolved.ProductName == "" {
			resolved.ProductName = pbInfo.ProductName
		}
		if resolved.BrandID == "" {
			resolved.BrandID = pbInfo.BrandID
		}
	}

	if resolved.TokenBlueprintID != "" {
		tbInfo := q.resolveTokenBlueprintInfo(
			ctx,
			resolved.TokenBlueprintID,
		)

		if resolved.TokenName == "" {
			resolved.TokenName = tbInfo.TokenName
		}
		if resolved.TokenIcon == "" {
			resolved.TokenIcon = tbInfo.TokenIcon
		}
		if resolved.BrandID == "" {
			resolved.BrandID = tbInfo.BrandID
		}
	}

	if resolved.BrandID != "" {
		brandInfo := q.resolveBrandInfo(ctx, resolved.BrandID)

		if resolved.BrandName == "" {
			resolved.BrandName = brandInfo.BrandName
		}
		if resolved.BrandIcon == "" {
			resolved.BrandIcon = brandInfo.BrandIcon
		}
	}

	return resolved, nil
}

func (q *HistoryQuery) resolveHistoryModelByID(
	ctx context.Context,
	in historydto.HistoryResolveModelInput,
) (historydto.HistoryResolvedModel, error) {
	if q == nil || q.displayResolver == nil {
		return historydto.HistoryResolvedModel{}, ErrHistoryQueryNotConfigured
	}

	if in.ModelID == "" {
		return historydto.HistoryResolvedModel{}, ErrHistoryModelIDEmpty
	}

	model, err := q.displayResolver.ResolveModelByModelID(
		ctx,
		in.ModelID,
	)
	if err != nil {
		return historydto.HistoryResolvedModel{}, err
	}

	return historyResolvedModelFromDisplayResolver(in, model), nil
}

func historyResolvedModelFromDisplayResolver(
	in historydto.HistoryResolveModelInput,
	model mallshared.ModelDisplay,
) historydto.HistoryResolvedModel {
	out := historydto.HistoryResolvedModel{
		ItemType: in.ItemType,

		ModelID:     in.ModelID,
		InventoryID: in.InventoryID,
		ListID:      in.ListID,

		ResaleID: in.ResaleID,

		ProductID:          in.ProductID,
		ProductBlueprintID: in.ProductBlueprintID,
		TokenBlueprintID:   in.TokenBlueprintID,
		BrandID:            in.BrandID,

		Kind:         model.Kind,
		ModelNumber:  model.ModelNumber,
		Size:         model.Size,
		Measurements: cloneMeasurements(model.Measurements),
		VolumeValue:  model.VolumeValue,
		VolumeUnit:   model.VolumeUnit,
	}

	if model.ModelID != "" {
		out.ModelID = model.ModelID
	}

	if out.ProductBlueprintID == "" && model.ProductBlueprintID != "" {
		out.ProductBlueprintID = model.ProductBlueprintID
	}

	if model.ColorName != "" || model.ColorRGB != 0 {
		rgb := model.ColorRGB

		out.Color = &historydto.HistoryColor{
			Name: model.ColorName,
			RGB:  &rgb,
		}
	}

	return out
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

func applyResolvedModelToItem(
	item *historydto.HistoryOrderItem,
	resolved historydto.HistoryResolvedModel,
) {
	if item == nil {
		return
	}

	if resolved.ItemType != "" {
		item.ItemType = resolved.ItemType
	}

	if resolved.ModelID != "" {
		item.ModelID = resolved.ModelID
	}

	if resolved.InventoryID != "" {
		item.InventoryID = resolved.InventoryID
	}

	if resolved.ListID != "" {
		item.ListID = resolved.ListID
	}

	if resolved.ResaleID != "" {
		item.ResaleID = resolved.ResaleID
	}

	if resolved.ProductID != "" {
		item.ProductID = resolved.ProductID
	}

	if resolved.ProductBlueprintID != "" {
		item.ProductBlueprintID = resolved.ProductBlueprintID
	}

	if resolved.TokenBlueprintID != "" {
		item.TokenBlueprintID = resolved.TokenBlueprintID
	}

	if resolved.ProductName != "" {
		item.ProductName = resolved.ProductName
	}

	if resolved.BrandID != "" {
		item.BrandID = resolved.BrandID
	}

	if resolved.Kind != "" {
		item.Kind = resolved.Kind
	}

	if resolved.ModelNumber != "" {
		item.ModelNumber = resolved.ModelNumber
	}

	if resolved.Size != "" {
		item.Size = resolved.Size
	}

	if resolved.Color != nil {
		item.Color = resolved.Color
	}

	if len(resolved.Measurements) > 0 {
		item.Measurements = cloneMeasurements(
			resolved.Measurements,
		)
	}

	if resolved.VolumeValue != nil {
		item.VolumeValue = resolved.VolumeValue
	}

	if resolved.VolumeUnit != "" {
		item.VolumeUnit = resolved.VolumeUnit
	}

	if resolved.TokenName != "" {
		item.TokenName = resolved.TokenName
	}

	if resolved.TokenIcon != "" {
		item.TokenIcon = resolved.TokenIcon
	}

	if resolved.BrandName != "" {
		item.BrandName = resolved.BrandName
	}

	if resolved.BrandIcon != "" {
		item.BrandIcon = resolved.BrandIcon
	}
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
