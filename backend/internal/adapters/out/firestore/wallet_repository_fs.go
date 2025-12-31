// backend/internal/adapters/out/firestore/wallet_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
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
	AvatarID      string    `firestore:"avatarId"`
	WalletAddress string    `firestore:"walletAddress"`
	Tokens        []string  `firestore:"tokens"`
	LastUpdatedAt time.Time `firestore:"lastUpdatedAt"`
	Status        string    `firestore:"status"`
}

// GetByAddress は walletAddress（= ドキュメントID）で 1 件取得します。
func (r *WalletRepositoryFS) GetByAddress(ctx context.Context, addr string) (walletdom.Wallet, error) {
	if r == nil || r.Client == nil {
		return walletdom.Wallet{}, errors.New("wallet_repository_fs: firestore client is nil")
	}

	a := strings.TrimSpace(addr)
	if a == "" {
		return walletdom.Wallet{}, walletdom.ErrInvalidWalletAddress
	}

	docRef := r.col().Doc(a)
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
	if strings.TrimSpace(d.WalletAddress) == "" {
		d.WalletAddress = a
	}

	// Status が空なら active をデフォルトとする
	if strings.TrimSpace(d.Status) == "" {
		d.Status = string(walletdom.StatusActive)
	}

	// lastUpdatedAt が空なら（過去データ互換）updated 相当として now を入れる
	if d.LastUpdatedAt.IsZero() {
		d.LastUpdatedAt = time.Now().UTC()
	}

	w, err := walletdom.NewFull(
		d.AvatarID,
		d.WalletAddress,
		d.Tokens,
		d.LastUpdatedAt,
		walletdom.WalletStatus(d.Status),
	)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	return w, nil
}

// Save は Wallet を Firestore に保存（upsert）します。
func (r *WalletRepositoryFS) Save(ctx context.Context, w walletdom.Wallet) error {
	if r == nil || r.Client == nil {
		return errors.New("wallet_repository_fs: firestore client is nil")
	}

	addr := strings.TrimSpace(w.WalletAddress)
	if addr == "" {
		return walletdom.ErrInvalidWalletAddress
	}

	d := walletDoc{
		AvatarID:      strings.TrimSpace(w.AvatarID),
		WalletAddress: addr,
		Tokens:        w.Tokens,
		LastUpdatedAt: w.LastUpdatedAt.UTC(),
		Status:        string(w.Status),
	}

	_, err := r.col().Doc(addr).Set(ctx, d)
	return err
}
