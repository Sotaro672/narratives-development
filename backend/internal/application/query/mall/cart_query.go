// backend/internal/application/query/mall/cart_query.go
package mall

import (
	"context"
	"errors"
	"time"

	malldto "narratives/internal/application/query/mall/dto"
	mallshared "narratives/internal/application/query/mall/shared"
	branddom "narratives/internal/domain/brand"
	cartdom "narratives/internal/domain/cart"
	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	productblueprintcategorydom "narratives/internal/domain/productBlueprintCategory"
	resaledom "narratives/internal/domain/resale"
)

type CartReader interface {
	GetByAvatarID(ctx context.Context, avatarID string) (*cartdom.Cart, error)
}

type ListReader interface {
	GetByID(ctx context.Context, id string) (ldom.List, error)
}

type ListImageReader interface {
	ListByListID(ctx context.Context, listID string) ([]ldom.ListImage, error)
}

type ResaleReader interface {
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
}

type ResaleImageReader interface {
	ListByResaleID(ctx context.Context, resaleID string) ([]resaledom.ResaleImage, error)
}

type CartQuery struct {
	CartRepo CartReader

	ListRepo             ListReader
	ListImageRepo        ListImageReader
	InventoryRepo        invdom.RepositoryPort
	ProductBlueprintRepo ProductBlueprintReader

	ResaleRepo      ResaleReader
	ResaleImageRepo ResaleImageReader

	BrandRepo branddom.Repository

	DisplayResolver mallshared.MallDisplayResolver
}

type CartQueryOption func(*CartQuery)

func WithCartQueryBrandRepo(repo branddom.Repository) CartQueryOption {
	return func(query *CartQuery) {
		if query != nil {
			query.BrandRepo = repo
		}
	}
}

func NewCartQuery(
	cartRepo CartReader,
	listRepo ListReader,
	listImageRepo ListImageReader,
	inventoryRepo invdom.RepositoryPort,
	productBlueprintRepo ProductBlueprintReader,
	resaleRepo ResaleReader,
	resaleImageRepo ResaleImageReader,
	displayResolver mallshared.MallDisplayResolver,
	opts ...CartQueryOption,
) *CartQuery {
	query := &CartQuery{
		CartRepo:             cartRepo,
		ListRepo:             listRepo,
		ListImageRepo:        listImageRepo,
		InventoryRepo:        inventoryRepo,
		ProductBlueprintRepo: productBlueprintRepo,
		ResaleRepo:           resaleRepo,
		ResaleImageRepo:      resaleImageRepo,
		DisplayResolver:      displayResolver,
	}

	for _, option := range opts {
		if option != nil {
			option(query)
		}
	}

	return query
}

func (q *CartQuery) GetByAvatarID(ctx context.Context, avatarID string) (malldto.CartDTO, error) {
	if q == nil || q.CartRepo == nil {
		return malldto.CartDTO{}, errors.New("mall cart query: cart repo is nil")
	}
	if avatarID == "" {
		return malldto.CartDTO{}, errors.New("avatarId is required")
	}

	cart, err := q.CartRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		return malldto.CartDTO{}, err
	}

	// 未作成のカートはNotFoundではなく空カートとして返す。
	if cart == nil {
		return malldto.CartDTO{
			AvatarID: avatarID,
			Items:    map[string]malldto.CartItemDTO{},
		}, nil
	}

	if cart.ID == "" {
		cart.ID = avatarID
	}

	cart = normalizeCart(cart)

	priceIndex, listMetaIndex := q.fetchLists(ctx, cart)
	inventoryIndex := q.fetchInventories(ctx, cart)
	modelIndex := q.fetchModels(ctx, cart)
	resaleIndex := q.fetchResales(ctx, cart)
	resaleImageIndex := q.fetchResaleImages(ctx, cart)
	resaleDisplayIndex := q.fetchResaleDisplayMeta(ctx, cart, resaleIndex)
	productDisplayIndex := q.fetchProductDisplayMeta(
		ctx,
		cart,
		inventoryIndex,
		resaleIndex,
		resaleDisplayIndex,
	)

	result := toCartDTO(
		cart,
		priceIndex,
		listMetaIndex,
		inventoryIndex,
		modelIndex,
		productDisplayIndex,
		resaleIndex,
		resaleImageIndex,
		resaleDisplayIndex,
	)

	return result, nil
}

