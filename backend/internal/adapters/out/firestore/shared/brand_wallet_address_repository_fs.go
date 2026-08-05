package shared

import (
	"context"

	"cloud.google.com/go/firestore"
)

// BrandWalletAddressReaderFS implements sharedquery.BrandWalletAddressReader.
type BrandWalletAddressReaderFS struct {
	lookup walletAddressLookupFS
}

func NewBrandWalletAddressReaderFS(
	fs *firestore.Client,
	collection string,
) *BrandWalletAddressReaderFS {
	return &BrandWalletAddressReaderFS{
		lookup: newWalletAddressLookupFS(
			fs,
			collection,
		),
	}
}

func (r *BrandWalletAddressReaderFS) FindBrandIDByWalletAddress(
	ctx context.Context,
	walletAddress string,
) (string, error) {
	if r == nil {
		return "", nil
	}

	return r.lookup.findIDByWalletAddress(
		ctx,
		walletAddress,
	)
}
