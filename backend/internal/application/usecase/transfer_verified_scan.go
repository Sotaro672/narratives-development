// backend/internal/application/usecase/transfer_verified_scan.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	applicationport "narratives/internal/application/port"
	orderdom "narratives/internal/domain/order"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
)

// TransferToAvatarByVerifiedScan verifies the scan and transfers the token to
// the authenticated avatar.
func (u *TransferUsecase) TransferToAvatarByVerifiedScan(
	ctx context.Context,
	in TransferByVerifiedScanInput,
) (TransferByVerifiedScanResult, error) {
	if u == nil ||
		u.verifier == nil ||
		u.orderRepo == nil ||
		u.tokenRepo == nil ||
		u.brandWallet == nil ||
		u.avatarWallet == nil ||
		u.executionUC == nil ||
		u.now == nil {
		return TransferByVerifiedScanResult{},
			ErrTransferNotConfigured
	}

	avatarID := in.AvatarID
	productID := in.ProductID
	operationID := in.OperationID

	if avatarID == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferAvatarIDEmpty
	}
	if productID == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferProductIDEmpty
	}
	if operationID == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferOperationIDEmpty
	}

	recovered, ok, err := u.recoverSucceededTransferByOperationID(
		ctx,
		TransferByVerifiedScanInput{
			AvatarID:    avatarID,
			ProductID:   productID,
			OperationID: operationID,
		},
	)
	if err != nil {
		return TransferByVerifiedScanResult{}, err
	}
	if ok {
		return recovered, nil
	}

	verifyResult, err := u.verifier.VerifyMatch(
		ctx,
		VerifyInput{
			AvatarID:  avatarID,
			ProductID: productID,
		},
	)
	if err != nil {
		return TransferByVerifiedScanResult{},
			fmt.Errorf("transfer_uc: verify failed: %w", err)
	}
	if !verifyResult.Matched {
		return TransferByVerifiedScanResult{},
			ErrTransferNotMatched
	}

	scannedModelID := verifyResult.ScannedModelID
	scannedTokenBlueprintID :=
		verifyResult.ScannedTokenBlueprintID

	token, err := u.tokenRepo.ResolveTokenByProductID(
		ctx,
		productID,
	)
	if err != nil {
		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: resolve token failed productId=%s: %w",
				productID,
				err,
			)
	}

	brandID := token.BrandID
	assetID := token.AssetID
	tokenBlueprintID := token.TokenBlueprintID

	if brandID == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferBrandIDEmpty
	}
	if assetID == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferAssetIDEmpty
	}

	if scannedTokenBlueprintID == "" {
		scannedTokenBlueprintID = tokenBlueprintID
	}
	if scannedTokenBlueprintID == "" {
		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: scanned tokenBlueprintId empty productId=%s",
				productID,
			)
	}

	if tokenBlueprintID != "" &&
		tokenBlueprintID != scannedTokenBlueprintID {
		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: tokenBlueprint mismatch productId=%s scanned=%s tokenDoc=%s",
				productID,
				scannedTokenBlueprintID,
				tokenBlueprintID,
			)
	}

	target, err := u.orderRepo.FindEligibleTransferItem(
		ctx,
		applicationport.FindEligibleTransferItemInput{
			AvatarID:         avatarID,
			ProductID:        productID,
			ModelID:          scannedModelID,
			TokenBlueprintID: scannedTokenBlueprintID,
		},
	)
	if err != nil {
		if errors.Is(err, orderdom.ErrNotFound) {
			return TransferByVerifiedScanResult{},
				ErrTransferNoEligibleOrder
		}

		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: find eligible transfer item failed avatarId=%s productId=%s: %w",
				avatarID,
				productID,
				err,
			)
	}
	if target.OrderID == "" || target.ItemIndex < 0 {
		return TransferByVerifiedScanResult{},
			ErrTransferNoEligibleOrder
	}

	matchedResult := TransferByVerifiedScanResult{
		MatchedOrderID:     target.OrderID,
		MatchedInventoryID: target.InventoryID,
		MatchedModelID:     target.ModelID,

		MatchedItemIndex: target.ItemIndex,
		MatchedItemType:  target.ItemType,
		MatchedResaleID:  target.ResaleID,

		ProductID:        productID,
		AssetID:          assetID,
		TokenBlueprintID: scannedTokenBlueprintID,
	}

	verifiedAt := u.now().UTC()

	verifiedItem, err :=
		u.orderRepo.MarkTokenTransferVerified(
			ctx,
			target.OrderID,
			target.ItemIndex,
			verifiedAt,
		)
	if err != nil {
		return matchedResult,
			fmt.Errorf(
				"transfer_uc: mark token transfer verified failed orderId=%s itemIndex=%d: %w",
				target.OrderID,
				target.ItemIndex,
				err,
			)
	}

	if verifiedItem.IsReturnRequested {
		if err := u.promoteReturnOpened(
			ctx,
			target,
			avatarID,
			productID,
		); err != nil {
			return matchedResult,
				fmt.Errorf(
					"%w: promote return failed orderId=%s itemIndex=%d: %v",
					ErrTransferBlockedByReturn,
					target.OrderID,
					target.ItemIndex,
					err,
				)
		}

		return matchedResult,
			ErrTransferBlockedByReturn
	}

	lockAt := u.now().UTC()

	if err := u.orderRepo.LockTransferItem(
		ctx,
		target.OrderID,
		target.ItemIndex,
		lockAt,
	); err != nil {
		if errors.Is(
			err,
			orderdom.ErrConflict,
		) {
			latestItem, verifiedErr :=
				u.orderRepo.MarkTokenTransferVerified(
					ctx,
					target.OrderID,
					target.ItemIndex,
					verifiedAt,
				)
			if verifiedErr != nil {
				return matchedResult,
					fmt.Errorf(
						"%w: refresh token transfer verified failed orderId=%s itemIndex=%d: %v",
						ErrTransferBlockedByReturn,
						target.OrderID,
						target.ItemIndex,
						verifiedErr,
					)
			}

			if latestItem.IsReturnRequested {
				if promoteErr := u.promoteReturnOpened(
					ctx,
					target,
					avatarID,
					productID,
				); promoteErr != nil {
					return matchedResult,
						fmt.Errorf(
							"%w: promote return failed orderId=%s itemIndex=%d: %v",
							ErrTransferBlockedByReturn,
							target.OrderID,
							target.ItemIndex,
							promoteErr,
						)
				}
			}

			return matchedResult,
				ErrTransferBlockedByReturn
		}

		return matchedResult,
			fmt.Errorf(
				"transfer_uc: lock failed orderId=%s itemIndex=%d: %w",
				target.OrderID,
				target.ItemIndex,
				err,
			)
	}

	locked := true
	defer func() {
		if locked {
			_ = u.orderRepo.UnlockTransferItem(
				context.Background(),
				target.OrderID,
				target.ItemIndex,
			)
		}
	}()

	toWallet, err :=
		u.avatarWallet.ResolveAvatarWalletAddress(
			ctx,
			avatarID,
		)
	if err != nil {
		return matchedResult,
			fmt.Errorf(
				"transfer_uc: resolve receiver avatar wallet failed avatarId=%s: %w",
				avatarID,
				err,
			)
	}
	if toWallet == "" {
		return matchedResult,
			ErrTransferToWalletEmpty
	}

	source, err := u.resolveTransferSource(
		ctx,
		target,
		brandID,
		avatarID,
	)
	if err != nil {
		return matchedResult, err
	}

	removeFromSenderWallet :=
		target.ItemType == orderdom.OrderItemTypeResale
	syncSenderWallet :=
		target.ItemType == orderdom.OrderItemTypeResale
	syncReceiverWallet :=
		target.ItemType == orderdom.OrderItemTypeResale

	var resaleReceivable salesreceivabledom.SalesReceivable
	if target.ItemType == orderdom.OrderItemTypeResale {
		if u.bankPayoutUC == nil {
			return matchedResult,
				ErrTransferBankPayoutNotConfigured
		}

		resaleReceivable, err = u.requirePendingResaleReceivable(
			ctx,
			target.OrderID,
			target.ItemIndex,
			target.ResaleID,
			verifiedItem,
			source.FromAvatarID,
		)
		if err != nil {
			return matchedResult, err
		}
	}

	afterOnChain := func(
		ctx context.Context,
		txSignature string,
		now time.Time,
	) error {
		if target.ItemType == orderdom.OrderItemTypeResale {
			completedReceivable, err := u.orderRepo.CompleteResaleReceivableFulfillment(
				ctx,
				target.OrderID,
				target.ItemIndex,
				resaleReceivable,
				now,
			)
			if err != nil {
				return fmt.Errorf(
					"complete resale transfer fulfillment failed orderId=%s itemIndex=%d receivableId=%s tx=%s: %w",
					target.OrderID,
					target.ItemIndex,
					resaleReceivable.ID,
					txSignature,
					err,
				)
			}

			if err := validateCompletedResaleReceivable(
				completedReceivable,
				resaleReceivable,
			); err != nil {
				return err
			}

			return nil
		}

		if err := u.orderRepo.MarkTransferredItem(
			ctx,
			target.OrderID,
			target.ItemIndex,
			now,
		); err != nil {
			return fmt.Errorf(
				"mark transferred failed orderId=%s itemIndex=%d tx=%s: %w",
				target.OrderID,
				target.ItemIndex,
				txSignature,
				err,
			)
		}

		return nil
	}

	var beforeSuccess TokenTransferExecutionHook
	if target.ItemType == orderdom.OrderItemTypeList &&
		u.inventoryUC != nil {
		beforeSuccess = func(
			ctx context.Context,
			txSignature string,
			now time.Time,
		) error {
			if err := u.inventoryUC.ReleaseAfterTransfer(
				ctx,
				target.InventoryID,
				target.ModelID,
				productID,
				target.OrderID,
				now,
			); err != nil {
				return fmt.Errorf(
					"inventory cleanup failed inventoryId=%s modelId=%s productId=%s orderId=%s tx=%s: %w",
					target.InventoryID,
					target.ModelID,
					productID,
					target.OrderID,
					txSignature,
					err,
				)
			}

			return nil
		}
	}

	executionResult, err := u.executionUC.Execute(
		ctx,
		TokenTransferExecutionInput{
			ProductID:   productID,
			OperationID: operationID,

			AttemptReference: target.OrderID,
			OrderItemIndex:   target.ItemIndex,
			OrderItemType:    target.ItemType,

			FromAvatarID: source.FromAvatarID,
			ToAvatarID:   avatarID,
			FromBrandID:  source.FromBrandID,

			BrandID:          brandID,
			ModelID:          target.ModelID,
			TokenBlueprintID: scannedTokenBlueprintID,

			AssetID: assetID,

			FromWallet: source.FromWallet,
			ToWallet:   toWallet,

			RemoveFromSenderWallet: removeFromSenderWallet,
			SyncSenderWallet:       syncSenderWallet,
			SyncReceiverWallet:     syncReceiverWallet,

			AfterOnChain:  afterOnChain,
			BeforeSuccess: beforeSuccess,
		},
	)
	if err != nil {
		return matchedResult,
			mapTransferExecutionError(err)
	}

	matchedResult.FromWallet = source.FromWallet
	matchedResult.ToWallet = toWallet
	matchedResult.TxSignature = executionResult.TxSignature

	if target.ItemType == orderdom.OrderItemTypeResale {
		payoutCtx, cancelPayout := context.WithTimeout(
			context.WithoutCancel(ctx),
			30*time.Second,
		)
		_, payoutErr := u.bankPayoutUC.ExecuteForSalesReceivable(
			payoutCtx,
			resaleReceivable.ID,
		)
		cancelPayout()

		if payoutErr != nil {
			log.Printf(
				"[transfer] bank payout failed after token transfer orderId=%s itemIndex=%d receivableId=%s tx=%s err=%v",
				target.OrderID,
				target.ItemIndex,
				resaleReceivable.ID,
				executionResult.TxSignature,
				payoutErr,
			)
		}
	}

	fromDisplayName := ""
	if source.FromAvatarID != "" {
		fromDisplayName = u.resolveAvatarDisplayName(
			ctx,
			source.FromAvatarID,
		)
	} else {
		fromDisplayName = u.resolveBrandDisplayName(
			ctx,
			source.FromBrandID,
		)
	}

	toDisplayName := u.resolveAvatarDisplayName(
		ctx,
		avatarID,
	)

	locked = false

	return TransferByVerifiedScanResult{
		MatchedOrderID:     target.OrderID,
		MatchedInventoryID: target.InventoryID,
		MatchedModelID:     target.ModelID,

		MatchedItemIndex: target.ItemIndex,
		MatchedItemType:  target.ItemType,
		MatchedResaleID:  target.ResaleID,

		ProductID:        productID,
		AssetID:          assetID,
		TokenBlueprintID: scannedTokenBlueprintID,

		FromWallet:  source.FromWallet,
		ToWallet:    toWallet,
		TxSignature: executionResult.TxSignature,

		FromDisplayName: fromDisplayName,
		ToDisplayName:   toDisplayName,
	}, nil
}

func (u *TransferUsecase) promoteReturnOpened(
	ctx context.Context,
	target applicationport.TransferTargetItem,
	avatarID string,
	productID string,
) error {
	if u == nil ||
		u.returnOpening == nil {
		return ErrTransferNotConfigured
	}

	_, err :=
		u.returnOpening.PromoteUnopenedToOpened(
			ctx,
			PromoteUnopenedToOpenedInput{
				OrderID:   target.OrderID,
				AvatarID:  avatarID,
				ItemIndex: target.ItemIndex,
				ProductID: productID,
			},
		)
	if err != nil {
		return err
	}

	return nil
}
