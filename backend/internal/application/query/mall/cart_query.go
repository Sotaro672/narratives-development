// backend/internal/application/query/mall/cart_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"time"

	malldto "narratives/internal/application/query/mall/dto"
	mallshared "narratives/internal/application/query/mall/shared"
	appresolver "narratives/internal/application/resolver"
	branddom "narratives/internal/domain/brand"
	cartdom "narratives/internal/domain/cart"
	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	resaledom "narratives/internal/domain/resale"
)

type CartReader interface {
	GetByAvatarID(
		ctx context.Context,
		avatarID string,
	) (*cartdom.Cart, error)
}

type ListReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (ldom.List, error)
}

type ResaleReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (resaledom.Resale, error)
}

type ResaleImageReader interface {
	ListByResaleID(
		ctx context.Context,
		resaleID string,
	) ([]resaledom.ResaleImage, error)
}

type CartQuery struct {
	CartRepo CartReader

	ListRepo             ListReader
	InventoryRepo        invdom.RepositoryPort
	ProductBlueprintRepo ProductBlueprintReader

	ResaleRepo      ResaleReader
	ResaleImageRepo ResaleImageReader

	ProductRepo productdom.Repository
	ModelRepo   modeldom.RepositoryPort
	BrandRepo   branddom.Repository

	Resolver *appresolver.NameResolver
}

type CartQueryOption func(*CartQuery)

func WithCartQueryProductRepo(
	repo productdom.Repository,
) CartQueryOption {
	return func(query *CartQuery) {
		if query != nil {
			query.ProductRepo = repo
		}
	}
}

func WithCartQueryModelRepo(
	repo modeldom.RepositoryPort,
) CartQueryOption {
	return func(query *CartQuery) {
		if query != nil {
			query.ModelRepo = repo
		}
	}
}

func WithCartQueryBrandRepo(
	repo branddom.Repository,
) CartQueryOption {
	return func(query *CartQuery) {
		if query != nil {
			query.BrandRepo = repo
		}
	}
}

func NewCartQuery(
	cartRepo CartReader,
	listRepo ListReader,
	inventoryRepo invdom.RepositoryPort,
	productBlueprintRepo ProductBlueprintReader,
	resaleRepo ResaleReader,
	resaleImageRepo ResaleImageReader,
	resolver *appresolver.NameResolver,
	opts ...CartQueryOption,
) *CartQuery {
	query := &CartQuery{
		CartRepo:             cartRepo,
		ListRepo:             listRepo,
		InventoryRepo:        inventoryRepo,
		ProductBlueprintRepo: productBlueprintRepo,
		ResaleRepo:           resaleRepo,
		ResaleImageRepo:      resaleImageRepo,
		Resolver:             resolver,
	}

	for _, option := range opts {
		if option != nil {
			option(query)
		}
	}

	return query
}

type cartQueryPort interface {
	GetCartQuery(
		ctx context.Context,
		avatarID string,
	) (any, error)
}

var _ cartQueryPort = (*CartQuery)(nil)

func (q *CartQuery) GetCartQuery(
	ctx context.Context,
	avatarID string,
) (any, error) {
	return q.GetByAvatarID(ctx, avatarID)
}

func (q *CartQuery) GetByAvatarID(
	ctx context.Context,
	avatarID string,
) (malldto.CartDTO, error) {
	if q == nil || q.CartRepo == nil {
		return malldto.CartDTO{},
			errors.New("mall cart query: cart repo is nil")
	}

	if avatarID == "" {
		return malldto.CartDTO{},
			errors.New("avatarId is required")
	}

	cart, err := q.CartRepo.GetByAvatarID(
		ctx,
		avatarID,
	)
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

	resaleDisplayIndex := q.fetchResaleDisplayMeta(
		ctx,
		cart,
		resaleIndex,
	)

	productNameIndex := q.fetchProductNames(
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
		productNameIndex,
		resaleIndex,
		resaleImageIndex,
		resaleDisplayIndex,
	)

	return result, nil
}

