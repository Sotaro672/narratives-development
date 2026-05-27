package mall

import (
	"context"
	"errors"
	"strings"

	historydto "narratives/internal/application/query/mall/dto"

	branddom "narratives/internal/domain/brand"
	pbdom "narratives/internal/domain/productBlueprint"
	tokenbpdom "narratives/internal/domain/tokenBlueprint"
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

// HistoryProductBlueprintResolver resolves product display base data
// from productBlueprintId.
//
// Concrete implementation can be productBlueprint.Repository because it has:
//
//	GetByID(ctx, productBlueprintID)
//
// ProductBlueprint provides:
// - ProductName
// - BrandID
type HistoryProductBlueprintResolver interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// HistoryTokenBlueprintResolver resolves token display base data
// from tokenBlueprintId.
//
// Concrete implementation can be tokenBlueprint.RepositoryPort because it has:
//
//	GetPatchByID(ctx, tokenBlueprintID)
//
// Patch provides:
// - TokenName
// - IconURL
// - BrandID
type HistoryTokenBlueprintResolver interface {
	GetPatchByID(ctx context.Context, id string) (tokenbpdom.Patch, error)
}

// HistoryBrandResolver resolves brand display data from brandId.
//
// Concrete implementation can be brand.Service because it has:
//
//	GetNameIconByID(ctx, brandID)
//
// NameIcon provides:
// - Name
// - BrandIcon
type HistoryBrandResolver interface {
	GetNameIconByID(ctx context.Context, brandID string) (branddom.NameIcon, error)
}

// HistoryModelResolver resolves model variation display information from modelId.
//
// Expected responsibility:
// modelId
// -> ModelVariation
// -> size / color / modelNumber / measurements
type HistoryModelResolver interface {
	ResolveHistoryModelByID(
		ctx context.Context,
		in historydto.HistoryResolveModelInput,
	) (historydto.HistoryResolvedModel, error)
}

type HistoryQuery struct {
	inventoryBlueprintResolver HistoryInventoryBlueprintResolver
	productBlueprintResolver   HistoryProductBlueprintResolver
	tokenBlueprintResolver     HistoryTokenBlueprintResolver
	brandResolver              HistoryBrandResolver
	modelResolver              HistoryModelResolver
}

func NewHistoryQuery(
	inventoryBlueprintResolver HistoryInventoryBlueprintResolver,
	productBlueprintResolver HistoryProductBlueprintResolver,
	tokenBlueprintResolver HistoryTokenBlueprintResolver,
	brandResolver HistoryBrandResolver,
	modelResolver HistoryModelResolver,
) *HistoryQuery {
	return &HistoryQuery{
		inventoryBlueprintResolver: inventoryBlueprintResolver,
		productBlueprintResolver:   productBlueprintResolver,
		tokenBlueprintResolver:     tokenBlueprintResolver,
		brandResolver:              brandResolver,
		modelResolver:              modelResolver,
	}
}

// EnrichOrderPage enriches an already fetched order page for Wallet history.
//
// This query does not fetch orders by itself.
// Order listing remains the responsibility of OrderUsecase / OrderHandler.
//
// Enrichment flow per order item:
//  1. inventoryId -> productBlueprintId / tokenBlueprintId
//  2. productBlueprintId -> productName / brandId
//  3. tokenBlueprintId -> tokenName / tokenIcon / brandId
//  4. brandId -> brandName / brandIcon
//  5. modelId -> size / color / modelNumber / measurements
func (q *HistoryQuery) EnrichOrderPage(
	ctx context.Context,
	in historydto.EnrichHistoryOrderPageInput,
) (historydto.HistoryOrderPage, error) {
	if q == nil ||
		q.inventoryBlueprintResolver == nil ||
		q.productBlueprintResolver == nil ||
		q.tokenBlueprintResolver == nil ||
		q.brandResolver == nil ||
		q.modelResolver == nil {
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
	productBlueprintCache := make(map[string]historyProductBlueprintInfo)
	tokenBlueprintCache := make(map[string]historyTokenBlueprintInfo)
	brandCache := make(map[string]historyBrandInfo)
	modelCache := make(map[string]historydto.HistoryResolvedModel)

	for orderIndex := range out.Items {
		for itemIndex := range out.Items[orderIndex].Items {
			item := &out.Items[orderIndex].Items[itemIndex]

			inventoryID := strings.TrimSpace(item.InventoryID)
			modelID := strings.TrimSpace(item.ModelID)

			if inventoryID == "" && modelID == "" {
				continue
			}

			var blueprintIDs historyBlueprintIDs
			if inventoryID != "" {
				cached, ok := blueprintCache[inventoryID]
				if ok {
					blueprintIDs = cached
				} else {
					productBlueprintID, tokenBlueprintID, err :=
						q.inventoryBlueprintResolver.ResolveBlueprintIDsByInventoryID(ctx, inventoryID)
					if err == nil {
						blueprintIDs = historyBlueprintIDs{
							ProductBlueprintID: strings.TrimSpace(productBlueprintID),
							TokenBlueprintID:   strings.TrimSpace(tokenBlueprintID),
						}
						blueprintCache[inventoryID] = blueprintIDs
					}
				}

				if blueprintIDs.ProductBlueprintID != "" {
					item.ProductBlueprintID = blueprintIDs.ProductBlueprintID
				}
				if blueprintIDs.TokenBlueprintID != "" {
					item.TokenBlueprintID = blueprintIDs.TokenBlueprintID
				}
			}

			if blueprintIDs.ProductBlueprintID != "" {
				pbInfo, ok := productBlueprintCache[blueprintIDs.ProductBlueprintID]
				if !ok {
					pbInfo = q.resolveProductBlueprintInfo(ctx, blueprintIDs.ProductBlueprintID)
					productBlueprintCache[blueprintIDs.ProductBlueprintID] = pbInfo
				}

				if pbInfo.ProductName != "" {
					item.ProductName = pbInfo.ProductName
				}
				if pbInfo.BrandID != "" {
					item.BrandID = pbInfo.BrandID
				}
			}

			if blueprintIDs.TokenBlueprintID != "" {
				tbInfo, ok := tokenBlueprintCache[blueprintIDs.TokenBlueprintID]
				if !ok {
					tbInfo = q.resolveTokenBlueprintInfo(ctx, blueprintIDs.TokenBlueprintID)
					tokenBlueprintCache[blueprintIDs.TokenBlueprintID] = tbInfo
				}

				if tbInfo.TokenName != "" {
					item.TokenName = tbInfo.TokenName
				}
				if tbInfo.TokenIcon != "" {
					item.TokenIcon = tbInfo.TokenIcon
				}
				if tbInfo.BrandID != "" {
					item.BrandID = tbInfo.BrandID
				}
			}

			if item.BrandID != "" {
				brandInfo, ok := brandCache[item.BrandID]
				if !ok {
					brandInfo = q.resolveBrandInfo(ctx, item.BrandID)
					brandCache[item.BrandID] = brandInfo
				}

				if brandInfo.BrandName != "" {
					item.BrandName = brandInfo.BrandName
				}
				if brandInfo.BrandIcon != "" {
					item.BrandIcon = brandInfo.BrandIcon
				}
			}

			if modelID == "" {
				continue
			}

			cacheKey := buildHistoryModelCacheKey(
				modelID,
				inventoryID,
				blueprintIDs.ProductBlueprintID,
				blueprintIDs.TokenBlueprintID,
			)

			resolved, ok := modelCache[cacheKey]
			if !ok {
				nextResolved, err := q.modelResolver.ResolveHistoryModelByID(ctx, historydto.HistoryResolveModelInput{
					ModelID:            modelID,
					InventoryID:        inventoryID,
					ProductBlueprintID: blueprintIDs.ProductBlueprintID,
					TokenBlueprintID:   blueprintIDs.TokenBlueprintID,
				})
				if err != nil {
					continue
				}

				resolved = nextResolved
				modelCache[cacheKey] = nextResolved
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

	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return "", "", ErrHistoryInventoryIDEmpty
	}

	return q.inventoryBlueprintResolver.ResolveBlueprintIDsByInventoryID(ctx, id)
}

func (q *HistoryQuery) ResolveProductBlueprintInfo(
	ctx context.Context,
	productBlueprintID string,
) (productName string, brandID string, err error) {
	if q == nil || q.productBlueprintResolver == nil {
		return "", "", ErrHistoryQueryNotConfigured
	}

	id := strings.TrimSpace(productBlueprintID)
	if id == "" {
		return "", "", nil
	}

	pb, pbErr := q.productBlueprintResolver.GetByID(ctx, id)
	if pbErr != nil {
		return "", "", pbErr
	}

	return strings.TrimSpace(pb.ProductName), strings.TrimSpace(pb.BrandID), nil
}

func (q *HistoryQuery) ResolveTokenBlueprintInfo(
	ctx context.Context,
	tokenBlueprintID string,
) (tokenName string, tokenIcon string, brandID string, err error) {
	if q == nil || q.tokenBlueprintResolver == nil {
		return "", "", "", ErrHistoryQueryNotConfigured
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return "", "", "", nil
	}

	patch, patchErr := q.tokenBlueprintResolver.GetPatchByID(ctx, id)
	if patchErr != nil {
		return "", "", "", patchErr
	}

	return strings.TrimSpace(patch.TokenName),
		strings.TrimSpace(patch.IconURL),
		strings.TrimSpace(patch.BrandID),
		nil
}

func (q *HistoryQuery) ResolveBrandInfo(
	ctx context.Context,
	brandID string,
) (brandName string, brandIcon string, err error) {
	if q == nil || q.brandResolver == nil {
		return "", "", ErrHistoryQueryNotConfigured
	}

	id := strings.TrimSpace(brandID)
	if id == "" {
		return "", "", nil
	}

	nameIcon, nameIconErr := q.brandResolver.GetNameIconByID(ctx, id)
	if nameIconErr != nil {
		return "", "", nameIconErr
	}

	return strings.TrimSpace(nameIcon.Name), strings.TrimSpace(nameIcon.BrandIcon), nil
}

func (q *HistoryQuery) ResolveModel(
	ctx context.Context,
	in historydto.HistoryResolveModelInput,
) (historydto.HistoryResolvedModel, error) {
	if q == nil || q.modelResolver == nil {
		return historydto.HistoryResolvedModel{}, ErrHistoryQueryNotConfigured
	}

	nextInput := historydto.HistoryResolveModelInput{
		ModelID:            strings.TrimSpace(in.ModelID),
		InventoryID:        strings.TrimSpace(in.InventoryID),
		ProductBlueprintID: strings.TrimSpace(in.ProductBlueprintID),
		TokenBlueprintID:   strings.TrimSpace(in.TokenBlueprintID),
	}

	if nextInput.ModelID == "" {
		return historydto.HistoryResolvedModel{}, ErrHistoryModelIDEmpty
	}

	if nextInput.InventoryID != "" &&
		(nextInput.ProductBlueprintID == "" || nextInput.TokenBlueprintID == "") &&
		q.inventoryBlueprintResolver != nil {
		productBlueprintID, tokenBlueprintID, err :=
			q.inventoryBlueprintResolver.ResolveBlueprintIDsByInventoryID(ctx, nextInput.InventoryID)
		if err == nil {
			if nextInput.ProductBlueprintID == "" {
				nextInput.ProductBlueprintID = strings.TrimSpace(productBlueprintID)
			}
			if nextInput.TokenBlueprintID == "" {
				nextInput.TokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
			}
		}
	}

	resolved, err := q.modelResolver.ResolveHistoryModelByID(ctx, nextInput)
	if err != nil {
		return historydto.HistoryResolvedModel{}, err
	}

	if resolved.ProductBlueprintID == "" {
		resolved.ProductBlueprintID = nextInput.ProductBlueprintID
	}
	if resolved.TokenBlueprintID == "" {
		resolved.TokenBlueprintID = nextInput.TokenBlueprintID
	}

	if q.productBlueprintResolver != nil && resolved.ProductBlueprintID != "" {
		pbInfo := q.resolveProductBlueprintInfo(ctx, resolved.ProductBlueprintID)

		if resolved.ProductName == "" {
			resolved.ProductName = pbInfo.ProductName
		}
		if resolved.BrandID == "" {
			resolved.BrandID = pbInfo.BrandID
		}
	}

	if q.tokenBlueprintResolver != nil && resolved.TokenBlueprintID != "" {
		tbInfo := q.resolveTokenBlueprintInfo(ctx, resolved.TokenBlueprintID)

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

	if q.brandResolver != nil && resolved.BrandID != "" {
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

func (q *HistoryQuery) resolveProductBlueprintInfo(
	ctx context.Context,
	productBlueprintID string,
) historyProductBlueprintInfo {
	id := strings.TrimSpace(productBlueprintID)
	if id == "" || q == nil || q.productBlueprintResolver == nil {
		return historyProductBlueprintInfo{}
	}

	productName, brandID, err := q.ResolveProductBlueprintInfo(ctx, id)
	if err != nil {
		return historyProductBlueprintInfo{}
	}

	return historyProductBlueprintInfo{
		ProductName: productName,
		BrandID:     brandID,
	}
}

func (q *HistoryQuery) resolveTokenBlueprintInfo(
	ctx context.Context,
	tokenBlueprintID string,
) historyTokenBlueprintInfo {
	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" || q == nil || q.tokenBlueprintResolver == nil {
		return historyTokenBlueprintInfo{}
	}

	tokenName, tokenIcon, brandID, err := q.ResolveTokenBlueprintInfo(ctx, id)
	if err != nil {
		return historyTokenBlueprintInfo{}
	}

	return historyTokenBlueprintInfo{
		TokenName: tokenName,
		TokenIcon: tokenIcon,
		BrandID:   brandID,
	}
}

func (q *HistoryQuery) resolveBrandInfo(
	ctx context.Context,
	brandID string,
) historyBrandInfo {
	id := strings.TrimSpace(brandID)
	if id == "" || q == nil || q.brandResolver == nil {
		return historyBrandInfo{}
	}

	brandName, brandIcon, err := q.ResolveBrandInfo(ctx, id)
	if err != nil {
		return historyBrandInfo{}
	}

	return historyBrandInfo{
		BrandName: brandName,
		BrandIcon: brandIcon,
	}
}

type historyBlueprintIDs struct {
	ProductBlueprintID string
	TokenBlueprintID   string
}

type historyProductBlueprintInfo struct {
	ProductName string
	BrandID     string
}

type historyTokenBlueprintInfo struct {
	TokenName string
	TokenIcon string
	BrandID   string
}

type historyBrandInfo struct {
	BrandName string
	BrandID   string
	BrandIcon string
}

func buildHistoryModelCacheKey(
	modelID string,
	inventoryID string,
	productBlueprintID string,
	tokenBlueprintID string,
) string {
	return strings.Join([]string{
		strings.TrimSpace(modelID),
		strings.TrimSpace(inventoryID),
		strings.TrimSpace(productBlueprintID),
		strings.TrimSpace(tokenBlueprintID),
	}, "|")
}

func cloneHistoryOrders(in []historydto.HistoryOrder) []historydto.HistoryOrder {
	if len(in) == 0 {
		return nil
	}

	out := make([]historydto.HistoryOrder, 0, len(in))

	for _, order := range in {
		next := order

		if len(order.Items) > 0 {
			next.Items = make([]historydto.HistoryOrderItem, len(order.Items))
			copy(next.Items, order.Items)
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
		item.Measurements = cloneMeasurements(resolved.Measurements)
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
