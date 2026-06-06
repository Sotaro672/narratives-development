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
// - docId: avatarId  ✅ (docId is the source of truth)
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
// - returns (zero, error) if repo is invalid
// - returns (zero, nil) if not found (nil policy)
func (r *CartRepositoryFS) GetByID(ctx context.Context, id string) (cartdom.Cart, error) {
	c, err := r.GetByAvatarID(ctx, id)
	if err != nil {
		return cartdom.Cart{}, err
	}
	if c == nil {
		// not found
		return cartdom.Cart{}, nil
	}
	return *c, nil
}

// GetByAvatarID returns (nil, nil) if not found (nil policy).
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
	// ✅ docId が source of truth（doc内に id フィールドが無くても必ず埋める）
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

	aid := c.ID
	if aid == "" {
		return errors.New("cart_repository_fs: Upsert requires cart.ID (= avatarId) as docId")
	}

	doc := cartDocFromDomain(c)

	// Overwrite full doc (simple & predictable).
	_, err := r.col().Doc(aid).Set(ctx, doc)
	return err
}

func (r *CartRepositoryFS) DeleteByAvatarID(ctx context.Context, avatarID string) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}

	aid := avatarID
	if aid == "" {
		return errors.New("cart_repository_fs: avatarID is empty")
	}

	_, err := r.col().Doc(aid).Delete(ctx)
	return err
}

// Clear empties cart items by docId (= avatarId / cartId).
// - items を空にして updatedAt を更新する
// - doc が存在しない場合は、空カートを作って成功扱いにする（冪等）
func (r *CartRepositoryFS) Clear(ctx context.Context, cartID string) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}

	id := cartID
	if id == "" {
		return errors.New("cart_repository_fs: cartID is empty")
	}

	now := time.Now().UTC()

	// try update first
	_, err := r.col().Doc(id).Update(ctx, []firestore.Update{
		{Path: "items", Value: map[string]any{}},
		{Path: "updatedAt", Value: now},
	})
	if err == nil {
		return nil
	}

	// If missing, create a new empty cart doc (idempotent behavior).
	if status.Code(err) == codes.NotFound {
		// expiresAt は TTL 運用のために一応入れておく（未使用なら TTL 設定側で無視されるだけ）
		expiresAt := now.Add(30 * 24 * time.Hour)

		doc := cartDoc{
			Items:     map[string]cartItemDoc{},
			CreatedAt: now,
			UpdatedAt: now,
			ExpiresAt: expiresAt,
		}
		_, setErr := r.col().Doc(id).Set(ctx, doc)
		return setErr
	}

	return err
}

// -----------------------------------------
// Firestore DTO
// -----------------------------------------

type cartDoc struct {
	// ✅ Items: itemKey -> CartItem
	Items map[string]cartItemDoc `firestore:"items"`

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
func cartDocFromSnapshot(snap *firestore.DocumentSnapshot) (cartDoc, error) {
	if snap == nil {
		return cartDoc{}, errors.New("cart_repository_fs: snapshot is nil")
	}

	raw := snap.Data()
	if raw == nil {
		// empty doc is unusual but handle defensively
		return cartDoc{
			Items: map[string]cartItemDoc{},
		}, nil
	}

	out := cartDoc{
		Items: map[string]cartItemDoc{},
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

	// items
	itemsAny, _ := raw["items"]
	m, ok := itemsAny.(map[string]any)
	if !ok || m == nil {
		// no items
		return out, nil
	}

	for k, v := range m {
		itemKey := k
		if itemKey == "" {
			continue
		}

		mv, ok := v.(map[string]any)
		if !ok {
			// unexpected shape -> skip
			continue
		}

		inv := asString(mv["inventoryId"])
		lid := asString(mv["listId"])
		mid := asString(mv["modelId"])
		qty := asInt(mv["qty"])

		// 必須チェック（qty > 0 は必須）
		if qty <= 0 {
			continue
		}

		out.Items[itemKey] = cartItemDoc{
			InventoryID: inv,
			ListID:      lid,
			ModelID:     mid,
			Qty:         qty,
		}
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

			inv := it.InventoryID
			lid := it.ListID
			mid := it.ModelID
			qty := it.Qty

			// qty は必須、ID も必須（空は捨てる）
			if qty <= 0 || inv == "" || lid == "" || mid == "" {
				continue
			}

			normalized := cartItemDoc{
				InventoryID: inv,
				ListID:      lid,
				ModelID:     mid,
				Qty:         qty,
			}

			if existing, ok := items[k]; ok {
				// 同一キーは qty を合算（IDs は既存優先、ただし空なら埋める）
				if existing.InventoryID == "" {
					existing.InventoryID = inv
				}
				if existing.ListID == "" {
					existing.ListID = lid
				}
				if existing.ModelID == "" {
					existing.ModelID = mid
				}
				existing.Qty = existing.Qty + qty
				items[k] = existing
			} else {
				items[k] = normalized
			}
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

			inv := it.InventoryID
			lid := it.ListID
			mid := it.ModelID
			qty := it.Qty

			if qty <= 0 {
				continue
			}

			normalized := cartdom.CartItem{
				InventoryID: inv,
				ListID:      lid,
				ModelID:     mid,
				Qty:         qty,
			}

			if existing, ok := items[k]; ok {
				// 重複キーは qty を合算
				existing.Qty = existing.Qty + qty
				// IDs が空なら埋める
				if existing.InventoryID == "" {
					existing.InventoryID = inv
				}
				if existing.ListID == "" {
					existing.ListID = lid
				}
				if existing.ModelID == "" {
					existing.ModelID = mid
				}
				items[k] = existing
			} else {
				items[k] = normalized
			}
		}
	}

	return &cartdom.Cart{
		// ID は呼び出し元（docId）で必ず埋める
		Items:     items,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		ExpiresAt: d.ExpiresAt,
	}
}
