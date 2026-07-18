// backend/internal/application/usecase/share_transfer_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	transferdom "narratives/internal/domain/transfer"
	walletdom "narratives/internal/domain/wallet"
)

// AvatarSecretProvider provides a signing capability for an avatar wallet owner.
type AvatarSecretProvider interface {
	GetAvatarSigner(ctx context.Context, avatarID string) (any, error)
}

// AvatarWalletItemTransferUpdater updates sender / receiver wallet token caches.
type AvatarWalletItemTransferUpdater interface {
	RemoveMintFromAvatarWalletItems(ctx context.Context, avatarID string, mintAddress string, now time.Time) error
	AddMintToAvatarWalletItems(ctx context.Context, avatarID string, mintAddress string, now time.Time) error
}

// AvatarWalletSyncer fully syncs wallet tokens from on-chain after transfer.
type AvatarWalletSyncer interface {
	SyncWalletTokens(ctx context.Context, avatarID string) (walletdom.Wallet, error)
}

type ShareTransferUsecase struct {
	tokenRepo    TokenResolver
	tokenUpdate  TokenOwnerUpdater
	walletUpdate AvatarWalletItemTransferUpdater
	walletSync   AvatarWalletSyncer
	transferRepo TransferRepo

	avatarWallet AvatarWalletResolver
	secrets      AvatarSecretProvider
	executor     TokenTransferExecutor

	resolveWarmer PostTransferResolveWarmer
	now           func() time.Time
}

func NewShareTransferUsecase(
	tokenRepo TokenResolver,
	tokenUpdate TokenOwnerUpdater,
	walletUpdate AvatarWalletItemTransferUpdater,
	walletSync AvatarWalletSyncer,
	transferRepo TransferRepo,
	avatarWallet AvatarWalletResolver,
	secrets AvatarSecretProvider,
	executor TokenTransferExecutor,
) *ShareTransferUsecase {
	return &ShareTransferUsecase{
		tokenRepo:     tokenRepo,
		tokenUpdate:   tokenUpdate,
		walletUpdate:  walletUpdate,
		walletSync:    walletSync,
		transferRepo:  transferRepo,
		avatarWallet:  avatarWallet,
		secrets:       secrets,
		executor:      executor,
		resolveWarmer: nil,
		now:           time.Now,
	}
}

func (u *ShareTransferUsecase) WithPostTransferResolveWarmer(w PostTransferResolveWarmer) *ShareTransferUsecase {
	if u != nil {
		u.resolveWarmer = w
	}
	return u
}

var (
	ErrShareTransferNotConfigured      = errors.New("share_transfer_uc: not configured")
	ErrShareTransferFromAvatarEmpty    = errors.New("share_transfer_uc: fromAvatarId is empty")
	ErrShareTransferToAvatarEmpty      = errors.New("share_transfer_uc: toAvatarId is empty")
	ErrShareTransferProductIDEmpty     = errors.New("share_transfer_uc: productId is empty")
	ErrShareTransferSameAvatar         = errors.New("share_transfer_uc: fromAvatarId and toAvatarId must be different")
	ErrShareTransferMintEmpty          = errors.New("share_transfer_uc: mintAddress is empty")
	ErrShareTransferFromWalletEmpty    = errors.New("share_transfer_uc: from avatar walletAddress is empty")
	ErrShareTransferToWalletEmpty      = errors.New("share_transfer_uc: to avatar walletAddress is empty")
	ErrShareTransferOwnerMismatch      = errors.New("share_transfer_uc: token current owner mismatch")
	ErrShareTransferResolveAfterFailed = errors.New("share_transfer_uc: post-transfer resolve failed")
	ErrShareTransferWalletSyncFailed   = errors.New("share_transfer_uc: wallet sync failed")
)

type ShareTransferInput struct {
	FromAvatarID string
	ToAvatarID   string
	ProductID    string
}

type ShareTransferResult struct {
	ProductID        string
	MintAddress      string
	TokenBlueprintID string

	FromAvatarID string
	ToAvatarID   string

	FromWallet  string
	ToWallet    string
	TxSignature string
}

