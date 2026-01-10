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

	// ✅ IMPORTANT:
	// 過去に items を map[string]int で保存していた場合や、途中で schema が変わった場合、
	// DataTo(&struct{ Items map[string]X }) が型不一致で 500 になり得る。
	// そこで snap.Data() を取り、後方互換で自前パースする。
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

	aid := strings.TrimSpace(c.ID)
	if aid == "" {
		return errors.New("cart_repository_fs: Upsert requires cart.ID (= avatarId) as docId")
	}

	doc := cartDocFromDomain(c)

	// Overwrite full doc (simple & predictable).
	_, err := r.col().Doc(aid).Set(ctx, doc)
	return err
}

// ✅ (optional) explicit docId upsert API (kept for compatibility / explicitness)
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
// - items を空にして updatedAt を更新する
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
	// NOTE: domain struct を直接 firestore DTO にしない（後方互換 & 柔軟にするため）
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

// cartDocFromSnapshot parses Firestore document data with backward compatibility.
//
// Supported shapes:
// 1) items: map[itemKey] = {inventoryId, listId, modelId, qty}
// 2) items: map[itemKey] = qty (legacy)
//   - in this case we keep ModelID=itemKey and Qty=qty, other IDs empty
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
		itemKey := strings.TrimSpace(k)
		if itemKey == "" {
			continue
		}

		// new shape: map[string]any
		if mv, ok := v.(map[string]any); ok {
			inv := strings.TrimSpace(asString(mv["inventoryId"]))
			lid := strings.TrimSpace(asString(mv["listId"]))
			mid := strings.TrimSpace(asString(mv["modelId"]))
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
			continue
		}

		// legacy shape: qty only
		qty := asInt(v)
		if qty <= 0 {
			continue
		}
		out.Items[itemKey] = cartItemDoc{
			InventoryID: "",
			ListID:      "",
			ModelID:     itemKey,
			Qty:         qty,
		}
	}

	return out, nil
}

func cartDocFromDomain(c *cartdom.Cart) cartDoc {
	items := map[string]cartItemDoc{}
	if c != nil && c.Items != nil {
		for k, it := range c.Items {
			k2 := strings.TrimSpace(k)
			if k2 == "" {
				continue
			}

			inv := strings.TrimSpace(it.InventoryID)
			lid := strings.TrimSpace(it.ListID)
			mid := strings.TrimSpace(it.ModelID)
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

			// normalize key if it had spaces
			if existing, ok := items[k2]; ok {
				// 同一キーは qty を合算（IDs は既存優先、ただし空なら埋める）
				if strings.TrimSpace(existing.InventoryID) == "" {
					existing.InventoryID = inv
				}
				if strings.TrimSpace(existing.ListID) == "" {
					existing.ListID = lid
				}
				if strings.TrimSpace(existing.ModelID) == "" {
					existing.ModelID = mid
				}
				existing.Qty = existing.Qty + qty
				items[k2] = existing
			} else {
				items[k2] = normalized
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
			k2 := strings.TrimSpace(k)
			if k2 == "" {
				continue
			}

			inv := strings.TrimSpace(it.InventoryID)
			lid := strings.TrimSpace(it.ListID)
			mid := strings.TrimSpace(it.ModelID)
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

			if existing, ok := items[k2]; ok {
				// 重複キーは qty を合算
				existing.Qty = existing.Qty + qty
				// IDs が空なら埋める
				if strings.TrimSpace(existing.InventoryID) == "" {
					existing.InventoryID = inv
				}
				if strings.TrimSpace(existing.ListID) == "" {
					existing.ListID = lid
				}
				if strings.TrimSpace(existing.ModelID) == "" {
					existing.ModelID = mid
				}
				items[k2] = existing
			} else {
				items[k2] = normalized
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
