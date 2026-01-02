// backend/internal/application/query/sns/preview_query.go
package sns

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"

	snsdto "narratives/internal/application/query/sns/dto"
	appresolver "narratives/internal/application/resolver"
	cartdom "narratives/internal/domain/cart"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// SNSPreviewQuery resolves (for item preview UI):
//   - avatarId + itemKey -> cart.items[itemKey] (inventoryId/listId/modelId/qty)
//   - listId -> title/listImage + price(modelId)
//   - inventoryId -> pbId/tbId (inventories or parse pb__tb)
//   - pbId -> productName + brandId/companyId
//   - tbId -> tokenName + brandId/companyId/iconUrl
//   - modelId -> modelNumber/size/color/rgb
//   - (brandId/companyId) -> brandName/companyName via NameResolver
type SNSPreviewQuery struct {
	FS *firestore.Client

	// ✅ optional: inject from DI
	Resolver *appresolver.NameResolver

	CartCol              string
	ListsCol             string
	InventoriesCol       string
	ProductBlueprintsCol string
	TokenBlueprintsCol   string
}

func NewSNSPreviewQuery(fs *firestore.Client) *SNSPreviewQuery {
	return &SNSPreviewQuery{
		FS:                   fs,
		Resolver:             nil,
		CartCol:              "carts",
		ListsCol:             "lists",
		InventoriesCol:       "inventories",
		ProductBlueprintsCol: "productBlueprints",
		TokenBlueprintsCol:   "tokenBlueprints",
	}
}

// GetByAvatarIDAndItemKey resolves a single cart item preview by (avatarId, itemKey).
func (q *SNSPreviewQuery) GetByAvatarIDAndItemKey(ctx context.Context, avatarID string, itemKey string) (snsdto.PreviewDTO, error) {
	if q == nil || q.FS == nil {
		return snsdto.PreviewDTO{}, errors.New("sns preview query: firestore client is nil")
	}

	avatarID = strings.TrimSpace(avatarID)
	itemKey = strings.TrimSpace(itemKey)
	if avatarID == "" || itemKey == "" {
		return snsdto.PreviewDTO{}, errors.New("avatarId and itemKey are required")
	}

	cartCol := strings.TrimSpace(q.CartCol)
	if cartCol == "" {
		cartCol = "carts"
	}

	// carts/{avatarId}
	snap, err := q.FS.Collection(cartCol).Doc(avatarID).Get(ctx)
	if err != nil {
		if isFirestoreNotFound(err) {
			return snsdto.PreviewDTO{}, ErrNotFound
		}
		return snsdto.PreviewDTO{}, err
	}
	if snap == nil || !snap.Exists() {
		return snsdto.PreviewDTO{}, ErrNotFound
	}

	var c cartdom.Cart
	if derr := snap.DataTo(&c); derr != nil {
		log.Printf("[sns_preview_query] DataTo(cart) failed avatarId=%q err=%v", maskUID(avatarID), derr)
		return snsdto.PreviewDTO{}, derr
	}
	c.ID = avatarID

	if c.Items == nil {
		return snsdto.PreviewDTO{}, ErrNotFound
	}
	it, ok := c.Items[itemKey]
	if !ok {
		return snsdto.PreviewDTO{}, ErrNotFound
	}

	invID := strings.TrimSpace(it.InventoryID)
	listID := strings.TrimSpace(it.ListID)
	modelID := strings.TrimSpace(it.ModelID)

	out := snsdto.PreviewDTO{
		AvatarID:    avatarID,
		ItemKey:     itemKey,
		InventoryID: invID,
		ListID:      listID,
		ModelID:     modelID,
		Qty:         it.Qty,
	}

	// --------------------------
	// list (title/image/price)
	// --------------------------
	if listID != "" {
		q.fillListFields(ctx, &out, listID, modelID)
	}

	// --------------------------
	// inventory -> pbId/tbId
	// --------------------------
	pbID, tbID := q.resolvePBAndTBByInventory(ctx, invID)
	out.ProductBlueprintID = pbID
	out.TokenBlueprintID = tbID

	// --------------------------
	// product blueprint meta
	// --------------------------
	if pbID != "" {
		q.fillProductFields(ctx, &out, pbID)
	}

	// --------------------------
	// token blueprint meta
	// --------------------------
	if tbID != "" {
		q.fillTokenFields(ctx, &out, tbID)
	}

	// --------------------------
	// model resolved
	// --------------------------
	if q.Resolver != nil && modelID != "" {
		mr := q.Resolver.ResolveModelResolved(ctx, modelID)
		if s := strings.TrimSpace(mr.ModelNumber); s != "" {
			out.ModelNumber = s
		}
		if s := strings.TrimSpace(mr.Size); s != "" {
			out.Size = s
		}
		if s := strings.TrimSpace(mr.Color); s != "" {
			out.Color = s
		}
		if mr.RGB != nil {
			out.RGB = mr.RGB
		}
	}

	// --------------------------
	// brand/company names (resolver)
	// --------------------------
	if q.Resolver != nil {
		// token
		if strings.TrimSpace(out.BrandID) != "" {
			out.BrandName = strings.TrimSpace(q.Resolver.ResolveBrandName(ctx, out.BrandID))
		}
		if strings.TrimSpace(out.CompanyID) == "" && strings.TrimSpace(out.BrandID) != "" {
			if cid := strings.TrimSpace(q.Resolver.ResolveBrandCompanyID(ctx, out.BrandID)); cid != "" {
				out.CompanyID = cid
			}
		}
		if strings.TrimSpace(out.CompanyID) != "" {
			out.CompanyName = strings.TrimSpace(q.Resolver.ResolveCompanyName(ctx, out.CompanyID))
		}

		// product
		if strings.TrimSpace(out.ProductBrandID) != "" {
			out.ProductBrandName = strings.TrimSpace(q.Resolver.ResolveBrandName(ctx, out.ProductBrandID))
		}
		if strings.TrimSpace(out.ProductCompanyID) == "" && strings.TrimSpace(out.ProductBrandID) != "" {
			if cid := strings.TrimSpace(q.Resolver.ResolveBrandCompanyID(ctx, out.ProductBrandID)); cid != "" {
				out.ProductCompanyID = cid
			}
		}
		if strings.TrimSpace(out.ProductCompanyID) != "" {
			out.ProductCompanyName = strings.TrimSpace(q.Resolver.ResolveCompanyName(ctx, out.ProductCompanyID))
		}
	}

	log.Printf("[sns_preview_query] get ok avatarId=%q itemKey=%q", maskUID(avatarID), itemKey)
	return out, nil
}