func normalizeCart(cart *cartdom.Cart) *cartdom.Cart {
	if cart == nil {
		return nil
	}

	if cart.Items == nil {
		cart.Items = map[string]cartdom.CartItem{}
		return cart
	}

	items := map[string]cartdom.CartItem{}

	for itemKey, item := range cart.Items {
		if itemKey == "" {
			continue
		}

		switch mallshared.InferCartItemType(item) {
		case cartdom.CartItemTypeList:
			if item.InventoryID == "" || item.ListID == "" || item.ModelID == "" || item.Qty <= 0 {
				continue
			}

			items[itemKey] = cartdom.CartItem{
				Type:        cartdom.CartItemTypeList,
				InventoryID: item.InventoryID,
				ListID:      item.ListID,
				ModelID:     item.ModelID,
				Qty:         item.Qty,
			}

		case cartdom.CartItemTypeResale:
			if item.ResaleID == "" || item.ProductID == "" {
				continue
			}

			items[itemKey] = cartdom.CartItem{
				Type:      cartdom.CartItemTypeResale,
				ResaleID:  item.ResaleID,
				ProductID: item.ProductID,
				Qty:       1,
			}
		}
	}

	cart.Items = items
	return cart
}

type invParts struct {
	ProductBlueprintID string
	TokenBlueprintID   string
}

type listMeta struct {
	Title    string
	ImageURL string
}

type productDisplayMeta struct {
	ProductName string
	BrandID     string
	BrandName   string

	ProductBlueprintCategoryPath []string
	ConsumptionTaxRate           int
}

