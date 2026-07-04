// backend/internal/application/query/mall/cart_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"time"

	malldto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"
	cartdom "narratives/internal/domain/cart"
	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	resaledom "narratives/internal/domain/resale"
)

// CartReader is the minimal cart read port required by CartQuery.
//
// cartdom.Repository satisfies this interface.
type CartReader interface {
	GetByAvatarID(ctx context.Context, avatarID string) (*cartdom.Cart, error)
}

// ListReader is the minimal list read port required by CartQuery.
//
// ldom.Repository satisfies this interface.
type ListReader interface {
	GetByID(ctx context.Context, id string) (ldom.List, error)
}

// ResaleReader is the minimal resale read port required by CartQuery.
//
// resaledom.Repository satisfies this interface.
type ResaleReader interface {
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
}

// ResaleImageReader is the minimal resale image read port required by CartQuery.
//
// resaledom.ImageRepository satisfies this interface.
type ResaleImageReader interface {
	ListByResaleID(ctx context.Context, resaleID string) ([]resaledom.ResaleImage, error)
}

type CartQuery struct {
	// CartRepo is used to read the cart document.
	CartRepo CartReader

	// ListReader is used to resolve list title, image, and prices.
	ListRepo ListReader

	// InventoryRepo is used to resolve inventory document ID to blueprint IDs.
	InventoryRepo invdom.RepositoryPort

	// ProductBlueprintRepo is used to resolve productName from productBlueprintId.
	ProductBlueprintRepo ProductBlueprintReader

	// ResaleRepo is used to resolve resale price and related product/token/brand IDs.
	ResaleRepo ResaleReader

	// ResaleImageRepo is used to resolve resale display image.
	ResaleImageRepo ResaleImageReader

	Resolver *appresolver.NameResolver
}

func NewCartQuery(
	cartRepo CartReader,
	listRepo ListReader,
	inventoryRepo invdom.RepositoryPort,
	productBlueprintRepo ProductBlueprintReader,
	resaleRepo ResaleReader,
	resaleImageRepo ResaleImageReader,
	resolver *appresolver.NameResolver,
) *CartQuery {
	return &CartQuery{
		CartRepo:             cartRepo,
		ListRepo:             listRepo,
		InventoryRepo:        inventoryRepo,
		ProductBlueprintRepo: productBlueprintRepo,
		ResaleRepo:           resaleRepo,
		ResaleImageRepo:      resaleImageRepo,
		Resolver:             resolver,
	}
}

// CartHandler 側の CartQueryService（GetCartQuery）に明示的に合わせる。
type cartQueryPort interface {
	GetCartQuery(ctx context.Context, avatarID string) (any, error)
}

var _ cartQueryPort = (*CartQuery)(nil)

func (q *CartQuery) GetCartQuery(ctx context.Context, avatarID string) (any, error) {
	return q.GetByAvatarID(ctx, avatarID)
}

func (q *CartQuery) GetByAvatarID(ctx context.Context, avatarID string) (malldto.CartDTO, error) {
	if q == nil || q.CartRepo == nil {
		return malldto.CartDTO{}, errors.New("mall cart query: cart repo is nil")
	}

	if avatarID == "" {
		return malldto.CartDTO{}, errors.New("avatarId is required")
	}

	c, err := q.CartRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		return malldto.CartDTO{}, err
	}
	if c == nil {
		return malldto.CartDTO{}, ErrNotFound
	}

	// Cart.ID は Firestore docId (= avatarId) が正。
	// repository 側で未設定だった場合でも read-model では avatarID を補完する。
	if c.ID == "" {
		c.ID = avatarID
	}

	// Query 側では itemKey を分解しない。
	// CartItem の中身だけを見て、不正 item は read-model から除外する。
	// list item と resale item の両方を保持する。
	c = normalizeCart(c)

	priceIndex, listMetaIndex := q.fetchLists(ctx, c)
	invIndex := q.fetchInventories(ctx, c)
	modelIndex := q.fetchModels(ctx, c)
	resaleIndex := q.fetchResales(ctx, c)
	resaleImageIndex := q.fetchResaleImages(ctx, c)
	productNameIndex := q.fetchProductNames(ctx, c, invIndex, resaleIndex)

	out := toCartDTO(
		c,
		priceIndex,
		listMetaIndex,
		invIndex,
		modelIndex,
		productNameIndex,
		resaleIndex,
		resaleImageIndex,
	)

	return out, nil
}

