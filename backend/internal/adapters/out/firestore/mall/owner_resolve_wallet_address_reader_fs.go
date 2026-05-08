// backend/internal/adapters/out/firestore/mall/owner_resolve_wallet_address_reader_fs.go
package mall

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"

	"google.golang.org/api/iterator"

	sharedquery "narratives/internal/application/query/shared"
)

var (
	errOwnerResolveCollectionEmpty = errors.New("firestore.mall: owner resolve collection is empty")
)

type brandWalletAddressReaderFS struct {
	fs  *firestore.Client
	col string
}

func NewBrandWalletAddressReaderFS(fs *firestore.Client, col string) interface {
	FindBrandIDByWalletAddress(ctx context.Context, walletAddress string) (string, error)
} {
	return brandWalletAddressReaderFS{fs: fs, col: col}
}

func (r brandWalletAddressReaderFS) FindBrandIDByWalletAddress(ctx context.Context, walletAddress string) (string, error) {
	if r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}
	addr := walletAddress
	if addr == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	col := r.col
	if col == "" {
		return "", errOwnerResolveCollectionEmpty
	}

	it := r.fs.Collection(col).
		Where("walletAddress", "==", addr).
		Limit(1).
		Documents(ctx)

	doc, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return "", nil
		}
		return "", err
	}
	if doc == nil || doc.Ref == nil {
		return "", nil
	}
	return doc.Ref.ID, nil
}

type avatarWalletAddressReaderFS struct {
	fs  *firestore.Client
	col string
}

func NewAvatarWalletAddressReaderFS(fs *firestore.Client, col string) interface {
	FindAvatarIDByWalletAddress(ctx context.Context, walletAddress string) (string, error)
} {
	return avatarWalletAddressReaderFS{fs: fs, col: col}
}

func (r avatarWalletAddressReaderFS) FindAvatarIDByWalletAddress(ctx context.Context, walletAddress string) (string, error) {
	if r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}
	addr := walletAddress
	if addr == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	col := r.col
	if col == "" {
		return "", errOwnerResolveCollectionEmpty
	}

	it := r.fs.Collection(col).
		Where("walletAddress", "==", addr).
		Limit(1).
		Documents(ctx)

	doc, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return "", nil
		}
		return "", err
	}
	if doc == nil || doc.Ref == nil {
		return "", nil
	}
	return doc.Ref.ID, nil
}
