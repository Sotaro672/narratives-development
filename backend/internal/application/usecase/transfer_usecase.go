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
// Ports
// ============================================================

// ModelTokenPair is a minimal pair used for matching scanned product and purchased items.
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

	// scan side
	ScannedModelID          string `json:"scannedModelId"`
	ScannedTokenBlueprintID string `json:"scannedTokenBlueprintId"`

	// purchased side (dedup list)
	PurchasedPairs []ModelTokenPair `json:"purchasedPairs"`

	// verdict
	Matched bool            `json:"matched"`
	Match   *ModelTokenPair `json:"match,omitempty"`
}

// ScanVerifier verifies whether scan(productId) matches purchased(untransferred) items for avatar.
type ScanVerifier interface {
	VerifyMatch(ctx context.Context, in VerifyInput) (VerifyResult, error)
}

// OrderRepoForTransfer is the minimal port needed for transfer orchestration.
//
// Lock/Mark は item 単位の排他・確定更新をトランザクションで担保する想定。
//
// itemKey policy:
// - list item:   "list:" + modelId
// - resale item: "resale:" + resaleId
//
// NOTE:
// 旧実装では modelId を item key として扱っていたが、resale item は modelId を持たない。
// そのため itemModelID ではなく itemKey を渡す。
type OrderRepoForTransfer interface {
	ListPaidByAvatarID(ctx context.Context, avatarID string) ([]orderdom.Order, error)

	LockTransferItem(ctx context.Context, orderID string, itemKey string, now time.Time) error
	UnlockTransferItem(ctx context.Context, orderID string, itemKey string) error

	MarkTransferredItem(ctx context.Context, orderID string, itemKey string, at time.Time) error
}

// TokenResolver resolves token doc by productId (tokens/{productId}).
type TokenResolver interface {
	ResolveTokenByProductID(ctx context.Context, productID string) (TokenForTransfer, error)
}

type TokenForTransfer struct {
	ProductID string

	BrandID string // tokens/{productId}.brandId

	// transfer target
	MintAddress string // tokens/{productId}.mintAddress

	// for validation/logging
	TokenBlueprintID string // tokens/{productId}.tokenBlueprintId (optional but recommended)
	ToAddress        string // tokens/{productId}.toAddress (current owner cache)
}

// TokenOwnerUpdater updates "current owner" cache in tokens/{productId}.
type TokenOwnerUpdater interface {
	// UpdateToAddressByProductID sets tokens/{productId}.toAddress = newToAddress.
	// Implementation can also set updatedAt/transferredAt/lastTxSignature if you want.
	UpdateToAddressByProductID(ctx context.Context, productID string, newToAddress string, now time.Time, txSignature string) error
}

// WalletItemUpdater updates wallet table items for avatar.
//
// This is enough for list transfer because the sender is brand wallet and only
// receiver avatar wallet cache must be updated.
type WalletItemUpdater interface {
	// AddMintToAvatarWalletItems ensures mintAddress exists in wallet.tokens (dedup / idempotent).
	AddMintToAvatarWalletItems(ctx context.Context, avatarID string, mintAddress string, now time.Time) error
}

// TransferRepo persists transfer attempts (pending/succeeded/failed).
type TransferRepo interface {
	// NextAttempt returns the next monotonically increasing attempt number for a productId.
	NextAttempt(ctx context.Context, productID string) (int, error)

	// Create creates a new transfer attempt record (typically pending).
	Create(ctx context.Context, t transferdom.Transfer) error

	// Update updates an existing transfer attempt record by (productId, attempt).
	Update(ctx context.Context, productID string, attempt int, p transferdom.TransferPatch) error
}

// BrandWalletResolver resolves brand walletAddress by brandId.
type BrandWalletResolver interface {
	ResolveBrandWalletAddress(ctx context.Context, brandID string) (string, error)
}

// AvatarWalletResolver resolves avatar walletAddress by avatarId.
type AvatarWalletResolver interface {
	ResolveAvatarWalletAddress(ctx context.Context, avatarID string) (string, error)
}

