// backend/internal/application/usecase/transfer_execution_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	transferdom "narratives/internal/domain/transfer"
)

// -----------------------------------------------------------------------------
// Usecase
// -----------------------------------------------------------------------------

// TokenTransferExecutionUsecase executes the common Bubblegum V2 cNFT transfer
// flow shared by TransferUsecase and ShareTransferUsecase.
//
// This use case is responsible only for the common transfer execution:
//
//  1. Create transfer attempt as PENDING.
//  2. Execute the on-chain Bubblegum V2 cNFT transfer.
//  3. Run caller-specific post-on-chain processing when configured.
//  4. Update the cached token owner.
//  5. Update sender / receiver wallet caches.
//  6. Synchronize wallets from on-chain when configured.
//  7. Warm the post-transfer token resolver when configured.
//  8. Run caller-specific finalization when configured.
//  9. Mark the transfer as SUCCEEDED.
//
// Current cNFT ownership verification and signer resolution are responsibilities
// of the Bubblegum service. The Go backend must not load or send private keys.
//
// Scan verification, order lookup / locking, transfer source resolution,
// order item state transition, and inventory-specific business rules remain
// responsibilities of the caller.
type TokenTransferExecutionUsecase struct {
	tokenUpdate  TokenOwnerUpdater
	walletUpdate AvatarWalletItemTransferUpdater
	walletSync   AvatarWalletSyncer
	transferRepo transferdom.RepositoryPort
	executor     TokenTransferExecutor

	resolveWarmer PostTransferResolveWarmer

	now func() time.Time
}

func NewTokenTransferExecutionUsecase(
	tokenUpdate TokenOwnerUpdater,
	walletUpdate AvatarWalletItemTransferUpdater,
	walletSync AvatarWalletSyncer,
	transferRepo transferdom.RepositoryPort,
	executor TokenTransferExecutor,
	resolveWarmer PostTransferResolveWarmer,
) *TokenTransferExecutionUsecase {
	return &TokenTransferExecutionUsecase{
		tokenUpdate:   tokenUpdate,
		walletUpdate:  walletUpdate,
		walletSync:    walletSync,
		transferRepo:  transferRepo,
		executor:      executor,
		resolveWarmer: resolveWarmer,
		now:           time.Now,
	}
}

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

var (
	ErrTokenTransferExecutionNotConfigured = errors.New(
		"token_transfer_execution_uc: not configured",
	)

	ErrTokenTransferExecutionProductIDEmpty = errors.New(
		"token_transfer_execution_uc: productId is empty",
	)

	ErrTokenTransferExecutionOperationIDEmpty = errors.New(
		"token_transfer_execution_uc: operationId is empty",
	)

	ErrTokenTransferExecutionAttemptReferenceEmpty = errors.New(
		"token_transfer_execution_uc: attempt reference is empty",
	)

	ErrTokenTransferExecutionFromAvatarIDEmpty = errors.New(
		"token_transfer_execution_uc: fromAvatarId is empty",
	)

	ErrTokenTransferExecutionToAvatarIDEmpty = errors.New(
		"token_transfer_execution_uc: toAvatarId is empty",
	)

	ErrTokenTransferExecutionAssetIDEmpty = errors.New(
		"token_transfer_execution_uc: assetId is empty",
	)

	ErrTokenTransferExecutionFromWalletEmpty = errors.New(
		"token_transfer_execution_uc: from walletAddress is empty",
	)

	ErrTokenTransferExecutionToWalletEmpty = errors.New(
		"token_transfer_execution_uc: to walletAddress is empty",
	)

	// Bubblegum service / DAS side ownership validation error mappingとの
	// 互換用に残します。Go側でFirestoreのownerを正として検証しません。
	ErrTokenTransferExecutionOwnerMismatch = errors.New(
		"token_transfer_execution_uc: token current owner mismatch",
	)

	ErrTokenTransferExecutionAttemptNotCreated = errors.New(
		"token_transfer_execution_uc: transfer attempt was not created",
	)

	ErrTokenTransferExecutionWalletSyncNotConfigured = errors.New(
		"token_transfer_execution_uc: wallet sync is not configured",
	)

	ErrTokenTransferExecutionWalletSyncFailed = errors.New(
		"token_transfer_execution_uc: wallet sync failed",
	)

	ErrTokenTransferExecutionResolveAfterFailed = errors.New(
		"token_transfer_execution_uc: post-transfer resolve failed",
	)

	ErrTokenTransferExecutionAfterOnChainFailed = errors.New(
		"token_transfer_execution_uc: after-on-chain processing failed",
	)

	ErrTokenTransferExecutionBeforeSuccessFailed = errors.New(
		"token_transfer_execution_uc: before-success processing failed",
	)
)

