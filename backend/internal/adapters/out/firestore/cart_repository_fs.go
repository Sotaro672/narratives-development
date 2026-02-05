// backend/internal/adapters/out/firestore/cart_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
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
// - docId: avatarId  ✅ (docId is the source of truth)
// - fields: items([]), createdAt, updatedAt, expiresAt
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

// GetByAvatarID returns (nil, nil) if not found (nil policy).
func (r *CartRepositoryFS) GetByAvatarID(ctx context.Context, avatarID string) (*cartdom.Cart, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("cart_repository_fs: firestore client is nil")
	}

	aid := strings.TrimSpace(avatarID)
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
	// ✅ docId is the source of truth
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

	aid := strings.TrimSpace(c.ID)
	if aid == "" {
		return errors.New("cart_repository_fs: Upsert requires cart.ID (= avatarId) as docId")
	}

	doc := cartDocFromDomain(c)

	// Overwrite full doc (simple & predictable).
	_, err := r.col().Doc(aid).Set(ctx, doc)
	return err
}

// (optional) explicit docId upsert API
func (r *CartRepositoryFS) UpsertByAvatarID(ctx context.Context, avatarID string, c *cartdom.Cart) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}
	if c == nil {
		return errors.New("cart_repository_fs: cart is nil")
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return errors.New("cart_repository_fs: avatarID is empty")
	}

	doc := cartDocFromDomain(c)

	_, err := r.col().Doc(aid).Set(ctx, doc)
	return err
}

func (r *CartRepositoryFS) DeleteByAvatarID(ctx context.Context, avatarID string) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return errors.New("cart_repository_fs: avatarID is empty")
	}

	_, err := r.col().Doc(aid).Delete(ctx)
	return err
}

// Clear empties cart items by docId (= avatarId / cartId).
// - items を空配列にして updatedAt を更新する
// - doc が存在しない場合は、空カートを作って成功扱いにする（冪等）
func (r *CartRepositoryFS) Clear(ctx context.Context, cartID string) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}

	id := strings.TrimSpace(cartID)
	if id == "" {
		return errors.New("cart_repository_fs: cartID is empty")
	}

	now := time.Now().UTC()

	// try update first
	_, err := r.col().Doc(id).Update(ctx, []firestore.Update{
		{Path: "items", Value: []any{}},
		{Path: "updatedAt", Value: now},
		{Path: "expiresAt", Value: now.Add(cartdom.DefaultCartTTL)},
	})
	if err == nil {
		return nil
	}

	// If missing, create a new empty cart doc (idempotent behavior).
	if status.Code(err) == codes.NotFound {
		doc := cartDoc{
			Items:     []cartItemDoc{},
			CreatedAt: now,
			UpdatedAt: now,
			ExpiresAt: now.Add(cartdom.DefaultCartTTL),
		}
		_, setErr := r.col().Doc(id).Set(ctx, doc)
		return setErr
	}

	return err
}

// -----------------------------------------
// Firestore DTO (NO backward-compat)
// -----------------------------------------

type cartDoc struct {
	// ✅ Items: []CartItem (no map key)
	Items []cartItemDoc `firestore:"items"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
	ExpiresAt time.Time `firestore:"expiresAt"`
}

type cartItemDoc struct {
	InventoryID string `firestore:"inventoryId"`
	ListID      string `firestore:"listId"`
	ModelID     string `firestore:"modelId"`
	Qty         int    `firestore:"qty"`
}

// cartDocFromSnapshot parses Firestore document data.
//
// Supported shape ONLY:
// - items: [{inventoryId, listId, modelId, qty}, ...]
//
// ❌ Backward compatibility is removed intentionally.
func cartDocFromSnapshot(snap *firestore.DocumentSnapshot) (cartDoc, error) {
	if snap == nil {
		return cartDoc{}, errors.New("cart_repository_fs: snapshot is nil")
	}

	raw := snap.Data()
	if raw == nil {
		// empty doc is unusual but handle defensively
		return cartDoc{
			Items: []cartItemDoc{},
		}, nil
	}

	out := cartDoc{
		Items: []cartItemDoc{},
	}

	// times
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

	// items (array only)
	itemsAny, has := raw["items"]
	if !has || itemsAny == nil {
		return out, nil
	}

	arr, ok := itemsAny.([]any)
	if !ok {
		return cartDoc{}, errors.New("cart_repository_fs: invalid items type (expected array)")
	}

	for _, v := range arr {
		mv, ok := v.(map[string]any)
		if !ok || mv == nil {
			// skip invalid entry
			continue
		}

		inv := strings.TrimSpace(asString(mv["inventoryId"]))
		lid := strings.TrimSpace(asString(mv["listId"]))
		mid := strings.TrimSpace(asString(mv["modelId"]))
		qty := asInt(mv["qty"])

		// strict: all required
		if inv == "" || lid == "" || mid == "" || qty <= 0 {
			continue
		}

		out.Items = append(out.Items, cartItemDoc{
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		})
	}

	return out, nil
}

func cartDocFromDomain(c *cartdom.Cart) cartDoc {
	items := make([]cartItemDoc, 0)
	if c != nil && len(c.Items) > 0 {
		for _, it := range c.Items {
			inv := strings.TrimSpace(it.InventoryID)
			lid := strings.TrimSpace(it.ListID)
			mid := strings.TrimSpace(it.ModelID)
			qty := it.Qty

			if inv == "" || lid == "" || mid == "" || qty <= 0 {
				continue
			}

			items = append(items, cartItemDoc{
				InventoryID: inv,
				ListID:      lid,
				ModelID:     mid,
				Qty:         qty,
			})
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
	items := make([]cartdom.CartItem, 0)

	for _, it := range d.Items {
		inv := strings.TrimSpace(it.InventoryID)
		lid := strings.TrimSpace(it.ListID)
		mid := strings.TrimSpace(it.ModelID)
		qty := it.Qty

		if inv == "" || lid == "" || mid == "" || qty <= 0 {
			continue
		}

		items = append(items, cartdom.CartItem{
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		})
	}

	return &cartdom.Cart{
		// ID は呼び出し元（docId）で必ず埋める
		Items:     items,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		ExpiresAt: d.ExpiresAt,
	}
}
