// backend/internal/adapters/out/firestore/wallet_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
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
	ErrInvalidAvatarID = errors.New("wallet_repository_fs: invalid avatarId")
	ErrInvalidAssetID  = errors.New("wallet_repository_fs: invalid assetId")
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
func (r *WalletRepositoryFS) GetByAvatarID(ctx context.Context, avatarID string) (walletdom.Wallet, error) {
	if r == nil || r.Client == nil {
		return walletdom.Wallet{}, errors.New("wallet_repository_fs: firestore client is nil")
	}
	if avatarID == "" {
		return walletdom.Wallet{}, ErrInvalidAvatarID
	}

	snap, err := r.col().Doc(avatarID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return walletdom.Wallet{}, walletdom.ErrNotFound
		}
		return walletdom.Wallet{}, err
	}

	return decodeWalletSnapshot(snap)
}

// GetByAddress は walletAddress で取得します。
func (r *WalletRepositoryFS) GetByAddress(ctx context.Context, addr string) (walletdom.Wallet, error) {
	if r == nil || r.Client == nil {
		return walletdom.Wallet{}, errors.New("wallet_repository_fs: firestore client is nil")
	}
	if addr == "" {
		return walletdom.Wallet{}, walletdom.ErrInvalidWalletAddress
	}

	iter := r.col().Where("walletAddress", "==", addr).Limit(2).Documents(ctx)
	defer iter.Stop()

	first, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return walletdom.Wallet{}, walletdom.ErrNotFound
		}
		return walletdom.Wallet{}, err
	}

	wallet, err := decodeWalletSnapshot(first)
	if err != nil {
		return walletdom.Wallet{}, err
	}
	if wallet.WalletAddress != addr {
		return walletdom.Wallet{}, fmt.Errorf("wallet_repository_fs: walletAddress mismatch for avatarId %q", first.Ref.ID)
	}

	second, err := iter.Next()
	if err == nil {
		return walletdom.Wallet{}, fmt.Errorf("wallet_repository_fs: duplicate walletAddress %q in avatarIds %q and %q", addr, first.Ref.ID, second.Ref.ID)
	}
	if !errors.Is(err, iterator.Done) {
		return walletdom.Wallet{}, err
	}

	return wallet, nil
}

// GetWalletAddressByAssetID は assetIds に assetId を含む wallet の walletAddress を返します。
func (r *WalletRepositoryFS) GetWalletAddressByAssetID(ctx context.Context, assetID string) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New("wallet_repository_fs: firestore client is nil")
	}
	if assetID == "" {
		return "", ErrInvalidAssetID
	}

	iter := r.col().Where("assetIds", "array-contains", assetID).Limit(2).Documents(ctx)
	defer iter.Stop()

	first, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return "", walletdom.ErrNotFound
		}
		return "", err
	}

	wallet, err := decodeWalletSnapshot(first)
	if err != nil {
		return "", err
	}
	if !wallet.HasAssetID(assetID) {
		return "", fmt.Errorf("wallet_repository_fs: assetId %q is not present in avatarId %q", assetID, first.Ref.ID)
	}

	second, err := iter.Next()
	if err == nil {
		return "", fmt.Errorf("wallet_repository_fs: assetId %q is owned by multiple avatarIds %q and %q", assetID, first.Ref.ID, second.Ref.ID)
	}
	if !errors.Is(err, iterator.Done) {
		return "", err
	}

	return wallet.WalletAddress, nil
}

// Save は Wallet を Firestore に保存（upsert）します。
func (r *WalletRepositoryFS) Save(ctx context.Context, avatarID string, wallet walletdom.Wallet) error {
	if r == nil || r.Client == nil {
		return errors.New("wallet_repository_fs: firestore client is nil")
	}
	if avatarID == "" {
		return ErrInvalidAvatarID
	}
	if err := validateCanonicalWallet(wallet); err != nil {
		return err
	}

	d := walletDoc{
		WalletAddress: wallet.WalletAddress,
		AssetIDs:      cloneWalletAssetIDs(wallet.AssetIDs),
		LastUpdatedAt: wallet.LastUpdatedAt,
		Status:        string(wallet.Status),
	}

	_, err := r.col().Doc(avatarID).Set(ctx, d)
	return err
}

