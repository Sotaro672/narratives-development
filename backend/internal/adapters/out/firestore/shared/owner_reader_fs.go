// backend/internal/adapters/out/firestore/shared/owner_reader_fs.go
package shared

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"
)

// ------------------------------------------------------------
// Avatar wallet -> avatarId
// ------------------------------------------------------------

type AvatarWalletOwnerReaderFS struct {
	fs  *firestore.Client
	col string // default: "avatars"
}

func NewAvatarWalletOwnerReaderFS(fs *firestore.Client) *AvatarWalletOwnerReaderFS {
	return &AvatarWalletOwnerReaderFS{fs: fs, col: "avatars"}
}

// FindAvatarIDByWalletAddress: avatars where walletAddress == addr -> docID(=avatarId)
func (r *AvatarWalletOwnerReaderFS) FindAvatarIDByWalletAddress(
	ctx context.Context,
	walletAddress string,
) (string, bool, error) {
	if r == nil || r.fs == nil {
		return "", false, nil
	}
	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return "", false, nil
	}

	for _, field := range []string{"walletAddress", "wallet_address"} {
		iter := r.fs.Collection(r.col).Where(field, "==", addr).Limit(1).Documents(ctx)
		docs, err := iter.GetAll()
		if err != nil {
			return "", false, err
		}
		if len(docs) > 0 {
			return docs[0].Ref.ID, true, nil
		}
	}

	return "", false, nil
}

// ------------------------------------------------------------
// Brand wallet -> brandId
// ------------------------------------------------------------

type BrandWalletOwnerReaderFS struct {
	fs  *firestore.Client
	col string // default: "brands"
}

func NewBrandWalletOwnerReaderFS(fs *firestore.Client) *BrandWalletOwnerReaderFS {
	return &BrandWalletOwnerReaderFS{fs: fs, col: "brands"}
}

// FindBrandIDByWalletAddress: brands where walletAddress == addr -> docID(=brandId)
func (r *BrandWalletOwnerReaderFS) FindBrandIDByWalletAddress(
	ctx context.Context,
	walletAddress string,
) (string, bool, error) {
	if r == nil || r.fs == nil {
		return "", false, nil
	}
	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return "", false, nil
	}

	for _, field := range []string{"walletAddress", "wallet_address"} {
		iter := r.fs.Collection(r.col).Where(field, "==", addr).Limit(1).Documents(ctx)
		docs, err := iter.GetAll()
		if err != nil {
			return "", false, err
		}
		if len(docs) > 0 {
			return docs[0].Ref.ID, true, nil
		}
	}

	return "", false, nil
}
