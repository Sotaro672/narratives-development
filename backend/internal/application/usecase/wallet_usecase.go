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
	GetByAvatarID(ctx context.Context, avatarID string) (walletdom.Wallet, error)
	GetByAddress(ctx context.Context, addr string) (walletdom.Wallet, error) // 互換 or 逆引き用途
	Save(ctx context.Context, avatarID string, w walletdom.Wallet) error
}

type OnchainWalletReader interface {
	ListOwnedTokenMints(ctx context.Context, walletAddress string) ([]string, error)
}

// WalletUsecase は Wallet 同期ユースケース
type WalletUsecase struct {
	WalletRepo    WalletRepository
	OnchainReader OnchainWalletReader // DI で未設定の場合は nil
}

// コンストラクタ（DI コンテナの呼び出しに合わせて 1 引数）
func NewWalletUsecase(walletRepo WalletRepository) *WalletUsecase {
	return &WalletUsecase{
		WalletRepo:    walletRepo,
		OnchainReader: nil,
	}
}

// 任意: OnchainReader を後から差し込むためのセッター
func (uc *WalletUsecase) WithOnchainReader(r OnchainWalletReader) *WalletUsecase {
	uc.OnchainReader = r
	return uc
}

// ✅ docId=avatarId 前提に合わせる
// - avatarID は必須
// - addr は「新規作成時 or 不整合修正時」に使う（空なら既存Walletから採用）
func (uc *WalletUsecase) SyncWalletTokens(ctx context.Context, avatarID string, addr string) (walletdom.Wallet, error) {
	if uc == nil || uc.WalletRepo == nil {
		return walletdom.Wallet{}, errors.New("wallet usecase: repo not configured")
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return walletdom.Wallet{}, errors.New("wallet usecase: avatarID is empty")
	}

	// 1) まず avatarID で Wallet を取る（docId=avatarId）
	w, err := uc.WalletRepo.GetByAvatarID(ctx, aid)
	if err != nil {
		// NotFound なら作る（addr 必須）
		if !errors.Is(err, walletdom.ErrNotFound) {
			return walletdom.Wallet{}, err
		}

		a := strings.TrimSpace(addr)
		if a == "" {
			return walletdom.Wallet{}, errors.New("wallet usecase: walletAddress is required for create")
		}

		now := time.Now().UTC()
		w, err = walletdom.NewFull(a, nil, now, walletdom.StatusActive)
		if err != nil {
			return walletdom.Wallet{}, err
		}
		// 保存（docId=avatarId）
		if err := uc.WalletRepo.Save(ctx, aid, w); err != nil {
			return walletdom.Wallet{}, err
		}
	} else {
		// 既存がある場合、addr が空なら既存を採用
		if strings.TrimSpace(addr) == "" {
			addr = w.WalletAddress
		}
	}

	// 2) OnchainReader があれば mint 一覧で同期
	if uc.OnchainReader != nil {
		a := strings.TrimSpace(addr)
		if a == "" {
			// 既存Walletにも addr が無いのは不正
			return walletdom.Wallet{}, errors.New("wallet usecase: walletAddress is empty")
		}

		mints, err := uc.OnchainReader.ListOwnedTokenMints(ctx, a)
		if err != nil {
			return walletdom.Wallet{}, err
		}
		if err := w.ReplaceTokens(mints, time.Now().UTC()); err != nil {
			return walletdom.Wallet{}, err
		}
	}

	// 3) 永続化（更新）
	if err := uc.WalletRepo.Save(ctx, aid, w); err != nil {
		return walletdom.Wallet{}, err
	}

	return w, nil
}
