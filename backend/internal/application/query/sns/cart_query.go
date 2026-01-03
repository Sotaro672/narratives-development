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
	ldom "narratives/internal/domain/list"
)

type SNSCartQuery struct {
	FS *firestore.Client

	// ✅ prefer domain repository (same as catalog_query)
	ListRepo ldom.Repository

	// ✅ optional: inject from DI
	Resolver *appresolver.NameResolver

	CartCol              string
	ListsCol             string // fallback only (when ListRepo is nil)
	InventoriesCol       string
	ProductBlueprintsCol string
}

func NewSNSCartQuery(fs *firestore.Client) *SNSCartQuery {
	return &SNSCartQuery{
		FS:                   fs,
		ListRepo:             nil,
		Resolver:             nil,
		CartCol:              "carts",
		ListsCol:             "lists",
		InventoriesCol:       "inventories",
		ProductBlueprintsCol: "product_blueprints",
	}
}

func NewSNSCartQueryWithListRepo(fs *firestore.Client, listRepo ldom.Repository) *SNSCartQuery {
	q := NewSNSCartQuery(fs)
	q.ListRepo = listRepo
	return q
}

// ✅ CartHandler 側の CartQueryService（GetCartQuery）に “明示的に” 合わせる。
// これで reflect 探索に依存せず、GET /sns/cart を read-model に寄せられる。
type cartQueryPort interface {
	GetCartQuery(ctx context.Context, avatarID string) (any, error)
}

var _ cartQueryPort = (*SNSCartQuery)(nil)

func (q *SNSCartQuery) GetCartQuery(ctx context.Context, avatarID string) (any, error) {
	return q.GetByAvatarID(ctx, avatarID)
}

func (q *SNSCartQuery) GetByAvatarID(ctx context.Context, avatarID string) (snsdto.CartDTO, error) {
	if q == nil || q.FS == nil {
		return snsdto.CartDTO{}, errors.New("sns cart query: firestore client is nil")
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return snsdto.CartDTO{}, errors.New("avatarId is required")
	}

	log.Printf("[sns_cart_query_diag] start avatarId=%q fs=%t listRepo=%t resolver=%t cartCol=%q listsCol=%q invCol=%q pbCol=%q",
		maskUID(avatarID),
		q.FS != nil,
		q.ListRepo != nil,
		q.Resolver != nil,
		strings.TrimSpace(q.CartCol),
		strings.TrimSpace(q.ListsCol),
		strings.TrimSpace(q.InventoriesCol),
		strings.TrimSpace(q.ProductBlueprintsCol),
	)

	cartCol := strings.TrimSpace(q.CartCol)
	if cartCol == "" {
		cartCol = "carts"
	}

	snap, err := q.FS.Collection(cartCol).Doc(avatarID).Get(ctx)
	if err != nil {
		if isFirestoreNotFound(err) {
			log.Printf("[sns_cart_query_diag] cart doc not found avatarId=%q col=%q", maskUID(avatarID), cartCol)
			return snsdto.CartDTO{}, ErrNotFound
		}
		log.Printf("[sns_cart_query_diag] cart doc get failed avatarId=%q col=%q err=%v", maskUID(avatarID), cartCol, err)
		return snsdto.CartDTO{}, err
	}
	if snap == nil || !snap.Exists() {
		log.Printf("[sns_cart_query_diag] cart snap missing avatarId=%q col=%q", maskUID(avatarID), cartCol)
		return snsdto.CartDTO{}, ErrNotFound
	}

	// ✅ IMPORTANT:
	// carts.items のスキーマが過去に変わっている可能性があるため、
	// DataTo(&cartdom.Cart) は使わず “後方互換パース” する。
	c, perr := cartFromSnapshotCompat(avatarID, snap)
	if perr != nil {
		log.Printf("[sns_cart_query] cart parse failed avatarId=%q err=%v", maskUID(avatarID), perr)
		return snsdto.CartDTO{}, perr
	}

	itemN := 0
	sample := ""
	if c.Items != nil {
		itemN = len(c.Items)
		for _, it := range c.Items {
			sample = fmt.Sprintf("inv=%q list=%q model=%q qty=%d",
				maskID(it.InventoryID), maskID(it.ListID), maskID(it.ModelID), it.Qty,
			)
			break
		}
	}
	log.Printf("[sns_cart_query_diag] cart loaded avatarId=%q items=%d sample=%s",
		maskUID(avatarID), itemN, sample,
	)

	priceIndex, listMetaIndex := q.fetchListIndicesByCart(ctx, c)
	invIndex := q.fetchInventoryIndexByCart(ctx, c)
	modelIndex := q.fetchModelSimpleIndexByCart(ctx, c)
	productNameIndex := q.fetchProductNameIndexByCart(ctx, c, invIndex)

	out := toCartDTO(c, priceIndex, listMetaIndex, invIndex, modelIndex, productNameIndex)

	cover := diagCartDTOMetaCoverage(out)
	log.Printf("[sns_cart_query_diag] dto built avatarId=%q items=%d title=%d image=%d price=%d size=%d color=%d productName=%d",
		maskUID(avatarID),
		cover.Items,
		cover.WithTitle,
		cover.WithImage,
		cover.WithPrice,
		cover.WithSize,
		cover.WithColor,
		cover.WithProductName,
	)

	log.Printf("[sns_cart_query] get ok avatarId=%q items=%d", maskUID(avatarID), len(out.Items))
	return out, nil
}

