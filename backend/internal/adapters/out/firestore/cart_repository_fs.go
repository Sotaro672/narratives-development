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
// - docId: avatarId
// - fields: avatarId, items(map), createdAt, updatedAt, expiresAt, ordered
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

	// docId is the source of truth for avatarId
	doc.AvatarID = aid

	return doc.toDomain(), nil
}

func (r *CartRepositoryFS) Upsert(ctx context.Context, c *cartdom.Cart) error {
	if r == nil || r.Client == nil {
		return errors.New("cart_repository_fs: firestore client is nil")
	}
	if c == nil {
		return errors.New("cart_repository_fs: cart is nil")
	}

	aid := strings.TrimSpace(c.AvatarID)
	if aid == "" {
		return errors.New("cart_repository_fs: avatarID is empty")
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
	AvatarID string         `firestore:"avatarId"`
	Items    map[string]int `firestore:"items"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
	ExpiresAt time.Time `firestore:"expiresAt"`

	Ordered bool `firestore:"ordered"`
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
		AvatarID:  strings.TrimSpace(c.AvatarID),
		Items:     items,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		ExpiresAt: c.ExpiresAt,
		Ordered:   c.Ordered,
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
		AvatarID:  strings.TrimSpace(d.AvatarID),
		Items:     items,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		ExpiresAt: d.ExpiresAt,
		Ordered:   d.Ordered,
	}
}
