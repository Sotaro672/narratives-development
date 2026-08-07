// backend/internal/application/usecase/share_transfer_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	walletdom "narratives/internal/domain/wallet"
)

// AvatarSecretProvider provides a signing capability for an avatar wallet owner.
type AvatarSecretProvider interface {
	GetAvatarSigner(
		ctx context.Context,
		avatarID string,
	) (any, error)
}

// AvatarWalletItemTransferUpdater updates sender / receiver wallet token caches.
type AvatarWalletItemTransferUpdater interface {
	RemoveMintFromAvatarWalletItems(
		ctx context.Context,
		avatarID string,
		mintAddress string,
		now time.Time,
	) error

	AddMintToAvatarWalletItems(
		ctx context.Context,
		avatarID string,
		mintAddress string,
		now time.Time,
	) error
}

// AvatarWalletSyncer fully syncs wallet tokens from on-chain after transfer.
type AvatarWalletSyncer interface {
	SyncWalletTokens(
		ctx context.Context,
		avatarID string,
	) (walletdom.Wallet, error)
}

type ShareTransferUsecase struct {
	tokenRepo TokenResolver

	avatarWallet AvatarWalletResolver
	secrets      AvatarSecretProvider

	executionUC *TokenTransferExecutionUsecase
}

func NewShareTransferUsecase(
	tokenRepo TokenResolver,
	avatarWallet AvatarWalletResolver,
	secrets AvatarSecretProvider,
	executionUC *TokenTransferExecutionUsecase,
) *ShareTransferUsecase {
	return &ShareTransferUsecase{
		tokenRepo: tokenRepo,

		avatarWallet: avatarWallet,
		secrets:      secrets,

		executionUC: executionUC,
	}
}