// ============================================================
// cart snapshot parsing (backward compatible)
// ============================================================

// carts doc supported shapes:
//
// 1) items: map[itemKey] = {inventoryId, listId, modelId, qty, ...}
// 2) items: map[itemKey] = qty (legacy)
//   - in this case ModelID=itemKey, Qty=qty, other IDs empty (will be filtered out later)
func cartFromSnapshotCompat(avatarID string, snap *firestore.DocumentSnapshot) (*cartdom.Cart, error) {
	if snap == nil {
		return nil, errors.New("sns cart query: snapshot is nil")
	}

	raw := snap.Data()
	if raw == nil {
		// empty doc is unusual but handle defensively
		return &cartdom.Cart{
			ID:    strings.TrimSpace(avatarID),
			Items: map[string]cartdom.CartItem{},
		}, nil
	}

	c := &cartdom.Cart{
		ID:    strings.TrimSpace(avatarID),
		Items: map[string]cartdom.CartItem{},
	}

	// times (best-effort)
	if t, ok := raw["createdAt"]; ok {
		if tt, ok2 := timeAnyToTime(t); ok2 {
			c.CreatedAt = tt
		}
	}
	if t, ok := raw["updatedAt"]; ok {
		if tt, ok2 := timeAnyToTime(t); ok2 {
			c.UpdatedAt = tt
		}
	}
	if t, ok := raw["expiresAt"]; ok {
		if tt, ok2 := timeAnyToTime(t); ok2 {
			c.ExpiresAt = tt
		}
	}

	// items
	itemsAny, _ := raw["items"]
	m, ok := itemsAny.(map[string]any)
	if !ok || m == nil {
		return c, nil
	}

	for k, v := range m {
		itemKey := strings.TrimSpace(k)
		if itemKey == "" {
			continue
		}

		// new shape: map[string]any
		if mv, ok := v.(map[string]any); ok {
			inv := strings.TrimSpace(stringAny(mv["inventoryId"]))
			lid := strings.TrimSpace(stringAny(mv["listId"]))
			mid := strings.TrimSpace(stringAny(mv["modelId"]))
			qty := intAny(mv["qty"])

			if qty <= 0 {
				continue
			}

			c.Items[itemKey] = cartdom.CartItem{
				InventoryID: inv,
				ListID:      lid,
				ModelID:     mid,
				Qty:         qty,
			}
			continue
		}

		// legacy shape: qty only
		qty := intAny(v)
		if qty <= 0 {
			continue
		}
		c.Items[itemKey] = cartdom.CartItem{
			InventoryID: "",
			ListID:      "",
			ModelID:     itemKey,
			Qty:         qty,
		}
	}

	return c, nil
}

func timeAnyToTime(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		if x.IsZero() {
			return time.Time{}, false
		}
		return x.UTC(), true
	default:
		// Firestore の Timestamp は Data() だと time.Time で来る想定だが、念のため fmt 経由はしない
		return time.Time{}, false
	}
}

func stringAny(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprint(v)
	}
}

func intAny(v any) int {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case int:
		return x
	case int8:
		return int(x)
	case int16:
		return int(x)
	case int32:
		return int(x)
	case int64:
		return int(x)
	case uint:
		return int(x)
	case uint8:
		return int(x)
	case uint16:
		return int(x)
	case uint32:
		return int(x)
	case uint64:
		return int(x)
	case float32:
		return int(x)
	case float64:
		return int(x)
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0
		}
		var n int
		_, err := fmt.Sscanf(s, "%d", &n)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
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
	Size  string
	Color string
}

