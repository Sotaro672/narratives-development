// backend/internal/adapters/out/firestore/mall/transfer_token_owner_updater_fs.go
package mall

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"

	usecase "narratives/internal/application/usecase"
)

type tokenOwnerUpdaterFS struct {
	fs  *firestore.Client
	col string
}

func NewTokenOwnerUpdaterFS(fs *firestore.Client, col string) usecase.TokenOwnerUpdater {
	return &tokenOwnerUpdaterFS{fs: fs, col: col}
}

func (r *tokenOwnerUpdaterFS) UpdateToAddressByProductID(ctx context.Context, productID string, newToAddress string, now time.Time, txSignature string) error {
	if r == nil || r.fs == nil {
		return errTokenResolverNotConfigured
	}
	pid := productID
	if pid == "" {
		return errors.New("tokenOwnerUpdaterFS: productId is empty")
	}
	addr := newToAddress
	if addr == "" {
		return errors.New("tokenOwnerUpdaterFS: newToAddress is empty")
	}
	col := r.col
	if col == "" {
		col = "tokens"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	ref := r.fs.Collection(col).Doc(pid)

	_, err := ref.Set(ctx, map[string]any{
		"toAddress":       addr,
		"updatedAt":       now,
		"lastTxSignature": txSignature,
		"ownerUpdatedAt":  now,
	}, firestore.MergeAll)
	return err
}