// ============================================================
// internal helpers
// ============================================================

func (q *SNSPreviewQuery) fillListFields(ctx context.Context, out *snsdto.PreviewDTO, listID string, modelID string) {
	if q == nil || q.FS == nil || out == nil {
		return
	}
	listsCol := strings.TrimSpace(q.ListsCol)
	if listsCol == "" {
		listsCol = "lists"
	}

	snap, err := q.FS.Collection(listsCol).Doc(listID).Get(ctx)
	if err != nil || snap == nil || !snap.Exists() {
		return
	}

	m := snap.Data()
	out.Title = strings.TrimSpace(pickString(m, "title", "Title"))
	out.ListImage = strings.TrimSpace(pickString(m, "imageId", "ImageID", "imageID", "ImageId"))

	// price from prices[] by modelId
	if modelID == "" {
		return
	}
	raw, ok := m["prices"]
	if !ok {
		return
	}
	rows, _ := raw.([]any)
	if len(rows) == 0 {
		return
	}
	for _, row := range rows {
		rm, _ := row.(map[string]any)
		if rm == nil {
			continue
		}
		mid := strings.TrimSpace(pickString(rm, "modelId", "ModelID", "modelID", "ModelId"))
		if mid == "" || mid != modelID {
			continue
		}
		pv := pickAny(rm, "price", "Price")
		if p, ok := asIntAny(pv); ok {
			pp := p
			out.Price = &pp
			return
		}
	}
}

func (q *SNSPreviewQuery) resolvePBAndTBByInventory(ctx context.Context, inventoryID string) (string, string) {
	invID := strings.TrimSpace(inventoryID)
	if invID == "" {
		return "", ""
	}

	// (A) inventories doc
	if q != nil && q.FS != nil {
		invCol := strings.TrimSpace(q.InventoriesCol)
		if invCol == "" {
			invCol = "inventories"
		}
		snap, err := q.FS.Collection(invCol).Doc(invID).Get(ctx)
		if err == nil && snap != nil && snap.Exists() {
			m := snap.Data()
			pb := strings.TrimSpace(pickString(m,
				"productBlueprintId", "productBlueprintID", "ProductBlueprintID", "ProductBlueprintId",
			))
			tb := strings.TrimSpace(pickString(m,
				"tokenBlueprintId", "tokenBlueprintID", "TokenBlueprintID", "TokenBlueprintId",
			))
			if pb != "" || tb != "" {
				// fill missing from parse if needed
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
				return pb, tb
			}
		}
	}

	// (B) parse from inventoryId = pb__tb
	if p, t, ok := parseInventoryID(invID); ok {
		return p, t
	}
	return "", ""
}