func toCartDTO(
	c *cartdom.Cart,
	priceIndex map[string]map[string]int,
	listMetaIndex map[string]listMeta,
	invIndex map[string]invParts,
	modelIndex map[string]modelSimple,
	productNameIndex map[string]string,
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

		dto := snsdto.CartItemDTO{
			InventoryID: invID,
			ListID:      listID,
			ModelID:     modelID,
			Qty:         it.Qty,
		}

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

		if priceIndex != nil {
			if m, ok := priceIndex[listID]; ok {
				if p, ok2 := m[modelID]; ok2 {
					pp := p
					dto.Price = &pp
				}
			}
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
		if pbID != "" && productNameIndex != nil {
			if name, ok := productNameIndex[pbID]; ok {
				if s := strings.TrimSpace(name); s != "" {
					dto.ProductName = s
				}
			}
		}

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
// list lookup
// ============================================================

func (q *SNSCartQuery) fetchListIndicesByCart(ctx context.Context, c *cartdom.Cart) (map[string]map[string]int, map[string]listMeta) {
	if q == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil, nil
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
		log.Printf("[sns_cart_query_diag] listIndex: no listIds in cart")
		return nil, nil
	}

	if q.ListRepo != nil {
		log.Printf("[sns_cart_query_diag] listIndex: via ListRepo listIds=%d sample=%q", len(listIDs), maskID(listIDs[0]))
		price, meta := q.fetchListIndicesByCartViaRepo(ctx, listIDs)
		log.Printf("[sns_cart_query_diag] listIndex: via ListRepo done priceLists=%d metaLists=%d",
			countMapKeys(price), countMetaKeys(meta),
		)
		return price, meta
	}

	log.Printf("[sns_cart_query_diag] listIndex: via Firestore (ListRepo is nil) listIds=%d sample=%q", len(listIDs), maskID(listIDs[0]))
	price, meta := q.fetchListIndicesByCartViaFirestore(ctx, listIDs)
	log.Printf("[sns_cart_query_diag] listIndex: via Firestore done priceLists=%d metaLists=%d",
		countMapKeys(price), countMetaKeys(meta),
	)
	return price, meta
}

func (q *SNSCartQuery) fetchListIndicesByCartViaRepo(ctx context.Context, listIDs []string) (map[string]map[string]int, map[string]listMeta) {
	if q == nil || q.ListRepo == nil || len(listIDs) == 0 {
		return nil, nil
	}

	priceOut := map[string]map[string]int{}
	metaOut := map[string]listMeta{}

	okN := 0
	errN := 0
	for _, lid0 := range listIDs {
		lid := strings.TrimSpace(lid0)
		if lid == "" {
			continue
		}

		l, err := q.ListRepo.GetByID(ctx, lid)
		if err != nil {
			errN++
			log.Printf("[sns_cart_query_diag] listRepo.GetByID failed listId=%q err=%v", maskID(lid), err)
			continue
		}
		okN++

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

		log.Printf("[sns_cart_query_diag] listRepo ok listId=%q title=%t image=%t prices=%d",
			maskID(lid),
			strings.TrimSpace(mt.Title) != "",
			strings.TrimSpace(mt.ImageID) != "",
			len(priceOut[lid]),
		)
	}
	log.Printf("[sns_cart_query_diag] listRepo summary ok=%d err=%d", okN, errN)

	if len(priceOut) == 0 {
		priceOut = nil
	}
	if len(metaOut) == 0 {
		metaOut = nil
	}
	return priceOut, metaOut
}

func (q *SNSCartQuery) fetchListIndicesByCartViaFirestore(ctx context.Context, listIDs []string) (map[string]map[string]int, map[string]listMeta) {
	if q == nil || q.FS == nil || len(listIDs) == 0 {
		return nil, nil
	}

	listsCol := strings.TrimSpace(q.ListsCol)
	if listsCol == "" {
		listsCol = "lists"
	}

	refs := make([]*firestore.DocumentRef, 0, len(listIDs))
	for _, lid := range listIDs {
		refs = append(refs, q.FS.Collection(listsCol).Doc(lid))
	}

	snaps, err := q.FS.GetAll(ctx, refs)
	if err != nil {
		log.Printf("[sns_cart_query] GetAll(lists) failed col=%q listIds=%d err=%v", listsCol, len(refs), err)
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
			log.Printf("[sns_cart_query_diag] list fs missing listId=%q col=%q exists=%t", maskID(lid), listsCol, snap != nil && snap.Exists())
			continue
		}

		var l ldom.List
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

			log.Printf("[sns_cart_query_diag] list fs(DataTo) ok listId=%q title=%t image=%t prices=%d",
				maskID(lid),
				strings.TrimSpace(mt.Title) != "",
				strings.TrimSpace(mt.ImageID) != "",
				len(priceOut[lid]),
			)
			continue
		}

		m := snap.Data()
		title := pickString(m, "title", "Title")
		image := pickString(m, "imageId", "ImageID", "imageID", "ImageId", "image", "Image", "listImage", "ListImage", "imageUrl", "ImageUrl")

		if strings.TrimSpace(title) != "" || strings.TrimSpace(image) != "" {
			metaOut[lid] = listMeta{Title: strings.TrimSpace(title), ImageID: strings.TrimSpace(image)}
		}

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

		log.Printf("[sns_cart_query_diag] list fs(map) ok listId=%q title=%t image=%t prices=%d",
			maskID(lid),
			strings.TrimSpace(title) != "",
			strings.TrimSpace(image) != "",
			len(priceOut[lid]),
		)
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
		log.Printf("[sns_cart_query] GetAll(inventories) failed col=%q invIds=%d err=%v", invCol, len(refs), err)
		return nil
	}

	out := map[string]invParts{}
	okN := 0
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

		fallbackParsed := false
		if pb == "" || tb == "" {
			if p, t, ok := parseInventoryID(invID); ok {
				fallbackParsed = true
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

		okN++
		out[invID] = invParts{ProductBlueprintID: pb, TokenBlueprintID: tb}
		log.Printf("[sns_cart_query_diag] inv ok invId=%q pb=%q tb=%q parsedFallback=%t",
			maskID(invID), maskID(pb), maskID(tb), fallbackParsed,
		)
	}

	log.Printf("[sns_cart_query_diag] inv summary invIds=%d resolved=%d col=%q", len(invIDs), okN, invCol)

	if len(out) == 0 {
		return nil
	}
	return out
}