type resaleMeta struct {
	ID                 string
	Price              int
	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

type resaleDisplayMeta struct {
	BrandName          string
	ModelID            string
	ProductBlueprintID string
	Model              mallshared.ModelDisplay
}

func toCartDTO(
	cart *cartdom.Cart,
	priceIndex map[string]map[string]int,
	listMetaIndex map[string]listMeta,
	inventoryIndex map[string]invParts,
	modelIndex map[string]mallshared.ModelDisplay,
	productDisplayIndex map[string]productDisplayMeta,
	resaleIndex map[string]resaleMeta,
	resaleImageIndex map[string]string,
	resaleDisplayIndex map[string]resaleDisplayMeta,
) malldto.CartDTO {
	result := malldto.CartDTO{
		AvatarID:  cart.ID,
		Items:     map[string]malldto.CartItemDTO{},
		CreatedAt: toRFC3339Ptr(cart.CreatedAt),
		UpdatedAt: toRFC3339Ptr(cart.UpdatedAt),
		ExpiresAt: toRFC3339Ptr(cart.ExpiresAt),
	}

	if cart.Items == nil {
		return result
	}

	for itemKey, item := range cart.Items {
		if itemKey == "" {
			continue
		}

		switch mallshared.InferCartItemType(item) {
		case cartdom.CartItemTypeList:
			itemDTO, ok := toListCartItemDTO(
				item,
				priceIndex,
				listMetaIndex,
				inventoryIndex,
				modelIndex,
				productDisplayIndex,
			)
			if !ok {
				continue
			}

			result.Items[itemKey] = itemDTO

		case cartdom.CartItemTypeResale:
			itemDTO, ok := toResaleCartItemDTO(
				item,
				resaleIndex,
				resaleImageIndex,
				resaleDisplayIndex,
				productDisplayIndex,
			)
			if !ok {
				continue
			}

			result.Items[itemKey] = itemDTO
		}
	}

	return result
}

func toListCartItemDTO(
	item cartdom.CartItem,
	priceIndex map[string]map[string]int,
	listMetaIndex map[string]listMeta,
	inventoryIndex map[string]invParts,
	modelIndex map[string]mallshared.ModelDisplay,
	productDisplayIndex map[string]productDisplayMeta,
) (malldto.CartItemDTO, bool) {
	inventoryID := item.InventoryID
	listID := item.ListID
	modelID := item.ModelID

	if inventoryID == "" || listID == "" || modelID == "" || item.Qty <= 0 {
		return malldto.CartItemDTO{}, false
	}

	result := malldto.CartItemDTO{
		Type:        string(cartdom.CartItemTypeList),
		InventoryID: inventoryID,
		ListID:      listID,
		ModelID:     modelID,
		Qty:         item.Qty,
	}

	if metadata, ok := listMetaIndex[listID]; ok {
		result.Title = metadata.Title
		result.ImageURL = metadata.ImageURL
	}

	if prices, ok := priceIndex[listID]; ok {
		if price, exists := prices[modelID]; exists {
			value := price
			result.Price = &value
		}
	}

	productBlueprintID := ""

	if parts, ok := inventoryIndex[inventoryID]; ok {
		productBlueprintID = parts.ProductBlueprintID
		result.ProductBlueprintID = parts.ProductBlueprintID
		result.TokenBlueprintID = parts.TokenBlueprintID
	}

	if productBlueprintID != "" {
		if display, ok := productDisplayIndex[productBlueprintID]; ok {
			result.ProductName = display.ProductName
			result.BrandID = display.BrandID
			result.BrandName = display.BrandName

			result.ProductBlueprintCategoryPath =
				append(
					[]string(nil),
					display.ProductBlueprintCategoryPath...,
				)

			result.ConsumptionTaxRate =
				display.ConsumptionTaxRate

			if result.Title == "" {
				result.Title = display.ProductName
			}
		}
	}

	if model, ok := modelIndex[modelID]; ok {
		mallshared.ApplyCartModelDisplay(
			&result,
			cartModelDisplayFromModelDisplay(model),
		)
	}

	return result, true
}

func toResaleCartItemDTO(
	item cartdom.CartItem,
	resaleIndex map[string]resaleMeta,
	resaleImageIndex map[string]string,
	resaleDisplayIndex map[string]resaleDisplayMeta,
	productDisplayIndex map[string]productDisplayMeta,
) (malldto.CartItemDTO, bool) {
	if item.ResaleID == "" || item.ProductID == "" {
		return malldto.CartItemDTO{}, false
	}

	var metadata *mallshared.ResaleCartItemMeta
	productBlueprintID := ""

	if resale, ok := resaleIndex[item.ResaleID]; ok {
		metadata = &mallshared.ResaleCartItemMeta{
			ID:                 resale.ID,
			Price:              resale.Price,
			ProductID:          resale.ProductID,
			ProductBlueprintID: resale.ProductBlueprintID,
			TokenBlueprintID:   resale.TokenBlueprintID,
			BrandID:            resale.BrandID,
		}

		productBlueprintID = resale.ProductBlueprintID
	}

	imageURL := resaleImageIndex[item.ResaleID]
	brandName := ""
	modelID := ""
	displayProductBlueprintID := ""
	model := mallshared.CartModelDisplay{}

	if display, ok := resaleDisplayIndex[item.ResaleID]; ok {
		brandName = display.BrandName
		modelID = display.ModelID
		displayProductBlueprintID = display.ProductBlueprintID
		model = cartModelDisplayFromModelDisplay(display.Model)

		if productBlueprintID == "" {
			productBlueprintID = display.ProductBlueprintID
		}
	}

	productName := ""
	if productBlueprintID != "" {
		if productDisplay, ok := productDisplayIndex[productBlueprintID]; ok {
			productName = productDisplay.ProductName
			if brandName == "" {
				brandName = productDisplay.BrandName
			}
		}
	}

	result, ok :=
		mallshared.ResaleCartItemToDTO(
			mallshared.ResaleCartItemDisplayInput{
				Item:               item,
				Meta:               metadata,
				ImageURL:           imageURL,
				BrandName:          brandName,
				ModelID:            modelID,
				Model:              model,
				ProductBlueprintID: displayProductBlueprintID,
				ProductName:        productName,
			},
		)
	if !ok {
		return malldto.CartItemDTO{}, false
	}

	if productBlueprintID != "" {
		if display, exists :=
			productDisplayIndex[productBlueprintID]; exists {
			result.ProductBlueprintCategoryPath =
				append(
					[]string(nil),
					display.ProductBlueprintCategoryPath...,
				)

			result.ConsumptionTaxRate =
				display.ConsumptionTaxRate
		}
	}

	return result, true
}

func cartModelDisplayFromModelDisplay(model mallshared.ModelDisplay) mallshared.CartModelDisplay {
	return mallshared.CartModelDisplay{
		Kind:        model.Kind,
		ModelNumber: model.ModelNumber,
		ModelLabel:  model.ModelLabel,
		Size:        model.Size,
		Color:       model.ColorName,
		VolumeValue: model.VolumeValue,
		VolumeUnit:  model.VolumeUnit,
	}
}

func toRFC3339Ptr(value time.Time) *string {
	if value.IsZero() {
		return nil
	}

	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func (q *CartQuery) fetchLists(
	ctx context.Context,
	cart *cartdom.Cart,
) (map[string]map[string]int, map[string]listMeta) {
	if q == nil || q.ListRepo == nil || cart == nil || len(cart.Items) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	listIDs := make([]string, 0, 8)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) != cartdom.CartItemTypeList {
			continue
		}

		listID := item.ListID
		if listID == "" {
			continue
		}
		if _, exists := seen[listID]; exists {
			continue
		}

		seen[listID] = struct{}{}
		listIDs = append(listIDs, listID)
	}

	if len(listIDs) == 0 {
		return nil, nil
	}

	priceIndex := map[string]map[string]int{}
	metadataIndex := map[string]listMeta{}

	for _, listID := range listIDs {
		list, err := q.ListRepo.GetByID(ctx, listID)
		if err != nil {
			continue
		}

		metadata := listMeta{
			Title: list.Title,
		}

		if q.ListImageRepo != nil {
			images, imageErr := q.ListImageRepo.ListByListID(ctx, listID)
			if imageErr == nil {
				metadata.ImageURL = mallshared.SelectPrimaryListImageURL(list, images)
			}
		}

		if metadata.Title != "" || metadata.ImageURL != "" {
			metadataIndex[listID] = metadata
		}

		if len(list.Prices) == 0 {
			continue
		}

		prices := map[string]int{}

		for _, row := range list.Prices {
			if row.ModelID == "" {
				continue
			}

			prices[row.ModelID] = row.Price
		}

		if len(prices) > 0 {
			priceIndex[listID] = prices
		}
	}

	if len(priceIndex) == 0 {
		priceIndex = nil
	}
	if len(metadataIndex) == 0 {
		metadataIndex = nil
	}

	return priceIndex, metadataIndex
}

