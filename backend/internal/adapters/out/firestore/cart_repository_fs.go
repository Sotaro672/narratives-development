// backend/internal/adapters/out/firestore/cart_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cartdom "narratives/internal/domain/cart"
)

// CartRepositoryFS implements cart.Repository using Firestore.
//
// Collection design (recommended):
// - collection: carts
// - docId: avatarId (docId is the source of truth)
// - fields: items(map), createdAt, updatedAt, expiresAt
//
// TTL:
// - Configure Firestore TTL on "expiresAt".
type CartRepositoryFS struct {
	Client *firestore.Client
}

func NewCartRepositoryFS(client *firestore.Client) *CartRepositoryFS {
	return &CartRepositoryFS{Client: client}
}

func (r *CartRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("carts")
}

// GetByID returns cart by docId (= avatarId).
func (r *CartRepositoryFS) GetByID(ctx context.Context, id string) (cartdom.Cart, error) {
	c, err := r.GetByAvatarID(ctx, id)
	if err != nil {
		return cartdom.Cart{}, err
	}
	if c == nil {
		return cartdom.Cart{}, nil
	}
	return *c, nil
}

// GetByAvatarID returns (nil, nil) if not found.
func (r *CartRepositoryFS) GetByAvatarID(ctx context.Context, avatarID string) (*cartdom.Cart, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("cart_repository_fs: firestore client is nil")
	}

	aid := avatarID
	if aid == "" {
		return nil, errors.New("cart_repository_fs: avatarID is empty")
	}

	snap, err := r.col().Doc(aid).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	doc, err := cartDocFromSnapshot(snap)
	if err != nil {
		return nil, err
	}

	d := doc.toDomain()
	d.ID = aid
	return d, nil
}

// Upsert saves cart by docId=cart.ID (= avatarId).
func (r *CartRepositoryFS) Upsert(ctx context.Context, c *cartdom.Cart) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}
	if c == nil {
		return errors.New("cart_repository_fs: cart is nil")
	}
	if c.ID == "" {
		return errors.New("cart_repository_fs: Upsert requires cart.ID (= avatarId) as docId")
	}

	doc := cartDocFromDomain(c)

	_, err := r.col().Doc(c.ID).Set(ctx, doc)
	return err
}

func (r *CartRepositoryFS) DeleteByAvatarID(ctx context.Context, avatarID string) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}
	if avatarID == "" {
		return errors.New("cart_repository_fs: avatarID is empty")
	}

	_, err := r.col().Doc(avatarID).Delete(ctx)
	return err
}

// Clear empties cart items by docId (= avatarId / cartId).
func (r *CartRepositoryFS) Clear(ctx context.Context, cartID string) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}
	if cartID == "" {
		return errors.New("cart_repository_fs: cartID is empty")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(cartdom.DefaultCartTTL)

	_, err := r.col().Doc(cartID).Update(ctx, []firestore.Update{
		{Path: "items", Value: map[string]any{}},
		{Path: "updatedAt", Value: now},
		{Path: "expiresAt", Value: expiresAt},
	})
	if err == nil {
		return nil
	}

	if status.Code(err) == codes.NotFound {
		doc := cartDoc{
			Items:     map[string]cartItemDoc{},
			CreatedAt: now,
			UpdatedAt: now,
			ExpiresAt: expiresAt,
		}
		_, setErr := r.col().Doc(cartID).Set(ctx, doc)
		return setErr
	}

	return err
}

// -----------------------------------------
// Firestore DTO
// -----------------------------------------