// ============================================================
// model resolver lookup
// ============================================================

func (q *SNSCartQuery) fetchModelSimpleIndexByCart(ctx context.Context, c *cartdom.Cart) map[string]modelSimple {
	if q == nil || c == nil || c.Items == nil || len(c.Items) == 0 {
		return nil
	}
	if q.Resolver == nil {
		log.Printf("[sns_cart_query_diag] modelIndex: resolver is nil -> skip (size/color will be empty)")
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
	okN := 0
	emptyN := 0

	for _, mid := range modelIDs {
		mr := q.Resolver.ResolveModelResolved(ctx, mid)
		sz := strings.TrimSpace(mr.Size)
		cl := strings.TrimSpace(mr.Color)
		if sz == "" && cl == "" {
			emptyN++
			continue
		}
		okN++
		out[mid] = modelSimple{Size: sz, Color: cl}
	}

	log.Printf("[sns_cart_query_diag] modelIndex: modelIds=%d resolved=%d empty=%d", len(modelIDs), okN, emptyN)

	if len(out) == 0 {
		return nil
	}
	return out
}

// ============================================================
// productName lookup（ここを強化ログ）
// ============================================================

type pbDeriveDiag struct {
	InvID  string
	PbID   string
	Source string // invIndex / parseInventoryID
}

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
		// ✅ repo 側と揃える
		pbCol = "product_blueprints"
	}

	pbSeen := map[string]struct{}{}
	pbIDs := make([]string, 0, 16)

	diags := make([]pbDeriveDiag, 0, 4)

	for _, it := range c.Items {
		invID := strings.TrimSpace(it.InventoryID)
		if invID == "" {
			continue
		}

		pbID := ""
		src := ""
		if invIndex != nil {
			if parts, ok := invIndex[invID]; ok {
				pbID = strings.TrimSpace(parts.ProductBlueprintID)
				if pbID != "" {
					src = "invIndex"
				}
			}
		}
		if pbID == "" {
			if p, _, ok := parseInventoryID(invID); ok {
				pbID = strings.TrimSpace(p)
				if pbID != "" {
					src = "parseInventoryID"
				}
			}
		}

		if len(diags) < 3 {
			diags = append(diags, pbDeriveDiag{InvID: invID, PbID: pbID, Source: src})
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
		log.Printf("[sns_cart_query_diag] productNameIndex: no pbIds (resolver=%t col=%q)", q.Resolver != nil, pbCol)
		for _, d := range diags {
			log.Printf("[sns_cart_query_diag] productNameIndex derive inv=%q pb=%q src=%s",
				maskID(d.InvID), maskID(d.PbID), d.Source,
			)
		}
		return nil
	}

	log.Printf("[sns_cart_query_diag] productNameIndex: pbIds=%d sample=%q col=%q resolver=%t",
		len(pbIDs), maskID(pbIDs[0]), pbCol, q.Resolver != nil,
	)
	for _, d := range diags {
		log.Printf("[sns_cart_query_diag] productNameIndex derive inv=%q pb=%q src=%s",
			maskID(d.InvID), maskID(d.PbID), d.Source,
		)
	}

	refs := make([]*firestore.DocumentRef, 0, len(pbIDs))
	for _, id := range pbIDs {
		refs = append(refs, q.FS.Collection(pbCol).Doc(id))
	}

	snaps, err := q.FS.GetAll(ctx, refs)
	if err != nil {
		log.Printf("[sns_cart_query] GetAll(productBlueprints) failed col=%q pbIds=%d err=%v", pbCol, len(refs), err)

		if q.Resolver != nil {
			out := map[string]string{}
			for _, id := range pbIDs {
				name := strings.TrimSpace(q.Resolver.ResolveProductName(ctx, id))
				log.Printf("[sns_cart_query_diag] productNameIndex resolver-only pb=%q name=%t",
					maskID(id), name != "",
				)
				if name != "" {
					out[id] = name
				}
			}
			log.Printf("[sns_cart_query_diag] productNameIndex: resolver-only resolved=%d", len(out))
			if len(out) == 0 {
				return nil
			}
			return out
		}
		return nil
	}

	out := map[string]string{}

	resolvedFS := 0
	resolvedResolver := 0
	miss := 0

	for i, snap := range snaps {
		pbID := ""
		if i >= 0 && i < len(pbIDs) {
			pbID = strings.TrimSpace(pbIDs[i])
		}
		if pbID == "" {
			continue
		}

		exists := snap != nil && snap.Exists()
		if !exists {
			miss++
			log.Printf("[sns_cart_query_diag] productNameIndex miss pb=%q col=%q -> will try resolver=%t",
				maskID(pbID), pbCol, q.Resolver != nil,
			)

			if q.Resolver != nil {
				rn := strings.TrimSpace(q.Resolver.ResolveProductName(ctx, pbID))
				log.Printf("[sns_cart_query_diag] productNameIndex resolver pb=%q name=%t",
					maskID(pbID), rn != "",
				)
				if rn != "" {
					out[pbID] = rn
					resolvedResolver++
				}
			}
			continue
		}

		m := snap.Data()
		name := strings.TrimSpace(pickString(m, "productName", "ProductName", "name", "Name"))
		if name != "" {
			out[pbID] = name
			resolvedFS++
			log.Printf("[sns_cart_query_diag] productNameIndex fs ok pb=%q name=%t", maskID(pbID), true)
			continue
		}

		if q.Resolver != nil {
			rn := strings.TrimSpace(q.Resolver.ResolveProductName(ctx, pbID))
			log.Printf("[sns_cart_query_diag] productNameIndex fs empty -> resolver pb=%q name=%t",
				maskID(pbID), rn != "",
			)
			if rn != "" {
				out[pbID] = rn
				resolvedResolver++
			}
		} else {
			log.Printf("[sns_cart_query_diag] productNameIndex fs empty and resolver nil pb=%q", maskID(pbID))
		}
	}

	log.Printf("[sns_cart_query_diag] productNameIndex summary pbIds=%d resolved=%d(fs=%d resolver=%d) miss=%d",
		len(pbIDs), len(out), resolvedFS, resolvedResolver, miss,
	)

	if len(out) == 0 {
		return nil
	}
	return out
}

