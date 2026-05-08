// backend/internal/adapters/out/firestore/shared/brand_wallet_address_repository_fs.go
package shared

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	sharedquery "narratives/internal/application/query/shared"
)

// BrandWalletAddressReaderFS implements sharedquery.BrandWalletAddressReader.
type BrandWalletAddressReaderFS struct {
	fs         *firestore.Client
	collection string
}

func NewBrandWalletAddressReaderFS(fs *firestore.Client, collection string) *BrandWalletAddressReaderFS {
	return &BrandWalletAddressReaderFS{
		fs:         fs,
		collection: collection,
	}
}

func (r *BrandWalletAddressReaderFS) FindBrandIDByWalletAddress(ctx context.Context, walletAddress string) (string, error) {
	if r == nil || r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}
	addr := walletAddress
	if addr == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	col := r.collection
	if col == "" {
		col = "brands"
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
