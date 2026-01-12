// backend/internal/adapters/out/firestore/shared/avatar_wallet_address_repository_fs.go
package shared

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	sharedquery "narratives/internal/application/query/shared"
)

// AvatarWalletAddressReaderFS implements sharedquery.AvatarWalletAddressReader.
type AvatarWalletAddressReaderFS struct {
	fs         *firestore.Client
	collection string
}

func NewAvatarWalletAddressReaderFS(fs *firestore.Client, collection string) *AvatarWalletAddressReaderFS {
	return &AvatarWalletAddressReaderFS{
		fs:         fs,
		collection: strings.TrimSpace(collection),
	}
}

func (r *AvatarWalletAddressReaderFS) FindAvatarIDByWalletAddress(ctx context.Context, walletAddress string) (string, error) {
	if r == nil || r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}
	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	col := strings.TrimSpace(r.collection)
	if col == "" {
		col = "avatars"
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
	return strings.TrimSpace(doc.Ref.ID), nil
}
