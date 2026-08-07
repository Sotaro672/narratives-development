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

// TokenTransferExecutionUsecase executes the common token transfer flow shared
// by TransferUsecase and ShareTransferUsecase.
//
// This use case is responsible only for the common transfer execution:
//
//  1. Create transfer attempt as PENDING.
//  2. Verify the current token owner.
//  3. Execute the on-chain token transfer.
//  4. Run caller-specific post-on-chain processing when configured.
//  5. Update the token owner.
//  6. Update sender / receiver wallet caches.
//  7. Synchronize wallets from on-chain when configured.
//  8. Warm the post-transfer token resolver when configured.
//  9. Run caller-specific finalization when configured.
//
// 10. Mark the transfer as SUCCEEDED.
//
// Scan verification, order lookup / locking, transfer source resolution,
// signer resolution, order item state transition, and inventory-specific
// business rules remain responsibilities of the caller.
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

	ErrTokenTransferExecutionAttemptReferenceEmpty = errors.New(
		"token_transfer_execution_uc: attempt reference is empty",
	)

	ErrTokenTransferExecutionFromAvatarIDEmpty = errors.New(
		"token_transfer_execution_uc: fromAvatarId is empty",
	)

	ErrTokenTransferExecutionToAvatarIDEmpty = errors.New(
		"token_transfer_execution_uc: toAvatarId is empty",
	)

	ErrTokenTransferExecutionMintAddressEmpty = errors.New(
		"token_transfer_execution_uc: mintAddress is empty",
	)

	ErrTokenTransferExecutionFromWalletEmpty = errors.New(
		"token_transfer_execution_uc: from walletAddress is empty",
	)

	ErrTokenTransferExecutionToWalletEmpty = errors.New(
		"token_transfer_execution_uc: to walletAddress is empty",
	)

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

	// AttemptReference is persisted to transfer.OrderID.
	//
	// TransferUsecase passes the actual order ID.
	// ShareTransferUsecase passes the generated share reference.
	AttemptReference string

	FromAvatarID string
	ToAvatarID   string

	BrandID          string
	ModelID          string
	TokenBlueprintID string

	MintAddress string

	// CurrentOwner is the current tokens/{productId}.toAddress.
	// When empty, owner verification is skipped.
	CurrentOwner string

	FromWallet string
	ToWallet   string

	FromSigner any
	ToSigner   any

	// Amount defaults to 1 when zero.
	Amount uint64

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

	ProductID string

	MintAddress string

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

	if in.AttemptReference == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionAttemptReferenceEmpty
	}

	if in.ToAvatarID == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionToAvatarIDEmpty
	}

	if in.MintAddress == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionMintAddressEmpty
	}

	if in.FromWallet == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionFromWalletEmpty
	}

	if in.ToWallet == "" {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionToWalletEmpty
	}

	if in.RemoveFromSenderWallet || in.SyncSenderWallet {
		if in.FromAvatarID == "" {
			return TokenTransferExecutionResult{},
				ErrTokenTransferExecutionFromAvatarIDEmpty
		}
	}

	if in.SyncSenderWallet || in.SyncReceiverWallet {
		if u.walletSync == nil {
			return TokenTransferExecutionResult{},
				ErrTokenTransferExecutionWalletSyncNotConfigured
		}
	}

	now := u.now().UTC()

	amount := in.Amount
	if amount == 0 {
		amount = 1
	}

	createdTransfer, err := u.transferRepo.CreateAttempt(
		ctx,
		transferdom.CreateAttemptInput{
			ProductID:       in.ProductID,
			OrderID:         in.AttemptReference,
			AvatarID:        in.ToAvatarID,
			ToWalletAddress: in.ToWallet,
			MintAddress:     in.MintAddress,
			CreatedAt:       now,
		},
	)
	if err != nil {
		return TokenTransferExecutionResult{},
			fmt.Errorf(
				"token_transfer_execution_uc: create transfer attempt failed productId=%s: %w",
				in.ProductID,
				err,
			)
	}

	if createdTransfer == nil ||
		createdTransfer.Attempt <= 0 {
		return TokenTransferExecutionResult{},
			ErrTokenTransferExecutionAttemptNotCreated
	}

	transferAttempt := createdTransfer.Attempt
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
	// Owner validation
	// -------------------------------------------------------------------------

	if in.CurrentOwner != "" &&
		in.CurrentOwner != in.FromWallet {
		message := fmt.Sprintf(
			"productId=%s tokenOwner=%s expectedFromWallet=%s",
			in.ProductID,
			in.CurrentOwner,
			in.FromWallet,
		)

		markFailed(
			transferdom.ErrorTypeMismatch,
			message,
			nil,
		)

		return TokenTransferExecutionResult{},
			fmt.Errorf(
				"%w: %s",
				ErrTokenTransferExecutionOwnerMismatch,
				message,
			)
	}

	// -------------------------------------------------------------------------
	// On-chain transfer
	// -------------------------------------------------------------------------

	executeResult, err := u.executor.ExecuteTransfer(
		ctx,
		ExecuteTransferInput{
			ProductID:        in.ProductID,
			AvatarID:         in.ToAvatarID,
			BrandID:          in.BrandID,
			ModelID:          in.ModelID,
			TokenBlueprintID: in.TokenBlueprintID,

			MintAddress: in.MintAddress,
			Amount:      amount,

			FromWalletAddress: in.FromWallet,
			ToWalletAddress:   in.ToWallet,

			FromSigner: in.FromSigner,
			ToSigner:   in.ToSigner,
		},
	)
	if err != nil {
		message := fmt.Sprintf(
			"execute transfer failed productId=%s fromWallet=%s toWallet=%s mint=%s: %v",
			in.ProductID,
			in.FromWallet,
			in.ToWallet,
			in.MintAddress,
			err,
		)

		markFailed(
			transferdom.ErrorTypeTransferFailed,
			message,
			nil,
		)

		return TokenTransferExecutionResult{},
			fmt.Errorf(
				"token_transfer_execution_uc: execute transfer failed productId=%s mint=%s: %w",
				in.ProductID,
				in.MintAddress,
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
				"after-on-chain processing failed productId=%s tx=%s: %v",
				in.ProductID,
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
	// Token owner update
	// -------------------------------------------------------------------------

	if err := u.tokenUpdate.UpdateToAddressByProductID(
		ctx,
		in.ProductID,
		in.ToWallet,
		now,
		txSignature,
	); err != nil {
		message := fmt.Sprintf(
			"update token owner failed productId=%s to=%s tx=%s: %v",
			in.ProductID,
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
				"token_transfer_execution_uc: update token owner failed productId=%s to=%s tx=%s: %w",
				in.ProductID,
				in.ToWallet,
				txSignature,
				err,
			)
	}

	// -------------------------------------------------------------------------
	// Wallet cache update
	// -------------------------------------------------------------------------

	if in.RemoveFromSenderWallet {
		if err := u.walletUpdate.RemoveMintFromAvatarWalletItems(
			ctx,
			in.FromAvatarID,
			in.MintAddress,
			now,
		); err != nil {
			message := fmt.Sprintf(
				"remove sender wallet item failed avatarId=%s mint=%s tx=%s: %v",
				in.FromAvatarID,
				in.MintAddress,
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

	if err := u.walletUpdate.AddMintToAvatarWalletItems(
		ctx,
		in.ToAvatarID,
		in.MintAddress,
		now,
	); err != nil {
		message := fmt.Sprintf(
			"add receiver wallet item failed avatarId=%s mint=%s tx=%s: %v",
			in.ToAvatarID,
			in.MintAddress,
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
		if _, err := u.walletSync.SyncWalletTokens(
			ctx,
			in.FromAvatarID,
		); err != nil {
			message := fmt.Sprintf(
				"sync sender wallet failed avatarId=%s wallet=%s mint=%s tx=%s: %v",
				in.FromAvatarID,
				in.FromWallet,
				in.MintAddress,
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
		if _, err := u.walletSync.SyncWalletTokens(
			ctx,
			in.ToAvatarID,
		); err != nil {
			message := fmt.Sprintf(
				"sync receiver wallet failed avatarId=%s wallet=%s mint=%s tx=%s: %v",
				in.ToAvatarID,
				in.ToWallet,
				in.MintAddress,
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
			in.MintAddress,
		); err != nil {
			message := fmt.Sprintf(
				"post-transfer resolve failed avatarId=%s mint=%s tx=%s: %v",
				in.ToAvatarID,
				in.MintAddress,
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
				"before-success processing failed productId=%s tx=%s: %v",
				in.ProductID,
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
				"token_transfer_execution_uc: mark transfer succeeded failed productId=%s attempt=%d tx=%s: %w",
				in.ProductID,
				transferAttempt,
				txSignature,
				err,
			)
	}

	return TokenTransferExecutionResult{
		Attempt: transferAttempt,

		ProductID: in.ProductID,

		MintAddress: in.MintAddress,

		FromWallet: in.FromWallet,
		ToWallet:   in.ToWallet,

		TxSignature: txSignature,

		ExecutedAt: now,
	}, nil
}
