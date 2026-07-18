// backend/internal/application/usecase/transfer_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	orderdom "narratives/internal/domain/order"
	resaledom "narratives/internal/domain/resale"
	transferdom "narratives/internal/domain/transfer"
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

// OrderRepoForTransfer provides the order operations required by transfer.
//
// itemKey:
//   - list:<modelId>
//   - resale:<resaleId>
type OrderRepoForTransfer interface {
	ListPaidByAvatarID(
		ctx context.Context,
		avatarID string,
	) ([]orderdom.Order, error)

	LockTransferItem(
		ctx context.Context,
		orderID string,
		itemKey string,
		now time.Time,
	) error

	UnlockTransferItem(
		ctx context.Context,
		orderID string,
		itemKey string,
	) error

	MarkTransferredItem(
		ctx context.Context,
		orderID string,
		itemKey string,
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

	MintAddress string

	TokenBlueprintID string
	ToAddress        string
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

type WalletItemUpdater interface {
	AddMintToAvatarWalletItems(
		ctx context.Context,
		avatarID string,
		mintAddress string,
		now time.Time,
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

type WalletSecretProvider interface {
	GetBrandSigner(
		ctx context.Context,
		brandID string,
	) (any, error)
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
	AvatarID         string
	BrandID          string
	ModelID          string
	TokenBlueprintID string

	MintAddress string
	Amount      uint64

	FromWalletAddress string
	ToWalletAddress   string

	FromSigner any
	ToSigner   any
}

type ExecuteTransferResult struct {
	TxSignature string
}

type PostTransferResolveWarmer interface {
	ResolveAfterTransfer(
		ctx context.Context,
		avatarID string,
		mintAddress string,
	) error
}

// ============================================================
// Usecase
// ============================================================

type TransferUsecase struct {
	verifier     ScanVerifier
	orderRepo    OrderRepoForTransfer
	tokenRepo    TokenResolver
	tokenUpdate  TokenOwnerUpdater
	walletUpdate WalletItemUpdater

	// Transferの永続化契約はDomain RepositoryPortへ統一する。
	transferRepo transferdom.RepositoryPort

	brandWallet  BrandWalletResolver
	avatarWallet AvatarWalletResolver

	brandDisplay  BrandDisplayResolver
	avatarDisplay AvatarDisplayResolver

	secrets  WalletSecretProvider
	executor TokenTransferExecutor

	resaleRepo           ResaleReaderForTransfer
	avatarSecrets        AvatarSecretProvider
	walletTransferUpdate AvatarWalletItemTransferUpdater
	walletSync           AvatarWalletSyncer

	resolveWarmer PostTransferResolveWarmer

	inventoryUC *InventoryUsecase

	now func() time.Time
}

func NewTransferUsecase(
	verifier ScanVerifier,
	orderRepo OrderRepoForTransfer,
	tokenRepo TokenResolver,
	tokenUpdate TokenOwnerUpdater,
	walletUpdate WalletItemUpdater,
	transferRepo transferdom.RepositoryPort,
	brandWallet BrandWalletResolver,
	avatarWallet AvatarWalletResolver,
	brandDisplay BrandDisplayResolver,
	avatarDisplay AvatarDisplayResolver,
	secrets WalletSecretProvider,
	executor TokenTransferExecutor,
	resolveWarmer PostTransferResolveWarmer,
	inventoryUC *InventoryUsecase,
) *TransferUsecase {
	return &TransferUsecase{
		verifier:     verifier,
		orderRepo:    orderRepo,
		tokenRepo:    tokenRepo,
		tokenUpdate:  tokenUpdate,
		walletUpdate: walletUpdate,
		transferRepo: transferRepo,

		brandWallet:  brandWallet,
		avatarWallet: avatarWallet,

		brandDisplay:  brandDisplay,
		avatarDisplay: avatarDisplay,

		secrets:  secrets,
		executor: executor,

		resolveWarmer: resolveWarmer,
		inventoryUC:   inventoryUC,

		now: time.Now,
	}
}

func (u *TransferUsecase) WithResaleTransferDependencies(
	resaleRepo ResaleReaderForTransfer,
	avatarSecrets AvatarSecretProvider,
	walletTransferUpdate AvatarWalletItemTransferUpdater,
	walletSync AvatarWalletSyncer,
) *TransferUsecase {
	if u != nil {
		u.resaleRepo = resaleRepo
		u.avatarSecrets = avatarSecrets
		u.walletTransferUpdate = walletTransferUpdate
		u.walletSync = walletSync
	}

	return u
}

var (
	ErrTransferNotConfigured          = errors.New("transfer_uc: not configured")
	ErrTransferAvatarIDEmpty          = errors.New("transfer_uc: avatarId is empty")
	ErrTransferProductIDEmpty         = errors.New("transfer_uc: productId is empty")
	ErrTransferNotMatched             = errors.New("transfer_uc: scan is not matched")
	ErrTransferNoEligibleOrder        = errors.New("transfer_uc: no eligible order/item found")
	ErrTransferMintEmpty              = errors.New("transfer_uc: mintAddress is empty")
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

	MatchedItemKey  string
	MatchedItemType orderdom.OrderItemType
	MatchedResaleID string

	ProductID        string
	MintAddress      string
	TokenBlueprintID string

	FromWallet  string
	ToWallet    string
	TxSignature string

	FromDisplayName string
	ToDisplayName   string
}

type transferTargetItem struct {
	OrderID string

	ItemKey  string
	ItemType orderdom.OrderItemType

	InventoryID string
	ModelID     string

	ResaleID string

	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

type transferExecutionSource struct {
	FromAvatarID string
	FromBrandID  string

	FromWallet string
	FromSigner any
}

// TransferToAvatarByVerifiedScan verifies the scan and transfers the token to
// the authenticated avatar.
func (u *TransferUsecase) TransferToAvatarByVerifiedScan(
	ctx context.Context,
	in TransferByVerifiedScanInput,
) (
	result TransferByVerifiedScanResult,
	retErr error,
) {
	if u == nil ||
		u.verifier == nil ||
		u.orderRepo == nil ||
		u.tokenRepo == nil ||
		u.tokenUpdate == nil ||
		u.walletUpdate == nil ||
		u.transferRepo == nil ||
		u.brandWallet == nil ||
		u.avatarWallet == nil ||
		u.secrets == nil ||
		u.executor == nil ||
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
	mintAddress := token.MintAddress
	tokenBlueprintID := token.TokenBlueprintID
	currentOwner := token.ToAddress

	if brandID == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferBrandIDEmpty
	}
	if mintAddress == "" {
		return TransferByVerifiedScanResult{},
			ErrTransferMintEmpty
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

	orders, err := u.orderRepo.ListPaidByAvatarID(
		ctx,
		avatarID,
	)
	if err != nil {
		return TransferByVerifiedScanResult{}, err
	}
	if len(orders) == 0 {
		return TransferByVerifiedScanResult{},
			ErrTransferNoEligibleOrder
	}

	target, found := findUntransferredTransferTarget(
		orders,
		productID,
		scannedModelID,
		scannedTokenBlueprintID,
	)
	if !found {
		return TransferByVerifiedScanResult{},
			ErrTransferNoEligibleOrder
	}

	now := u.now().UTC()

	if err := u.orderRepo.LockTransferItem(
		ctx,
		target.OrderID,
		target.ItemKey,
		now,
	); err != nil {
		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: lock failed orderId=%s itemKey=%s: %w",
				target.OrderID,
				target.ItemKey,
				err,
			)
	}

	locked := true
	transferAttempt := 0
	transferCreated := false
	transferFailed := false

	patchTransfer := func(
		patch transferdom.TransferPatch,
	) error {
		if !transferCreated {
			return nil
		}

		_, err := u.transferRepo.Patch(
			context.Background(),
			productID,
			transferAttempt,
			patch,
			nil,
		)
		return err
	}

	defer func() {
		if retErr != nil &&
			transferCreated &&
			!transferFailed {
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
		}

		if locked {
			_ = u.orderRepo.UnlockTransferItem(
				context.Background(),
				target.OrderID,
				target.ItemKey,
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

	createdTransfer, err := u.transferRepo.CreateAttempt(
		ctx,
		transferdom.CreateAttemptInput{
			ProductID:       productID,
			OrderID:         target.OrderID,
			AvatarID:        avatarID,
			ToWalletAddress: toWallet,
			MintAddress:     mintAddress,
			CreatedAt:       now,
		},
	)
	if err != nil {
		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: create transfer attempt failed productId=%s: %w",
				productID,
				err,
			)
	}
	if createdTransfer == nil ||
		createdTransfer.Attempt <= 0 {
		return TransferByVerifiedScanResult{},
			ErrTransferAttemptNotCreated
	}

	transferAttempt = createdTransfer.Attempt
	transferCreated = true

	markFailed := func(
		errorType transferdom.ErrorType,
		message string,
		txSignature *string,
	) {
		if !transferCreated {
			return
		}

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

		_ = patchTransfer(patch)
		transferFailed = true
	}

	markSucceeded := func(
		txSignature string,
	) error {
		if !transferCreated {
			return ErrTransferAttemptNotCreated
		}
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

	if currentOwner != "" &&
		currentOwner != source.FromWallet {
		message := fmt.Sprintf(
			"productId=%s tokenOwner=%s expectedFromWallet=%s itemType=%s",
			productID,
			currentOwner,
			source.FromWallet,
			target.ItemType,
		)

		markFailed(
			transferdom.ErrorTypeMismatch,
			message,
			nil,
		)

		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"%w: %s",
				ErrTransferOwnerMismatch,
				message,
			)
	}

	executeResult, err := u.executor.ExecuteTransfer(
		ctx,
		ExecuteTransferInput{
			ProductID:        productID,
			AvatarID:         avatarID,
			BrandID:          brandID,
			ModelID:          target.ModelID,
			TokenBlueprintID: scannedTokenBlueprintID,

			MintAddress: mintAddress,
			Amount:      1,

			FromWalletAddress: source.FromWallet,
			ToWalletAddress:   toWallet,

			FromSigner: source.FromSigner,
			ToSigner:   nil,
		},
	)
	if err != nil {
		message := fmt.Sprintf(
			"execute transfer failed orderId=%s itemKey=%s mint=%s: %v",
			target.OrderID,
			target.ItemKey,
			mintAddress,
			err,
		)

		markFailed(
			transferdom.ErrorTypeTransferFailed,
			message,
			nil,
		)

		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: execute transfer failed orderId=%s itemKey=%s mint=%s: %w",
				target.OrderID,
				target.ItemKey,
				mintAddress,
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

		return TransferByVerifiedScanResult{},
			transferdom.ErrEmptyTxSignature
	}

	if err := markSucceeded(txSignature); err != nil {
		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: mark transfer succeeded failed productId=%s attempt=%d tx=%s: %w",
				productID,
				transferAttempt,
				txSignature,
				err,
			)
	}

	if err := u.orderRepo.MarkTransferredItem(
		ctx,
		target.OrderID,
		target.ItemKey,
		now,
	); err != nil {
		message := fmt.Sprintf(
			"mark transferred failed orderId=%s itemKey=%s tx=%s: %v",
			target.OrderID,
			target.ItemKey,
			txSignature,
			err,
		)

		markFailed(
			transferdom.ErrorTypeUnknown,
			message,
			&txSignature,
		)

		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: mark transferred failed orderId=%s itemKey=%s tx=%s: %w",
				target.OrderID,
				target.ItemKey,
				txSignature,
				err,
			)
	}

	if err := u.tokenUpdate.UpdateToAddressByProductID(
		ctx,
		productID,
		toWallet,
		now,
		txSignature,
	); err != nil {
		message := fmt.Sprintf(
			"update token owner failed productId=%s to=%s tx=%s: %v",
			productID,
			toWallet,
			txSignature,
			err,
		)

		markFailed(
			transferdom.ErrorTypeUnknown,
			message,
			&txSignature,
		)

		return TransferByVerifiedScanResult{},
			fmt.Errorf(
				"transfer_uc: update token owner failed productId=%s to=%s tx=%s: %w",
				productID,
				toWallet,
				txSignature,
				err,
			)
	}

	if err := u.updateWalletsAfterTransfer(
		ctx,
		target,
		source.FromAvatarID,
		avatarID,
		mintAddress,
		now,
		txSignature,
	); err != nil {
		message := err.Error()

		markFailed(
			transferdom.ErrorTypeUnknown,
			message,
			&txSignature,
		)

		return TransferByVerifiedScanResult{}, err
	}

	if u.resolveWarmer != nil {
		if err := u.resolveWarmer.ResolveAfterTransfer(
			ctx,
			avatarID,
			mintAddress,
		); err != nil {
			message := fmt.Sprintf(
				"post-transfer resolve failed avatarId=%s mint=%s tx=%s: %v",
				avatarID,
				mintAddress,
				txSignature,
				err,
			)

			markFailed(
				transferdom.ErrorTypeUnknown,
				message,
				&txSignature,
			)

			return TransferByVerifiedScanResult{},
				fmt.Errorf(
					"%w: %s",
					ErrTransferResolveAfterFailed,
					message,
				)
		}
	}

	if target.ItemType == orderdom.OrderItemTypeList &&
		u.inventoryUC != nil {
		if err := u.inventoryUC.ReleaseAfterTransfer(
			ctx,
			target.InventoryID,
			target.ModelID,
			productID,
			target.OrderID,
			now,
		); err != nil {
			message := fmt.Sprintf(
				"inventory cleanup failed inventoryId=%s modelId=%s productId=%s orderId=%s tx=%s: %v",
				target.InventoryID,
				target.ModelID,
				productID,
				target.OrderID,
				txSignature,
				err,
			)

			markFailed(
				transferdom.ErrorTypeUnknown,
				message,
				&txSignature,
			)

			return TransferByVerifiedScanResult{},
				fmt.Errorf(
					"%w: %s",
					ErrTransferInventoryCleanupFailed,
					message,
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

		MatchedItemKey:  target.ItemKey,
		MatchedItemType: target.ItemType,
		MatchedResaleID: target.ResaleID,

		ProductID:        productID,
		MintAddress:      mintAddress,
		TokenBlueprintID: scannedTokenBlueprintID,

		FromWallet:  source.FromWallet,
		ToWallet:    toWallet,
		TxSignature: txSignature,

		FromDisplayName: fromDisplayName,
		ToDisplayName:   toDisplayName,
	}, nil
}

// ============================================================
// Transfer source helpers
// ============================================================

func (u *TransferUsecase) resolveTransferSource(
	ctx context.Context,
	target transferTargetItem,
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

	fromSigner, err := u.secrets.GetBrandSigner(
		ctx,
		brandID,
	)
	if err != nil {
		return transferExecutionSource{},
			fmt.Errorf(
				"transfer_uc: get brand signer failed brandId=%s wallet=%s: %w",
				brandID,
				fromWallet,
				err,
			)
	}

	return transferExecutionSource{
		FromBrandID: brandID,
		FromWallet:  fromWallet,
		FromSigner:  fromSigner,
	}, nil
}

func (u *TransferUsecase) resolveResaleTransferSource(
	ctx context.Context,
	target transferTargetItem,
	buyerAvatarID string,
) (transferExecutionSource, error) {
	if u.resaleRepo == nil ||
		u.avatarSecrets == nil ||
		u.walletTransferUpdate == nil ||
		u.walletSync == nil {
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

	fromSigner, err :=
		u.avatarSecrets.GetAvatarSigner(
			ctx,
			fromAvatarID,
		)
	if err != nil {
		return transferExecutionSource{},
			fmt.Errorf(
				"transfer_uc: get seller avatar signer failed avatarId=%s wallet=%s: %w",
				fromAvatarID,
				fromWallet,
				err,
			)
	}

	return transferExecutionSource{
		FromAvatarID: fromAvatarID,
		FromWallet:   fromWallet,
		FromSigner:   fromSigner,
	}, nil
}

// ============================================================
// Wallet update helpers
// ============================================================

func (u *TransferUsecase) updateWalletsAfterTransfer(
	ctx context.Context,
	target transferTargetItem,
	fromAvatarID string,
	toAvatarID string,
	mintAddress string,
	now time.Time,
	txSignature string,
) error {
	switch target.ItemType {
	case orderdom.OrderItemTypeList:
		if err := u.walletUpdate.AddMintToAvatarWalletItems(
			ctx,
			toAvatarID,
			mintAddress,
			now,
		); err != nil {
			return fmt.Errorf(
				"transfer_uc: update receiver wallet items failed avatarId=%s mint=%s tx=%s: %w",
				toAvatarID,
				mintAddress,
				txSignature,
				err,
			)
		}

		return nil

	case orderdom.OrderItemTypeResale:
		return u.updateResaleWalletsAfterTransfer(
			ctx,
			fromAvatarID,
			toAvatarID,
			mintAddress,
			now,
			txSignature,
		)

	default:
		return ErrTransferNoEligibleOrder
	}
}

func (u *TransferUsecase) updateResaleWalletsAfterTransfer(
	ctx context.Context,
	fromAvatarID string,
	toAvatarID string,
	mintAddress string,
	now time.Time,
	txSignature string,
) error {
	if u.walletTransferUpdate == nil ||
		u.walletSync == nil {
		return ErrTransferResaleNotConfigured
	}
	if fromAvatarID == "" {
		return ErrTransferResaleSellerAvatarIDEmpty
	}
	if toAvatarID == "" {
		return ErrTransferAvatarIDEmpty
	}
	if mintAddress == "" {
		return ErrTransferMintEmpty
	}

	if err := u.walletTransferUpdate.RemoveMintFromAvatarWalletItems(
		ctx,
		fromAvatarID,
		mintAddress,
		now,
	); err != nil {
		return fmt.Errorf(
			"transfer_uc: remove seller wallet item failed avatarId=%s mint=%s tx=%s: %w",
			fromAvatarID,
			mintAddress,
			txSignature,
			err,
		)
	}

	if err := u.walletTransferUpdate.AddMintToAvatarWalletItems(
		ctx,
		toAvatarID,
		mintAddress,
		now,
	); err != nil {
		return fmt.Errorf(
			"transfer_uc: add buyer wallet item failed avatarId=%s mint=%s tx=%s: %w",
			toAvatarID,
			mintAddress,
			txSignature,
			err,
		)
	}

	if _, err := u.walletSync.SyncWalletTokens(
		ctx,
		fromAvatarID,
	); err != nil {
		return fmt.Errorf(
			"%w: sync seller wallet failed avatarId=%s mint=%s tx=%s: %v",
			ErrTransferWalletSyncFailed,
			fromAvatarID,
			mintAddress,
			txSignature,
			err,
		)
	}

	if _, err := u.walletSync.SyncWalletTokens(
		ctx,
		toAvatarID,
	); err != nil {
		return fmt.Errorf(
			"%w: sync buyer wallet failed avatarId=%s mint=%s tx=%s: %v",
			ErrTransferWalletSyncFailed,
			toAvatarID,
			mintAddress,
			txSignature,
			err,
		)
	}

	return nil
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

// ============================================================
// Order item matching helpers
// ============================================================

func findUntransferredTransferTarget(
	orders []orderdom.Order,
	productID string,
	scannedModelID string,
	scannedTokenBlueprintID string,
) (transferTargetItem, bool) {
	if productID == "" ||
		scannedTokenBlueprintID == "" {
		return transferTargetItem{}, false
	}

	for _, order := range orders {
		if !order.Paid {
			continue
		}

		if target, found := findUntransferredResaleItem(
			order,
			productID,
			scannedTokenBlueprintID,
		); found {
			return target, true
		}

		if target, found := findUntransferredListItem(
			order,
			scannedModelID,
			scannedTokenBlueprintID,
		); found {
			return target, true
		}
	}

	return transferTargetItem{}, false
}

func findUntransferredResaleItem(
	order orderdom.Order,
	productID string,
	scannedTokenBlueprintID string,
) (transferTargetItem, bool) {
	if productID == "" ||
		scannedTokenBlueprintID == "" {
		return transferTargetItem{}, false
	}

	for _, item := range order.Items {
		// Order domainのTypeを正として使用し、項目内容から推測しない。
		if item.Type != orderdom.OrderItemTypeResale {
			continue
		}
		if item.Transferred {
			continue
		}
		if item.ResaleID == "" {
			continue
		}
		if item.ProductID != productID {
			continue
		}
		if item.TokenBlueprintID == "" {
			continue
		}
		if item.TokenBlueprintID !=
			scannedTokenBlueprintID {
			continue
		}

		itemKey := buildTransferItemKey(
			item.Type,
			item,
		)
		if itemKey == "" {
			continue
		}

		return transferTargetItem{
			OrderID: order.ID,

			ItemKey:  itemKey,
			ItemType: item.Type,

			ResaleID: item.ResaleID,

			ProductID:          item.ProductID,
			ProductBlueprintID: item.ProductBlueprintID,
			TokenBlueprintID:   item.TokenBlueprintID,
			BrandID:            item.BrandID,
		}, true
	}

	return transferTargetItem{}, false
}

func findUntransferredListItem(
	order orderdom.Order,
	scannedModelID string,
	scannedTokenBlueprintID string,
) (transferTargetItem, bool) {
	if scannedModelID == "" ||
		scannedTokenBlueprintID == "" {
		return transferTargetItem{}, false
	}

	for _, item := range order.Items {
		// Order domainのTypeを正として使用し、項目内容から推測しない。
		if item.Type != orderdom.OrderItemTypeList {
			continue
		}
		if item.ModelID != scannedModelID {
			continue
		}
		if item.InventoryID == "" {
			continue
		}
		if item.Transferred {
			continue
		}

		itemTokenBlueprintID :=
			extractTokenBlueprintIDFromInventoryID(
				item.InventoryID,
			)
		if itemTokenBlueprintID == "" {
			continue
		}
		if itemTokenBlueprintID !=
			scannedTokenBlueprintID {
			continue
		}

		itemKey := buildTransferItemKey(
			item.Type,
			item,
		)
		if itemKey == "" {
			continue
		}

		return transferTargetItem{
			OrderID: order.ID,

			ItemKey:  itemKey,
			ItemType: item.Type,

			InventoryID: item.InventoryID,
			ModelID:     item.ModelID,

			TokenBlueprintID: itemTokenBlueprintID,
		}, true
	}

	return transferTargetItem{}, false
}

func buildTransferItemKey(
	itemType orderdom.OrderItemType,
	item orderdom.OrderItemSnapshot,
) string {
	switch itemType {
	case orderdom.OrderItemTypeList:
		if item.ModelID == "" {
			return ""
		}
		return "list:" + item.ModelID

	case orderdom.OrderItemTypeResale:
		if item.ResaleID == "" {
			return ""
		}
		return "resale:" + item.ResaleID

	default:
		return ""
	}
}

// inventoryId is expected to contain tokenBlueprintId as the second segment
// separated by "__".
func extractTokenBlueprintIDFromInventoryID(
	inventoryID string,
) string {
	if inventoryID == "" {
		return ""
	}

	parts := strings.Split(inventoryID, "__")
	if len(parts) < 2 {
		return ""
	}

	return parts[1]
}