func (q *CartQuery) fetchInventories(ctx context.Context, cart *cartdom.Cart) map[string]invParts {
	if q == nil || q.InventoryRepo == nil || cart == nil || len(cart.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	inventoryIDs := make([]string, 0, 8)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) != cartdom.CartItemTypeList {
			continue
		}

		inventoryID := item.InventoryID
		if inventoryID == "" {
			continue
		}
		if _, exists := seen[inventoryID]; exists {
			continue
		}

		seen[inventoryID] = struct{}{}
		inventoryIDs = append(inventoryIDs, inventoryID)
	}

	result := map[string]invParts{}

	for _, inventoryID := range inventoryIDs {
		productBlueprintID, tokenBlueprintID, err :=
			q.InventoryRepo.ResolveBlueprintIDsByInventoryID(ctx, inventoryID)
		if err != nil {
			continue
		}

		if productBlueprintID == "" || tokenBlueprintID == "" {
			continue
		}

		result[inventoryID] = invParts{
			ProductBlueprintID: productBlueprintID,
			TokenBlueprintID:   tokenBlueprintID,
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func (q *CartQuery) fetchResales(ctx context.Context, cart *cartdom.Cart) map[string]resaleMeta {
	if q == nil || q.ResaleRepo == nil || cart == nil || len(cart.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	resaleIDs := make([]string, 0, 8)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) != cartdom.CartItemTypeResale {
			continue
		}

		resaleID := item.ResaleID
		if resaleID == "" {
			continue
		}
		if _, exists := seen[resaleID]; exists {
			continue
		}

		seen[resaleID] = struct{}{}
		resaleIDs = append(resaleIDs, resaleID)
	}

	result := map[string]resaleMeta{}

	for _, resaleID := range resaleIDs {
		resale, err := q.ResaleRepo.GetByID(ctx, resaleID)
		if err != nil {
			continue
		}

		if resale.ID == "" {
			resale.ID = resaleID
		}

		result[resaleID] = resaleMeta{
			ID:                 resale.ID,
			Price:              resale.Price,
			ProductID:          resale.ProductID,
			ProductBlueprintID: resale.ProductBlueprintID,
			TokenBlueprintID:   resale.TokenBlueprintID,
			BrandID:            resale.BrandID,
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func (q *CartQuery) fetchResaleImages(ctx context.Context, cart *cartdom.Cart) map[string]string {
	if q == nil || q.ResaleImageRepo == nil || cart == nil || len(cart.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	resaleIDs := make([]string, 0, 8)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) != cartdom.CartItemTypeResale {
			continue
		}

		resaleID := item.ResaleID
		if resaleID == "" {
			continue
		}
		if _, exists := seen[resaleID]; exists {
			continue
		}

		seen[resaleID] = struct{}{}
		resaleIDs = append(resaleIDs, resaleID)
	}

	result := map[string]string{}

	for _, resaleID := range resaleIDs {
		images, err := q.ResaleImageRepo.ListByResaleID(ctx, resaleID)
		if err != nil {
			continue
		}

		imageURL := mallshared.FirstResaleImageURL(images)
		if imageURL == "" {
			continue
		}

		result[resaleID] = imageURL
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func (q *CartQuery) fetchResaleDisplayMeta(
	ctx context.Context,
	cart *cartdom.Cart,
	resaleIndex map[string]resaleMeta,
) map[string]resaleDisplayMeta {
	if q == nil || cart == nil || len(cart.Items) == 0 || len(resaleIndex) == 0 {
		return nil
	}

	result := map[string]resaleDisplayMeta{}

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) != cartdom.CartItemTypeResale {
			continue
		}

		resaleID := item.ResaleID
		if resaleID == "" {
			continue
		}

		metadata, ok := resaleIndex[resaleID]
		if !ok {
			continue
		}

		display := resaleDisplayMeta{
			ProductBlueprintID: metadata.ProductBlueprintID,
		}

		if q.BrandRepo != nil && metadata.BrandID != "" {
			brand, err := q.BrandRepo.GetByID(ctx, metadata.BrandID)
			if err == nil {
				display.BrandName = brand.Name
			}
		}

		productID := metadata.ProductID
		if productID == "" {
			productID = item.ProductID
		}

		if q.DisplayResolver != nil && productID != "" {
			model, err := q.DisplayResolver.ResolveModelByProductID(ctx, productID)
			if err == nil {
				display.Model = model
				display.ModelID = model.ModelID

				if display.ProductBlueprintID == "" {
					display.ProductBlueprintID = model.ProductBlueprintID
				}
			}
		}

		if display.BrandName == "" &&
			display.ModelID == "" &&
			display.ProductBlueprintID == "" {
			continue
		}

		result[resaleID] = display
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func (q *CartQuery) fetchModels(
	ctx context.Context,
	cart *cartdom.Cart,
) map[string]mallshared.ModelDisplay {
	if q == nil || q.DisplayResolver == nil || cart == nil || len(cart.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	modelIDs := make([]string, 0, 16)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) != cartdom.CartItemTypeList {
			continue
		}

		modelID := item.ModelID
		if modelID == "" {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}

		seen[modelID] = struct{}{}
		modelIDs = append(modelIDs, modelID)
	}

	result := map[string]mallshared.ModelDisplay{}

	for _, modelID := range modelIDs {
		model, err := q.DisplayResolver.ResolveModelByModelID(ctx, modelID)
		if err != nil {
			continue
		}

		result[modelID] = model
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func (q *CartQuery) fetchProductDisplayMeta(
	ctx context.Context,
	cart *cartdom.Cart,
	inventoryIndex map[string]invParts,
	resaleIndex map[string]resaleMeta,
	resaleDisplayIndex map[string]resaleDisplayMeta,
) map[string]productDisplayMeta {
	if q == nil || q.ProductBlueprintRepo == nil || cart == nil || len(cart.Items) == 0 {
		return nil
	}

	result := map[string]productDisplayMeta{}
	seen := map[string]struct{}{}

	for _, item := range cart.Items {
		productBlueprintID := ""

		switch mallshared.InferCartItemType(item) {
		case cartdom.CartItemTypeList:
			if parts, ok := inventoryIndex[item.InventoryID]; ok {
				productBlueprintID = parts.ProductBlueprintID
			}

		case cartdom.CartItemTypeResale:
			if metadata, ok := resaleIndex[item.ResaleID]; ok {
				productBlueprintID = metadata.ProductBlueprintID
			}

			if productBlueprintID == "" {
				if display, ok := resaleDisplayIndex[item.ResaleID]; ok {
					productBlueprintID = display.ProductBlueprintID
				}
			}

		default:
			continue
		}

		if productBlueprintID == "" {
			continue
		}
		if _, exists := seen[productBlueprintID]; exists {
			continue
		}

		seen[productBlueprintID] = struct{}{}

		productBlueprint, err := q.ProductBlueprintRepo.GetByID(ctx, productBlueprintID)
		if err != nil {
			continue
		}

		productBlueprintCategoryPath :=
			append(
				[]string(nil),
				productBlueprint.
					ProductBlueprintCategoryPath...,
			)

		consumptionTaxRate, err :=
			productblueprintcategorydom.
				GetConsumptionTaxRate(
					productBlueprintCategoryPath,
				)
		if err != nil {
			continue
		}

		display := productDisplayMeta{
			ProductName: productBlueprint.ProductName,
			BrandID:     productBlueprint.BrandID,

			ProductBlueprintCategoryPath: productBlueprintCategoryPath,

			ConsumptionTaxRate: int(consumptionTaxRate),
		}

		if q.BrandRepo != nil && productBlueprint.BrandID != "" {
			brand, brandErr := q.BrandRepo.GetByID(ctx, productBlueprint.BrandID)
			if brandErr == nil {
				display.BrandName = brand.Name
			}
		}

		result[productBlueprintID] = display
	}

	if len(result) == 0 {
		return nil
	}

	return result
}
