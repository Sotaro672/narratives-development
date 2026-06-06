// backend/internal/application/query/mall/cart_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	malldto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"
	cartdom "narratives/internal/domain/cart"
	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
)

// ListReader is the minimal list read port required by CartQuery.
//
// ldom.Repository satisfies this interface.
type ListReader interface {
	GetByID(ctx context.Context, id string) (ldom.List, error)
}

type CartQuery struct {
	FS *firestore.Client

	// ListReader is used to resolve list title, image, and prices.
	ListRepo ListReader

	// InventoryRepo is used to resolve inventory document ID to blueprint IDs.
	InventoryRepo invdom.RepositoryPort

	// optional: inject from DI
	Resolver *appresolver.NameResolver

	CartCol string
}

func NewCartQuery(fs *firestore.Client) *CartQuery {
	return &CartQuery{
		FS:            fs,
		ListRepo:      nil,
		InventoryRepo: nil,
		Resolver:      nil,
		CartCol:       "carts",
	}
}

func NewCartQueryWithListRepo(fs *firestore.Client, listRepo ListReader) *CartQuery {
	q := NewCartQuery(fs)
	q.ListRepo = listRepo
	return q
}

func NewCartQueryWithRepos(
	fs *firestore.Client,
	listRepo ListReader,
	inventoryRepo invdom.RepositoryPort,
) *CartQuery {
	q := NewCartQuery(fs)
	q.ListRepo = listRepo
	q.InventoryRepo = inventoryRepo
	return q
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
	if q == nil || q.FS == nil {
		return malldto.CartDTO{}, errors.New("mall cart query: firestore client is nil")
	}

	if avatarID == "" {
		return malldto.CartDTO{}, errors.New("avatarId is required")
	}

	cartCol := q.CartCol
	if cartCol == "" {
		cartCol = "carts"
	}

	snap, err := q.FS.Collection(cartCol).Doc(avatarID).Get(ctx)
	if err != nil {
		if isFirestoreNotFound(err) {
			return malldto.CartDTO{}, ErrNotFound
		}
		return malldto.CartDTO{}, err
	}
	if snap == nil || !snap.Exists() {
		return malldto.CartDTO{}, ErrNotFound
	}

	// carts doc は domain/cart.Cart の firestore tag 付き構造を正とする。
	c, perr := cartFromSnapshot(avatarID, snap)
	if perr != nil {
		return malldto.CartDTO{}, perr
	}

	priceIndex, listMetaIndex := q.fetchListIndicesByCart(ctx, c)
	invIndex := q.fetchInventoryIndexByCart(ctx, c)
	modelIndex := q.fetchModelSimpleIndexByCart(ctx, c)
	productNameIndex := q.fetchProductNameIndexByCart(ctx, c, invIndex)

	out := toCartDTO(
		c,
		priceIndex,
		listMetaIndex,
		invIndex,
		modelIndex,
		productNameIndex,
	)

	return out, nil
}

// ============================================================
// cart snapshot parsing (current schema only)
// ============================================================

// carts doc supported shape:
// - docId = avatarId
// - items: map[itemKey] = {inventoryId, listId, modelId, qty}
// - createdAt / updatedAt / expiresAt are Firestore timestamps
func cartFromSnapshot(avatarID string, snap *firestore.DocumentSnapshot) (*cartdom.Cart, error) {
	if snap == nil {
		return nil, errors.New("mall cart query: snapshot is nil")
	}

	c := &cartdom.Cart{}
	if err := snap.DataTo(c); err != nil {
		return nil, err
	}

	// Cart.ID is firestore:"-" and must come from docId.
	c.ID = avatarID

	if c.Items == nil {
		c.Items = map[string]cartdom.CartItem{}
		return c, nil
	}

	// Query 側では itemKey を分解しない。
	// 正規 CartItem の中身だけを見て、不正 item は read-model から除外する。
	items := map[string]cartdom.CartItem{}
	for itemKey, it := range c.Items {
		if itemKey == "" {
			continue
		}
		if it.InventoryID == "" || it.ListID == "" || it.ModelID == "" || it.Qty <= 0 {
			continue
		}

		items[itemKey] = cartdom.CartItem{
			InventoryID: it.InventoryID,
			ListID:      it.ListID,
			ModelID:     it.ModelID,
			Qty:         it.Qty,
		}
	}
	c.Items = items

	return c, nil
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

func toCartDTO(
	c *cartdom.Cart,
	priceIndex map[string]map[string]int,
	listMetaIndex map[string]listMeta,
	invIndex map[string]invParts,
	modelIndex map[string]modelSimple,
	productNameIndex map[string]string,
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

		invID := it.InventoryID
		listID := it.ListID
		modelID := it.ModelID
		if invID == "" || listID == "" || modelID == "" || it.Qty <= 0 {
			continue
		}

		item := malldto.CartItemDTO{
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
			}
		}

		if pbID != "" && productNameIndex != nil {
			if name, ok := productNameIndex[pbID]; ok && name != "" {
				item.ProductName = name
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

		out.Items[key] = item
	}

	return out
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

func (q *CartQuery) fetchListIndicesByCart(
	ctx context.Context,
	c *cartdom.Cart,
) (map[string]map[string]int, map[string]listMeta) {
	if q == nil || q.ListRepo == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	listIDs := make([]string, 0, 8)

	for _, it := range c.Items {
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

func (q *CartQuery) fetchInventoryIndexByCart(
	ctx context.Context,
	c *cartdom.Cart,
) map[string]invParts {
	if q == nil || q.InventoryRepo == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	invIDs := make([]string, 0, 8)

	for _, it := range c.Items {
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
// model resolver lookup
// ============================================================

func (q *CartQuery) fetchModelSimpleIndexByCart(
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

			Size:  mr.Size,
			Color: mr.Color,

			VolumeValue: mr.VolumeValue,
			VolumeUnit:  mr.VolumeUnit,
		}

		ms.ModelLabel = buildCartModelLabel(ms)

		if isEmptyModelSimple(ms) {
			continue
		}

		out[mid] = ms
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func isEmptyModelSimple(ms modelSimple) bool {
	return ms.Kind == "" &&
		ms.ModelNumber == "" &&
		ms.ModelLabel == "" &&
		ms.Size == "" &&
		ms.Color == "" &&
		ms.VolumeValue == nil &&
		ms.VolumeUnit == ""
}

func buildCartModelLabel(ms modelSimple) string {
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

func (q *CartQuery) fetchProductNameIndexByCart(
	ctx context.Context,
	c *cartdom.Cart,
	invIndex map[string]invParts,
) map[string]string {
	if q == nil || q.Resolver == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	out := map[string]string{}
	seen := map[string]struct{}{}

	for _, it := range c.Items {
		invID := it.InventoryID
		if invID == "" {
			continue
		}

		pbID := ""
		if invIndex != nil {
			if parts, ok := invIndex[invID]; ok {
				pbID = parts.ProductBlueprintID
			}
		}
		if pbID == "" {
			continue
		}

		if _, ok := seen[pbID]; ok {
			continue
		}
		seen[pbID] = struct{}{}

		name := q.Resolver.ResolveProductName(ctx, pbID)
		if name != "" {
			out[pbID] = name
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
