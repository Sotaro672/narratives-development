// backend\internal\adapters\out\firestore\shared\wallet_address_lookup_fs.go
package shared

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	sharedquery "narratives/internal/application/query/shared"
)

type walletAddressLookupFS struct {
	fs         *firestore.Client
	collection string
}

func newWalletAddressLookupFS(
	fs *firestore.Client,
	collection string,
) walletAddressLookupFS {
	return walletAddressLookupFS{
		fs:         fs,
		collection: collection,
	}
}

func (r walletAddressLookupFS) findIDByWalletAddress(
	ctx context.Context,
	walletAddress string,
) (string, error) {
	if r.fs == nil {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}

	if walletAddress == "" {
		return "", sharedquery.ErrInvalidWalletAddress
	}

	if r.collection == "" {
		return "", sharedquery.ErrOwnerResolveNotConfigured
	}

	it := r.fs.Collection(r.collection).
		Where("walletAddress", "==", walletAddress).
		Limit(1).
		Documents(ctx)
	defer it.Stop()

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
