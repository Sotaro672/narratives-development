// backend/internal/adapters/out/firestore/wallet_repository_fs.go
package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	walletdom "narratives/internal/domain/wallet"
)

// WalletRepositoryFS は WalletRepository の Firestore 実装です。
type WalletRepositoryFS struct {
	Client *firestore.Client
}

// NewWalletRepositoryFS は WalletRepositoryFS を生成します。
func NewWalletRepositoryFS(client *firestore.Client) *WalletRepositoryFS {
	return &WalletRepositoryFS{Client: client}
}

func (r *WalletRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("wallets")
}

// Firestore 上のスキーマ用 DTO
type walletDoc struct {
	WalletAddress string    `firestore:"walletAddress"`
	Tokens        []string  `firestore:"tokens"`
	LastUpdatedAt time.Time `firestore:"lastUpdatedAt"`
	Status        string    `firestore:"status"`
	CreatedAt     time.Time `firestore:"createdAt"`
	UpdatedAt     time.Time `firestore:"updatedAt"`
}

// GetByAddress は walletAddress（= ドキュメントID）で 1 件取得します。
func (r *WalletRepositoryFS) GetByAddress(ctx context.Context, addr string) (walletdom.Wallet, error) {
	if addr == "" {
		return walletdom.Wallet{}, walletdom.ErrInvalidWalletAddress
	}

	docRef := r.col().Doc(addr)
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return walletdom.Wallet{}, walletdom.ErrNotFound
		}
		return walletdom.Wallet{}, err
	}

	var d walletDoc
	if err := snap.DataTo(&d); err != nil {
		return walletdom.Wallet{}, err
	}

	// ドキュメントに walletAddress フィールドが無い場合は ID を採用
	if d.WalletAddress == "" {
		d.WalletAddress = addr
	}

	// Status が空なら active をデフォルトとする
	if d.Status == "" {
		d.Status = string(walletdom.StatusActive)
	}

	w, err := walletdom.NewFull(
		d.WalletAddress,
		d.Tokens,
		d.LastUpdatedAt,
		d.CreatedAt,
		d.UpdatedAt,
		walletdom.WalletStatus(d.Status),
	)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	return w, nil
}

// Save は Wallet を Firestore に保存（upsert）します。
func (r *WalletRepositoryFS) Save(ctx context.Context, w walletdom.Wallet) error {
	d := walletDoc{
		WalletAddress: w.WalletAddress,
		Tokens:        w.Tokens,
		LastUpdatedAt: w.LastUpdatedAt.UTC(),
		Status:        string(w.Status),
		CreatedAt:     w.CreatedAt.UTC(),
		UpdatedAt:     w.UpdatedAt.UTC(),
	}

	_, err := r.col().Doc(w.WalletAddress).Set(ctx, d)
	return err
}
