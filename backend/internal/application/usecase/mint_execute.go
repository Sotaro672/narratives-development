// backend/internal/application/usecase/mint_execute.go
package usecase

import (
	"context"
	"errors"
	"fmt"

	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
)

// ExecuteNextMintTask は mintID に紐づく次の実行可能 task を1件だけ処理します.
//
// フロー:
//  1. 親 Mint を取得
//  2. 次の PENDING / FAILED_RETRYABLE task を1件取得
//  3. task を MINTING に更新
//  4. productId 1件だけ on-chain mint
//  5. token record / task / inventory を更新
//  6. 未完了 task が残っていれば次の worker task を enqueue
//  7. 全件完了なら親 Mint を MINTED にする
func (u *MintUsecase) ExecuteNextMintTask(
	ctx context.Context,
	mintRequestID string,
) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}

	if mintRequestID == "" {
		return nil, errors.New("mintRequestID is empty")
	}

	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}

	if u.mintTaskRepo == nil {
		return nil, errors.New("mint task repo is nil")
	}

	if u.mintResultMapper == nil {
		return nil, errors.New("mint result mapper is nil")
	}

	if u.mintProductMintRecord == nil {
		return nil, errors.New("mint product recorder is nil")
	}

	mintEntValue, err := u.mintRepo.GetByID(ctx, mintRequestID)
	if err != nil {
		return nil, err
	}

	mintEnt := &mintEntValue

	if mintEnt.Status == mintdom.MintStatusMinted {
		return u.mintResultMapper.FromMint(*mintEnt), nil
	}

	tbID := mintEnt.TokenBlueprintID
	if tbID == "" {
		return nil, errors.New(
			"tokenBlueprintID is empty on mint",
		)
	}

	brandID := mintEnt.BrandID
	if brandID == "" {
		return nil, errors.New(
			"brandID is empty on mint",
		)
	}

	pbID := u.resolveProductBlueprintIDFromProduction(
		ctx,
		mintRequestID,
	)
	if pbID == "" {
		return nil, errors.New(
			"productBlueprintID is empty (cannot upsert inventory)",
		)
	}

	if len(mintEnt.Products) == 0 {
		return nil, errors.New(
			"no products for this mint request",
		)
	}

	if u.tokenMinter == nil {
		return nil, errors.New("token minter is nil")
	}

	if u.mintRequestPort == nil {
		return nil, errors.New("mint request port is nil")
	}

	if err := mintEnt.MarkMinting(); err == nil {
		if _, updateErr := u.mintRepo.Update(
			ctx,
			*mintEnt,
		); updateErr != nil {
			return nil, fmt.Errorf(
				"mark parent minting: %w",
				updateErr,
			)
		}
	}

	req, err := u.mintRequestPort.LoadForMinting(
		ctx,
		mintRequestID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load mint request for minting: %w",
			err,
		)
	}

	if req == nil {
		return nil, fmt.Errorf(
			"mint request %s is nil",
			mintRequestID,
		)
	}

	reqID := req.ID
	if reqID == "" {
		reqID = mintRequestID
	}

	reqTBID := req.TokenBlueprintID
	if reqTBID == "" {
		reqTBID = tbID
	}

	actorID := req.ActorID
	if actorID == "" {
		actorID = mintEnt.CreatedBy
	}

	if actorID == "" {
		actorID = MemberIDFromContext(ctx)
	}

	metadataURI, err := u.ensureMetadataURI(
		ctx,
		reqTBID,
		actorID,
		req.MetadataURI,
	)
	if err != nil {
		return nil, err
	}

	if metadataURI == "" {
		return nil, fmt.Errorf(
			"mint request %s has empty MetadataURI",
			reqID,
		)
	}

	toAddress := req.ToAddress
	if toAddress == "" {
		return nil, fmt.Errorf(
			"mint request %s has empty ToAddress",
			reqID,
		)
	}

	name := req.BlueprintName
	symbol := req.BlueprintSymbol
	if name == "" || symbol == "" {
		return nil, fmt.Errorf(
			"mint request %s has empty name or symbol",
			reqID,
		)
	}

	task, err := u.mintTaskRepo.GetNextExecutableTask(
		ctx,
		mintRequestID,
	)
	if err != nil {
		if errors.Is(
			err,
			mintdom.ErrMintProductTaskNotFound,
		) {
			return u.finalizeMintIfAllTasksCompleted(
				ctx,
				mintEnt,
				reqTBID,
				actorID,
			)
		}

		return nil, fmt.Errorf(
			"get next executable mint task: %w",
			err,
		)
	}

	task, err = u.mintTaskRepo.MarkMinting(
		ctx,
		mintRequestID,
		task.ProductID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"mark mint product task minting: %w",
			err,
		)
	}

	minted, err := u.tokenMinter.MintProducts(
		ctx,
		MintProductsInput{
			ToAddress:        toAddress,
			ProductIDs:       []string{task.ProductID},
			TokenBlueprintID: reqTBID,
			BrandID:          brandID,
			BlueprintName:    name,
			BlueprintSymbol:  symbol,
			MetadataURI:      metadataURI,
		},
	)
	if err != nil {
		if failErr := u.markTaskFailed(
			ctx,
			mintRequestID,
			task.ProductID,
			err,
		); failErr != nil {
			return nil, fmt.Errorf(
				"mint product failed: %w; also failed to update task: %v",
				err,
				failErr,
			)
		}

		if parentErr := u.markParentFailedRetryable(
			ctx,
			mintEnt,
		); parentErr != nil {
			return nil, fmt.Errorf(
				"mint product failed: %w; also failed to update parent: %v",
				err,
				parentErr,
			)
		}

		return nil, err
	}

	if len(minted) != 1 {
		return nil, fmt.Errorf(
			"expected exactly one minted result, got %d (mintRequestId=%s productId=%s)",
			len(minted),
			mintRequestID,
			task.ProductID,
		)
	}

	mintedOne := minted[0]
	if mintedOne.ProductID == "" {
		mintedOne.ProductID = task.ProductID
	}

	if mintedOne.ProductID != task.ProductID {
		return nil, fmt.Errorf(
			"minted productID mismatch: task=%s minted=%s",
			task.ProductID,
			mintedOne.ProductID,
		)
	}

	if mintedOne.Result == nil {
		return nil, fmt.Errorf(
			"onchain mint succeeded but result is nil (mintRequestId=%s productId=%s)",
			mintRequestID,
			task.ProductID,
		)
	}

	if err := u.recordMintedProduct(
		ctx,
		reqID,
		mintedOne,
	); err != nil {
		return mintedOne.Result, fmt.Errorf(
			"record minted product: %w",
			err,
		)
	}

	if _, err := u.mintTaskRepo.MarkMinted(
		ctx,
		mintRequestID,
		task.ProductID,
		mintedOne.Result.AssetID,
		mintedOne.Result.TreeAddress,
		mintedOne.Result.LeafIndex,
		mintedOne.Result.Signature,
	); err != nil {
		return mintedOne.Result, fmt.Errorf(
			"mark mint product task minted: %w",
			err,
		)
	}

	if u.inventoryUC == nil {
		return mintedOne.Result, errors.New(
			"inventory usecase is nil (cannot upsert inventory)",
		)
	}

	if _, invErr := u.inventoryUC.UpsertFromMint(
		ctx,
		reqTBID,
		pbID,
		[]string{task.ProductID},
	); invErr != nil {
		return mintedOne.Result, invErr
	}

	if err := u.updateParentAndMaybeEnqueueNext(
		ctx,
		mintEnt,
		reqTBID,
		actorID,
		mintedOne.Result.Signature,
	); err != nil {
		return mintedOne.Result, err
	}

	return mintedOne.Result, nil
}