var (
	ErrShareTransferNotConfigured = errors.New(
		"share_transfer_uc: not configured",
	)
	ErrShareTransferFromAvatarEmpty = errors.New(
		"share_transfer_uc: fromAvatarId is empty",
	)
	ErrShareTransferToAvatarEmpty = errors.New(
		"share_transfer_uc: toAvatarId is empty",
	)
	ErrShareTransferProductIDEmpty = errors.New(
		"share_transfer_uc: productId is empty",
	)
	ErrShareTransferSameAvatar = errors.New(
		"share_transfer_uc: fromAvatarId and toAvatarId must be different",
	)
	ErrShareTransferMintEmpty = errors.New(
		"share_transfer_uc: mintAddress is empty",
	)
	ErrShareTransferFromWalletEmpty = errors.New(
		"share_transfer_uc: from avatar walletAddress is empty",
	)
	ErrShareTransferToWalletEmpty = errors.New(
		"share_transfer_uc: to avatar walletAddress is empty",
	)
	ErrShareTransferOwnerMismatch = errors.New(
		"share_transfer_uc: token current owner mismatch",
	)
	ErrShareTransferResolveAfterFailed = errors.New(
		"share_transfer_uc: post-transfer resolve failed",
	)
	ErrShareTransferWalletSyncFailed = errors.New(
		"share_transfer_uc: wallet sync failed",
	)
	ErrShareTransferAttemptNotCreated = errors.New(
		"share_transfer_uc: transfer attempt was not created",
	)
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
// ShareTransferUsecase is responsible for:
//  1. Validate sender / receiver avatar IDs.
//  2. Resolve tokens/{productId}.
//  3. Resolve sender / receiver avatar wallets.
//  4. Resolve the sender avatar signer.
//  5. Build the share transfer reference.
//  6. Delegate common transfer execution to TokenTransferExecutionUsecase.
//
// TokenTransferExecutionUsecase is responsible for:
//  1. Create transfer(PENDING).
//  2. Validate token.toAddress == sender wallet.
//  3. Execute the on-chain transfer.
//  4. Update tokens/{productId}.toAddress.
//  5. Remove the mint from the sender wallet cache.
//  6. Add the mint to the receiver wallet cache.
//  7. Sync sender / receiver wallets from on-chain.
//  8. Warm the receiver token resolver.
//  9. Mark transfer(SUCCEEDED).
//
// 10. Mark transfer(FAILED) when execution or post-processing fails.
func (u *ShareTransferUsecase) ShareToAvatar(
	ctx context.Context,
	in ShareTransferInput,
) (ShareTransferResult, error) {
	if u == nil ||
		u.tokenRepo == nil ||
		u.avatarWallet == nil ||
		u.secrets == nil ||
		u.executionUC == nil {
		return ShareTransferResult{},
			ErrShareTransferNotConfigured
	}

	fromAvatarID := in.FromAvatarID
	toAvatarID := in.ToAvatarID
	productID := in.ProductID

	if fromAvatarID == "" {
		return ShareTransferResult{},
			ErrShareTransferFromAvatarEmpty
	}

	if toAvatarID == "" {
		return ShareTransferResult{},
			ErrShareTransferToAvatarEmpty
	}

	if productID == "" {
		return ShareTransferResult{},
			ErrShareTransferProductIDEmpty
	}

	if fromAvatarID == toAvatarID {
		return ShareTransferResult{},
			ErrShareTransferSameAvatar
	}

	token, err := u.tokenRepo.ResolveTokenByProductID(
		ctx,
		productID,
	)
	if err != nil {
		return ShareTransferResult{},
			fmt.Errorf(
				"share_transfer_uc: resolve token failed productId=%s: %w",
				productID,
				err,
			)
	}

	mintAddress := token.MintAddress
	tokenBlueprintID := token.TokenBlueprintID
	currentOwner := token.ToAddress
	brandID := token.BrandID

	if mintAddress == "" {
		return ShareTransferResult{},
			ErrShareTransferMintEmpty
	}

	fromWallet, err :=
		u.avatarWallet.ResolveAvatarWalletAddress(
			ctx,
			fromAvatarID,
		)
	if err != nil {
		return ShareTransferResult{},
			fmt.Errorf(
				"share_transfer_uc: resolve sender avatar wallet failed avatarId=%s: %w",
				fromAvatarID,
				err,
			)
	}

	if fromWallet == "" {
		return ShareTransferResult{},
			ErrShareTransferFromWalletEmpty
	}

	toWallet, err :=
		u.avatarWallet.ResolveAvatarWalletAddress(
			ctx,
			toAvatarID,
		)
	if err != nil {
		return ShareTransferResult{},
			fmt.Errorf(
				"share_transfer_uc: resolve receiver avatar wallet failed avatarId=%s: %w",
				toAvatarID,
				err,
			)
	}

	if toWallet == "" {
		return ShareTransferResult{},
			ErrShareTransferToWalletEmpty
	}

	fromSigner, err := u.secrets.GetAvatarSigner(
		ctx,
		fromAvatarID,
	)
	if err != nil {
		return ShareTransferResult{},
			fmt.Errorf(
				"share_transfer_uc: get sender avatar signer failed avatarId=%s wallet=%s: %w",
				fromAvatarID,
				fromWallet,
				err,
			)
	}

	shareRef := buildShareTransferRef(
		fromAvatarID,
		toAvatarID,
		productID,
	)

	executionResult, err := u.executionUC.Execute(
		ctx,
		TokenTransferExecutionInput{
			ProductID: productID,

			AttemptReference: shareRef,

			FromAvatarID: fromAvatarID,
			ToAvatarID:   toAvatarID,

			BrandID:          brandID,
			ModelID:          "",
			TokenBlueprintID: tokenBlueprintID,

			MintAddress:  mintAddress,
			CurrentOwner: currentOwner,

			FromWallet: fromWallet,
			ToWallet:   toWallet,

			FromSigner: fromSigner,
			ToSigner:   nil,

			Amount: 1,

			RemoveFromSenderWallet: true,
			SyncSenderWallet:       true,
			SyncReceiverWallet:     true,

			AfterOnChain:  nil,
			BeforeSuccess: nil,
		},
	)
	if err != nil {
		return ShareTransferResult{},
			mapShareTransferExecutionError(err)
	}

	return ShareTransferResult{
		ProductID:        productID,
		MintAddress:      mintAddress,
		TokenBlueprintID: tokenBlueprintID,

		FromAvatarID: fromAvatarID,
		ToAvatarID:   toAvatarID,

		FromWallet:  fromWallet,
		ToWallet:    toWallet,
		TxSignature: executionResult.TxSignature,
	}, nil
}

func mapShareTransferExecutionError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		ErrTokenTransferExecutionNotConfigured,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferNotConfigured,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionProductIDEmpty,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferProductIDEmpty,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionFromAvatarIDEmpty,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferFromAvatarEmpty,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionToAvatarIDEmpty,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferToAvatarEmpty,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionMintAddressEmpty,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferMintEmpty,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionFromWalletEmpty,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferFromWalletEmpty,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionToWalletEmpty,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferToWalletEmpty,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionOwnerMismatch,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferOwnerMismatch,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionAttemptNotCreated,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferAttemptNotCreated,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionWalletSyncNotConfigured,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferNotConfigured,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionWalletSyncFailed,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferWalletSyncFailed,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionResolveAfterFailed,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferResolveAfterFailed,
			err,
		)

	default:
		return err
	}
}

func buildShareTransferRef(
	fromAvatarID string,
	toAvatarID string,
	productID string,
) string {
	return fmt.Sprintf(
		"share:%s:%s:%s",
		fromAvatarID,
		toAvatarID,
		productID,
	)
}