// ShareToAvatar transfers the token currently owned by fromAvatar to toAvatar.
//
// Flow:
//  1. resolve tokens/{productId} to get mintAddress / tokenBlueprintId / current owner(toAddress)
//  2. resolve sender / receiver avatar wallets
//  3. create transfer(PENDING)
//  4. validate token.toAddress == sender wallet
//  5. get sender avatar signer
//  6. execute avatar -> avatar transfer
//     - fee payer is handled by TokenTransferExecutor implementation
//     - current solana executor uses mint authority(master wallet) as fee payer
//  7. transfer(SUCCEEDED)
//  8. update tokens/{productId}.toAddress = receiver wallet
//  9. remove mint from sender wallet cache
//
// 10. add mint to receiver wallet cache
// 11. sync sender wallet from on-chain
// 12. sync receiver wallet from on-chain
// 13. resolve warmup for receiver
// 14. on failure, transfer(FAILED) update
func (u *ShareTransferUsecase) ShareToAvatar(ctx context.Context, in ShareTransferInput) (res ShareTransferResult, retErr error) {
	if u == nil ||
		u.tokenRepo == nil ||
		u.tokenUpdate == nil ||
		u.walletUpdate == nil ||
		u.walletSync == nil ||
		u.transferRepo == nil ||
		u.avatarWallet == nil ||
		u.secrets == nil ||
		u.executor == nil {
		return ShareTransferResult{}, ErrShareTransferNotConfigured
	}

	fromAvatarID := in.FromAvatarID
	toAvatarID := in.ToAvatarID
	productID := in.ProductID

	if fromAvatarID == "" {
		return ShareTransferResult{}, ErrShareTransferFromAvatarEmpty
	}
	if toAvatarID == "" {
		return ShareTransferResult{}, ErrShareTransferToAvatarEmpty
	}
	if productID == "" {
		return ShareTransferResult{}, ErrShareTransferProductIDEmpty
	}
	if fromAvatarID == toAvatarID {
		return ShareTransferResult{}, ErrShareTransferSameAvatar
	}

	now := u.now().UTC()

	tok, err := u.tokenRepo.ResolveTokenByProductID(ctx, productID)
	if err != nil {
		return ShareTransferResult{}, fmt.Errorf("share_transfer_uc: resolve token failed productId=%s: %w", productID, err)
	}

	mint := tok.MintAddress
	tokenTBID := tok.TokenBlueprintID
	currentOwner := tok.ToAddress
	brandID := tok.BrandID

	if mint == "" {
		return ShareTransferResult{}, ErrShareTransferMintEmpty
	}

	fromWallet, err := u.avatarWallet.ResolveAvatarWalletAddress(ctx, fromAvatarID)
	if err != nil {
		return ShareTransferResult{}, fmt.Errorf(
			"share_transfer_uc: resolve sender avatar wallet failed avatarId=%s: %w",
			fromAvatarID, err,
		)
	}
	if fromWallet == "" {
		return ShareTransferResult{}, ErrShareTransferFromWalletEmpty
	}

	toWallet, err := u.avatarWallet.ResolveAvatarWalletAddress(ctx, toAvatarID)
	if err != nil {
		return ShareTransferResult{}, fmt.Errorf(
			"share_transfer_uc: resolve receiver avatar wallet failed avatarId=%s: %w",
			toAvatarID, err,
		)
	}
	if toWallet == "" {
		return ShareTransferResult{}, ErrShareTransferToWalletEmpty
	}

	attempt, err := u.transferRepo.NextAttempt(ctx, productID)
	if err != nil {
		return ShareTransferResult{}, fmt.Errorf("share_transfer_uc: next attempt failed productId=%s: %w", productID, err)
	}

	shareRef := buildShareTransferRef(fromAvatarID, toAvatarID, productID)

	tr, err := transferdom.NewPending(
		attempt,
		productID,
		shareRef,
		toAvatarID,
		toWallet,
		mint,
		now,
	)
	if err != nil {
		return ShareTransferResult{}, fmt.Errorf(
			"share_transfer_uc: build transfer entity failed productId=%s attempt=%d: %w",
			productID, attempt, err,
		)
	}

	if err := u.transferRepo.Create(ctx, tr); err != nil {
		return ShareTransferResult{}, fmt.Errorf(
			"share_transfer_uc: create transfer failed productId=%s attempt=%d: %w",
			productID, attempt, err,
		)
	}

	transferCreated := true
	transferAttempt := attempt

	defer func() {
		if retErr != nil && transferCreated {
			et := transferdom.ErrorTypeUnknown
			msg := retErr.Error()
			st := transferdom.StatusFailed
			p := transferdom.TransferPatch{
				Status:    &st,
				ErrorType: &et,
				ErrorMsg:  &msg,
			}
			_ = u.transferRepo.Update(context.Background(), productID, transferAttempt, p)
		}
	}()

	markFailed := func(et transferdom.ErrorType, msg string, txSig *string) {
		if !transferCreated {
			return
		}
		st := transferdom.StatusFailed
		m := msg
		p := transferdom.TransferPatch{
			Status:    &st,
			ErrorType: &et,
			ErrorMsg:  &m,
		}
		if txSig != nil {
			s := *txSig
			p.TxSignature = &s
		}
		_ = u.transferRepo.Update(context.Background(), productID, transferAttempt, p)
	}

	markSucceeded := func(txSig string) {
		if !transferCreated {
			return
		}
		st := transferdom.StatusSucceeded
		s := txSig
		p := transferdom.TransferPatch{
			Status:      &st,
			TxSignature: &s,
		}
		_ = u.transferRepo.Update(context.Background(), productID, transferAttempt, p)
	}

	if currentOwner != "" && currentOwner != fromWallet {
		msg := fmt.Sprintf(
			"productId=%s tokenOwner=%s expectedSenderWallet=%s",
			productID, currentOwner, fromWallet,
		)
		markFailed(transferdom.ErrorTypeMismatch, msg, nil)
		return ShareTransferResult{}, fmt.Errorf("%w: %s", ErrShareTransferOwnerMismatch, msg)
	}

	fromSigner, err := u.secrets.GetAvatarSigner(ctx, fromAvatarID)
	if err != nil {
		msg := fmt.Sprintf("get sender avatar signer failed avatarId=%s wallet=%s: %v", fromAvatarID, fromWallet, err)
		markFailed(transferdom.ErrorTypeSecretInvalid, msg, nil)
		return ShareTransferResult{}, fmt.Errorf(
			"share_transfer_uc: get sender avatar signer failed avatarId=%s wallet=%s: %w",
			fromAvatarID, fromWallet, err,
		)
	}

	var toSigner any = nil

	execOut, err := u.executor.ExecuteTransfer(ctx, ExecuteTransferInput{
		ProductID:        productID,
		AvatarID:         toAvatarID,
		BrandID:          brandID,
		ModelID:          "",
		TokenBlueprintID: tokenTBID,

		MintAddress: mint,
		Amount:      1,

		FromWalletAddress: fromWallet,
		ToWalletAddress:   toWallet,

		FromSigner: fromSigner,
		ToSigner:   toSigner,
	})
	if err != nil {
		msg := fmt.Sprintf(
			"execute share transfer failed fromAvatarId=%s toAvatarId=%s mint=%s: %v",
			fromAvatarID, toAvatarID, mint, err,
		)
		markFailed(transferdom.ErrorTypeTransferFailed, msg, nil)
		return ShareTransferResult{}, fmt.Errorf(
			"share_transfer_uc: execute share transfer failed fromAvatarId=%s toAvatarId=%s mint=%s: %w",
			fromAvatarID, toAvatarID, mint, err,
		)
	}

	tx := execOut.TxSignature

	markSucceeded(tx)

	if err := u.tokenUpdate.UpdateToAddressByProductID(ctx, productID, toWallet, now, tx); err != nil {
		msg := fmt.Sprintf("update token owner failed productId=%s to=%s tx=%s: %v", productID, toWallet, tx, err)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return ShareTransferResult{}, fmt.Errorf(
			"share_transfer_uc: update token owner failed productId=%s to=%s tx=%s: %w",
			productID, toWallet, tx, err,
		)
	}

	if err := u.walletUpdate.RemoveMintFromAvatarWalletItems(ctx, fromAvatarID, mint, now); err != nil {
		msg := fmt.Sprintf("remove sender wallet item failed avatarId=%s mint=%s tx=%s: %v", fromAvatarID, mint, tx, err)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return ShareTransferResult{}, fmt.Errorf("share_transfer_uc: %s", msg)
	}

	if err := u.walletUpdate.AddMintToAvatarWalletItems(ctx, toAvatarID, mint, now); err != nil {
		msg := fmt.Sprintf("add receiver wallet item failed avatarId=%s mint=%s tx=%s: %v", toAvatarID, mint, tx, err)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return ShareTransferResult{}, fmt.Errorf("share_transfer_uc: %s", msg)
	}

	if _, err := u.walletSync.SyncWalletTokens(ctx, fromAvatarID); err != nil {
		msg := fmt.Sprintf("sync sender wallet failed avatarId=%s wallet=%s mint=%s tx=%s: %v", fromAvatarID, fromWallet, mint, tx, err)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return ShareTransferResult{}, fmt.Errorf("%w: %s", ErrShareTransferWalletSyncFailed, msg)
	}

	if _, err := u.walletSync.SyncWalletTokens(ctx, toAvatarID); err != nil {
		msg := fmt.Sprintf("sync receiver wallet failed avatarId=%s wallet=%s mint=%s tx=%s: %v", toAvatarID, toWallet, mint, tx, err)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return ShareTransferResult{}, fmt.Errorf("%w: %s", ErrShareTransferWalletSyncFailed, msg)
	}

	if u.resolveWarmer != nil {
		if err := u.resolveWarmer.ResolveAfterTransfer(ctx, toAvatarID, mint); err != nil {
			msg := fmt.Sprintf("post-transfer resolve failed avatarId=%s mint=%s tx=%s: %v", toAvatarID, mint, tx, err)
			markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
			return ShareTransferResult{}, fmt.Errorf("%w: %s", ErrShareTransferResolveAfterFailed, msg)
		}
	}

	return ShareTransferResult{
		ProductID:        productID,
		MintAddress:      mint,
		TokenBlueprintID: tokenTBID,
		FromAvatarID:     fromAvatarID,
		ToAvatarID:       toAvatarID,
		FromWallet:       fromWallet,
		ToWallet:         toWallet,
		TxSignature:      tx,
	}, nil
}

func buildShareTransferRef(fromAvatarID string, toAvatarID string, productID string) string {
	return fmt.Sprintf("share:%s:%s:%s", fromAvatarID, toAvatarID, productID)
}
