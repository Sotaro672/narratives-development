// backend/internal/application/usecase/transfer_recovery.go
package usecase

import (
	"context"
	"errors"
	"fmt"

	transferdom "narratives/internal/domain/transfer"
)

func (u *TransferUsecase) recoverSucceededTransferByOperationID(
	ctx context.Context,
	in TransferByVerifiedScanInput,
) (TransferByVerifiedScanResult, bool, error) {
	if u == nil ||
		u.executionUC == nil ||
		u.executionUC.transferRepo == nil {
		return TransferByVerifiedScanResult{},
			false,
			ErrTransferNotConfigured
	}

	transfer, err := u.executionUC.transferRepo.GetByOperationID(
		ctx,
		in.OperationID,
	)
	if err != nil {
		if errors.Is(err, transferdom.ErrNotFound) {
			return TransferByVerifiedScanResult{}, false, nil
		}

		return TransferByVerifiedScanResult{},
			false,
			fmt.Errorf(
				"transfer_uc: resolve transfer by operationId failed operationId=%s: %w",
				in.OperationID,
				err,
			)
	}

	if transfer == nil {
		return TransferByVerifiedScanResult{}, false, nil
	}

	if transfer.OperationID != in.OperationID ||
		transfer.ProductID != in.ProductID ||
		transfer.AvatarID != in.AvatarID {
		return TransferByVerifiedScanResult{},
			false,
			ErrTransferOperationMismatch
	}

	if transfer.Status != transferdom.StatusSucceeded {
		return TransferByVerifiedScanResult{}, false, nil
	}
	if transfer.TxSignature == nil || *transfer.TxSignature == "" {
		return TransferByVerifiedScanResult{},
			false,
			transferdom.ErrEmptyTxSignature
	}

	fromDisplayName := ""
	if transfer.FromAvatarID != "" {
		fromDisplayName = u.resolveAvatarDisplayName(
			ctx,
			transfer.FromAvatarID,
		)
	} else if transfer.FromBrandID != "" {
		fromDisplayName = u.resolveBrandDisplayName(
			ctx,
			transfer.FromBrandID,
		)
	}

	toDisplayName := u.resolveAvatarDisplayName(
		ctx,
		transfer.AvatarID,
	)

	return TransferByVerifiedScanResult{
		MatchedOrderID: transfer.OrderID,

		MatchedItemIndex: transfer.OrderItemIndex,
		MatchedItemType:  transfer.OrderItemType,

		ProductID: transfer.ProductID,
		AssetID:   transfer.AssetID,

		ToWallet:    transfer.ToWalletAddress,
		TxSignature: *transfer.TxSignature,

		FromDisplayName: fromDisplayName,
		ToDisplayName:   toDisplayName,
	}, true, nil
}