// -----------------------------------------------------------------------------
// Input / Result
// -----------------------------------------------------------------------------

// TokenTransferExecutionHook allows the caller to execute business-specific
// processing while the transfer attempt is still PENDING.
//
// Typical usage:
//
// TransferUsecase:
//   - AfterOnChain: MarkTransferredItem
//   - BeforeSuccess: inventory cleanup
//
// ShareTransferUsecase:
//   - no hook required
//
// Returning an error causes the transfer attempt to be marked FAILED.
// When the on-chain transaction already exists, its transaction signature is
// preserved in the failed transfer record.
type TokenTransferExecutionHook func(
	ctx context.Context,
	txSignature string,
	now time.Time,
) error

type TokenTransferExecutionInput struct {
	ProductID string

	// OperationID is the stable idempotency key for one logical transfer.
	//
	// The same logical retry must reuse the exact same OperationID.
	// It is persisted with the transfer attempt and forwarded to the
	// Bubblegum service as the transfer idempotency key.
	OperationID string

	// AttemptReference is persisted to transfer.OrderID.
	//
	// TransferUsecase passes the actual order ID.
	// ShareTransferUsecase passes the generated share reference.
	AttemptReference string

	// Exactly one of FromAvatarID / FromBrandID identifies the logical sender.
	// The Bubblegum service resolves the corresponding signer securely.
	FromAvatarID string
	FromBrandID  string
	ToAvatarID   string

	BrandID          string
	ModelID          string
	TokenBlueprintID string

	// AssetID is the Bubblegum V2 cNFT asset identifier.
	AssetID string

	// FromWallet / ToWallet are public wallet addresses only.
	// Private keys or signer objects must not be passed through this usecase.
	FromWallet string
	ToWallet   string

	// RemoveFromSenderWallet is true for avatar -> avatar transfer such as
	// resale and share.
	//
	// It is false for brand -> avatar transfer.
	RemoveFromSenderWallet bool

	// SyncSenderWallet synchronizes the sender avatar wallet from on-chain
	// after the transfer.
	SyncSenderWallet bool

	// SyncReceiverWallet synchronizes the receiver avatar wallet from on-chain
	// after the transfer.
	SyncReceiverWallet bool

	// AfterOnChain is executed immediately after the blockchain transaction
	// succeeds and before token / wallet state updates.
	//
	// TransferUsecase can use this to mark the matched order item transferred.
	AfterOnChain TokenTransferExecutionHook

	// BeforeSuccess is executed after common token / wallet processing and
	// resolver warmup, but before transfer status becomes SUCCEEDED.
	//
	// TransferUsecase can use this for inventory cleanup.
	BeforeSuccess TokenTransferExecutionHook
}

type TokenTransferExecutionResult struct {
	Attempt int

	ProductID   string
	OperationID string

	AssetID string

	FromWallet string
	ToWallet   string

	TxSignature string

	ExecutedAt time.Time
}

// -----------------------------------------------------------------------------
// Execute
// -----------------------------------------------------------------------------