// ============================================================
// cart read-model normalization
// ============================================================

func normalizeCart(c *cartdom.Cart) *cartdom.Cart {
	if c == nil {
		return nil
	}

	if c.Items == nil {
		c.Items = map[string]cartdom.CartItem{}
		return c
	}

	items := map[string]cartdom.CartItem{}

	for itemKey, it := range c.Items {
		if itemKey == "" {
			continue
		}

		switch inferCartItemType(it) {
		case cartdom.CartItemTypeList:
			if it.InventoryID == "" || it.ListID == "" || it.ModelID == "" || it.Qty <= 0 {
				continue
			}

			items[itemKey] = cartdom.CartItem{
				Type:        cartdom.CartItemTypeList,
				InventoryID: it.InventoryID,
				ListID:      it.ListID,
				ModelID:     it.ModelID,
				Qty:         it.Qty,
			}

		case cartdom.CartItemTypeResale:
			if it.ResaleID == "" || it.ProductID == "" {
				continue
			}

			items[itemKey] = cartdom.CartItem{
				Type:      cartdom.CartItemTypeResale,
				ResaleID:  it.ResaleID,
				ProductID: it.ProductID,
				Qty:       1,
			}
		}
	}

	c.Items = items

	return c
}

func inferCartItemType(it cartdom.CartItem) cartdom.CartItemType {
	switch it.Type {
	case cartdom.CartItemTypeList, cartdom.CartItemTypeResale:
		return it.Type
	}

	if it.ResaleID != "" || it.ProductID != "" {
		return cartdom.CartItemTypeResale
	}

	if it.InventoryID != "" || it.ListID != "" || it.ModelID != "" {
		return cartdom.CartItemTypeList
	}

	return ""
}

// ============================================================
// mappers
// ============================================================

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

	// apparel
	Size  string
	Color string

	// alcohol
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

func toCartDTO(
	c *cartdom.Cart,
	priceIndex map[string]map[string]int,
	listMetaIndex map[string]listMeta,
	invIndex map[string]invParts,
	modelIndex map[string]modelSimple,
	productNameIndex map[string]string,
	resaleIndex map[string]resaleMeta,
	resaleImageIndex map[string]string,
) malldto.CartDTO {
	out := malldto.CartDTO{
		AvatarID:  c.ID,
		Items:     map[string]malldto.CartItemDTO{},
		CreatedAt: toRFC3339Ptr(c.CreatedAt),
		UpdatedAt: toRFC3339Ptr(c.UpdatedAt),
		ExpiresAt: toRFC3339Ptr(c.ExpiresAt),
	}

	if c.Items == nil {
		return out
	}

	for k, it := range c.Items {
		key := k
		if key == "" {
			continue
		}

		switch inferCartItemType(it) {
		case cartdom.CartItemTypeList:
			item, ok := toListCartItemDTO(
				it,
				priceIndex,
				listMetaIndex,
				invIndex,
				modelIndex,
				productNameIndex,
			)
			if !ok {
				continue
			}

			out.Items[key] = item

		case cartdom.CartItemTypeResale:
			item, ok := toResaleCartItemDTO(
				it,
				resaleIndex,
				resaleImageIndex,
				productNameIndex,
			)
			if !ok {
				continue
			}

			out.Items[key] = item
		}
	}

	return out
}

