// backend/internal/domain/wallet/service.go
package wallet

import (
	"context"
	"errors"
	"time"
)

// Repository is the domain-side port for wallet persistence.
//
// Adapter examples:
//   - Firestore implementation
//   - PostgreSQL implementation
type Repository interface {
	// GetByAddress returns the wallet for the given walletAddress.
	// It MUST return ErrNotFound when the wallet does not exist.
	GetByAddress(ctx context.Context, addr string) (Wallet, error)

	// Save creates or updates the wallet.
	Save(ctx context.Context, w Wallet) error
}

// AddressGenerator is a port for generating a new Solana wallet address.
//
// 実際の秘密鍵生成・管理はクライアント（Flutter / Web）側で行い、
// バックエンドは walletAddress（公開鍵）だけを扱う構成もあり得るが、
// バックエンドで生成したい場合はこのインターフェースを実装してアダプタ側で
// Solana SDK を呼び出すことを想定している。
type AddressGenerator interface {
	Generate(ctx context.Context) (string, error)
}

// NowFunc is injectable clock function for testability.
type NowFunc func() time.Time

var (
	// ErrWalletAlreadyExists は同一 walletAddress のウォレットが既に存在する場合に返される。
	ErrWalletAlreadyExists = errors.New("wallet: already exists")

	// ErrAddressGeneratorMissing はアドレス生成が必要なケースで AddressGenerator が未設定の場合に返される。
	ErrAddressGeneratorMissing = errors.New("wallet: address generator is not configured")
)

// Service encapsulates wallet domain use cases.
type Service struct {
	repo    Repository
	addrGen AddressGenerator
	now     NowFunc
}

// NewService constructs a wallet Service.
func NewService(repo Repository, addrGen AddressGenerator, now NowFunc) *Service {
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &Service{
		repo:    repo,
		addrGen: addrGen,
		now:     now,
	}
}

// OpenWalletWithGeneratedAddress creates a new wallet using AddressGenerator.
//
// - addrGen が nil の場合は ErrAddressGeneratorMissing
// - 生成したアドレスのウォレットが既に存在する場合は ErrWalletAlreadyExists
// - 成功時は StatusActive / tokens: []string{} / 時刻は now() を用いて Wallet を作成する
func (s *Service) OpenWalletWithGeneratedAddress(ctx context.Context) (Wallet, error) {
	if s.addrGen == nil {
		return Wallet{}, ErrAddressGeneratorMissing
	}

	addr, err := s.addrGen.Generate(ctx)
	if err != nil {
		return Wallet{}, err
	}

	return s.openWalletInternal(ctx, addr)
}

// OpenWalletWithAddress registers a wallet using an already-generated address.
//
// クライアント側（Flutter 等）で Solana ウォレットを作成し、
// その公共鍵（walletAddress）だけをバックエンドに登録する場合はこのメソッドを利用する。
//
// - addr が不正な形式の場合は ErrInvalidWalletAddress
// - 同一アドレスのウォレットが既に存在する場合は ErrWalletAlreadyExists
func (s *Service) OpenWalletWithAddress(ctx context.Context, addr string) (Wallet, error) {
	return s.openWalletInternal(ctx, addr)
}

// openWalletInternal is shared logic for opening a wallet.
func (s *Service) openWalletInternal(ctx context.Context, addr string) (Wallet, error) {
	// 1. アドレス形式チェック（早期バリデーション）
	if !isValidWallet(addr) {
		return Wallet{}, ErrInvalidWalletAddress
	}

	// 2. 既存チェック
	if _, err := s.repo.GetByAddress(ctx, addr); err == nil {
		// 既に存在する
		return Wallet{}, ErrWalletAlreadyExists
	} else if !errors.Is(err, ErrNotFound) {
		// 予期しないエラー
		return Wallet{}, err
	}

	// 3. 新規 Wallet を作成（トークンは空、ステータスは active）
	now := s.now().UTC()
	w, err := NewFull(addr, nil, now, now, now, StatusActive)
	if err != nil {
		return Wallet{}, err
	}

	// 4. 永続化
	if err := s.repo.Save(ctx, w); err != nil {
		return Wallet{}, err
	}

	return w, nil
}

// Activate sets wallet status to active.
func (s *Service) Activate(ctx context.Context, addr string) (Wallet, error) {
	return s.setStatus(ctx, addr, StatusActive)
}

// Deactivate sets wallet status to inactive.
func (s *Service) Deactivate(ctx context.Context, addr string) (Wallet, error) {
	return s.setStatus(ctx, addr, StatusInactive)
}

// setStatus is shared logic for status transitions.
func (s *Service) setStatus(ctx context.Context, addr string, status WalletStatus) (Wallet, error) {
	w, err := s.repo.GetByAddress(ctx, addr)
	if err != nil {
		return Wallet{}, err
	}

	if err := w.SetStatus(status, s.now()); err != nil {
		return Wallet{}, err
	}

	if err := s.repo.Save(ctx, w); err != nil {
		return Wallet{}, err
	}

	return w, nil
}
