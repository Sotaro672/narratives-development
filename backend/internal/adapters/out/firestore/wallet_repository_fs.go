// backend/internal/adapters/out/firestore/wallet_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
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
)

// Firestore 上のスキーマ用 DTO
//
// ✅ Collection design (after change):
// - collection: wallets
// - docId: avatarId
// - fields: walletAddress, tokens, lastUpdatedAt, status
// - ❌ avatarId field is NOT stored (docId is the source of truth).
type walletDoc struct {
	WalletAddress string    `firestore:"walletAddress"`
	Tokens        []string  `firestore:"tokens"`
	LastUpdatedAt time.Time `firestore:"lastUpdatedAt"`
	Status        string    `firestore:"status"`
}

// GetByAvatarID は avatarId（= ドキュメントID）で 1 件取得します。
func (r *WalletRepositoryFS) GetByAvatarID(ctx context.Context, avatarID string) (walletdom.Wallet, error) {
	if r == nil || r.Client == nil {
		return walletdom.Wallet{}, errors.New("wallet_repository_fs: firestore client is nil")
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return walletdom.Wallet{}, ErrInvalidAvatarID
	}

	// ✅ 新仕様: docId = avatarId
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

	addr := strings.TrimSpace(d.WalletAddress)
	if addr == "" {
		return walletdom.Wallet{}, walletdom.ErrInvalidWalletAddress
	}

	// Status が空なら active をデフォルト
	if strings.TrimSpace(d.Status) == "" {
		d.Status = string(walletdom.StatusActive)
	}

	// lastUpdatedAt が空なら（過去データ互換）now
	if d.LastUpdatedAt.IsZero() {
		d.LastUpdatedAt = time.Now().UTC()
	}

	w, err := walletdom.NewFull(
		addr,
		d.Tokens,
		d.LastUpdatedAt.UTC(),
		walletdom.WalletStatus(strings.TrimSpace(d.Status)),
	)
	if err != nil {
		return walletdom.Wallet{}, err
	}
	return w, nil
}

// GetByAddress は walletAddress で取得します。
// ✅ 新仕様では docId ではないため、where(walletAddress==addr) で引きます。
// ✅ 互換: 旧仕様(docId=walletAddress)のデータも読めます。
func (r *WalletRepositoryFS) GetByAddress(ctx context.Context, addr string) (walletdom.Wallet, error) {
	if r == nil || r.Client == nil {
		return walletdom.Wallet{}, errors.New("wallet_repository_fs: firestore client is nil")
	}

	a := strings.TrimSpace(addr)
	if a == "" {
		return walletdom.Wallet{}, walletdom.ErrInvalidWalletAddress
	}

	// 1) ✅ 新仕様: where で検索（walletAddress == a）
	iter := r.col().Where("walletAddress", "==", a).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == nil {
		var d walletDoc
		if err := doc.DataTo(&d); err != nil {
			return walletdom.Wallet{}, err
		}

		if strings.TrimSpace(d.Status) == "" {
			d.Status = string(walletdom.StatusActive)
		}
		if d.LastUpdatedAt.IsZero() {
			d.LastUpdatedAt = time.Now().UTC()
		}

		w, err := walletdom.NewFull(
			strings.TrimSpace(d.WalletAddress),
			d.Tokens,
			d.LastUpdatedAt.UTC(),
			walletdom.WalletStatus(strings.TrimSpace(d.Status)),
		)
		if err != nil {
			return walletdom.Wallet{}, err
		}
		return w, nil
	}
	if !errors.Is(err, iterator.Done) {
		return walletdom.Wallet{}, err
	}

	// 2) ✅ 旧仕様: docId=walletAddress を読む（互換）
	snap, err := r.col().Doc(a).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return walletdom.Wallet{}, walletdom.ErrNotFound
		}
		return walletdom.Wallet{}, err
	}

	// 旧データは avatarId フィールドを持っている可能性があるが、ドメインからは削除したので無視する
	var raw struct {
		WalletAddress string    `firestore:"walletAddress"`
		Tokens        []string  `firestore:"tokens"`
		LastUpdatedAt time.Time `firestore:"lastUpdatedAt"`
		Status        string    `firestore:"status"`

		// legacy (ignore)
		AvatarID string `firestore:"avatarId"`
	}
	if err := snap.DataTo(&raw); err != nil {
		return walletdom.Wallet{}, err
	}

	if strings.TrimSpace(raw.WalletAddress) == "" {
		raw.WalletAddress = a
	}
	if strings.TrimSpace(raw.Status) == "" {
		raw.Status = string(walletdom.StatusActive)
	}
	if raw.LastUpdatedAt.IsZero() {
		raw.LastUpdatedAt = time.Now().UTC()
	}

	w, err := walletdom.NewFull(
		strings.TrimSpace(raw.WalletAddress),
		raw.Tokens,
		raw.LastUpdatedAt.UTC(),
		walletdom.WalletStatus(strings.TrimSpace(raw.Status)),
	)
	if err != nil {
		return walletdom.Wallet{}, err
	}
	return w, nil
}

// Save は Wallet を Firestore に保存（upsert）します。
// ✅ 新仕様: docId = avatarId / avatarId field is not stored
func (r *WalletRepositoryFS) Save(ctx context.Context, avatarID string, w walletdom.Wallet) error {
	if r == nil || r.Client == nil {
		return errors.New("wallet_repository_fs: firestore client is nil")
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return ErrInvalidAvatarID
	}

	addr := strings.TrimSpace(w.WalletAddress)
	if addr == "" {
		return walletdom.ErrInvalidWalletAddress
	}

	now := time.Now().UTC()
	last := w.LastUpdatedAt
	if last.IsZero() {
		last = now
	}

	st := w.Status
	if strings.TrimSpace(string(st)) == "" {
		st = walletdom.StatusActive
	}

	d := walletDoc{
		WalletAddress: addr,
		Tokens:        w.Tokens,
		LastUpdatedAt: last.UTC(),
		Status:        string(st),
	}

	_, err := r.col().Doc(aid).Set(ctx, d)
	return err
}
