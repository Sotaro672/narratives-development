package usecase

import (
    "context"
    "strings"

    walletdom "narratives/internal/domain/wallet"
)

// WalletRepo defines the minimal persistence port needed by WalletUsecase.
type WalletRepo interface {
    GetByID(ctx context.Context, id string) (walletdom.Wallet, error)
    Exists(ctx context.Context, id string) (bool, error)
    Create(ctx context.Context, v walletdom.Wallet) (walletdom.Wallet, error)
    Save(ctx context.Context, v walletdom.Wallet) (walletdom.Wallet, error)
    Delete(ctx context.Context, id string) error
}

// WalletUsecase orchestrates wallet operations.
type WalletUsecase struct {
    repo WalletRepo
}

func NewWalletUsecase(repo WalletRepo) *WalletUsecase {
    return &WalletUsecase{repo: repo}
}

// Queries

func (u *WalletUsecase) GetByID(ctx context.Context, id string) (walletdom.Wallet, error) {
    return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *WalletUsecase) Exists(ctx context.Context, id string) (bool, error) {
    return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *WalletUsecase) Create(ctx context.Context, v walletdom.Wallet) (walletdom.Wallet, error) {
    return u.repo.Create(ctx, v)
}

func (u *WalletUsecase) Save(ctx context.Context, v walletdom.Wallet) (walletdom.Wallet, error) {
    return u.repo.Save(ctx, v)
}

func (u *WalletUsecase) Delete(ctx context.Context, id string) error {
    return u.repo.Delete(ctx, strings.TrimSpace(id))
}