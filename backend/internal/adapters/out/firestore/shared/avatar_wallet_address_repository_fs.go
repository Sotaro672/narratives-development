package shared

import (
	"context"

	"cloud.google.com/go/firestore"
)

// AvatarWalletAddressReaderFS implements sharedquery.AvatarWalletAddressReader.
type AvatarWalletAddressReaderFS struct {
	lookup walletAddressLookupFS
}

func NewAvatarWalletAddressReaderFS(
	fs *firestore.Client,
	collection string,
) *AvatarWalletAddressReaderFS {
	return &AvatarWalletAddressReaderFS{
		lookup: newWalletAddressLookupFS(
			fs,
			collection,
		),
	}
}

func (r *AvatarWalletAddressReaderFS) FindAvatarIDByWalletAddress(
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
