// backend/internal/application/usecase/wallet_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	walletdom "narratives/internal/domain/wallet"
)

// ✅ usecase が必要とするIFをここで定義する（domain の Repository に依存しない）
type WalletRepository interface {
	// docId=avatarId
	GetByAvatarID(ctx context.Context, avatarID string) (walletdom.Wallet, error)
	Save(ctx context.Context, avatarID string, w walletdom.Wallet) error
}

type OnchainWalletReader interface {
	ListOwnedTokenMints(ctx context.Context, walletAddress string) ([]string, error)
}

// WalletUsecase は Wallet 同期ユースケース
// - 「同期API（案A）」を前提に、同期対象 wallet は既に存在することを要求する
// - 旧式互換（avatar から addr を拾って create する等）は行わない
type WalletUsecase struct {
	WalletRepo    WalletRepository
	OnchainReader OnchainWalletReader // 必須（同期APIとして使うなら）
}

// コンストラクタ（DI コンテナの呼び出しに合わせて 1 引数）
// OnchainReader は WithOnchainReader で差し込む（nil のまま同期を呼ぶとエラー）
func NewWalletUsecase(walletRepo WalletRepository) *WalletUsecase {
	return &WalletUsecase{
		WalletRepo:    walletRepo,
		OnchainReader: nil,
	}
}

// 任意: OnchainReader を後から差し込むためのセッター
func (uc *WalletUsecase) WithOnchainReader(r OnchainWalletReader) *WalletUsecase {
	if uc != nil {
		uc.OnchainReader = r
	}
	return uc
}

var (
	ErrWalletUsecaseNotConfigured     = errors.New("wallet usecase: not configured")
	ErrWalletSyncAvatarIDEmpty        = errors.New("wallet usecase: avatarID is empty")
	ErrWalletSyncOnchainNotConfigured = errors.New("wallet usecase: onchain reader not configured")
	ErrWalletSyncWalletAddressEmpty   = errors.New("wallet usecase: walletAddress is empty")
)

// SyncWalletTokens:
// - docId=avatarId の wallet を取得
// - wallet.walletAddress を owner として devnet RPC から token mints を取得
// - wallet.tokens を on-chain に合わせて置き換え、保存
//
// NOTE:
// - wallet が存在しない場合は ErrNotFound を返す（この usecase は create しない）
// - addr を handler から受け取らない（旧式互換排除）
// - replace は冪等
func (uc *WalletUsecase) SyncWalletTokens(ctx context.Context, avatarID string) (walletdom.Wallet, error) {
	if uc == nil || uc.WalletRepo == nil {
		return walletdom.Wallet{}, ErrWalletUsecaseNotConfigured
	}
	if uc.OnchainReader == nil {
		return walletdom.Wallet{}, ErrWalletSyncOnchainNotConfigured
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return walletdom.Wallet{}, ErrWalletSyncAvatarIDEmpty
	}

	// 1) docId=avatarId で wallet を取得（存在が前提）
	w, err := uc.WalletRepo.GetByAvatarID(ctx, aid)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	addr := strings.TrimSpace(w.WalletAddress)
	if addr == "" {
		return walletdom.Wallet{}, ErrWalletSyncWalletAddressEmpty
	}

	// 2) on-chain から mint 一覧を取得
	mints, err := uc.OnchainReader.ListOwnedTokenMints(ctx, addr)
	if err != nil {
		return walletdom.Wallet{}, err
	}

	// 3) 置換して保存
	now := time.Now().UTC()
	if err := w.ReplaceTokens(mints, now); err != nil {
		return walletdom.Wallet{}, err
	}

	if err := uc.WalletRepo.Save(ctx, aid, w); err != nil {
		return walletdom.Wallet{}, err
	}

	return w, nil
}