func (u *TokenTransferExecutionUsecase) Execute(
	ctx context.Context,
	in TokenTransferExecutionInput,
) (
	result TokenTransferExecutionResult,
	retErr error,
) {
	if u == nil ||
		u.tokenUpdate == nil ||
		u.walletUpdate == nil ||
		u.transferRepo == nil ||
		u.executor == nil ||
		u.now == nil {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionNotConfigured
	}

	if in.ProductID == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionProductIDEmpty
	}

	if in.OperationID == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionOperationIDEmpty
	}

	if in.AttemptReference == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionAttemptReferenceEmpty
	}

	if in.ToAvatarID == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionToAvatarIDEmpty
	}

	if in.AssetID == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionAssetIDEmpty
	}

	if in.FromWallet == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionFromWalletEmpty
	}

	if in.ToWallet == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionToWalletEmpty
	}

	if in.FromAvatarID == "" &&
		in.FromBrandID == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionFromAvatarIDEmpty
	}

	if in.RemoveFromSenderWallet ||
		in.SyncSenderWallet {
		if in.FromAvatarID == "" {
			return TokenTransferExecutionResult{},
				ErrTokenTransferExecutionFromAvatarIDEmpty
		}
	}

	if in.SyncSenderWallet ||
		in.SyncReceiverWallet {
		if u.walletSync == nil {
			return TokenTransferExecutionResult{},
				ErrTokenTransferExecutionWalletSyncNotConfigured
		}
	}

	now := u.now().UTC()

	createdTransfer, err := u.transferRepo.CreateAttempt(
		ctx,
		transferdom.CreateAttemptInput{
			ProductID:       in.ProductID,
			OperationID:     in.OperationID,
			OrderID:         in.AttemptReference,
			AvatarID:        in.ToAvatarID,
			ToWalletAddress: in.ToWallet,
			AssetID:         in.AssetID,
			CreatedAt:       now,
		},
	)
	if err != nil {
		return TokenTransferExecutionResult{},
			fmt.Errorf(
				"token_transfer_execution_uc: create transfer attempt failed productId=%s assetId=%s: %w",
				in.ProductID,
				in.AssetID,
				err,
			)
	}

	if createdTransfer == nil ||
		createdTransfer.Attempt <= 0 {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionAttemptNotCreated
	}

	transferAttempt := createdTransfer.Attempt

	// CreateAttempt is idempotent by OperationID.
	// If the same logical operation already completed successfully,
	// return the persisted result without executing another on-chain transfer.
	if createdTransfer.Status == transferdom.StatusSucceeded &&
		createdTransfer.TxSignature != nil &&
		*createdTransfer.TxSignature != "" {
		return TokenTransferExecutionResult{
			Attempt: transferAttempt,

			ProductID:   in.ProductID,
			OperationID: in.OperationID,

			AssetID: in.AssetID,

			FromWallet: in.FromWallet,
			ToWallet:   in.ToWallet,

			TxSignature: *createdTransfer.TxSignature,

			ExecutedAt: now,
		}, nil
	}

	transferFailed := false

	patchTransfer := func(
		patch transferdom.TransferPatch,
	) error {
		_, err := u.transferRepo.Patch(
			context.Background(),
			in.ProductID,
			transferAttempt,
			patch,
		)

		return err
	}

	markFailed := func(
		errorType transferdom.ErrorType,
		message string,
		txSignature *string,
	) {
		status := transferdom.StatusFailed

		patch := transferdom.TransferPatch{
			Status:    &status,
			ErrorType: &errorType,
			ErrorMsg:  &message,
		}

		if txSignature != nil {
			signature := *txSignature
			patch.TxSignature = &signature
		}

		if err := patchTransfer(patch); err == nil {
			transferFailed = true
		}
	}

	markSucceeded := func(
		txSignature string,
	) error {
		if txSignature == "" {
			return transferdom.ErrEmptyTxSignature
		}

		status := transferdom.StatusSucceeded
		signature := txSignature

		return patchTransfer(
			transferdom.TransferPatch{
				Status:      &status,
				TxSignature: &signature,
			},
		)
	}

	defer func() {
		if retErr == nil || transferFailed {
			return
		}

		status := transferdom.StatusFailed
		errorType := transferdom.ErrorTypeUnknown
		message := retErr.Error()

		_ = patchTransfer(
			transferdom.TransferPatch{
				Status:    &status,
				ErrorType: &errorType,
				ErrorMsg:  &message,
			},
		)
	}()

	// -------------------------------------------------------------------------
	// On-chain Bubblegum V2 transfer
	// -------------------------------------------------------------------------

	executeResult, err := u.executor.ExecuteTransfer(
		ctx,
		ExecuteTransferInput{
			ProductID:   in.ProductID,
			OperationID: in.OperationID,

			FromAvatarID: in.FromAvatarID,
			ToAvatarID:   in.ToAvatarID,
			FromBrandID:  in.FromBrandID,

			BrandID:          in.BrandID,
			ModelID:          in.ModelID,
			TokenBlueprintID: in.TokenBlueprintID,

			AssetID: in.AssetID,

			FromWalletAddress: in.FromWallet,
			ToWalletAddress:   in.ToWallet,
		},
	)
	if err != nil {
		message := fmt.Sprintf(
			"execute transfer failed productId=%s assetId=%s fromWallet=%s toWallet=%s: %v",
			in.ProductID,
			in.AssetID,
			in.FromWallet,
			in.ToWallet,
			err,
		)

		markFailed(
			transferdom.ErrorTypeTransferFailed,
			message,
			nil,
		)

		return TokenTransferExecutionResult{},
			fmt.Errorf(
				"token_transfer_execution_uc: execute transfer failed productId=%s assetId=%s: %w",
				in.ProductID,
				in.AssetID,
				err,
			)
	}

	txSignature := executeResult.TxSignature
	if txSignature == "" {
		message := "transfer executor returned an empty txSignature"

		markFailed(
			transferdom.ErrorTypeTransferFailed,
			message,
			nil,
		)

		return TokenTransferExecutionResult{},
			transferdom.ErrEmptyTxSignature
	}

	// -------------------------------------------------------------------------
	// Caller-specific post-on-chain processing
	// -------------------------------------------------------------------------

	if in.AfterOnChain != nil {
		if err := in.AfterOnChain(
			ctx,
			txSignature,
			now,
		); err != nil {
			message := fmt.Sprintf(
				"after-on-chain processing failed productId=%s assetId=%s tx=%s: %v",
				in.ProductID,
				in.AssetID,
				txSignature,
				err,
			)

			markFailed(
				transferdom.ErrorTypeUnknown,
				message,
				&txSignature,
			)

			return TokenTransferExecutionResult{},
				fmt.Errorf(
					"%w: %s",
					ErrTokenTransferExecutionAfterOnChainFailed,
					message,
				)
		}
	}

	// -------------------------------------------------------------------------
	// Cached token owner update
	// -------------------------------------------------------------------------

	if err := u.tokenUpdate.UpdateToAddressByProductID(
		ctx,
		in.ProductID,
		in.ToWallet,
		now,
		txSignature,
	); err != nil {
		message := fmt.Sprintf(
			"update token owner failed productId=%s assetId=%s to=%s tx=%s: %v",
			in.ProductID,
			in.AssetID,
			in.ToWallet,
			txSignature,
			err,
		)

		markFailed(
			transferdom.ErrorTypeUnknown,
			message,
			&txSignature,
		)

		return TokenTransferExecutionResult{},
			fmt.Errorf(
				"token_transfer_execution_uc: update token owner failed productId=%s assetId=%s to=%s tx=%s: %w",
				in.ProductID,
				in.AssetID,
				in.ToWallet,
				txSignature,
				err,
			)
	}

	// -------------------------------------------------------------------------
	// Wallet asset cache update
	// -------------------------------------------------------------------------

	if in.RemoveFromSenderWallet {
		if err := u.walletUpdate.RemoveAssetIDFromAvatarWalletItems(
			ctx,
			in.FromAvatarID,
			in.AssetID,
			now,
		); err != nil {
			message := fmt.Sprintf(
				"remove sender wallet asset failed avatarId=%s assetId=%s tx=%s: %v",
				in.FromAvatarID,
				in.AssetID,
				txSignature,
				err,
			)

			markFailed(
				transferdom.ErrorTypeUnknown,
				message,
				&txSignature,
			)

			return TokenTransferExecutionResult{},
				fmt.Errorf(
					"token_transfer_execution_uc: %s",
					message,
				)
		}
	}

	if err := u.walletUpdate.AddAssetIDToAvatarWalletItems(
		ctx,
		in.ToAvatarID,
		in.AssetID,
		now,
	); err != nil {
		message := fmt.Sprintf(
			"add receiver wallet asset failed avatarId=%s assetId=%s tx=%s: %v",
			in.ToAvatarID,
			in.AssetID,
			txSignature,
			err,
		)

		markFailed(
			transferdom.ErrorTypeUnknown,
			message,
			&txSignature,
		)

		return TokenTransferExecutionResult{},
			fmt.Errorf(
				"token_transfer_execution_uc: %s",
				message,
			)
	}

	// -------------------------------------------------------------------------
	// Wallet read-through sync
	// -------------------------------------------------------------------------

	if in.SyncSenderWallet {
		if _, err := u.walletSync.SyncWalletAssetIDs(
			ctx,
			in.FromAvatarID,
		); err != nil {
			message := fmt.Sprintf(
				"sync sender wallet failed avatarId=%s wallet=%s assetId=%s tx=%s: %v",
				in.FromAvatarID,
				in.FromWallet,
				in.AssetID,
				txSignature,
				err,
			)

			markFailed(
				transferdom.ErrorTypeUnknown,
				message,
				&txSignature,
			)

			return TokenTransferExecutionResult{},
				fmt.Errorf(
					"%w: %s",
					ErrTokenTransferExecutionWalletSyncFailed,
					message,
				)
		}
	}

	if in.SyncReceiverWallet {
		if _, err := u.walletSync.SyncWalletAssetIDs(
			ctx,
			in.ToAvatarID,
		); err != nil {
			message := fmt.Sprintf(
				"sync receiver wallet failed avatarId=%s wallet=%s assetId=%s tx=%s: %v",
				in.ToAvatarID,
				in.ToWallet,
				in.AssetID,
				txSignature,
				err,
			)

			markFailed(
				transferdom.ErrorTypeUnknown,
				message,
				&txSignature,
			)

			return TokenTransferExecutionResult{},
				fmt.Errorf(
					"%w: %s",
					ErrTokenTransferExecutionWalletSyncFailed,
					message,
				)
		}
	}

	// -------------------------------------------------------------------------
	// Post-transfer resolve warmup
	// -------------------------------------------------------------------------

	if u.resolveWarmer != nil {
		if err := u.resolveWarmer.ResolveAfterTransfer(
			ctx,
			in.ToAvatarID,
			in.AssetID,
		); err != nil {
			message := fmt.Sprintf(
				"post-transfer resolve failed avatarId=%s assetId=%s tx=%s: %v",
				in.ToAvatarID,
				in.AssetID,
				txSignature,
				err,
			)

			markFailed(
				transferdom.ErrorTypeUnknown,
				message,
				&txSignature,
			)

			return TokenTransferExecutionResult{},
				fmt.Errorf(
					"%w: %s",
					ErrTokenTransferExecutionResolveAfterFailed,
					message,
				)
		}
	}

	// -------------------------------------------------------------------------
	// Caller-specific finalization
	// -------------------------------------------------------------------------

	if in.BeforeSuccess != nil {
		if err := in.BeforeSuccess(
			ctx,
			txSignature,
			now,
		); err != nil {
			message := fmt.Sprintf(
				"before-success processing failed productId=%s assetId=%s tx=%s: %v",
				in.ProductID,
				in.AssetID,
				txSignature,
				err,
			)

			markFailed(
				transferdom.ErrorTypeUnknown,
				message,
				&txSignature,
			)

			return TokenTransferExecutionResult{},
				fmt.Errorf(
					"%w: %s",
					ErrTokenTransferExecutionBeforeSuccessFailed,
					message,
				)
		}
	}

	// -------------------------------------------------------------------------
	// Success
	// -------------------------------------------------------------------------

	// SUCCEEDED is intentionally written only after every common and
	// caller-specific post-transfer operation has completed successfully.
	if err := markSucceeded(txSignature); err != nil {
		return TokenTransferExecutionResult{},
			fmt.Errorf(
				"token_transfer_execution_uc: mark transfer succeeded failed productId=%s assetId=%s attempt=%d tx=%s: %w",
				in.ProductID,
				in.AssetID,
				transferAttempt,
				txSignature,
				err,
			)
	}

	return TokenTransferExecutionResult{
		Attempt: transferAttempt,

		ProductID:   in.ProductID,
		OperationID: in.OperationID,

		AssetID: in.AssetID,

		FromWallet: in.FromWallet,
		ToWallet:   in.ToWallet,

		TxSignature: txSignature,

		ExecutedAt: now,
	}, nil
}
