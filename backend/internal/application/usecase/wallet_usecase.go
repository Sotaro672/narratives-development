// backend/internal/application/usecase/wallet_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	walletdom "narratives/internal/domain/wallet"
)

// ドメイン側に移譲したポートを usecase パッケージ内ではエイリアスとして利用する
type WalletRepository = walletdom.Repository
type OnchainWalletReader = walletdom.OnchainReader

// WalletUsecase は Wallet 同期ユースケース
type WalletUsecase struct {
	WalletRepo    WalletRepository
	OnchainReader OnchainWalletReader // DI で未設定の場合は nil
}

// コンストラクタ（DI コンテナの呼び出しに合わせて 1 引数）
// OnchainReader は後からセットするか、未設定なら On-chain 同期なしで動作します。
func NewWalletUsecase(
	walletRepo WalletRepository,
) *WalletUsecase {
	return &WalletUsecase{
		WalletRepo:    walletRepo,
		OnchainReader: nil,
	}
}

// 任意: OnchainReader を後から差し込むためのセッター（必要になったら DI 側で利用）
func (uc *WalletUsecase) WithOnchainReader(r OnchainWalletReader) *WalletUsecase {
	uc.OnchainReader = r
	return uc
}

// Solana 上と Wallet エンティティを同期するユースケース
// - OnchainReader が設定されていれば Solana から mint 一覧を取得して Wallet.Tokens を更新
// - OnchainReader が nil の場合は、DB の Wallet を取得（または新規作成）するだけ
func (uc *WalletUsecase) SyncWalletTokens(ctx context.Context, addr string) (walletdom.Wallet, error) {
	a := strings.TrimSpace(addr)
	if a == "" {
		return walletdom.Wallet{}, walletdom.ErrInvalidWalletAddress
	}

	var mints []string
	var err error

	// 1. On-chain の mint 一覧を取得（OnchainReader があれば）
	if uc.OnchainReader != nil {
		mints, err = uc.OnchainReader.ListOwnedTokenMints(ctx, a)
		if err != nil {
			return walletdom.Wallet{}, err
		}
	}

	// 2. DB 上の Wallet を取得（なければ新規作成）
	w, err := uc.WalletRepo.GetByAddress(ctx, a)
	if err != nil {
		// NotFound の扱い：Wallet レコードを新規作成（それ以外のエラーは返す）
		if !errors.Is(err, walletdom.ErrNotFound) {
			return walletdom.Wallet{}, err
		}

		now := time.Now().UTC()
		initialTokens := mints
		if uc.OnchainReader == nil {
			// OnchainReader 無しの場合はトークン情報は空で作成
			initialTokens = nil
		}

		// ✅ 新シグネチャ: NewFull(avatarId, walletAddress, tokens, lastUpdatedAt, status)
		// avatarId はここでは不明なため空で作る（後で紐付け更新する想定）
		w, err = walletdom.NewFull("", a, initialTokens, now, walletdom.StatusActive)
		if err != nil {
			return walletdom.Wallet{}, err
		}
	} else if uc.OnchainReader != nil {
		// 既存 Wallet があり、かつ OnchainReader が有効なら Tokens を同期
		if err := w.ReplaceTokens(mints, time.Now().UTC()); err != nil {
			return walletdom.Wallet{}, err
		}
		// （OnchainReader が nil の場合は既存 Tokens をそのまま保持）
	}

	// 3. 永続化（新規作成 or 更新）
	if err := uc.WalletRepo.Save(ctx, w); err != nil {
		return walletdom.Wallet{}, err
	}

	return w, nil
}
