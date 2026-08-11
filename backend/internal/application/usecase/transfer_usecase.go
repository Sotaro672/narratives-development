// backend/internal/application/usecase/transfer_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	orderdom "narratives/internal/domain/order"
	resaledom "narratives/internal/domain/resale"
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

type FindEligibleTransferItemInput struct {
	AvatarID         string
	ProductID        string
	ModelID          string
	TokenBlueprintID string
}

type TransferTargetItem struct {
	OrderID   string
	ItemIndex int
	ItemType  orderdom.OrderItemType

	InventoryID string
	ModelID     string
	ResaleID    string

	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

// OrderRepoForTransfer provides exact lookup and state transitions for one
// transfer-eligible order item. Implementations must use the canonical
// orderTransferItems read model and must not scan Order documents in memory.
type OrderRepoForTransfer interface {
	// FindEligibleTransferItem returns orderdom.ErrNotFound when no paid,
	// untransferred item exactly matches the input.
	FindEligibleTransferItem(
		ctx context.Context,
		in FindEligibleTransferItemInput,
	) (TransferTargetItem, error)

	LockTransferItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
		now time.Time,
	) error

	UnlockTransferItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
	) error

	MarkTransferredItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
		at time.Time,
	) error
}

type TokenResolver interface {
	ResolveTokenByProductID(
		ctx context.Context,
		productID string,
	) (TokenForTransfer, error)
}

type TokenForTransfer struct {
	ProductID string

	BrandID string

	AssetID string

	TokenBlueprintID string
}

type TokenOwnerUpdater interface {
	UpdateToAddressByProductID(
		ctx context.Context,
		productID string,
		newToAddress string,
		now time.Time,
		txSignature string,
	) error
}

type BrandWalletResolver interface {
	ResolveBrandWalletAddress(
		ctx context.Context,
		brandID string,
	) (string, error)
}

type AvatarWalletResolver interface {
	ResolveAvatarWalletAddress(
		ctx context.Context,
		avatarID string,
	) (string, error)
}

type BrandDisplayResolver interface {
	GetByID(
		ctx context.Context,
		id string,
	) (branddom.Brand, error)
}

type AvatarDisplayResolver interface {
	GetByID(
		ctx context.Context,
		id string,
	) (avatardom.Avatar, error)
}

type ResaleReaderForTransfer interface {
	GetByID(
		ctx context.Context,
		id string,
	) (resaledom.Resale, error)
}

type TokenTransferExecutor interface {
	ExecuteTransfer(
		ctx context.Context,
		in ExecuteTransferInput,
	) (ExecuteTransferResult, error)
}

type ExecuteTransferInput struct {
	ProductID        string
	FromAvatarID     string
	ToAvatarID       string
	FromBrandID      string
	BrandID          string
	ModelID          string
	TokenBlueprintID string

	AssetID string

	FromWalletAddress string
	ToWalletAddress   string
}

type ExecuteTransferResult struct {
	TxSignature string
}

type PostTransferResolveWarmer interface {
	ResolveAfterTransfer(
		ctx context.Context,
		avatarID string,
		assetID string,
	) error
}

// ============================================================
// Usecase
// ============================================================

type TransferUsecase struct {
	verifier  ScanVerifier
	orderRepo OrderRepoForTransfer
	tokenRepo TokenResolver

	brandWallet  BrandWalletResolver
	avatarWallet AvatarWalletResolver

	brandDisplay  BrandDisplayResolver
	avatarDisplay AvatarDisplayResolver

	resaleRepo ResaleReaderForTransfer

	executionUC *TokenTransferExecutionUsecase
	inventoryUC *InventoryUsecase

	now func() time.Time
}

func NewTransferUsecase(
	verifier ScanVerifier,
	orderRepo OrderRepoForTransfer,
	tokenRepo TokenResolver,
	brandWallet BrandWalletResolver,
	avatarWallet AvatarWalletResolver,
	brandDisplay BrandDisplayResolver,
	avatarDisplay AvatarDisplayResolver,
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
	resaleRepo ResaleReaderForTransfer,
) *TransferUsecase {
	if u != nil {
		u.resaleRepo = resaleRepo
	}

	return u
}

var (
	ErrTransferNotConfigured          = errors.New("transfer_uc: not configured")
	ErrTransferAvatarIDEmpty          = errors.New("transfer_uc: avatarId is empty")
	ErrTransferProductIDEmpty         = errors.New("transfer_uc: productId is empty")
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

	ErrTransferResaleNotConfigured       = errors.New("transfer_uc: resale transfer dependencies are not configured")
	ErrTransferResaleIDEmpty             = errors.New("transfer_uc: resaleId is empty")
	ErrTransferResaleSellerAvatarIDEmpty = errors.New("transfer_uc: resale seller avatarId is empty")
	ErrTransferSameAvatar                = errors.New("transfer_uc: seller avatarId and buyer avatarId must be different")
	ErrTransferWalletSyncFailed          = errors.New("transfer_uc: wallet sync failed")
)

type TransferByVerifiedScanInput struct {
	AvatarID  string
	ProductID string
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

	if avatarID == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferAvatarIDEmpty
	}
	if productID == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferProductIDEmpty
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
		FindEligibleTransferItemInput{
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

	lockAt := u.now().UTC()

	if err := u.orderRepo.LockTransferItem(
		ctx,
		target.OrderID,
		target.ItemIndex,
		lockAt,
	); err != nil {
		return TransferByVerifiedScanResult{},
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
		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: resolve receiver avatar wallet failed avatarId=%s: %w",
				avatarID,
				err,
			)
	}
	if toWallet == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferToWalletEmpty
	}

	source, err := u.resolveTransferSource(
		ctx,
		target,
		brandID,
		avatarID,
	)
	if err != nil {
		return TransferByVerifiedScanResult{}, err
	}

	removeFromSenderWallet :=
		target.ItemType == orderdom.OrderItemTypeResale
	syncSenderWallet :=
		target.ItemType == orderdom.OrderItemTypeResale
	syncReceiverWallet :=
		target.ItemType == orderdom.OrderItemTypeResale

	afterOnChain := func(
		ctx context.Context,
		txSignature string,
		now time.Time,
	) error {
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
			ProductID: productID,

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
		return TransferByVerifiedScanResult{},
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
	target TransferTargetItem,
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
	target TransferTargetItem,
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
