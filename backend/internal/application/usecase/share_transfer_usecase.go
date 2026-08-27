// backend/internal/application/usecase/share_transfer_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	walletdom "narratives/internal/domain/wallet"
)

// AvatarWalletItemTransferUpdater updates sender / receiver wallet asset caches.
//
// NOTE:
// Method names follow the current assetId-based wallet repository contract.
// The identifier passed to these methods is a Bubblegum V2 assetId, not a
// legacy SPL mint address.
type AvatarWalletItemTransferUpdater interface {
	RemoveAssetIDFromAvatarWalletItems(
		ctx context.Context,
		avatarID string,
		assetID string,
		now time.Time,
	) error

	AddAssetIDToAvatarWalletItems(
		ctx context.Context,
		avatarID string,
		assetID string,
		now time.Time,
	) error
}

// AvatarWalletSyncer fully syncs wallet assets from on-chain after transfer.
type AvatarWalletSyncer interface {
	SyncWalletAssetIDs(
		ctx context.Context,
		avatarID string,
	) (walletdom.Wallet, error)
}

type ShareTransferUsecase struct {
	tokenRepo TokenResolver

	avatarWallet AvatarWalletResolver

	executionUC *TokenTransferExecutionUsecase
}

func NewShareTransferUsecase(
	tokenRepo TokenResolver,
	avatarWallet AvatarWalletResolver,
	executionUC *TokenTransferExecutionUsecase,
) *ShareTransferUsecase {
	return &ShareTransferUsecase{
		tokenRepo: tokenRepo,

		avatarWallet: avatarWallet,

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
	ErrShareTransferOperationIDEmpty = errors.New(
		"share_transfer_uc: operationId is empty",
	)
	ErrShareTransferSameAvatar = errors.New(
		"share_transfer_uc: fromAvatarId and toAvatarId must be different",
	)
	ErrShareTransferAssetIDEmpty = errors.New(
		"share_transfer_uc: assetId is empty",
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
	OperationID  string
}

type ShareTransferResult struct {
	ProductID        string
	AssetID          string
	TokenBlueprintID string

	FromAvatarID string
	ToAvatarID   string

	FromWallet  string
	ToWallet    string
	TxSignature string
}

// ShareToAvatar transfers the Bubblegum V2 cNFT currently owned by fromAvatar
// to toAvatar.
//
// ShareTransferUsecase is responsible for:
//  1. Validate sender / receiver avatar IDs.
//  2. Validate the stable operationId used for transfer idempotency.
//  3. Resolve tokens/{productId}.
//  4. Resolve sender / receiver avatar wallet addresses.
//  5. Build the share transfer reference.
//  6. Delegate common transfer execution to TokenTransferExecutionUsecase.
//
// TokenTransferExecutionUsecase is responsible for:
//  1. Create or reuse transfer(PENDING) by operationId.
//  2. Delegate the Bubblegum V2 transfer to TokenTransferExecutor.
//  3. Update cached tokens/{productId}.toAddress.
//  4. Remove the assetId from the sender wallet cache.
//  5. Add the assetId to the receiver wallet cache.
//  6. Sync sender / receiver wallets from on-chain.
//  7. Warm the receiver token resolver.
//  8. Mark transfer(SUCCEEDED).
//  9. Mark transfer(FAILED) when execution or post-processing fails.
//
// Current ownership verification and signer resolution are responsibilities of
// the Bubblegum service using DAS / on-chain state. The Go backend must not
// load or send avatar private keys.
func (u *ShareTransferUsecase) ShareToAvatar(
	ctx context.Context,
	in ShareTransferInput,
) (ShareTransferResult, error) {
	if u == nil ||
		u.tokenRepo == nil ||
		u.avatarWallet == nil ||
		u.executionUC == nil {
		return ShareTransferResult{},
			ErrShareTransferNotConfigured
	}

	fromAvatarID := in.FromAvatarID
	toAvatarID := in.ToAvatarID
	productID := in.ProductID
	operationID := in.OperationID

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

	if operationID == "" {
		return ShareTransferResult{},
			ErrShareTransferOperationIDEmpty
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

	assetID := token.AssetID
	tokenBlueprintID := token.TokenBlueprintID
	brandID := token.BrandID

	if assetID == "" {
		return ShareTransferResult{},
			ErrShareTransferAssetIDEmpty
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

	shareRef := buildShareTransferRef(
		fromAvatarID,
		toAvatarID,
		productID,
	)

	executionResult, err := u.executionUC.Execute(
		ctx,
		TokenTransferExecutionInput{
			ProductID:   productID,
			OperationID: operationID,

			AttemptReference: shareRef,

			FromAvatarID: fromAvatarID,
			FromBrandID:  "",
			ToAvatarID:   toAvatarID,

			BrandID:          brandID,
			ModelID:          "",
			TokenBlueprintID: tokenBlueprintID,

			AssetID: assetID,

			FromWallet: fromWallet,
			ToWallet:   toWallet,

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
		AssetID:          assetID,
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
		ErrTokenTransferExecutionOperationIDEmpty,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferOperationIDEmpty,
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
		ErrTokenTransferExecutionAssetIDEmpty,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrShareTransferAssetIDEmpty,
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