func normalizeCart(
	cart *cartdom.Cart,
) *cartdom.Cart {
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
			if item.InventoryID == "" ||
				item.ListID == "" ||
				item.ModelID == "" ||
				item.Qty <= 0 {
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
			if item.ResaleID == "" ||
				item.ProductID == "" {
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
	Title   string
	ImageID string
}

type modelSimple struct {
	Kind        string
	ModelNumber string
	ModelLabel  string

	Size  string
	Color string

	VolumeValue *int
	VolumeUnit  string
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
	Model              modelSimple
}

func toCartDTO(
	cart *cartdom.Cart,
	priceIndex map[string]map[string]int,
	listMetaIndex map[string]listMeta,
	inventoryIndex map[string]invParts,
	modelIndex map[string]modelSimple,
	productNameIndex map[string]string,
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
				productNameIndex,
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
				productNameIndex,
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
	modelIndex map[string]modelSimple,
	productNameIndex map[string]string,
) (malldto.CartItemDTO, bool) {
	inventoryID := item.InventoryID
	listID := item.ListID
	modelID := item.ModelID

	if inventoryID == "" ||
		listID == "" ||
		modelID == "" ||
		item.Qty <= 0 {
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
		result.ListImage = metadata.ImageID
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
		if name := productNameIndex[productBlueprintID]; name != "" {
			result.ProductName = name
			if result.Title == "" {
				result.Title = name
			}
		}
	}

	if model, ok := modelIndex[modelID]; ok {
		applyModelSimpleToCartItem(&result, model)
	}

	return result, true
}

func toResaleCartItemDTO(
	item cartdom.CartItem,
	resaleIndex map[string]resaleMeta,
	resaleImageIndex map[string]string,
	resaleDisplayIndex map[string]resaleDisplayMeta,
	productNameIndex map[string]string,
) (malldto.CartItemDTO, bool) {
	if item.ResaleID == "" ||
		item.ProductID == "" {
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
		model = cartModelDisplayFromSimple(display.Model)

		if productBlueprintID == "" {
			productBlueprintID = display.ProductBlueprintID
		}
	}

	productName := ""
	if productBlueprintID != "" {
		productName = productNameIndex[productBlueprintID]
	}

	return mallshared.ResaleCartItemToDTO(
		mallshared.ResaleCartItemDisplayInput{
			Item: item,
			Meta: metadata,

			ImageURL: imageURL,

			BrandName: brandName,
			ModelID:   modelID,
			Model:     model,

			ProductBlueprintID: displayProductBlueprintID,
			ProductName:        productName,
		},
	)
}

func cartModelDisplayFromSimple(
	model modelSimple,
) mallshared.CartModelDisplay {
	return mallshared.CartModelDisplay{
		Kind:        model.Kind,
		ModelNumber: model.ModelNumber,
		ModelLabel:  model.ModelLabel,
		Size:        model.Size,
		Color:       model.Color,
		VolumeValue: model.VolumeValue,
		VolumeUnit:  model.VolumeUnit,
	}
}

func applyModelSimpleToCartItem(
	item *malldto.CartItemDTO,
	model modelSimple,
) {
	if item == nil {
		return
	}

	if model.Kind != "" {
		item.ModelKind = model.Kind
	}
	if model.ModelNumber != "" {
		item.ModelNumber = model.ModelNumber
	}
	if model.ModelLabel != "" {
		item.ModelLabel = model.ModelLabel
	}
	if model.Size != "" {
		item.Size = model.Size
	}
	if model.Color != "" {
		item.Color = model.Color
	}
	if model.VolumeValue != nil {
		item.VolumeValue = model.VolumeValue
	}
	if model.VolumeUnit != "" {
		item.VolumeUnit = model.VolumeUnit
	}
}

func toRFC3339Ptr(
	value time.Time,
) *string {
	if value.IsZero() {
		return nil
	}

	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func (q *CartQuery) fetchLists(
	ctx context.Context,
	cart *cartdom.Cart,
) (
	map[string]map[string]int,
	map[string]listMeta,
) {
	if q == nil ||
		q.ListRepo == nil ||
		cart == nil ||
		len(cart.Items) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	listIDs := make([]string, 0, 8)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) !=
			cartdom.CartItemTypeList {
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
			Title:   list.Title,
			ImageID: list.ImageID,
		}

		if metadata.Title != "" ||
			metadata.ImageID != "" {
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

func (q *CartQuery) fetchInventories(
	ctx context.Context,
	cart *cartdom.Cart,
) map[string]invParts {
	if q == nil ||
		q.InventoryRepo == nil ||
		cart == nil ||
		len(cart.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	inventoryIDs := make([]string, 0, 8)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) !=
			cartdom.CartItemTypeList {
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
			q.InventoryRepo.ResolveBlueprintIDsByInventoryID(
				ctx,
				inventoryID,
			)
		if err != nil {
			continue
		}

		if productBlueprintID == "" ||
			tokenBlueprintID == "" {
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

func (q *CartQuery) fetchResales(
	ctx context.Context,
	cart *cartdom.Cart,
) map[string]resaleMeta {
	if q == nil ||
		q.ResaleRepo == nil ||
		cart == nil ||
		len(cart.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	resaleIDs := make([]string, 0, 8)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) !=
			cartdom.CartItemTypeResale {
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
		resale, err := q.ResaleRepo.GetByID(
			ctx,
			resaleID,
		)
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

func (q *CartQuery) fetchResaleImages(
	ctx context.Context,
	cart *cartdom.Cart,
) map[string]string {
	if q == nil ||
		q.ResaleImageRepo == nil ||
		cart == nil ||
		len(cart.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	resaleIDs := make([]string, 0, 8)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) !=
			cartdom.CartItemTypeResale {
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
		images, err := q.ResaleImageRepo.ListByResaleID(
			ctx,
			resaleID,
		)
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
	if q == nil ||
		cart == nil ||
		len(cart.Items) == 0 ||
		len(resaleIndex) == 0 {
		return nil
	}

	result := map[string]resaleDisplayMeta{}

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) !=
			cartdom.CartItemTypeResale {
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

		if q.BrandRepo != nil &&
			metadata.BrandID != "" {
			brand, err := q.BrandRepo.GetByID(
				ctx,
				metadata.BrandID,
			)
			if err == nil {
				display.BrandName = brand.Name
			}
		}

		productID := firstNonEmptyString(
			metadata.ProductID,
			item.ProductID,
		)

		if q.ProductRepo != nil &&
			productID != "" {
			product, err := q.ProductRepo.GetByID(
				ctx,
				productID,
			)
			if err == nil {
				display.ModelID = product.ModelID
			}
		}

		if q.ModelRepo != nil &&
			display.ModelID != "" {
			model, err := q.ModelRepo.GetByID(
				ctx,
				display.ModelID,
			)
			if err == nil {
				display.ModelID = firstNonEmptyString(
					display.ModelID,
					model.GetID(),
				)

				display.ProductBlueprintID =
					firstNonEmptyString(
						display.ProductBlueprintID,
						model.GetProductBlueprintID(),
					)

				display.Model =
					modelVariationToSimple(model)
			}
		}

		if display.BrandName == "" &&
			display.ModelID == "" &&
			display.ProductBlueprintID == "" &&
			isEmptyModel(display.Model) {
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
) map[string]modelSimple {
	if q == nil ||
		q.Resolver == nil ||
		cart == nil ||
		len(cart.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	modelIDs := make([]string, 0, 16)

	for _, item := range cart.Items {
		if mallshared.InferCartItemType(item) !=
			cartdom.CartItemTypeList {
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

	result := map[string]modelSimple{}

	for _, modelID := range modelIDs {
		resolved := q.Resolver.ResolveModelResolved(
			ctx,
			modelID,
		)

		model := modelSimple{
			Kind:        resolved.Kind,
			ModelNumber: resolved.ModelNumber,
			Size:        resolved.Size,
			Color:       resolved.Color,
			VolumeValue: resolved.VolumeValue,
			VolumeUnit:  resolved.VolumeUnit,
		}

		model.ModelLabel = buildModelLabel(model)

		if isEmptyModel(model) {
			continue
		}

		result[modelID] = model
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func modelVariationToSimple(
	variation modeldom.ModelVariation,
) modelSimple {
	if variation == nil {
		return modelSimple{}
	}

	result := modelSimple{
		ModelNumber: variation.GetModelNumber(),
	}

	switch model := variation.(type) {
	case modeldom.ApparelModelVariation:
		result.Kind =
			string(modeldom.ModelVariationKindApparel)
		result.ModelNumber = firstNonEmptyString(
			result.ModelNumber,
			model.ModelNumber,
		)
		result.Size = model.Size
		result.Color = model.Color.Name

	case *modeldom.ApparelModelVariation:
		if model != nil {
			result.Kind =
				string(modeldom.ModelVariationKindApparel)
			result.ModelNumber = firstNonEmptyString(
				result.ModelNumber,
				model.ModelNumber,
			)
			result.Size = model.Size
			result.Color = model.Color.Name
		}

	case modeldom.AlcoholModelVariation:
		result.Kind =
			string(modeldom.ModelVariationKindAlcohol)
		result.ModelNumber = firstNonEmptyString(
			result.ModelNumber,
			model.ModelNumber,
		)

		value := model.Volume.Value
		if value > 0 {
			result.VolumeValue = &value
		}

		result.VolumeUnit = model.Volume.Unit

	case *modeldom.AlcoholModelVariation:
		if model != nil {
			result.Kind =
				string(modeldom.ModelVariationKindAlcohol)
			result.ModelNumber = firstNonEmptyString(
				result.ModelNumber,
				model.ModelNumber,
			)

			value := model.Volume.Value
			if value > 0 {
				result.VolumeValue = &value
			}

			result.VolumeUnit = model.Volume.Unit
		}
	}

	result.ModelLabel = buildModelLabel(result)
	return result
}

func isEmptyModel(
	model modelSimple,
) bool {
	return model.Kind == "" &&
		model.ModelNumber == "" &&
		model.ModelLabel == "" &&
		model.Size == "" &&
		model.Color == "" &&
		model.VolumeValue == nil &&
		model.VolumeUnit == ""
}

func buildModelLabel(
	model modelSimple,
) string {
	if model.Kind == "alcohol" {
		if model.ModelNumber != "" &&
			model.VolumeValue != nil &&
			model.VolumeUnit != "" {
			return fmt.Sprintf(
				"%s / %d%s",
				model.ModelNumber,
				*model.VolumeValue,
				model.VolumeUnit,
			)
		}

		if model.VolumeValue != nil &&
			model.VolumeUnit != "" {
			return fmt.Sprintf(
				"%d%s",
				*model.VolumeValue,
				model.VolumeUnit,
			)
		}

		return model.ModelNumber
	}

	if model.Kind == "apparel" ||
		model.Kind == "" {
		if model.Size != "" &&
			model.Color != "" {
			return fmt.Sprintf(
				"%s / %s",
				model.Size,
				model.Color,
			)
		}

		if model.Size != "" {
			return model.Size
		}
		if model.Color != "" {
			return model.Color
		}
	}

	return model.ModelNumber
}

func firstNonEmptyString(
	values ...string,
) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func (q *CartQuery) fetchProductNames(
	ctx context.Context,
	cart *cartdom.Cart,
	inventoryIndex map[string]invParts,
	resaleIndex map[string]resaleMeta,
	resaleDisplayIndex map[string]resaleDisplayMeta,
) map[string]string {
	if q == nil ||
		q.ProductBlueprintRepo == nil ||
		cart == nil ||
		len(cart.Items) == 0 {
		return nil
	}

	result := map[string]string{}
	seen := map[string]struct{}{}

	for _, item := range cart.Items {
		productBlueprintID := ""

		switch mallshared.InferCartItemType(item) {
		case cartdom.CartItemTypeList:
			if parts, ok :=
				inventoryIndex[item.InventoryID]; ok {
				productBlueprintID =
					parts.ProductBlueprintID
			}

		case cartdom.CartItemTypeResale:
			if metadata, ok :=
				resaleIndex[item.ResaleID]; ok {
				productBlueprintID =
					metadata.ProductBlueprintID
			}

			if productBlueprintID == "" {
				if display, ok :=
					resaleDisplayIndex[item.ResaleID]; ok {
					productBlueprintID =
						display.ProductBlueprintID
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

		productBlueprint, err :=
			q.ProductBlueprintRepo.GetByID(
				ctx,
				productBlueprintID,
			)
		if err != nil {
			continue
		}

		if productBlueprint.ProductName != "" {
			result[productBlueprintID] =
				productBlueprint.ProductName
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}