// BrandDisplayResolver resolves brand display info for transfer result.
//
// brand.RepositoryPort / brand.Repository の GetByID(ctx, id string) に合わせる。
// transfer response では Brand.Name を表示名として使う。
type BrandDisplayResolver interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

// AvatarDisplayResolver resolves avatar display info for transfer result.
//
// avatar.Repository の GetByID(ctx, id string) に合わせる。
// transfer response では Avatar.AvatarName を表示名として使う。
type AvatarDisplayResolver interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

// WalletSecretProvider provides a signing capability for a brand.
type WalletSecretProvider interface {
	GetBrandSigner(ctx context.Context, brandID string) (any, error)
}

// ResaleReaderForTransfer resolves resale listing owner for resale transfer.
//
// Resale transfer is avatar wallet -> avatar wallet.
// The seller avatar is resale.Resale.AvatarID.
type ResaleReaderForTransfer interface {
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
}

// TokenTransferExecutor executes transfer using signers.
type TokenTransferExecutor interface {
	ExecuteTransfer(ctx context.Context, in ExecuteTransferInput) (ExecuteTransferResult, error)
}

type ExecuteTransferInput struct {
	// identifiers
	ProductID        string
	AvatarID         string
	BrandID          string
	ModelID          string
	TokenBlueprintID string

	// token info
	MintAddress string
	Amount      uint64 // SPL transfer amount（NFT想定=1）

	// wallets
	FromWalletAddress string
	ToWalletAddress   string

	// signers
	FromSigner any
	ToSigner   any
}

type ExecuteTransferResult struct {
	TxSignature string
}

// PostTransferResolveWarmer warms the resolve path after transfer.
//
// 期待値:
// - transfer 完了後に、その avatar から mintAddress を resolve できる状態まで進める
// - 実装側では cache warm / signed viewUri 発行 / ownership 再評価 などを含めてよい
type PostTransferResolveWarmer interface {
	ResolveAfterTransfer(ctx context.Context, avatarID string, mintAddress string) error
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
	transferRepo TransferRepo

	brandWallet  BrandWalletResolver
	avatarWallet AvatarWalletResolver

	brandDisplay  BrandDisplayResolver
	avatarDisplay AvatarDisplayResolver

	secrets  WalletSecretProvider
	executor TokenTransferExecutor

	// resale transfer dependencies.
	//
	// resale transfer uses:
	// - resaleRepo.GetByID(resaleId) to resolve seller avatar
	// - avatarSecrets.GetAvatarSigner(sellerAvatarID)
	// - walletTransferUpdate.RemoveMintFromAvatarWalletItems(...)
	// - walletSync.SyncWalletTokens(...) for seller and buyer
	resaleRepo           ResaleReaderForTransfer
	avatarSecrets        AvatarSecretProvider
	walletTransferUpdate AvatarWalletItemTransferUpdater
	walletSync           AvatarWalletSyncer

	// resolve warmup after successful transfer
	resolveWarmer PostTransferResolveWarmer

	// optional dependency
	inventoryUC *InventoryUsecase

	now func() time.Time
}

func NewTransferUsecase(
	verifier ScanVerifier,
	orderRepo OrderRepoForTransfer,
	tokenRepo TokenResolver,
	tokenUpdate TokenOwnerUpdater,
	walletUpdate WalletItemUpdater,
	transferRepo TransferRepo,
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

// WithResaleTransferDependencies enables resale order transfer.
//
// Keep this as a fluent optional setter so existing DI can compile while
// container wiring is updated. Resale item transfer will fail fast if these
// dependencies are missing.
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
	ErrTransferTokenDocNotReady       = errors.New("transfer_uc: token doc is not ready")
	ErrTransferResolveAfterFailed     = errors.New("transfer_uc: post-transfer resolve failed")
	ErrTransferInventoryCleanupFailed = errors.New("transfer_uc: inventory cleanup failed")

	ErrTransferResaleNotConfigured       = errors.New("transfer_uc: resale transfer dependencies are not configured")
	ErrTransferResaleIDEmpty             = errors.New("transfer_uc: resaleId is empty")
	ErrTransferResaleSellerAvatarIDEmpty = errors.New("transfer_uc: resale seller avatarId is empty")
	ErrTransferSameAvatar                = errors.New("transfer_uc: seller avatarId and buyer avatarId must be different")
	ErrTransferWalletSyncFailed          = errors.New("transfer_uc: wallet sync failed")
)