func (q *SNSPreviewQuery) fillProductFields(ctx context.Context, out *snsdto.PreviewDTO, productBlueprintID string) {
	if q == nil || q.FS == nil || out == nil {
		return
	}
	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return
	}

	pbCol := strings.TrimSpace(q.ProductBlueprintsCol)
	if pbCol == "" {
		pbCol = "productBlueprints"
	}

	snap, err := q.FS.Collection(pbCol).Doc(pbID).Get(ctx)
	if err != nil || snap == nil || !snap.Exists() {
		// fallback name only
		if q.Resolver != nil {
			out.ProductName = strings.TrimSpace(q.Resolver.ResolveProductName(ctx, pbID))
		}
		return
	}

	m := snap.Data()
	out.ProductName = strings.TrimSpace(pickString(m, "productName", "ProductName", "name", "Name"))
	out.ProductBrandID = strings.TrimSpace(pickString(m, "brandId", "BrandID", "brandID", "BrandId"))
	out.ProductCompanyID = strings.TrimSpace(pickString(m, "companyId", "CompanyID", "companyID", "CompanyId"))

	// fallback productName
	if out.ProductName == "" && q.Resolver != nil {
		out.ProductName = strings.TrimSpace(q.Resolver.ResolveProductName(ctx, pbID))
	}
}

func (q *SNSPreviewQuery) fillTokenFields(ctx context.Context, out *snsdto.PreviewDTO, tokenBlueprintID string) {
	if q == nil || q.FS == nil || out == nil {
		return
	}
	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return
	}

	tbCol := strings.TrimSpace(q.TokenBlueprintsCol)
	if tbCol == "" {
		tbCol = "tokenBlueprints"
	}

	snap, err := q.FS.Collection(tbCol).Doc(tbID).Get(ctx)
	if err != nil || snap == nil || !snap.Exists() {
		// fallback tokenName
		if q.Resolver != nil {
			out.TokenName = strings.TrimSpace(q.Resolver.ResolveTokenName(ctx, tbID))
		}
		return
	}

	// Try decode first (if tags match)
	var tb tbdom.TokenBlueprint
	if derr := snap.DataTo(&tb); derr == nil {
		out.BrandID = strings.TrimSpace(tb.BrandID)
		out.CompanyID = strings.TrimSpace(tb.CompanyID)
		out.IconURL = strings.TrimSpace(tb.IconURL)

		name := strings.TrimSpace(tb.Name)
		if name == "" {
			name = strings.TrimSpace(tb.Symbol)
		}
		out.TokenName = name
		return
	}

	// Fallback map read
	m := snap.Data()
	out.BrandID = strings.TrimSpace(pickString(m, "brandId", "BrandID", "brandID", "BrandId"))
	out.CompanyID = strings.TrimSpace(pickString(m, "companyId", "CompanyID", "companyID", "CompanyId"))
	out.IconURL = strings.TrimSpace(pickString(m, "iconUrl", "IconURL", "iconURL", "IconUrl"))

	name := strings.TrimSpace(pickString(m, "name", "Name"))
	if name == "" {
		name = strings.TrimSpace(pickString(m, "symbol", "Symbol"))
	}
	out.TokenName = name

	// final fallback tokenName
	if strings.TrimSpace(out.TokenName) == "" && q.Resolver != nil {
		out.TokenName = strings.TrimSpace(q.Resolver.ResolveTokenName(ctx, tbID))
	}
}

// (optional) for debug formatting only
func (q *SNSPreviewQuery) String() string {
	if q == nil {
		return "SNSPreviewQuery(nil)"
	}
	return fmt.Sprintf("SNSPreviewQuery(cart=%q lists=%q inv=%q pb=%q tb=%q)", q.CartCol, q.ListsCol, q.InventoriesCol, q.ProductBlueprintsCol, q.TokenBlueprintsCol)
}