func toListCartItemDTO(
	it cartdom.CartItem,
	priceIndex map[string]map[string]int,
	listMetaIndex map[string]listMeta,
	invIndex map[string]invParts,
	modelIndex map[string]modelSimple,
	productNameIndex map[string]string,
) (malldto.CartItemDTO, bool) {
	invID := it.InventoryID
	listID := it.ListID
	modelID := it.ModelID

	if invID == "" || listID == "" || modelID == "" || it.Qty <= 0 {
		return malldto.CartItemDTO{}, false
	}

	item := malldto.CartItemDTO{
		Type:        string(cartdom.CartItemTypeList),
		InventoryID: invID,
		ListID:      listID,
		ModelID:     modelID,
		Qty:         it.Qty,
	}

	if listMetaIndex != nil {
		if lm, ok := listMetaIndex[listID]; ok {
			if lm.Title != "" {
				item.Title = lm.Title
			}
			if lm.ImageID != "" {
				item.ListImage = lm.ImageID
			}
		}
	}

	if priceIndex != nil {
		if m, ok := priceIndex[listID]; ok {
			if p, ok2 := m[modelID]; ok2 {
				pp := p
				item.Price = &pp
			}
		}
	}

	pbID := ""
	if invIndex != nil {
		if parts, ok := invIndex[invID]; ok {
			pbID = parts.ProductBlueprintID
			item.ProductBlueprintID = parts.ProductBlueprintID
			item.TokenBlueprintID = parts.TokenBlueprintID
		}
	}

	if pbID != "" && productNameIndex != nil {
		if name, ok := productNameIndex[pbID]; ok && name != "" {
			item.ProductName = name

			if item.Title == "" {
				item.Title = name
			}
		}
	}

	if modelIndex != nil {
		if ms, ok := modelIndex[modelID]; ok {
			if ms.Kind != "" {
				item.ModelKind = ms.Kind
			}
			if ms.ModelNumber != "" {
				item.ModelNumber = ms.ModelNumber
			}
			if ms.ModelLabel != "" {
				item.ModelLabel = ms.ModelLabel
			}

			if ms.Size != "" {
				item.Size = ms.Size
			}
			if ms.Color != "" {
				item.Color = ms.Color
			}

			if ms.VolumeValue != nil {
				item.VolumeValue = ms.VolumeValue
			}
			if ms.VolumeUnit != "" {
				item.VolumeUnit = ms.VolumeUnit
			}
		}
	}

	return item, true
}

func toResaleCartItemDTO(
	it cartdom.CartItem,
	resaleIndex map[string]resaleMeta,
	resaleImageIndex map[string]string,
	productNameIndex map[string]string,
) (malldto.CartItemDTO, bool) {
	if it.ResaleID == "" || it.ProductID == "" {
		return malldto.CartItemDTO{}, false
	}

	item := malldto.CartItemDTO{
		Type:      string(cartdom.CartItemTypeResale),
		ResaleID:  it.ResaleID,
		ProductID: it.ProductID,
		Qty:       1,
	}

	pbID := ""

	if resaleIndex != nil {
		if meta, ok := resaleIndex[it.ResaleID]; ok {
			if meta.ProductID != "" {
				item.ProductID = meta.ProductID
			}

			if meta.ProductBlueprintID != "" {
				item.ProductBlueprintID = meta.ProductBlueprintID
				pbID = meta.ProductBlueprintID
			}

			if meta.TokenBlueprintID != "" {
				item.TokenBlueprintID = meta.TokenBlueprintID
			}

			if meta.BrandID != "" {
				item.BrandID = meta.BrandID
			}

			price := meta.Price
			item.Price = &price
		}
	}

	if resaleImageIndex != nil {
		if imageURL, ok := resaleImageIndex[it.ResaleID]; ok && imageURL != "" {
			item.ImageURL = imageURL

			// 既存 frontend が listImage を見ている場合でも表示されるように同じ URL を入れる。
			item.ListImage = imageURL
		}
	}

	if pbID != "" && productNameIndex != nil {
		if name, ok := productNameIndex[pbID]; ok && name != "" {
			item.ProductName = name
			item.Title = name
		}
	}

	return item, true
}

