// backend/internal/adapters/out/firestore/wallet_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
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

var (
	ErrInvalidAvatarID      = errors.New("wallet_repository_fs: invalid avatarId")
	ErrInvalidLastUpdatedAt = errors.New("wallet_repository_fs: lastUpdatedAt is required")
	ErrInvalidAssetID       = errors.New("wallet_repository_fs: invalid assetId")
)

// Firestore 上のスキーマ用 DTO
//
// Collection design:
// - collection: wallets
// - docId: avatarId
// - fields: walletAddress, assetIds, lastUpdatedAt, status
// - avatarId field is NOT stored (docId is the source of truth).
type walletDoc struct {
	WalletAddress string    `firestore:"walletAddress"`
	AssetIDs      []string  `firestore:"assetIds"`
	LastUpdatedAt time.Time `firestore:"lastUpdatedAt"`
	Status        string    `firestore:"status"`
}

// GetByAvatarID は avatarId（= ドキュメントID）で 1 件取得します。
func (r *WalletRepositoryFS) GetByAvatarID(
	ctx context.Context,
	avatarID string,
) (walletdom.Wallet, error) {
	if r == nil || r.Client == nil {
		return walletdom.Wallet{}, errors.New(
			"wallet_repository_fs: firestore client is nil",
		)
	}

	aid := avatarID
	if aid == "" {
		return walletdom.Wallet{}, ErrInvalidAvatarID
	}

	snap, err := r.col().Doc(aid).Get(ctx)
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

	addr := d.WalletAddress
	if addr == "" {
		return walletdom.Wallet{}, walletdom.ErrInvalidWalletAddress
	}

	if d.Status == "" {
		d.Status = string(walletdom.StatusActive)
	}

	if d.LastUpdatedAt.IsZero() {
		return walletdom.Wallet{}, ErrInvalidLastUpdatedAt
	}

	wallet, err := walletdom.NewFull(
		addr,
		d.AssetIDs,
		d.LastUpdatedAt.UTC(),
		walletdom.WalletStatus(d.Status),
	)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	return wallet, nil
}

// GetByAddress は walletAddress で取得します。
func (r *WalletRepositoryFS) GetByAddress(
	ctx context.Context,
	addr string,
) (walletdom.Wallet, error) {
	if r == nil || r.Client == nil {
		return walletdom.Wallet{}, errors.New(
			"wallet_repository_fs: firestore client is nil",
		)
	}

	address := addr
	if address == "" {
		return walletdom.Wallet{}, walletdom.ErrInvalidWalletAddress
	}

	iter := r.col().
		Where("walletAddress", "==", address).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return walletdom.Wallet{}, walletdom.ErrNotFound
		}
		return walletdom.Wallet{}, err
	}

	var d walletDoc
	if err := doc.DataTo(&d); err != nil {
		return walletdom.Wallet{}, err
	}

	if d.Status == "" {
		d.Status = string(walletdom.StatusActive)
	}

	if d.LastUpdatedAt.IsZero() {
		return walletdom.Wallet{}, ErrInvalidLastUpdatedAt
	}

	wallet, err := walletdom.NewFull(
		d.WalletAddress,
		d.AssetIDs,
		d.LastUpdatedAt.UTC(),
		walletdom.WalletStatus(d.Status),
	)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	return wallet, nil
}

// GetWalletAddressByAssetID は assetIds に assetId を含む
// wallet の walletAddress を返します。
func (r *WalletRepositoryFS) GetWalletAddressByAssetID(
	ctx context.Context,
	assetID string,
) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New(
			"wallet_repository_fs: firestore client is nil",
		)
	}

	if assetID == "" {
		return "", ErrInvalidAssetID
	}

	iter := r.col().
		Where("assetIds", "array-contains", assetID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return "", walletdom.ErrNotFound
		}
		return "", err
	}

	var d walletDoc
	if err := doc.DataTo(&d); err != nil {
		return "", err
	}

	if d.WalletAddress == "" {
		return "", walletdom.ErrInvalidWalletAddress
	}

	return d.WalletAddress, nil
}

// Save は Wallet を Firestore に保存（upsert）します。
func (r *WalletRepositoryFS) Save(
	ctx context.Context,
	avatarID string,
	wallet walletdom.Wallet,
) error {
	if r == nil || r.Client == nil {
		return errors.New("wallet_repository_fs: firestore client is nil")
	}

	aid := avatarID
	if aid == "" {
		return ErrInvalidAvatarID
	}

	addr := wallet.WalletAddress
	if addr == "" {
		return walletdom.ErrInvalidWalletAddress
	}

	now := time.Now().UTC()
	last := wallet.LastUpdatedAt
	if last.IsZero() {
		last = now
	}

	walletStatus := wallet.Status
	if string(walletStatus) == "" {
		walletStatus = walletdom.StatusActive
	}

	d := walletDoc{
		WalletAddress: addr,
		AssetIDs:      wallet.AssetIDs,
		LastUpdatedAt: last.UTC(),
		Status:        string(walletStatus),
	}

	_, err := r.col().Doc(aid).Set(ctx, d)
	return err
}

// AddAssetIDToAvatarWalletItems は avatar wallet の assetIds 配列に
// assetId を冪等追加します。
// - Firestore の arrayUnion を使うことで、重複追加を防ぎ、並行更新にも強くします。
// - lastUpdatedAt も更新します。
func (r *WalletRepositoryFS) AddAssetIDToAvatarWalletItems(
	ctx context.Context,
	avatarID string,
	assetID string,
	now time.Time,
) error {
	if r == nil || r.Client == nil {
		return errors.New("wallet_repository_fs: firestore client is nil")
	}

	aid := avatarID
	if aid == "" {
		return ErrInvalidAvatarID
	}

	if assetID == "" {
		return ErrInvalidAssetID
	}

	at := now
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}

	_, err := r.col().Doc(aid).Update(ctx, []firestore.Update{
		{
			Path:  "assetIds",
			Value: firestore.ArrayUnion(assetID),
		},
		{
			Path:  "lastUpdatedAt",
			Value: at,
		},
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return walletdom.ErrNotFound
		}
		return err
	}

	return nil
}

// RemoveAssetIDFromAvatarWalletItems は avatar wallet の assetIds 配列から
// assetId を冪等削除します。
// - Firestore の arrayRemove を使うことで、存在しない値でも安全に実行できます。
// - lastUpdatedAt も更新します。
func (r *WalletRepositoryFS) RemoveAssetIDFromAvatarWalletItems(
	ctx context.Context,
	avatarID string,
	assetID string,
	now time.Time,
) error {
	if r == nil || r.Client == nil {
		return errors.New("wallet_repository_fs: firestore client is nil")
	}

	aid := avatarID
	if aid == "" {
		return ErrInvalidAvatarID
	}

	if assetID == "" {
		return ErrInvalidAssetID
	}

	at := now
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}

	_, err := r.col().Doc(aid).Update(ctx, []firestore.Update{
		{
			Path:  "assetIds",
			Value: firestore.ArrayRemove(assetID),
		},
		{
			Path:  "lastUpdatedAt",
			Value: at,
		},
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return walletdom.ErrNotFound
		}
		return err
	}

	return nil
}
