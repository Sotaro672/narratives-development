// backend/internal/application/query/sns/cart_query.go
package sns

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	snsdto "narratives/internal/application/query/sns/dto"
	appresolver "narratives/internal/application/resolver"
	cartdom "narratives/internal/domain/cart"
	listdom "narratives/internal/domain/list"
)

// SNSCartQuery resolves (for cart list UI):
//   - avatarId -> cart (carts/{avatarId})
//   - cart.items[].listId -> title/listImage
//   - cart.items[].(listId, modelId) -> price from lists/{listId}.prices[]
//   - cart.items[].inventoryId (= productBlueprintId__tokenBlueprintId) -> productBlueprintId (best-effort)
//   - productBlueprintId -> productName (best-effort)
//   - cart.items[].modelId -> size/color (via NameResolver.ResolveModelResolved)
//   - qty
//
// IMPORTANT: CartDTO returns ONLY (per item):
//
//	inventoryId, listId, modelId, title, listImage, price, productName, size, color, qty
type SNSCartQuery struct {
	FS *firestore.Client

	// ✅ optional: inject from DI
	Resolver *appresolver.NameResolver

	// collection names (override if your firestore schema differs)
	CartCol              string
	ListsCol             string
	InventoriesCol       string
	ProductBlueprintsCol string
}

func NewSNSCartQuery(fs *firestore.Client) *SNSCartQuery {
	return &SNSCartQuery{
		FS:                   fs,
		Resolver:             nil,
		CartCol:              "carts",
		ListsCol:             "lists",
		InventoriesCol:       "inventories",
		ProductBlueprintsCol: "productBlueprints",
	}
}

// GetByAvatarID fetches cart document by docId (= avatarId).
// - If not found, returns ErrNotFound (defined in order_query.go in the same package).
func (q *SNSCartQuery) GetByAvatarID(ctx context.Context, avatarID string) (snsdto.CartDTO, error) {
	if q == nil || q.FS == nil {
		return snsdto.CartDTO{}, errors.New("sns cart query: firestore client is nil")
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return snsdto.CartDTO{}, errors.New("avatarId is required")
	}

	cartCol := strings.TrimSpace(q.CartCol)
	if cartCol == "" {
		cartCol = "carts"
	}

	// ✅ carts/{avatarId}
	snap, err := q.FS.Collection(cartCol).Doc(avatarID).Get(ctx)
	if err != nil {
		if isFirestoreNotFound(err) {
			return snsdto.CartDTO{}, ErrNotFound
		}
		return snsdto.CartDTO{}, err
	}
	if snap == nil || !snap.Exists() {
		return snsdto.CartDTO{}, ErrNotFound
	}

	var c cartdom.Cart
	if derr := snap.DataTo(&c); derr != nil {
		log.Printf("[sns_cart_query] DataTo(cart) failed avatarId=%q err=%v", maskUID(avatarID), derr)
		return snsdto.CartDTO{}, derr
	}

	// ✅ Firestore docId (= avatarId) を正として必ず入れる
	c.ID = avatarID

	// ✅ listId -> (modelId -> price) AND listId -> (title/listImage)
	priceIndex, listMetaIndex := q.fetchListIndicesByCart(ctx, &c)

	// ✅ inventoryId -> (productBlueprintId, tokenBlueprintId) (best-effort)
	invIndex := q.fetchInventoryIndexByCart(ctx, &c)

	// ✅ modelId -> size/color (best-effort)
	modelIndex := q.fetchModelSimpleIndexByCart(ctx, &c)

	// ✅ productBlueprintId -> productName (best-effort)
	productNameIndex := q.fetchProductNameIndexByCart(ctx, &c, invIndex)

	out := toCartDTO(&c, priceIndex, listMetaIndex, invIndex, modelIndex, productNameIndex)

	log.Printf("[sns_cart_query] get ok avatarId=%q items=%d", maskUID(avatarID), len(out.Items))
	return out, nil
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
	ImageID string // lists.imageId (or List.ImageID)
}

type modelSimple struct {
	Size  string
	Color string
}

