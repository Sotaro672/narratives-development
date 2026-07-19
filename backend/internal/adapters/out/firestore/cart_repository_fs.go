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

// Domain Repositoryを唯一の永続化契約としてcompile時に検証する。
var _ cartdom.Repository = (*CartRepositoryFS)(nil)

func NewCartRepositoryFS(client *firestore.Client) *CartRepositoryFS {
	return &CartRepositoryFS{Client: client}
}

func (r *CartRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("carts")
}

// GetByAvatarID returns (nil, nil) if not found.
func (r *CartRepositoryFS) GetByAvatarID(
	ctx context.Context,
	avatarID string,
) (*cartdom.Cart, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("cart_repository_fs: firestore client is nil")
	}

	if avatarID == "" {
		return nil, errors.New("cart_repository_fs: avatarID is empty")
	}

	snap, err := r.col().Doc(avatarID).Get(ctx)
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

	cart := doc.toDomain()
	cart.ID = avatarID

	return cart, nil
}

// Upsert saves cart by docId=cart.ID (= avatarId).
func (r *CartRepositoryFS) Upsert(
	ctx context.Context,
	cart *cartdom.Cart,
) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}
	if cart == nil {
		return errors.New("cart_repository_fs: cart is nil")
	}
	if cart.ID == "" {
		return errors.New(
			"cart_repository_fs: Upsert requires cart.ID (= avatarId) as docId",
		)
	}

	doc := cartDocFromDomain(cart)

	_, err := r.col().Doc(cart.ID).Set(ctx, doc)
	return err
}

// DeleteByAvatarID deletes carts/{avatarId}.
func (r *CartRepositoryFS) DeleteByAvatarID(
	ctx context.Context,
	avatarID string,
) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}
	if avatarID == "" {
		return errors.New("cart_repository_fs: avatarID is empty")
	}

	_, err := r.col().Doc(avatarID).Delete(ctx)
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

func cartDocFromSnapshot(
	snap *firestore.DocumentSnapshot,
) (cartDoc, error) {
	if snap == nil {
		return cartDoc{}, errors.New(
			"cart_repository_fs: snapshot is nil",
		)
	}

	raw := snap.Data()
	if raw == nil {
		return cartDoc{
			Items: map[string]cartItemDoc{},
		}, nil
	}

	out := cartDoc{
		Items: map[string]cartItemDoc{},
	}

	if value, ok := raw["createdAt"]; ok {
		if parsed, valid := asTime(value); valid {
			out.CreatedAt = parsed
		}
	}

	if value, ok := raw["updatedAt"]; ok {
		if parsed, valid := asTime(value); valid {
			out.UpdatedAt = parsed
		}
	}

	if value, ok := raw["expiresAt"]; ok {
		if parsed, valid := asTime(value); valid {
			out.ExpiresAt = parsed
		}
	}

	itemsValue, _ := raw["items"]
	itemsMap, ok := itemsValue.(map[string]any)
	if !ok || itemsMap == nil {
		return out, nil
	}

	for key, value := range itemsMap {
		if key == "" {
			continue
		}

		itemMap, ok := value.(map[string]any)
		if !ok {
			continue
		}

		item := cartItemDoc{
			Type:        asString(itemMap["type"]),
			InventoryID: asString(itemMap["inventoryId"]),
			ListID:      asString(itemMap["listId"]),
			ModelID:     asString(itemMap["modelId"]),
			ResaleID:    asString(itemMap["resaleId"]),
			ProductID:   asString(itemMap["productId"]),
			Qty:         asInt(itemMap["qty"]),
		}

		normalized, valid := normalizeCartItemDoc(item)
		if !valid {
			continue
		}

		out.Items[key] = normalized
	}

	return out, nil
}

func cartDocFromDomain(cart *cartdom.Cart) cartDoc {
	items := map[string]cartItemDoc{}

	if cart != nil && cart.Items != nil {
		for key, item := range cart.Items {
			if key == "" {
				continue
			}

			normalized, valid := cartItemDocFromDomain(item)
			if !valid {
				continue
			}

			items[key] = normalized
		}
	}

	return cartDoc{
		Items:     items,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
		ExpiresAt: cart.ExpiresAt,
	}
}

func (doc cartDoc) toDomain() *cartdom.Cart {
	items := map[string]cartdom.CartItem{}

	if doc.Items != nil {
		for key, item := range doc.Items {
			if key == "" {
				continue
			}

			normalized, valid := cartItemDomainFromDoc(item)
			if !valid {
				continue
			}

			items[key] = normalized
		}
	}

	return &cartdom.Cart{
		Items:     items,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
		ExpiresAt: doc.ExpiresAt,
	}
}

func cartItemDocFromDomain(
	item cartdom.CartItem,
) (cartItemDoc, bool) {
	itemType := inferCartItemType(
		string(item.Type),
		item.InventoryID,
		item.ListID,
		item.ModelID,
		item.ResaleID,
		item.ProductID,
	)

	switch itemType {
	case cartdom.CartItemTypeList:
		if item.Qty <= 0 ||
			item.InventoryID == "" ||
			item.ListID == "" ||
			item.ModelID == "" {
			return cartItemDoc{}, false
		}

		return cartItemDoc{
			Type:        string(cartdom.CartItemTypeList),
			InventoryID: item.InventoryID,
			ListID:      item.ListID,
			ModelID:     item.ModelID,
			Qty:         item.Qty,
		}, true

	case cartdom.CartItemTypeResale:
		if item.ResaleID == "" || item.ProductID == "" {
			return cartItemDoc{}, false
		}

		return cartItemDoc{
			Type:      string(cartdom.CartItemTypeResale),
			ResaleID:  item.ResaleID,
			ProductID: item.ProductID,
			Qty:       1,
		}, true

	default:
		return cartItemDoc{}, false
	}
}

func cartItemDomainFromDoc(
	item cartItemDoc,
) (cartdom.CartItem, bool) {
	normalized, valid := normalizeCartItemDoc(item)
	if !valid {
		return cartdom.CartItem{}, false
	}

	itemType := inferCartItemType(
		normalized.Type,
		normalized.InventoryID,
		normalized.ListID,
		normalized.ModelID,
		normalized.ResaleID,
		normalized.ProductID,
	)

	switch itemType {
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

func normalizeCartItemDoc(
	item cartItemDoc,
) (cartItemDoc, bool) {
	itemType := inferCartItemType(
		item.Type,
		item.InventoryID,
		item.ListID,
		item.ModelID,
		item.ResaleID,
		item.ProductID,
	)

	switch itemType {
	case cartdom.CartItemTypeList:
		if item.Qty <= 0 ||
			item.InventoryID == "" ||
			item.ListID == "" ||
			item.ModelID == "" {
			return cartItemDoc{}, false
		}

		return cartItemDoc{
			Type:        string(cartdom.CartItemTypeList),
			InventoryID: item.InventoryID,
			ListID:      item.ListID,
			ModelID:     item.ModelID,
			Qty:         item.Qty,
		}, true

	case cartdom.CartItemTypeResale:
		if item.ResaleID == "" || item.ProductID == "" {
			return cartItemDoc{}, false
		}

		return cartItemDoc{
			Type:      string(cartdom.CartItemTypeResale),
			ResaleID:  item.ResaleID,
			ProductID: item.ProductID,
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
	case cartdom.CartItemTypeList,
		cartdom.CartItemTypeResale:
		return itemType
	}

	if resaleID != "" || productID != "" {
		return cartdom.CartItemTypeResale
	}

	if inventoryID != "" ||
		listID != "" ||
		modelID != "" {
		return cartdom.CartItemTypeList
	}

	return ""
}