func toRFC3339Ptr(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

// ============================================================
// list lookup
// ============================================================

func (q *CartQuery) fetchLists(
	ctx context.Context,
	c *cartdom.Cart,
) (map[string]map[string]int, map[string]listMeta) {
	if q == nil || q.ListRepo == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	listIDs := make([]string, 0, 8)

	for _, it := range c.Items {
		if inferCartItemType(it) != cartdom.CartItemTypeList {
			continue
		}

		lid := it.ListID
		if lid == "" {
			continue
		}
		if _, ok := seen[lid]; ok {
			continue
		}
		seen[lid] = struct{}{}
		listIDs = append(listIDs, lid)
	}

	if len(listIDs) == 0 {
		return nil, nil
	}

	priceOut := map[string]map[string]int{}
	metaOut := map[string]listMeta{}

	for _, lid0 := range listIDs {
		lid := lid0
		if lid == "" {
			continue
		}

		l, err := q.ListRepo.GetByID(ctx, lid)
		if err != nil {
			continue
		}

		mt := listMeta{
			Title:   l.Title,
			ImageID: l.ImageID,
		}
		if mt.Title != "" || mt.ImageID != "" {
			metaOut[lid] = mt
		}

		if len(l.Prices) > 0 {
			m := map[string]int{}
			for _, row := range l.Prices {
				mid := row.ModelID
				if mid == "" {
					continue
				}
				m[mid] = row.Price
			}
			if len(m) > 0 {
				priceOut[lid] = m
			}
		}
	}

	if len(priceOut) == 0 {
		priceOut = nil
	}
	if len(metaOut) == 0 {
		metaOut = nil
	}

	return priceOut, metaOut
}

// ============================================================
// inventory lookup
// ============================================================

func (q *CartQuery) fetchInventories(
	ctx context.Context,
	c *cartdom.Cart,
) map[string]invParts {
	if q == nil || q.InventoryRepo == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	invIDs := make([]string, 0, 8)

	for _, it := range c.Items {
		if inferCartItemType(it) != cartdom.CartItemTypeList {
			continue
		}

		invID := it.InventoryID
		if invID == "" {
			continue
		}
		if _, ok := seen[invID]; ok {
			continue
		}
		seen[invID] = struct{}{}
		invIDs = append(invIDs, invID)
	}

	if len(invIDs) == 0 {
		return nil
	}

	out := map[string]invParts{}

	for _, invID := range invIDs {
		productBlueprintID, tokenBlueprintID, err :=
			q.InventoryRepo.ResolveBlueprintIDsByInventoryID(ctx, invID)
		if err != nil {
			// Cart read-model では、削除済み・不正な inventory は該当 item の補助情報だけ欠落させる。
			// ここで全体エラーにすると cart 表示全体が落ちるため、既存実装と同様に skip する。
			continue
		}

		if productBlueprintID == "" || tokenBlueprintID == "" {
			continue
		}

		out[invID] = invParts{
			ProductBlueprintID: productBlueprintID,
			TokenBlueprintID:   tokenBlueprintID,
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// ============================================================
// resale lookup
// ============================================================

func (q *CartQuery) fetchResales(
	ctx context.Context,
	c *cartdom.Cart,
) map[string]resaleMeta {
	if q == nil || q.ResaleRepo == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	resaleIDs := make([]string, 0, 8)

	for _, it := range c.Items {
		if inferCartItemType(it) != cartdom.CartItemTypeResale {
			continue
		}

		rid := it.ResaleID
		if rid == "" {
			continue
		}
		if _, ok := seen[rid]; ok {
			continue
		}
		seen[rid] = struct{}{}
		resaleIDs = append(resaleIDs, rid)
	}

	if len(resaleIDs) == 0 {
		return nil
	}

	out := map[string]resaleMeta{}

	for _, rid := range resaleIDs {
		r, err := q.ResaleRepo.GetByID(ctx, rid)
		if err != nil {
			// resale が削除済み等の場合でも、cart 全体は落とさず補助情報だけ欠落させる。
			continue
		}

		if r.ID == "" {
			r.ID = rid
		}

		out[rid] = resaleMeta{
			ID:                 r.ID,
			Price:              r.Price,
			ProductID:          r.ProductID,
			ProductBlueprintID: r.ProductBlueprintID,
			TokenBlueprintID:   r.TokenBlueprintID,
			BrandID:            r.BrandID,
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func (q *CartQuery) fetchResaleImages(
	ctx context.Context,
	c *cartdom.Cart,
) map[string]string {
	if q == nil || q.ResaleImageRepo == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	resaleIDs := make([]string, 0, 8)

	for _, it := range c.Items {
		if inferCartItemType(it) != cartdom.CartItemTypeResale {
			continue
		}

		rid := it.ResaleID
		if rid == "" {
			continue
		}
		if _, ok := seen[rid]; ok {
			continue
		}
		seen[rid] = struct{}{}
		resaleIDs = append(resaleIDs, rid)
	}

	if len(resaleIDs) == 0 {
		return nil
	}

	out := map[string]string{}

	for _, rid := range resaleIDs {
		images, err := q.ResaleImageRepo.ListByResaleID(ctx, rid)
		if err != nil {
			continue
		}

		imageURL := firstResaleImageURL(images)
		if imageURL == "" {
			continue
		}

		out[rid] = imageURL
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func firstResaleImageURL(images []resaledom.ResaleImage) string {
	if len(images) == 0 {
		return ""
	}

	var fallback string

	for _, img := range images {
		if img.URL == "" {
			continue
		}

		if fallback == "" {
			fallback = img.URL
		}

		if img.DisplayOrder == 0 {
			return img.URL
		}
	}

	return fallback
}

// ============================================================
// model resolver lookup
// ============================================================

func (q *CartQuery) fetchModels(
	ctx context.Context,
	c *cartdom.Cart,
) map[string]modelSimple {
	if q == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}
	if q.Resolver == nil {
		return nil
	}

	seen := map[string]struct{}{}
	modelIDs := make([]string, 0, 16)

	for _, it := range c.Items {
		if inferCartItemType(it) != cartdom.CartItemTypeList {
			continue
		}

		mid := it.ModelID
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		modelIDs = append(modelIDs, mid)
	}

	if len(modelIDs) == 0 {
		return nil
	}

	out := map[string]modelSimple{}

	for _, mid := range modelIDs {
		mr := q.Resolver.ResolveModelResolved(ctx, mid)

		ms := modelSimple{
			Kind:        mr.Kind,
			ModelNumber: mr.ModelNumber,
			Size:        mr.Size,
			Color:       mr.Color,
			VolumeValue: mr.VolumeValue,
			VolumeUnit:  mr.VolumeUnit,
		}

		ms.ModelLabel = buildModelLabel(ms)

		if isEmptyModel(ms) {
			continue
		}

		out[mid] = ms
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func isEmptyModel(ms modelSimple) bool {
	return ms.Kind == "" &&
		ms.ModelNumber == "" &&
		ms.ModelLabel == "" &&
		ms.Size == "" &&
		ms.Color == "" &&
		ms.VolumeValue == nil &&
		ms.VolumeUnit == ""
}

func buildModelLabel(ms modelSimple) string {
	if ms.Kind == "alcohol" {
		if ms.ModelNumber != "" && ms.VolumeValue != nil && ms.VolumeUnit != "" {
			return fmt.Sprintf("%s / %d%s", ms.ModelNumber, *ms.VolumeValue, ms.VolumeUnit)
		}

		if ms.VolumeValue != nil && ms.VolumeUnit != "" {
			return fmt.Sprintf("%d%s", *ms.VolumeValue, ms.VolumeUnit)
		}

		if ms.ModelNumber != "" {
			return ms.ModelNumber
		}

		return ""
	}

	if ms.Kind == "apparel" || ms.Kind == "" {
		if ms.Size != "" && ms.Color != "" {
			return fmt.Sprintf("%s / %s", ms.Size, ms.Color)
		}

		if ms.Size != "" {
			return ms.Size
		}

		if ms.Color != "" {
			return ms.Color
		}
	}

	if ms.ModelNumber != "" {
		return ms.ModelNumber
	}

	return ""
}

// ============================================================
// productName lookup
// ============================================================

func (q *CartQuery) fetchProductNames(
	ctx context.Context,
	c *cartdom.Cart,
	invIndex map[string]invParts,
	resaleIndex map[string]resaleMeta,
) map[string]string {
	if q == nil || q.ProductBlueprintRepo == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	out := map[string]string{}
	seen := map[string]struct{}{}

	for _, it := range c.Items {
		pbID := ""

		switch inferCartItemType(it) {
		case cartdom.CartItemTypeList:
			invID := it.InventoryID
			if invID == "" {
				continue
			}

			if invIndex != nil {
				if parts, ok := invIndex[invID]; ok {
					pbID = parts.ProductBlueprintID
				}
			}

		case cartdom.CartItemTypeResale:
			resaleID := it.ResaleID
			if resaleID == "" {
				continue
			}

			if resaleIndex != nil {
				if meta, ok := resaleIndex[resaleID]; ok {
					pbID = meta.ProductBlueprintID
				}
			}

		default:
			continue
		}

		if pbID == "" {
			continue
		}

		if _, ok := seen[pbID]; ok {
			continue
		}
		seen[pbID] = struct{}{}

		pb, err := q.ProductBlueprintRepo.GetByID(ctx, pbID)
		if err != nil {
			continue
		}

		if pb.ProductName != "" {
			out[pbID] = pb.ProductName
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
