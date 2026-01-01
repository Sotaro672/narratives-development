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

	var doc cartDoc
	if err := snap.DataTo(&doc); err != nil {
		return nil, err
	}

	d := doc.toDomain()
	// ✅ docId が source of truth（doc内に id フィールドが無くても必ず埋める）
	d.ID = aid
	return d, nil
}

// Upsert saves cart by docId=cart.ID (= avatarId).
// Cart ドメインに ID を持たせたので、Upsert はそれを docId として使う。
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

// -----------------------------------------
// Firestore DTO
// -----------------------------------------

type cartDoc struct {
	Items map[string]int `firestore:"items"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
	ExpiresAt time.Time `firestore:"expiresAt"`
}

func cartDocFromDomain(c *cartdom.Cart) cartDoc {
	items := map[string]int{}
	if c.Items != nil {
		for k, v := range c.Items {
			k2 := strings.TrimSpace(k)
			if k2 == "" || v <= 0 {
				continue
			}
			items[k2] = items[k2] + v
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
	items := map[string]int{}
	if d.Items != nil {
		for k, v := range d.Items {
			k2 := strings.TrimSpace(k)
			if k2 == "" || v <= 0 {
				continue
			}
			items[k2] = items[k2] + v
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