func toCartDTO(
	c *cartdom.Cart,
	priceIndex map[string]map[string]int, // listId -> (modelId -> price)
	listMetaIndex map[string]listMeta, // listId -> meta
	invIndex map[string]invParts, // inventoryId -> parts
	modelIndex map[string]modelSimple, // modelId -> (size,color)
	productNameIndex map[string]string, // productBlueprintId -> productName
) snsdto.CartDTO {
	out := snsdto.CartDTO{
		AvatarID:  strings.TrimSpace(c.ID),
		Items:     map[string]snsdto.CartItemDTO{},
		CreatedAt: toRFC3339Ptr(c.CreatedAt),
		UpdatedAt: toRFC3339Ptr(c.UpdatedAt),
		ExpiresAt: toRFC3339Ptr(c.ExpiresAt),
	}

	if c.Items == nil {
		return out
	}

	for k, it := range c.Items {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}

		invID := strings.TrimSpace(it.InventoryID)
		listID := strings.TrimSpace(it.ListID)
		modelID := strings.TrimSpace(it.ModelID)
		if invID == "" || listID == "" || modelID == "" || it.Qty <= 0 {
			continue
		}

		// ✅ 画面側がすぐ使えるように IDs も同梱して返す
		dto := snsdto.CartItemDTO{
			InventoryID: invID,
			ListID:      listID,
			ModelID:     modelID,
			Qty:         it.Qty,
		}

		// --------------------------
		// list meta: title / listImage
		// --------------------------
		if listMetaIndex != nil {
			if lm, ok := listMetaIndex[listID]; ok {
				if s := strings.TrimSpace(lm.Title); s != "" {
					dto.Title = s
				}
				if s := strings.TrimSpace(lm.ImageID); s != "" {
					dto.ListImage = s
				}
			}
		}

		// --------------------------
		// price: listId -> modelId
		// --------------------------
		if priceIndex != nil {
			if m, ok := priceIndex[listID]; ok {
				if p, ok2 := m[modelID]; ok2 {
					pp := p
					dto.Price = &pp
				}
			}
		}

		// --------------------------
		// productName: inventoryId -> pbId -> name
		// --------------------------
		pbID := ""
		if invIndex != nil {
			if parts, ok := invIndex[invID]; ok {
				pbID = strings.TrimSpace(parts.ProductBlueprintID)
			}
		}
		if pbID == "" {
			if p, _, ok := parseInventoryID(invID); ok {
				pbID = p
			}
		}
		if pbID != "" && productNameIndex != nil {
			if name, ok := productNameIndex[pbID]; ok {
				if s := strings.TrimSpace(name); s != "" {
					dto.ProductName = s
				}
			}
		}

		// --------------------------
		// modelId -> size/color
		// --------------------------
		if modelIndex != nil {
			if ms, ok := modelIndex[modelID]; ok {
				if s := strings.TrimSpace(ms.Size); s != "" {
					dto.Size = s
				}
				if s := strings.TrimSpace(ms.Color); s != "" {
					dto.Color = s
				}
			}
		}

		out.Items[key] = dto
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
// list lookup (best-effort)
// - listId -> prices(modelId->price) and title/imageId
// ============================================================

func (q *SNSCartQuery) fetchListIndicesByCart(ctx context.Context, c *cartdom.Cart) (map[string]map[string]int, map[string]listMeta) {
	if q == nil || q.FS == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil, nil
	}

	listsCol := strings.TrimSpace(q.ListsCol)
	if listsCol == "" {
		listsCol = "lists"
	}

	seen := map[string]struct{}{}
	listIDs := make([]string, 0, 8)

	for _, it := range c.Items {
		lid := strings.TrimSpace(it.ListID)
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

	refs := make([]*firestore.DocumentRef, 0, len(listIDs))
	for _, lid := range listIDs {
		refs = append(refs, q.FS.Collection(listsCol).Doc(lid))
	}

	snaps, err := q.FS.GetAll(ctx, refs)
	if err != nil {
		log.Printf("[sns_cart_query] GetAll(lists) failed listIds=%d err=%v", len(refs), err)
		return nil, nil
	}

	priceOut := map[string]map[string]int{}
	metaOut := map[string]listMeta{}

	for i, snap := range snaps {
		lid := ""
		if i >= 0 && i < len(listIDs) {
			lid = strings.TrimSpace(listIDs[i])
		}
		if lid == "" || snap == nil || !snap.Exists() {
			continue
		}

		// Prefer struct decode
		var l listdom.List
		if derr := snap.DataTo(&l); derr == nil {
			mt := listMeta{
				Title:   strings.TrimSpace(l.Title),
				ImageID: strings.TrimSpace(l.ImageID),
			}
			if mt.Title != "" || mt.ImageID != "" {
				metaOut[lid] = mt
			}

			if len(l.Prices) > 0 {
				m := map[string]int{}
				for _, row := range l.Prices {
					mid := strings.TrimSpace(row.ModelID)
					if mid == "" {
						continue
					}
					m[mid] = row.Price
				}
				if len(m) > 0 {
					priceOut[lid] = m
				}
			}
			continue
		}

		// Fallback map read
		m := snap.Data()
		title := pickString(m, "title", "Title")
		image := pickString(m, "imageId", "ImageID", "imageID", "ImageId")
		if strings.TrimSpace(title) != "" || strings.TrimSpace(image) != "" {
			metaOut[lid] = listMeta{Title: strings.TrimSpace(title), ImageID: strings.TrimSpace(image)}
		}

		// prices: [{modelId, price}]
		if raw, ok := m["prices"]; ok {
			rows, _ := raw.([]any)
			if len(rows) > 0 {
				pm := map[string]int{}
				for _, row := range rows {
					rm, _ := row.(map[string]any)
					if rm == nil {
						continue
					}
					mid := strings.TrimSpace(pickString(rm, "modelId", "ModelID", "modelID", "ModelId"))
					if mid == "" {
						continue
					}
					pv := pickAny(rm, "price", "Price")
					if p, ok := asIntAny(pv); ok {
						pm[mid] = p
					}
				}
				if len(pm) > 0 {
					priceOut[lid] = pm
				}
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
// inventory lookup (best-effort)
// - inventoryId -> (productBlueprintId, tokenBlueprintId)
// ============================================================

func (q *SNSCartQuery) fetchInventoryIndexByCart(ctx context.Context, c *cartdom.Cart) map[string]invParts {
	if q == nil || q.FS == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	invCol := strings.TrimSpace(q.InventoriesCol)
	if invCol == "" {
		invCol = "inventories"
	}

	seen := map[string]struct{}{}
	invIDs := make([]string, 0, 8)

	for _, it := range c.Items {
		inv := strings.TrimSpace(it.InventoryID)
		if inv == "" {
			continue
		}
		if _, ok := seen[inv]; ok {
			continue
		}
		seen[inv] = struct{}{}
		invIDs = append(invIDs, inv)
	}

	if len(invIDs) == 0 {
		return nil
	}

	refs := make([]*firestore.DocumentRef, 0, len(invIDs))
	for _, id := range invIDs {
		refs = append(refs, q.FS.Collection(invCol).Doc(id))
	}

	snaps, err := q.FS.GetAll(ctx, refs)
	if err != nil {
		log.Printf("[sns_cart_query] GetAll(inventories) failed invIds=%d err=%v", len(refs), err)
		// best-effort: allow parsing from inventoryId (= pb__tb)
		return nil
	}

	out := map[string]invParts{}

	for i, snap := range snaps {
		invID := ""
		if i >= 0 && i < len(invIDs) {
			invID = strings.TrimSpace(invIDs[i])
		}
		if invID == "" || snap == nil || !snap.Exists() {
			continue
		}

		m := snap.Data()
		pb := pickString(m, "productBlueprintId", "productBlueprintID", "ProductBlueprintID", "ProductBlueprintId")
		tb := pickString(m, "tokenBlueprintId", "tokenBlueprintID", "TokenBlueprintID", "TokenBlueprintId")

		if pb == "" || tb == "" {
			if p, t, ok := parseInventoryID(invID); ok {
				if pb == "" {
					pb = p
				}
				if tb == "" {
					tb = t
				}
			}
		}

		pb = strings.TrimSpace(pb)
		tb = strings.TrimSpace(tb)
		if pb == "" && tb == "" {
			continue
		}

		out[invID] = invParts{ProductBlueprintID: pb, TokenBlueprintID: tb}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// ============================================================
// model resolver lookup (best-effort)
// - modelId -> size/color only
// ============================================================

func (q *SNSCartQuery) fetchModelSimpleIndexByCart(ctx context.Context, c *cartdom.Cart) map[string]modelSimple {
	if q == nil || q.Resolver == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	modelIDs := make([]string, 0, 16)

	for _, it := range c.Items {
		mid := strings.TrimSpace(it.ModelID)
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
		sz := strings.TrimSpace(mr.Size)
		cl := strings.TrimSpace(mr.Color)
		if sz == "" && cl == "" {
			continue
		}
		out[mid] = modelSimple{Size: sz, Color: cl}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// ============================================================
// productName lookup (best-effort)
// - inventoryId -> pbId -> productName
// ============================================================

func (q *SNSCartQuery) fetchProductNameIndexByCart(
	ctx context.Context,
	c *cartdom.Cart,
	invIndex map[string]invParts,
) map[string]string {
	if q == nil || q.FS == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}

	pbCol := strings.TrimSpace(q.ProductBlueprintsCol)
	if pbCol == "" {
		pbCol = "productBlueprints"
	}

	pbSeen := map[string]struct{}{}
	pbIDs := make([]string, 0, 16)

	for _, it := range c.Items {
		invID := strings.TrimSpace(it.InventoryID)
		if invID == "" {
			continue
		}

		pbID := ""
		if invIndex != nil {
			if parts, ok := invIndex[invID]; ok {
				pbID = strings.TrimSpace(parts.ProductBlueprintID)
			}
		}
		if pbID == "" {
			if p, _, ok := parseInventoryID(invID); ok {
				pbID = p
			}
		}

		if pbID == "" {
			continue
		}
		if _, ok := pbSeen[pbID]; ok {
			continue
		}
		pbSeen[pbID] = struct{}{}
		pbIDs = append(pbIDs, pbID)
	}

	if len(pbIDs) == 0 {
		return nil
	}

	refs := make([]*firestore.DocumentRef, 0, len(pbIDs))
	for _, id := range pbIDs {
		refs = append(refs, q.FS.Collection(pbCol).Doc(id))
	}

	snaps, err := q.FS.GetAll(ctx, refs)
	if err != nil {
		log.Printf("[sns_cart_query] GetAll(productBlueprints) failed pbIds=%d err=%v", len(refs), err)
		// fallback: resolver only (best-effort)
		if q.Resolver != nil {
			out := map[string]string{}
			for _, id := range pbIDs {
				name := strings.TrimSpace(q.Resolver.ResolveProductName(ctx, id))
				if name != "" {
					out[id] = name
				}
			}
			if len(out) == 0 {
				return nil
			}
			return out
		}
		return nil
	}

	out := map[string]string{}

	for i, snap := range snaps {
		pbID := ""
		if i >= 0 && i < len(pbIDs) {
			pbID = strings.TrimSpace(pbIDs[i])
		}
		if pbID == "" || snap == nil || !snap.Exists() {
			continue
		}

		m := snap.Data()
		name := strings.TrimSpace(pickString(m, "productName", "ProductName", "name", "Name"))

		// fallback resolver
		if name == "" && q.Resolver != nil {
			name = strings.TrimSpace(q.Resolver.ResolveProductName(ctx, pbID))
		}

		if name == "" {
			continue
		}
		out[pbID] = name
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// ============================================================
// shared helpers (package scope; preview_query からも利用)
// ============================================================

// inventoryId is expected: productBlueprintId__tokenBlueprintId
func parseInventoryID(inventoryID string) (productBlueprintID string, tokenBlueprintID string, ok bool) {
	s := strings.TrimSpace(inventoryID)
	if s == "" {
		return "", "", false
	}

	parts := strings.Split(s, "__")
	if len(parts) != 2 {
		return "", "", false
	}

	pb := strings.TrimSpace(parts[0])
	tb := strings.TrimSpace(parts[1])
	if pb == "" || tb == "" {
		return "", "", false
	}
	return pb, tb, true
}

func pickString(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func pickAny(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func asIntAny(v any) (int, bool) {
	if v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case int:
		return x, true
	case int8:
		return int(x), true
	case int16:
		return int(x), true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint:
		return int(x), true
	case uint8:
		return int(x), true
	case uint16:
		return int(x), true
	case uint32:
		return int(x), true
	case uint64:
		return int(x), true
	case float32:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		var n int
		_, err := fmt.Sscanf(s, "%d", &n)
		return n, err == nil
	default:
		return 0, false
	}
}