// AddAssetIDToAvatarWalletItems は avatar wallet の assetIds 配列に assetId を冪等追加します。
// Firestore arrayUnion を使い、lastUpdatedAt も更新します。
func (r *WalletRepositoryFS) AddAssetIDToAvatarWalletItems(ctx context.Context, avatarID string, assetID string, now time.Time) error {
	if r == nil || r.Client == nil {
		return errors.New("wallet_repository_fs: firestore client is nil")
	}
	if avatarID == "" {
		return ErrInvalidAvatarID
	}
	if assetID == "" {
		return ErrInvalidAssetID
	}
	if now.IsZero() {
		return walletdom.ErrInvalidLastUpdatedAt
	}

	ref := r.col().Doc(avatarID)
	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return walletdom.ErrNotFound
			}
			return err
		}

		current, err := decodeWalletSnapshot(snap)
		if err != nil {
			return err
		}

		// AddAssetID の domain validation を利用して assetId の形式を検証する。
		probe := cloneWallet(current)
		if err := probe.AddAssetID(assetID, now); err != nil {
			return err
		}

		return tx.Update(ref, []firestore.Update{
			{Path: "assetIds", Value: firestore.ArrayUnion(assetID)},
			{Path: "lastUpdatedAt", Value: now},
		})
	})
}

// RemoveAssetIDFromAvatarWalletItems は avatar wallet の assetIds 配列から assetId を冪等削除します。
// Firestore arrayRemove を使い、lastUpdatedAt も更新します。
func (r *WalletRepositoryFS) RemoveAssetIDFromAvatarWalletItems(ctx context.Context, avatarID string, assetID string, now time.Time) error {
	if r == nil || r.Client == nil {
		return errors.New("wallet_repository_fs: firestore client is nil")
	}
	if avatarID == "" {
		return ErrInvalidAvatarID
	}
	if assetID == "" {
		return ErrInvalidAssetID
	}
	if now.IsZero() {
		return walletdom.ErrInvalidLastUpdatedAt
	}

	ref := r.col().Doc(avatarID)
	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return walletdom.ErrNotFound
			}
			return err
		}

		current, err := decodeWalletSnapshot(snap)
		if err != nil {
			return err
		}

		// RemoveAssetID は形式検証を行わないため、AddAssetID を検証用途だけに利用する。
		probe := cloneWallet(current)
		if err := probe.AddAssetID(assetID, now); err != nil {
			return err
		}

		return tx.Update(ref, []firestore.Update{
			{Path: "assetIds", Value: firestore.ArrayRemove(assetID)},
			{Path: "lastUpdatedAt", Value: now},
		})
	})
}

// ============================================================
// Firestore decode / validation
// ============================================================

func decodeWalletSnapshot(snap *firestore.DocumentSnapshot) (walletdom.Wallet, error) {
	if snap == nil || snap.Ref == nil || snap.Ref.ID == "" {
		return walletdom.Wallet{}, errors.New("wallet_repository_fs: invalid wallet document snapshot")
	}

	var d walletDoc
	if err := snap.DataTo(&d); err != nil {
		return walletdom.Wallet{}, fmt.Errorf("wallet_repository_fs: decode wallet document %q: %w", snap.Ref.ID, err)
	}

	wallet := walletdom.Wallet{
		WalletAddress: d.WalletAddress,
		AssetIDs:      cloneWalletAssetIDs(d.AssetIDs),
		LastUpdatedAt: d.LastUpdatedAt,
		Status:        walletdom.WalletStatus(d.Status),
	}
	if err := validateCanonicalWallet(wallet); err != nil {
		return walletdom.Wallet{}, fmt.Errorf("wallet_repository_fs: invalid wallet document %q: %w", snap.Ref.ID, err)
	}

	return wallet, nil
}

// validateCanonicalWallet は domain の NewFull を validation にだけ利用し、
// dedup / UTC normalize 後の値を永続データの代わりとして返さない。
// Firestore の assetIds が既に canonical でなければ error とする。
func validateCanonicalWallet(wallet walletdom.Wallet) error {
	validated, err := walletdom.NewFull(wallet.WalletAddress, wallet.AssetIDs, wallet.LastUpdatedAt, wallet.Status)
	if err != nil {
		return err
	}

	if len(validated.AssetIDs) != len(wallet.AssetIDs) {
		return walletdom.ErrInvalidAssetID
	}
	for i := range wallet.AssetIDs {
		if validated.AssetIDs[i] != wallet.AssetIDs[i] {
			return walletdom.ErrInvalidAssetID
		}
	}

	return nil
}

func cloneWallet(wallet walletdom.Wallet) walletdom.Wallet {
	return walletdom.Wallet{
		WalletAddress: wallet.WalletAddress,
		AssetIDs:      cloneWalletAssetIDs(wallet.AssetIDs),
		LastUpdatedAt: wallet.LastUpdatedAt,
		Status:        wallet.Status,
	}
}

func cloneWalletAssetIDs(assetIDs []string) []string {
	if assetIDs == nil {
		return nil
	}
	out := make([]string, len(assetIDs))
	copy(out, assetIDs)
	return out
}

// ============================================================
// compile-time interface check
// ============================================================

var _ walletdom.Repository = (*WalletRepositoryFS)(nil)
