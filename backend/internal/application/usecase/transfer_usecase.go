// backend/internal/application/usecase/transfer_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	applicationport "narratives/internal/application/port"
	orderdom "narratives/internal/domain/order"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
)

// ============================================================
// Ports and DTOs
// ============================================================

type ModelTokenPair struct {
	ModelID          string `json:"modelId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

type VerifyInput struct {
	AvatarID  string `json:"avatarId"`
	ProductID string `json:"productId"`
}

type VerifyResult struct {
	AvatarID  string `json:"avatarId"`
	ProductID string `json:"productId"`

	ScannedModelID          string `json:"scannedModelId"`
	ScannedTokenBlueprintID string `json:"scannedTokenBlueprintId"`

	PurchasedPairs []ModelTokenPair `json:"purchasedPairs"`

	Matched bool            `json:"matched"`
	Match   *ModelTokenPair `json:"match,omitempty"`
}

type ScanVerifier interface {
	VerifyMatch(
		ctx context.Context,
		in VerifyInput,
	) (VerifyResult, error)
}

type ReturnOpeningHandler interface {
	PromoteUnopenedToOpened(
		ctx context.Context,
		in PromoteUnopenedToOpenedInput,
	) (ReturnRequestResult, error)
}

// ResaleReceivableFulfillmentRepository is the atomic persistence boundary used
// after a resale token transfer has succeeded on-chain.
//
// The implementation must commit the canonical Order item, orderTransferItems
// projection, and SalesReceivable lifecycle in one Firestore transaction.
//
// A receivable may remain pending when other active resale items belonging to
// the same receivable are still untransferred. It becomes available only when
// every active resale item represented by that receivable is transferred.
type ResaleReceivableFulfillmentRepository interface {
	CompleteResaleReceivableFulfillment(
		ctx context.Context,
		orderID string,
		itemIndex int,
		receivable salesreceivabledom.SalesReceivable,
		at time.Time,
	) (salesreceivabledom.SalesReceivable, error)
}

// ============================================================
// Usecase
// ============================================================

type TransferUsecase struct {
	verifier  ScanVerifier
	orderRepo applicationport.OrderRepoForTransfer
	tokenRepo applicationport.TokenResolver

	brandWallet  applicationport.BrandWalletResolver
	avatarWallet applicationport.AvatarWalletResolver

	brandDisplay  applicationport.BrandGetter
	avatarDisplay applicationport.AvatarDisplayResolver

	resaleRepo applicationport.ResaleGetter

	salesReceivableUC *SalesReceivableUsecase

	returnOpening ReturnOpeningHandler

	executionUC *TokenTransferExecutionUsecase
	inventoryUC *InventoryUsecase

	now func() time.Time
}

func NewTransferUsecase(
	verifier ScanVerifier,
	orderRepo applicationport.OrderRepoForTransfer,
	tokenRepo applicationport.TokenResolver,
	brandWallet applicationport.BrandWalletResolver,
	avatarWallet applicationport.AvatarWalletResolver,
	brandDisplay applicationport.BrandGetter,
	avatarDisplay applicationport.AvatarDisplayResolver,
	executionUC *TokenTransferExecutionUsecase,
	inventoryUC *InventoryUsecase,
) *TransferUsecase {
	return &TransferUsecase{
		verifier:  verifier,
		orderRepo: orderRepo,
		tokenRepo: tokenRepo,

		brandWallet:  brandWallet,
		avatarWallet: avatarWallet,

		brandDisplay:  brandDisplay,
		avatarDisplay: avatarDisplay,

		executionUC: executionUC,
		inventoryUC: inventoryUC,

		now: time.Now,
	}
}

func (u *TransferUsecase) WithResaleTransferDependencies(
	resaleRepo applicationport.ResaleGetter,
) *TransferUsecase {
	if u != nil {
		u.resaleRepo = resaleRepo
	}

	return u
}

func (u *TransferUsecase) WithResaleReceivableDependencies(
	salesReceivableUC *SalesReceivableUsecase,
) *TransferUsecase {
	if u != nil {
		u.salesReceivableUC = salesReceivableUC
	}

	return u
}

func (u *TransferUsecase) WithReturnOpeningHandler(
	returnOpening ReturnOpeningHandler,
) *TransferUsecase {
	if u != nil {
		u.returnOpening = returnOpening
	}

	return u
}

var (
	ErrTransferNotConfigured          = errors.New("transfer_uc: not configured")
	ErrTransferAvatarIDEmpty          = errors.New("transfer_uc: avatarId is empty")
	ErrTransferProductIDEmpty         = errors.New("transfer_uc: productId is empty")
	ErrTransferOperationIDEmpty       = errors.New("transfer_uc: operationId is empty")
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
	ErrTransferSameAvatar                    = errors.New("transfer_uc: seller avatarId and buyer avatarId must be different")
	ErrTransferWalletSyncFailed              = errors.New("transfer_uc: wallet sync failed")
)

type TransferByVerifiedScanInput struct {
	AvatarID    string
	ProductID   string
	OperationID string
}

type TransferByVerifiedScanResult struct {
	MatchedOrderID     string
	MatchedInventoryID string
	MatchedModelID     string

	MatchedItemIndex int
	MatchedItemType  orderdom.OrderItemType
	MatchedResaleID  string

	ProductID        string
	AssetID          string
	TokenBlueprintID string

	FromWallet  string
	ToWallet    string
	TxSignature string

	FromDisplayName string
	ToDisplayName   string
}

type transferExecutionSource struct {
	FromAvatarID string
	FromBrandID  string

	FromWallet string
}

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
			// A return request may have started after the first verified-scan
			// transaction but before the transfer lock.
			//
			// Record the verified scan again against the latest Order state so
			// tokenTransferVerifiedAt cannot be lost by that competing return
			// update.
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
		resaleReceivable, err = u.requirePendingResaleReceivable(
			ctx,
			target.OrderID,
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
			fulfillmentRepo, ok := u.orderRepo.(ResaleReceivableFulfillmentRepository)
			if !ok {
				return ErrTransferResaleReceivableNotConfigured
			}

			completedReceivable, err := fulfillmentRepo.CompleteResaleReceivableFulfillment(
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

// ============================================================
// Transfer source helpers
// ============================================================

func (u *TransferUsecase) resolveTransferSource(
	ctx context.Context,
	target applicationport.TransferTargetItem,
	brandID string,
	buyerAvatarID string,
) (transferExecutionSource, error) {
	switch target.ItemType {
	case orderdom.OrderItemTypeList:
		return u.resolveListTransferSource(
			ctx,
			brandID,
		)

	case orderdom.OrderItemTypeResale:
		return u.resolveResaleTransferSource(
			ctx,
			target,
			buyerAvatarID,
		)

	default:
		return transferExecutionSource{},
			ErrTransferNoEligibleOrder
	}
}

func (u *TransferUsecase) resolveListTransferSource(
	ctx context.Context,
	brandID string,
) (transferExecutionSource, error) {
	if brandID == "" {
		return transferExecutionSource{},
			ErrTransferBrandIDEmpty
	}

	fromWallet, err :=
		u.brandWallet.ResolveBrandWalletAddress(
			ctx,
			brandID,
		)
	if err != nil {
		return transferExecutionSource{},
			fmt.Errorf(
				"transfer_uc: resolve brand wallet failed brandId=%s: %w",
				brandID,
				err,
			)
	}
	if fromWallet == "" {
		return transferExecutionSource{},
			ErrTransferFromWalletEmpty
	}

	return transferExecutionSource{
		FromBrandID: brandID,
		FromWallet:  fromWallet,
	}, nil
}

func (u *TransferUsecase) resolveResaleTransferSource(
	ctx context.Context,
	target applicationport.TransferTargetItem,
	buyerAvatarID string,
) (transferExecutionSource, error) {
	if u.resaleRepo == nil {
		return transferExecutionSource{},
			ErrTransferResaleNotConfigured
	}

	resaleID := target.ResaleID
	if resaleID == "" {
		return transferExecutionSource{},
			ErrTransferResaleIDEmpty
	}

	resale, err := u.resaleRepo.GetByID(
		ctx,
		resaleID,
	)
	if err != nil {
		return transferExecutionSource{},
			fmt.Errorf(
				"transfer_uc: resolve resale failed resaleId=%s: %w",
				resaleID,
				err,
			)
	}

	fromAvatarID := resale.AvatarID
	if fromAvatarID == "" {
		return transferExecutionSource{},
			ErrTransferResaleSellerAvatarIDEmpty
	}
	if fromAvatarID == buyerAvatarID {
		return transferExecutionSource{},
			ErrTransferSameAvatar
	}

	fromWallet, err :=
		u.avatarWallet.ResolveAvatarWalletAddress(
			ctx,
			fromAvatarID,
		)
	if err != nil {
		return transferExecutionSource{},
			fmt.Errorf(
				"transfer_uc: resolve seller avatar wallet failed avatarId=%s: %w",
				fromAvatarID,
				err,
			)
	}
	if fromWallet == "" {
		return transferExecutionSource{},
			ErrTransferFromWalletEmpty
	}

	return transferExecutionSource{
		FromAvatarID: fromAvatarID,
		FromWallet:   fromWallet,
	}, nil
}

// ============================================================
// Resale receivable helpers
// ============================================================

func (u *TransferUsecase) requirePendingResaleReceivable(
	ctx context.Context,
	paymentID string,
	item orderdom.OrderItemSnapshot,
	expectedAvatarID string,
) (salesreceivabledom.SalesReceivable, error) {
	if u == nil || u.salesReceivableUC == nil {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableNotConfigured
	}
	if paymentID == "" ||
		item.Type != orderdom.OrderItemTypeResale ||
		expectedAvatarID == "" {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}

	snapshot := item.SellerSnapshot
	if snapshot.AvatarID == "" ||
		snapshot.UserID == "" ||
		snapshot.PayoutAccountID == "" ||
		snapshot.PayoutAccountID != snapshot.UserID ||
		snapshot.AvatarID != expectedAvatarID ||
		snapshot.BrandID != "" ||
		snapshot.CompanyID != "" ||
		snapshot.AccountID != "" ||
		snapshot.StripeAccountID != "" {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}

	receivableID, err := salesreceivabledom.NewID(
		paymentID,
		snapshot.PayoutAccountID,
	)
	if err != nil {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}

	receivable, err := u.salesReceivableUC.GetByID(
		ctx,
		receivableID,
	)
	if err != nil {
		return salesreceivabledom.SalesReceivable{},
			fmt.Errorf(
				"%w: load receivable %s: %v",
				ErrTransferResaleReceivableUnavailable,
				receivableID,
				err,
			)
	}
	if receivable == nil {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableUnavailable
	}
	if receivable.ID != receivableID ||
		receivable.PaymentID != paymentID ||
		receivable.OrderID != paymentID ||
		receivable.AvatarID != snapshot.AvatarID ||
		receivable.UserID != snapshot.UserID ||
		receivable.PayoutAccountID != snapshot.PayoutAccountID {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}
	if err := receivable.Validate(); err != nil {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableMismatch
	}
	if receivable.Status != salesreceivabledom.StatusPending {
		return salesreceivabledom.SalesReceivable{},
			ErrTransferResaleReceivableUnavailable
	}

	return *receivable, nil
}

func validateCompletedResaleReceivable(
	actual salesreceivabledom.SalesReceivable,
	expected salesreceivabledom.SalesReceivable,
) error {
	if err := validateExistingSalesReceivableAllocation(
		actual,
		expected,
	); err != nil {
		return ErrTransferResaleReceivableMismatch
	}

	switch actual.Status {
	case salesreceivabledom.StatusPending,
		salesreceivabledom.StatusAvailable:
		return nil
	default:
		return ErrTransferResaleReceivableUnavailable
	}
}

// ============================================================
// Display helpers
// ============================================================

func (u *TransferUsecase) resolveBrandDisplayName(
	ctx context.Context,
	brandID string,
) string {
	if u == nil ||
		u.brandDisplay == nil ||
		brandID == "" {
		return ""
	}

	brand, err := u.brandDisplay.GetByID(
		ctx,
		brandID,
	)
	if err != nil {
		return ""
	}

	return brand.Name
}

func (u *TransferUsecase) resolveAvatarDisplayName(
	ctx context.Context,
	avatarID string,
) string {
	if u == nil ||
		u.avatarDisplay == nil ||
		avatarID == "" {
		return ""
	}

	avatar, err := u.avatarDisplay.GetByID(
		ctx,
		avatarID,
	)
	if err != nil {
		return ""
	}

	return avatar.AvatarName
}