// TransferByVerifiedScanInput is the entry point input.
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

// TransferToAvatarByVerifiedScan does:
// 0) verify scan (avatarId, productId) => matched?
// 1) tokens/{productId} から mintAddress/brandId/tokenBlueprintId/toAddress を取得
// 2) orders を avatarId + paid=true で検索し、未移転 item を特定
//   - list item: modelId + tokenBlueprintId
//   - resale item: productId + tokenBlueprintId
//
// 3) lock(item単位)
// 4) transfer source wallet / buyer avatar wallet を解決
//   - list item: brand wallet -> buyer avatar wallet
//   - resale item: seller avatar wallet -> buyer avatar wallet
//
// 5) transfer(PENDING) 起票（attempt採番→pending作成）
// 6) token.toAddress が transfer source wallet を指しているか検証
// 7) source signer を取得
// 8) mintAddress を source -> buyer avatar へ transfer
// 9) transfer(SUCCEEDED) 更新
// 10) orders item を transferred=true で確定更新
// 11) tokens/{productId}.toAddress を buyer avatar wallet に更新
// 12) wallet テーブルを更新
//   - list item: buyer wallet に mintAddress を追加
//   - resale item: seller wallet から削除 + buyer wallet に追加 + 両者 sync
//
// 13) resolve warmup を実行（期待値: resolve まで完了）
// 14) list item の場合のみ inventory cleanup
// 15) 失敗時は transfer(FAILED) 更新 + unlock（best-effort）
func (u *TransferUsecase) TransferToAvatarByVerifiedScan(ctx context.Context, in TransferByVerifiedScanInput) (res TransferByVerifiedScanResult, retErr error) {
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
		u.executor == nil {
		return TransferByVerifiedScanResult{}, ErrTransferNotConfigured
	}

	avatarID := in.AvatarID
	productID := in.ProductID

	if avatarID == "" {
		return TransferByVerifiedScanResult{}, ErrTransferAvatarIDEmpty
	}
	if productID == "" {
		return TransferByVerifiedScanResult{}, ErrTransferProductIDEmpty
	}

	// 0) verify
	vres, err := u.verifier.VerifyMatch(ctx, VerifyInput{
		AvatarID:  avatarID,
		ProductID: productID,
	})
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: verify failed: %w", err)
	}
	if !vres.Matched {
		return TransferByVerifiedScanResult{}, ErrTransferNotMatched
	}

	scannedModelID := vres.ScannedModelID
	scannedTBID := vres.ScannedTokenBlueprintID

	// 1) token doc
	tok, err := u.tokenRepo.ResolveTokenByProductID(ctx, productID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: resolve token failed productId=%s: %w", productID, err)
	}

	brandID := tok.BrandID
	mint := tok.MintAddress
	tokenTBID := tok.TokenBlueprintID
	currentOwner := tok.ToAddress

	if brandID == "" {
		return TransferByVerifiedScanResult{}, ErrTransferBrandIDEmpty
	}
	if mint == "" {
		return TransferByVerifiedScanResult{}, ErrTransferMintEmpty
	}

	if scannedTBID == "" {
		scannedTBID = tokenTBID
	}
	if scannedTBID == "" {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: scanned tokenBlueprintId empty (productId=%s)", productID)
	}

	if tokenTBID != "" && tokenTBID != scannedTBID {
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: tokenBlueprint mismatch productId=%s scanned=%s tokenDoc=%s",
			productID, scannedTBID, tokenTBID,
		)
	}

	// 2) locate eligible order/item
	orders, err := u.orderRepo.ListPaidByAvatarID(ctx, avatarID)
	if err != nil {
		return TransferByVerifiedScanResult{}, err
	}
	if len(orders) == 0 {
		return TransferByVerifiedScanResult{}, ErrTransferNoEligibleOrder
	}

	target, ok := findUntransferredTransferTarget(
		orders,
		productID,
		scannedModelID,
		scannedTBID,
	)
	if !ok {
		return TransferByVerifiedScanResult{}, ErrTransferNoEligibleOrder
	}

	now := u.now().UTC()

	// 3) lock
	if err := u.orderRepo.LockTransferItem(ctx, target.OrderID, target.ItemKey, now); err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: lock failed orderId=%s itemKey=%s: %w",
			target.OrderID, target.ItemKey, err,
		)
	}

	locked := true
	transferAttempt := 0
	transferCreated := false

	defer func() {
		if retErr != nil && transferCreated {
			et := transferdom.ErrorTypeUnknown
			msg := retErr.Error()
			st := transferdom.StatusFailed
			p := transferdom.TransferPatch{
				Status:    &st,
				ErrorType: &et,
				ErrorMsg:  &msg,
			}
			_ = u.transferRepo.Update(context.Background(), productID, transferAttempt, p)
		}

		if locked {
			_ = u.orderRepo.UnlockTransferItem(context.Background(), target.OrderID, target.ItemKey)
		}
	}()

	// 4) resolve receiver wallet
	toWallet, err := u.avatarWallet.ResolveAvatarWalletAddress(ctx, avatarID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: resolve receiver avatar wallet failed avatarId=%s: %w", avatarID, err)
	}
	if toWallet == "" {
		return TransferByVerifiedScanResult{}, ErrTransferToWalletEmpty
	}

	// 4.1) resolve source wallet + signer
	source, err := u.resolveTransferSource(ctx, target, brandID, avatarID)
	if err != nil {
		return TransferByVerifiedScanResult{}, err
	}

	// 4.5) create transfer record (PENDING)
	attempt, err := u.transferRepo.NextAttempt(ctx, productID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: next attempt failed productId=%s: %w", productID, err)
	}

	tr, err := transferdom.NewPending(
		attempt,
		productID,
		target.OrderID,
		avatarID,
		toWallet,
		mint,
		now,
	)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: build transfer entity failed productId=%s attempt=%d: %w", productID, attempt, err)
	}
	if err := u.transferRepo.Create(ctx, tr); err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: create transfer failed productId=%s attempt=%d: %w", productID, attempt, err)
	}
	transferAttempt = attempt
	transferCreated = true

	markFailed := func(et transferdom.ErrorType, msg string, txSig *string) {
		if !transferCreated {
			return
		}
		st := transferdom.StatusFailed
		m := msg
		p := transferdom.TransferPatch{
			Status:    &st,
			ErrorType: &et,
			ErrorMsg:  &m,
		}
		if txSig != nil {
			s := *txSig
			p.TxSignature = &s
		}
		_ = u.transferRepo.Update(context.Background(), productID, transferAttempt, p)
	}

	markSucceeded := func(txSig string) {
		if !transferCreated {
			return
		}
		st := transferdom.StatusSucceeded
		s := txSig
		p := transferdom.TransferPatch{
			Status:      &st,
			TxSignature: &s,
		}
		_ = u.transferRepo.Update(context.Background(), productID, transferAttempt, p)
	}

	// 5) safety: token current owner check
	if currentOwner != "" && currentOwner != source.FromWallet {
		msg := fmt.Sprintf("productId=%s tokenOwner=%s expectedFromWallet=%s itemType=%s",
			productID, currentOwner, source.FromWallet, target.ItemType,
		)
		markFailed(transferdom.ErrorTypeMismatch, msg, nil)
		return TransferByVerifiedScanResult{}, fmt.Errorf("%w: %s", ErrTransferOwnerMismatch, msg)
	}

	var toSigner any = nil

	// 7) execute transfer
	execOut, err := u.executor.ExecuteTransfer(ctx, ExecuteTransferInput{
		ProductID:        productID,
		AvatarID:         avatarID,
		BrandID:          brandID,
		ModelID:          target.ModelID,
		TokenBlueprintID: scannedTBID,

		MintAddress: mint,
		Amount:      1,

		FromWalletAddress: source.FromWallet,
		ToWalletAddress:   toWallet,

		FromSigner: source.FromSigner,
		ToSigner:   toSigner,
	})
	if err != nil {
		msg := fmt.Sprintf("execute transfer failed orderId=%s itemKey=%s mint=%s: %v",
			target.OrderID, target.ItemKey, mint, err,
		)
		markFailed(transferdom.ErrorTypeTransferFailed, msg, nil)
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: execute transfer failed orderId=%s itemKey=%s mint=%s: %w",
			target.OrderID, target.ItemKey, mint, err,
		)
	}

	tx := execOut.TxSignature

	// 7.5) transfer record -> SUCCEEDED
	markSucceeded(tx)

	// 8) mark transferred true
	if err := u.orderRepo.MarkTransferredItem(ctx, target.OrderID, target.ItemKey, now); err != nil {
		msg := fmt.Sprintf("mark transferred failed orderId=%s itemKey=%s tx=%s: %v",
			target.OrderID, target.ItemKey, tx, err,
		)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: mark transferred failed orderId=%s itemKey=%s tx=%s: %w",
			target.OrderID, target.ItemKey, tx, err,
		)
	}

	// 9) update tokens/{productId}.toAddress = buyer avatar wallet
	if err := u.tokenUpdate.UpdateToAddressByProductID(ctx, productID, toWallet, now, tx); err != nil {
		msg := fmt.Sprintf("update token owner failed productId=%s to=%s tx=%s: %v",
			productID, toWallet, tx, err,
		)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: update token owner failed productId=%s to=%s tx=%s: %w",
			productID, toWallet, tx, err,
		)
	}

	// 10) update wallet table tokens
	if err := u.updateWalletsAfterTransfer(ctx, target, source.FromAvatarID, avatarID, mint, now, tx); err != nil {
		msg := err.Error()
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return TransferByVerifiedScanResult{}, err
	}

	// 11) resolve warmup (期待値: resolve までセットでやりきる)
	if u.resolveWarmer != nil {
		if err := u.resolveWarmer.ResolveAfterTransfer(ctx, avatarID, mint); err != nil {
			msg := fmt.Sprintf("post-transfer resolve failed avatarId=%s mint=%s tx=%s: %v",
				avatarID, mint, tx, err,
			)
			markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
			return TransferByVerifiedScanResult{}, fmt.Errorf("%w: %s", ErrTransferResolveAfterFailed, msg)
		}
	}

	// 12) inventory cleanup
	// resale item は inventory reservation 対象ではないため実行しない。
	if target.ItemType == orderdom.OrderItemTypeList && u.inventoryUC != nil {
		if err := u.inventoryUC.ReleaseAfterTransfer(
			ctx,
			target.InventoryID,
			target.ModelID,
			productID,
			target.OrderID,
			now,
		); err != nil {
			msg := fmt.Sprintf("inventory cleanup failed inventoryId=%s modelId=%s productId=%s orderId=%s tx=%s: %v",
				target.InventoryID, target.ModelID, productID, target.OrderID, tx, err,
			)
			markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
			return TransferByVerifiedScanResult{}, fmt.Errorf("%w: %s", ErrTransferInventoryCleanupFailed, msg)
		}
	}

	fromDisplayName := ""
	if source.FromAvatarID != "" {
		fromDisplayName = u.resolveAvatarDisplayName(ctx, source.FromAvatarID)
	} else {
		fromDisplayName = u.resolveBrandDisplayName(ctx, source.FromBrandID)
	}
	toDisplayName := u.resolveAvatarDisplayName(ctx, avatarID)

	locked = false

	return TransferByVerifiedScanResult{
		MatchedOrderID:     target.OrderID,
		MatchedInventoryID: target.InventoryID,
		MatchedModelID:     target.ModelID,

		MatchedItemKey:  target.ItemKey,
		MatchedItemType: target.ItemType,
		MatchedResaleID: target.ResaleID,

		ProductID:        productID,
		MintAddress:      mint,
		TokenBlueprintID: scannedTBID,

		FromWallet:  source.FromWallet,
		ToWallet:    toWallet,
		TxSignature: tx,

		FromDisplayName: fromDisplayName,
		ToDisplayName:   toDisplayName,
	}, nil
}