// ============================================================
// diagnostics helpers
// ============================================================

type cartCover struct {
	Items           int
	WithTitle       int
	WithImage       int
	WithPrice       int
	WithSize        int
	WithColor       int
	WithProductName int
}

func diagCartDTOMetaCoverage(dto snsdto.CartDTO) cartCover {
	c := cartCover{}
	if dto.Items == nil {
		return c
	}
	c.Items = len(dto.Items)
	for _, it := range dto.Items {
		if strings.TrimSpace(it.Title) != "" {
			c.WithTitle++
		}
		if strings.TrimSpace(it.ListImage) != "" {
			c.WithImage++
		}
		if it.Price != nil {
			c.WithPrice++
		}
		if strings.TrimSpace(it.Size) != "" {
			c.WithSize++
		}
		if strings.TrimSpace(it.Color) != "" {
			c.WithColor++
		}
		if strings.TrimSpace(it.ProductName) != "" {
			c.WithProductName++
		}
	}
	return c
}

func countMapKeys(m map[string]map[string]int) int {
	if m == nil {
		return 0
	}
	return len(m)
}

func countMetaKeys(m map[string]listMeta) int {
	if m == nil {
		return 0
	}
	return len(m)
}

func maskID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 6 {
		return s
	}
	return s[:4] + "***" + s[len(s)-2:]
}

// ============================================================
// shared helpers
// ============================================================

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
