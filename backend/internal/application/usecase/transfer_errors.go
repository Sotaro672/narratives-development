// backend/internal/application/usecase/transfer_errors.go
package usecase

import (
	"errors"
	"fmt"
)

var (
	ErrTransferNotConfigured          = errors.New("transfer_uc: not configured")
	ErrTransferAvatarIDEmpty          = errors.New("transfer_uc: avatarId is empty")
	ErrTransferProductIDEmpty         = errors.New("transfer_uc: productId is empty")
	ErrTransferOperationIDEmpty       = errors.New("transfer_uc: operationId is empty")
	ErrTransferOperationMismatch      = errors.New("transfer_uc: operationId is associated with another transfer")
	ErrTransferNotMatched             = errors.New("transfer_uc: scan is not matched")
	ErrTransferNoEligibleOrder        = errors.New("transfer_uc: no eligible order/item found")
	ErrTransferAssetIDEmpty           = errors.New("transfer_uc: assetId is empty")
	ErrTransferBrandIDEmpty           = errors.New("transfer_uc: brandId is empty")
	ErrTransferFromWalletEmpty        = errors.New("transfer_uc: from walletAddress is empty")
	ErrTransferToWalletEmpty          = errors.New("transfer_uc: avatar walletAddress is empty")
	ErrTransferOwnerMismatch          = errors.New("transfer_uc: token current owner mismatch")
	ErrTransferResolveAfterFailed     = errors.New("transfer_uc: post-transfer resolve failed")
	ErrTransferInventoryCleanupFailed = errors.New("transfer_uc: inventory cleanup failed")
	ErrTransferAttemptNotCreated      = errors.New("transfer_uc: transfer attempt was not created")
	ErrTransferBlockedByReturn        = errors.New("transfer_uc: transfer blocked by return request")

	ErrTransferResaleNotConfigured           = errors.New("transfer_uc: resale transfer dependencies are not configured")
	ErrTransferResaleIDEmpty                 = errors.New("transfer_uc: resaleId is empty")
	ErrTransferResaleSellerAvatarIDEmpty     = errors.New("transfer_uc: resale seller avatarId is empty")
	ErrTransferResaleReceivableNotConfigured = errors.New("transfer_uc: resale receivable dependencies are not configured")
	ErrTransferResaleReceivableMismatch      = errors.New("transfer_uc: resale receivable identity mismatch")
	ErrTransferResaleReceivableUnavailable   = errors.New("transfer_uc: resale receivable is unavailable")
	ErrTransferBankPayoutNotConfigured       = errors.New("transfer_uc: bank payout dependencies are not configured")
	ErrTransferBankPayoutFailed              = errors.New("transfer_uc: bank payout failed after token transfer")
	ErrTransferSameAvatar                    = errors.New("transfer_uc: seller avatarId and buyer avatarId must be different")
	ErrTransferWalletSyncFailed              = errors.New("transfer_uc: wallet sync failed")
)

func mapTransferExecutionError(err error) error {
	switch {
	case errors.Is(
		err,
		ErrTokenTransferExecutionNotConfigured,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrTransferNotConfigured,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionAttemptNotCreated,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrTransferAttemptNotCreated,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionOwnerMismatch,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrTransferOwnerMismatch,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionWalletSyncNotConfigured,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrTransferResaleNotConfigured,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionWalletSyncFailed,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrTransferWalletSyncFailed,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionResolveAfterFailed,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrTransferResolveAfterFailed,
			err,
		)

	case errors.Is(
		err,
		ErrTokenTransferExecutionBeforeSuccessFailed,
	):
		return fmt.Errorf(
			"%w: %v",
			ErrTransferInventoryCleanupFailed,
			err,
		)

	default:
		return err
	}
}