// ============================================================
// Transfer source / wallet update helpers
// ============================================================

func (u *TransferUsecase) resolveTransferSource(
	ctx context.Context,
	target transferTargetItem,
	brandID string,
	buyerAvatarID string,
) (transferExecutionSource, error) {
	switch target.ItemType {
	case orderdom.OrderItemTypeResale:
		return u.resolveResaleTransferSource(ctx, target, buyerAvatarID)

	case orderdom.OrderItemTypeList:
		return u.resolveListTransferSource(ctx, brandID)

	default:
		return transferExecutionSource{}, ErrTransferNoEligibleOrder
	}
}

func (u *TransferUsecase) resolveListTransferSource(
	ctx context.Context,
	brandID string,
) (transferExecutionSource, error) {
	if brandID == "" {
		return transferExecutionSource{}, ErrTransferBrandIDEmpty
	}

	fromWallet, err := u.brandWallet.ResolveBrandWalletAddress(ctx, brandID)
	if err != nil {
		return transferExecutionSource{}, fmt.Errorf("transfer_uc: resolve brand wallet failed brandId=%s: %w", brandID, err)
	}
	if fromWallet == "" {
		return transferExecutionSource{}, ErrTransferFromWalletEmpty
	}

	fromSigner, err := u.secrets.GetBrandSigner(ctx, brandID)
	if err != nil {
		return transferExecutionSource{}, fmt.Errorf(
			"transfer_uc: get brand signer failed brandId=%s wallet=%s: %w",
			brandID, fromWallet, err,
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
		return transferExecutionSource{}, ErrTransferResaleNotConfigured
	}

	resaleID := target.ResaleID
	if resaleID == "" {
		return transferExecutionSource{}, ErrTransferResaleIDEmpty
	}

	r, err := u.resaleRepo.GetByID(ctx, resaleID)
	if err != nil {
		return transferExecutionSource{}, fmt.Errorf("transfer_uc: resolve resale failed resaleId=%s: %w", resaleID, err)
	}

	fromAvatarID := r.AvatarID
	if fromAvatarID == "" {
		return transferExecutionSource{}, ErrTransferResaleSellerAvatarIDEmpty
	}
	if fromAvatarID == buyerAvatarID {
		return transferExecutionSource{}, ErrTransferSameAvatar
	}

	fromWallet, err := u.avatarWallet.ResolveAvatarWalletAddress(ctx, fromAvatarID)
	if err != nil {
		return transferExecutionSource{}, fmt.Errorf(
			"transfer_uc: resolve seller avatar wallet failed avatarId=%s: %w",
			fromAvatarID, err,
		)
	}
	if fromWallet == "" {
		return transferExecutionSource{}, ErrTransferFromWalletEmpty
	}

	fromSigner, err := u.avatarSecrets.GetAvatarSigner(ctx, fromAvatarID)
	if err != nil {
		return transferExecutionSource{}, fmt.Errorf(
			"transfer_uc: get seller avatar signer failed avatarId=%s wallet=%s: %w",
			fromAvatarID, fromWallet, err,
		)
	}

	return transferExecutionSource{
		FromAvatarID: fromAvatarID,
		FromWallet:   fromWallet,
		FromSigner:   fromSigner,
	}, nil
}

func (u *TransferUsecase) updateWalletsAfterTransfer(
	ctx context.Context,
	target transferTargetItem,
	fromAvatarID string,
	toAvatarID string,
	mint string,
	now time.Time,
	tx string,
) error {
	switch target.ItemType {
	case orderdom.OrderItemTypeResale:
		return u.updateResaleWalletsAfterTransfer(ctx, fromAvatarID, toAvatarID, mint, now, tx)

	case orderdom.OrderItemTypeList:
		if err := u.walletUpdate.AddMintToAvatarWalletItems(ctx, toAvatarID, mint, now); err != nil {
			return fmt.Errorf(
				"transfer_uc: update receiver wallet items failed avatarId=%s mint=%s tx=%s: %w",
				toAvatarID, mint, tx, err,
			)
		}
		return nil

	default:
		return ErrTransferNoEligibleOrder
	}
}

func (u *TransferUsecase) updateResaleWalletsAfterTransfer(
	ctx context.Context,
	fromAvatarID string,
	toAvatarID string,
	mint string,
	now time.Time,
	tx string,
) error {
	if u.walletTransferUpdate == nil || u.walletSync == nil {
		return ErrTransferResaleNotConfigured
	}

	if fromAvatarID == "" {
		return ErrTransferResaleSellerAvatarIDEmpty
	}
	if toAvatarID == "" {
		return ErrTransferAvatarIDEmpty
	}
	if mint == "" {
		return ErrTransferMintEmpty
	}

	if err := u.walletTransferUpdate.RemoveMintFromAvatarWalletItems(ctx, fromAvatarID, mint, now); err != nil {
		return fmt.Errorf(
			"transfer_uc: remove seller wallet item failed avatarId=%s mint=%s tx=%s: %w",
			fromAvatarID, mint, tx, err,
		)
	}

	if err := u.walletTransferUpdate.AddMintToAvatarWalletItems(ctx, toAvatarID, mint, now); err != nil {
		return fmt.Errorf(
			"transfer_uc: add buyer wallet item failed avatarId=%s mint=%s tx=%s: %w",
			toAvatarID, mint, tx, err,
		)
	}

	if _, err := u.walletSync.SyncWalletTokens(ctx, fromAvatarID); err != nil {
		return fmt.Errorf(
			"%w: sync seller wallet failed avatarId=%s mint=%s tx=%s: %v",
			ErrTransferWalletSyncFailed, fromAvatarID, mint, tx, err,
		)
	}

	if _, err := u.walletSync.SyncWalletTokens(ctx, toAvatarID); err != nil {
		return fmt.Errorf(
			"%w: sync buyer wallet failed avatarId=%s mint=%s tx=%s: %v",
			ErrTransferWalletSyncFailed, toAvatarID, mint, tx, err,
		)
	}

	return nil
}

// ============================================================
// Local helpers
// ============================================================

func (u *TransferUsecase) resolveBrandDisplayName(ctx context.Context, brandID string) string {
	if u == nil || u.brandDisplay == nil || brandID == "" {
		return ""
	}

	b, err := u.brandDisplay.GetByID(ctx, brandID)
	if err != nil {
		return ""
	}

	return b.Name
}

func (u *TransferUsecase) resolveAvatarDisplayName(ctx context.Context, avatarID string) string {
	if u == nil || u.avatarDisplay == nil || avatarID == "" {
		return ""
	}

	a, err := u.avatarDisplay.GetByID(ctx, avatarID)
	if err != nil {
		return ""
	}

	return a.AvatarName
}

func findUntransferredTransferTarget(
	orders []orderdom.Order,
	productID string,
	scannedModelID string,
	scannedTBID string,
) (transferTargetItem, bool) {
	if productID == "" || scannedTBID == "" {
		return transferTargetItem{}, false
	}

	for _, o := range orders {
		if !o.Paid {
			continue
		}

		if target, ok := findUntransferredResaleItem(o, productID, scannedTBID); ok {
			return target, true
		}

		if target, ok := findUntransferredListItem(o, scannedModelID, scannedTBID); ok {
			return target, true
		}
	}

	return transferTargetItem{}, false
}

func findUntransferredResaleItem(
	o orderdom.Order,
	productID string,
	scannedTBID string,
) (transferTargetItem, bool) {
	if productID == "" || scannedTBID == "" {
		return transferTargetItem{}, false
	}

	for _, it := range o.Items {
		itemType := inferTransferOrderItemType(it)
		if itemType != orderdom.OrderItemTypeResale {
			continue
		}

		if it.Transferred {
			continue
		}

		if it.ResaleID == "" {
			continue
		}

		if it.ProductID != productID {
			continue
		}

		itemTBID := it.TokenBlueprintID
		if itemTBID == "" {
			continue
		}
		if itemTBID != scannedTBID {
			continue
		}

		itemKey := buildTransferItemKey(itemType, it)
		if itemKey == "" {
			continue
		}

		return transferTargetItem{
			OrderID: o.ID,

			ItemKey:  itemKey,
			ItemType: itemType,

			ResaleID: it.ResaleID,

			ProductID:          it.ProductID,
			ProductBlueprintID: it.ProductBlueprintID,
			TokenBlueprintID:   itemTBID,
			BrandID:            it.BrandID,
		}, true
	}

	return transferTargetItem{}, false
}

func findUntransferredListItem(
	o orderdom.Order,
	scannedModelID string,
	scannedTBID string,
) (transferTargetItem, bool) {
	if scannedModelID == "" || scannedTBID == "" {
		return transferTargetItem{}, false
	}

	for _, it := range o.Items {
		itemType := inferTransferOrderItemType(it)
		if itemType != orderdom.OrderItemTypeList {
			continue
		}

		if it.ModelID != scannedModelID {
			continue
		}
		if it.InventoryID == "" {
			continue
		}
		if it.Transferred {
			continue
		}

		itemTB := extractTokenBlueprintIDFromInventoryID(it.InventoryID)
		if itemTB == "" {
			continue
		}
		if itemTB != scannedTBID {
			continue
		}

		itemKey := buildTransferItemKey(itemType, it)
		if itemKey == "" {
			continue
		}

		return transferTargetItem{
			OrderID: o.ID,

			ItemKey:  itemKey,
			ItemType: itemType,

			InventoryID: it.InventoryID,
			ModelID:     it.ModelID,

			TokenBlueprintID: itemTB,
		}, true
	}

	return transferTargetItem{}, false
}

func inferTransferOrderItemType(it orderdom.OrderItemSnapshot) orderdom.OrderItemType {
	switch it.Type {
	case orderdom.OrderItemTypeList, orderdom.OrderItemTypeResale:
		return it.Type
	}

	if it.ResaleID != "" || it.ProductID != "" {
		return orderdom.OrderItemTypeResale
	}

	if it.ModelID != "" ||
		it.InventoryID != "" ||
		it.ListID != "" {
		return orderdom.OrderItemTypeList
	}

	return ""
}

func buildTransferItemKey(
	itemType orderdom.OrderItemType,
	it orderdom.OrderItemSnapshot,
) string {
	switch itemType {
	case orderdom.OrderItemTypeResale:
		resaleID := it.ResaleID
		if resaleID == "" {
			return ""
		}
		return "resale:" + resaleID

	case orderdom.OrderItemTypeList:
		modelID := it.ModelID
		if modelID == "" {
			return ""
		}
		return "list:" + modelID

	default:
		return ""
	}
}

// inventoryId は "__" 区切りで、2つめのセグメントが tokenBlueprintId
func extractTokenBlueprintIDFromInventoryID(inventoryID string) string {
	if inventoryID == "" {
		return ""
	}

	parts := strings.Split(inventoryID, "__")
	if len(parts) < 2 {
		return ""
	}

	return parts[1]
}
