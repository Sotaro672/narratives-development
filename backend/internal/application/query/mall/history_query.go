// backend/internal/application/query/mall/history_query.go
package mall

import (
	"context"
	"errors"
	"strings"

	historydto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"

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
//	GetByID(ctx, tokenBlueprintID)
//
// TokenBlueprint provides:
// - Name
// - IconURL
// - BrandID
type HistoryTokenBlueprintResolver interface {
	GetByID(ctx context.Context, id string) (*tokenbpdom.TokenBlueprint, error)
}

// HistoryBrandResolver resolves brand display data from brandId.
//
// Concrete implementation can be brand.RepositoryPort / brand.Repository because it has:
//
//	GetByID(ctx, brandID)
//
// Brand provides:
// - Name
// - BrandIcon
type HistoryBrandResolver interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type HistoryQuery struct {
	inventoryBlueprintResolver HistoryInventoryBlueprintResolver
	productBlueprintResolver   HistoryProductBlueprintResolver
	tokenBlueprintResolver     HistoryTokenBlueprintResolver
	brandResolver              HistoryBrandResolver
	nameResolver               *appresolver.NameResolver
}

func NewHistoryQuery(
	inventoryBlueprintResolver HistoryInventoryBlueprintResolver,
	productBlueprintResolver HistoryProductBlueprintResolver,
	tokenBlueprintResolver HistoryTokenBlueprintResolver,
	brandResolver HistoryBrandResolver,
	nameResolver *appresolver.NameResolver,
) *HistoryQuery {
	return &HistoryQuery{
		inventoryBlueprintResolver: inventoryBlueprintResolver,
		productBlueprintResolver:   productBlueprintResolver,
		tokenBlueprintResolver:     tokenBlueprintResolver,
		brandResolver:              brandResolver,
		nameResolver:               nameResolver,
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
//  5. modelId -> size / color / modelNumber / volume
func (q *HistoryQuery) EnrichOrderPage(
	ctx context.Context,
	in historydto.EnrichHistoryOrderPageInput,
) (historydto.HistoryOrderPage, error) {
	if q == nil ||
		q.inventoryBlueprintResolver == nil ||
		q.productBlueprintResolver == nil ||
		q.tokenBlueprintResolver == nil ||
		q.brandResolver == nil ||
		q.nameResolver == nil {
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

			inventoryID := item.InventoryID
			modelID := item.ModelID

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
							ProductBlueprintID: productBlueprintID,
							TokenBlueprintID:   tokenBlueprintID,
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
				nextResolved, err := q.resolveHistoryModelByID(ctx, historydto.HistoryResolveModelInput{
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

	id := inventoryID
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

	id := productBlueprintID
	if id == "" {
		return "", "", nil
	}

	pb, pbErr := q.productBlueprintResolver.GetByID(ctx, id)
	if pbErr != nil {
		return "", "", pbErr
	}

	return pb.ProductName, pb.BrandID, nil
}

func (q *HistoryQuery) ResolveTokenBlueprintInfo(
	ctx context.Context,
	tokenBlueprintID string,
) (tokenName string, tokenIcon string, brandID string, err error) {
	if q == nil || q.tokenBlueprintResolver == nil {
		return "", "", "", ErrHistoryQueryNotConfigured
	}

	id := tokenBlueprintID
	if id == "" {
		return "", "", "", nil
	}

	tb, tbErr := q.tokenBlueprintResolver.GetByID(ctx, id)
	if tbErr != nil {
		return "", "", "", tbErr
	}
	if tb == nil {
		return "", "", "", nil
	}

	return tb.Name,
		tb.IconURL,
		tb.BrandID,
		nil
}

func (q *HistoryQuery) ResolveBrandInfo(
	ctx context.Context,
	brandID string,
) (brandName string, brandIcon string, err error) {
	if q == nil || q.brandResolver == nil {
		return "", "", ErrHistoryQueryNotConfigured
	}

	id := brandID
	if id == "" {
		return "", "", nil
	}

	b, brandErr := q.brandResolver.GetByID(ctx, id)
	if brandErr != nil {
		return "", "", brandErr
	}

	return b.Name, b.BrandIcon, nil
}

func (q *HistoryQuery) ResolveModel(
	ctx context.Context,
	in historydto.HistoryResolveModelInput,
) (historydto.HistoryResolvedModel, error) {
	if q == nil || q.nameResolver == nil {
		return historydto.HistoryResolvedModel{}, ErrHistoryQueryNotConfigured
	}

	nextInput := historydto.HistoryResolveModelInput{
		ModelID:            in.ModelID,
		InventoryID:        in.InventoryID,
		ProductBlueprintID: in.ProductBlueprintID,
		TokenBlueprintID:   in.TokenBlueprintID,
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

func (q *HistoryQuery) resolveHistoryModelByID(
	ctx context.Context,
	in historydto.HistoryResolveModelInput,
) (historydto.HistoryResolvedModel, error) {
	if q == nil || q.nameResolver == nil {
		return historydto.HistoryResolvedModel{}, ErrHistoryQueryNotConfigured
	}

	if in.ModelID == "" {
		return historydto.HistoryResolvedModel{}, ErrHistoryModelIDEmpty
	}

	model := q.nameResolver.ResolveModelResolved(ctx, in.ModelID)

	return historyResolvedModelFromNameResolver(in, model), nil
}

func historyResolvedModelFromNameResolver(
	in historydto.HistoryResolveModelInput,
	model appresolver.ModelResolved,
) historydto.HistoryResolvedModel {
	out := historydto.HistoryResolvedModel{
		ModelID:            in.ModelID,
		InventoryID:        in.InventoryID,
		ProductBlueprintID: in.ProductBlueprintID,
		TokenBlueprintID:   in.TokenBlueprintID,
		Kind:               model.Kind,
		ModelNumber:        model.ModelNumber,
		Size:               model.Size,
		VolumeValue:        model.VolumeValue,
		VolumeUnit:         model.VolumeUnit,
	}

	if model.Color != "" || model.RGB != nil {
		out.Color = &historydto.HistoryColor{
			Name: model.Color,
			RGB:  model.RGB,
		}
	}

	return out
}

func (q *HistoryQuery) resolveProductBlueprintInfo(
	ctx context.Context,
	productBlueprintID string,
) historyProductBlueprintInfo {
	id := productBlueprintID
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
	id := tokenBlueprintID
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
	id := brandID
	if id == "" || q == nil || q.brandResolver == nil {
		return historyBrandInfo{}
	}

	brandName, brandIcon, err := q.ResolveBrandInfo(ctx, id)
	if err != nil {
		return historyBrandInfo{}
	}

	return historyBrandInfo{
		BrandName: brandName,
		BrandID:   id,
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
		modelID,
		inventoryID,
		productBlueprintID,
		tokenBlueprintID,
	}, "|")
}

func cloneHistoryOrders(in []historydto.HistoryOrder) []historydto.HistoryOrder {
	out := make([]historydto.HistoryOrder, 0, len(in))

	for _, order := range in {
		next := order

		if len(order.Items) > 0 {
			next.Items = make([]historydto.HistoryOrderItem, len(order.Items))
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
		item.Measurements = cloneHistoryModelMeasurements(resolved.Measurements)
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

func cloneHistoryModelMeasurements(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]int, len(in))
	for key, value := range in {
		if key == "" {
			continue
		}

		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}