type cartDoc struct {
	Items map[string]cartItemDoc `firestore:"items"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
	ExpiresAt time.Time `firestore:"expiresAt"`
}

type cartItemDoc struct {
	Type string `firestore:"type,omitempty"`

	InventoryID string `firestore:"inventoryId,omitempty"`
	ListID      string `firestore:"listId,omitempty"`
	ModelID     string `firestore:"modelId,omitempty"`

	ResaleID  string `firestore:"resaleId,omitempty"`
	ProductID string `firestore:"productId,omitempty"`

	Qty int `firestore:"qty"`
}

func cartDocFromSnapshot(snap *firestore.DocumentSnapshot) (cartDoc, error) {
	if snap == nil {
		return cartDoc{}, errors.New("cart_repository_fs: snapshot is nil")
	}

	raw := snap.Data()
	if raw == nil {
		return cartDoc{Items: map[string]cartItemDoc{}}, nil
	}

	out := cartDoc{Items: map[string]cartItemDoc{}}

	if t, ok := raw["createdAt"]; ok {
		if tt, ok2 := asTime(t); ok2 {
			out.CreatedAt = tt
		}
	}
	if t, ok := raw["updatedAt"]; ok {
		if tt, ok2 := asTime(t); ok2 {
			out.UpdatedAt = tt
		}
	}
	if t, ok := raw["expiresAt"]; ok {
		if tt, ok2 := asTime(t); ok2 {
			out.ExpiresAt = tt
		}
	}

	itemsAny, _ := raw["items"]
	m, ok := itemsAny.(map[string]any)
	if !ok || m == nil {
		return out, nil
	}

	for k, v := range m {
		if k == "" {
			continue
		}

		mv, ok := v.(map[string]any)
		if !ok {
			continue
		}

		item := cartItemDoc{
			Type:        asString(mv["type"]),
			InventoryID: asString(mv["inventoryId"]),
			ListID:      asString(mv["listId"]),
			ModelID:     asString(mv["modelId"]),
			ResaleID:    asString(mv["resaleId"]),
			ProductID:   asString(mv["productId"]),
			Qty:         asInt(mv["qty"]),
		}

		normalized, ok := normalizeCartItemDoc(item)
		if !ok {
			continue
		}

		out.Items[k] = normalized
	}

	return out, nil
}

func cartDocFromDomain(c *cartdom.Cart) cartDoc {
	items := map[string]cartItemDoc{}

	if c != nil && c.Items != nil {
		for k, it := range c.Items {
			if k == "" {
				continue
			}

			normalized, ok := cartItemDocFromDomain(it)
			if !ok {
				continue
			}

			items[k] = normalized
		}
	}

	return cartDoc{
		Items:     items,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		ExpiresAt: c.ExpiresAt,
	}
}

func (d cartDoc) toDomain() *cartdom.Cart {
	items := map[string]cartdom.CartItem{}

	if d.Items != nil {
		for k, it := range d.Items {
			if k == "" {
				continue
			}

			normalized, ok := cartItemDomainFromDoc(it)
			if !ok {
				continue
			}

			items[k] = normalized
		}
	}

	return &cartdom.Cart{
		Items:     items,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		ExpiresAt: d.ExpiresAt,
	}
}

func cartItemDocFromDomain(it cartdom.CartItem) (cartItemDoc, bool) {
	switch inferCartItemType(string(it.Type), it.InventoryID, it.ListID, it.ModelID, it.ResaleID, it.ProductID) {
	case cartdom.CartItemTypeList:
		if it.Qty <= 0 || it.InventoryID == "" || it.ListID == "" || it.ModelID == "" {
			return cartItemDoc{}, false
		}

		return cartItemDoc{
			Type:        string(cartdom.CartItemTypeList),
			InventoryID: it.InventoryID,
			ListID:      it.ListID,
			ModelID:     it.ModelID,
			Qty:         it.Qty,
		}, true

	case cartdom.CartItemTypeResale:
		if it.ResaleID == "" || it.ProductID == "" {
			return cartItemDoc{}, false
		}

		return cartItemDoc{
			Type:      string(cartdom.CartItemTypeResale),
			ResaleID:  it.ResaleID,
			ProductID: it.ProductID,
			Qty:       1,
		}, true

	default:
		return cartItemDoc{}, false
	}
}

func cartItemDomainFromDoc(it cartItemDoc) (cartdom.CartItem, bool) {
	normalized, ok := normalizeCartItemDoc(it)
	if !ok {
		return cartdom.CartItem{}, false
	}

	switch inferCartItemType(normalized.Type, normalized.InventoryID, normalized.ListID, normalized.ModelID, normalized.ResaleID, normalized.ProductID) {
	case cartdom.CartItemTypeList:
		return cartdom.CartItem{
			Type:        cartdom.CartItemTypeList,
			InventoryID: normalized.InventoryID,
			ListID:      normalized.ListID,
			ModelID:     normalized.ModelID,
			Qty:         normalized.Qty,
		}, true

	case cartdom.CartItemTypeResale:
		return cartdom.CartItem{
			Type:      cartdom.CartItemTypeResale,
			ResaleID:  normalized.ResaleID,
			ProductID: normalized.ProductID,
			Qty:       1,
		}, true

	default:
		return cartdom.CartItem{}, false
	}
}

func normalizeCartItemDoc(it cartItemDoc) (cartItemDoc, bool) {
	switch inferCartItemType(it.Type, it.InventoryID, it.ListID, it.ModelID, it.ResaleID, it.ProductID) {
	case cartdom.CartItemTypeList:
		if it.Qty <= 0 || it.InventoryID == "" || it.ListID == "" || it.ModelID == "" {
			return cartItemDoc{}, false
		}

		return cartItemDoc{
			Type:        string(cartdom.CartItemTypeList),
			InventoryID: it.InventoryID,
			ListID:      it.ListID,
			ModelID:     it.ModelID,
			Qty:         it.Qty,
		}, true

	case cartdom.CartItemTypeResale:
		if it.ResaleID == "" || it.ProductID == "" {
			return cartItemDoc{}, false
		}

		return cartItemDoc{
			Type:      string(cartdom.CartItemTypeResale),
			ResaleID:  it.ResaleID,
			ProductID: it.ProductID,
			Qty:       1,
		}, true

	default:
		return cartItemDoc{}, false
	}
}

func inferCartItemType(
	rawType string,
	inventoryID string,
	listID string,
	modelID string,
	resaleID string,
	productID string,
) cartdom.CartItemType {
	itemType := cartdom.CartItemType(rawType)

	switch itemType {
	case cartdom.CartItemTypeList, cartdom.CartItemTypeResale:
		return itemType
	}

	if resaleID != "" || productID != "" {
		return cartdom.CartItemTypeResale
	}

	if inventoryID != "" || listID != "" || modelID != "" {
		return cartdom.CartItemTypeList
	}

	return ""
}
