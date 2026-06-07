// backend/internal/application/usecase/transfer_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	mallquery "narratives/internal/application/query/mall"
	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	orderdom "narratives/internal/domain/order"
	transferdom "narratives/internal/domain/transfer"
)

// ============================================================
// Ports
// ============================================================

// ScanVerifier verifies whether scan(productId) matches purchased(untransferred) items for avatar.
type ScanVerifier interface {
	VerifyMatch(ctx context.Context, in mallquery.VerifyInput) (mallquery.VerifyResult, error)
}

// OrderRepoForTransfer is the minimal port needed for transfer orchestration.
//
// Lock/Mark は item 単位の排他・確定更新をトランザクションで担保する想定。
// itemKey は現状 modelId を採用（同一order内で modelId が重複しない前提）。
type OrderRepoForTransfer interface {
	ListPaidByAvatarID(ctx context.Context, avatarID string) ([]orderdom.Order, error)

	LockTransferItem(ctx context.Context, orderID string, itemModelID string, now time.Time) error
	UnlockTransferItem(ctx context.Context, orderID string, itemModelID string) error

	MarkTransferredItem(ctx context.Context, orderID string, itemModelID string, at time.Time) error
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
type WalletItemUpdater interface {
	// AddMintToAvatarWalletItems ensures mintAddress exists in wallet.tokens (dedup / idempotent).
	AddMintToAvatarWalletItems(ctx context.Context, avatarID string, mintAddress string, now time.Time, txSignature string) error
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
	FromWalletAddress string // brand wallet
	ToWalletAddress   string // avatar wallet

	// signers
	FromSigner any // brand signer (required)
	ToSigner   any // usually nil (optional)
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

	// resolve warmup after successful transfer
	resolveWarmer PostTransferResolveWarmer

	// optional injection
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
	secrets WalletSecretProvider,
	executor TokenTransferExecutor,
) *TransferUsecase {
	return &TransferUsecase{
		verifier:      verifier,
		orderRepo:     orderRepo,
		tokenRepo:     tokenRepo,
		tokenUpdate:   tokenUpdate,
		walletUpdate:  walletUpdate,
		transferRepo:  transferRepo,
		brandWallet:   brandWallet,
		avatarWallet:  avatarWallet,
		brandDisplay:  nil,
		avatarDisplay: nil,
		secrets:       secrets,
		executor:      executor,
		resolveWarmer: nil,
		inventoryUC:   nil,
		now:           time.Now,
	}
}

// WithInventoryUsecase injects InventoryUsecase for post-transfer cleanup.
func (u *TransferUsecase) WithInventoryUsecase(inventoryUC *InventoryUsecase) *TransferUsecase {
	if u != nil {
		u.inventoryUC = inventoryUC
	}
	return u
}

// WithTransferDisplayResolvers injects display resolvers for transfer response.
// These are used only for response display values.
func (u *TransferUsecase) WithTransferDisplayResolvers(
	brandDisplay BrandDisplayResolver,
	avatarDisplay AvatarDisplayResolver,
) *TransferUsecase {
	if u != nil {
		u.brandDisplay = brandDisplay
		u.avatarDisplay = avatarDisplay
	}
	return u
}

// WithPostTransferResolveWarmer injects resolve warmup logic.
// 例:
// - wallet resolve cache warm
// - signed viewUri issuance
// - ownership visibility confirmation
func (u *TransferUsecase) WithPostTransferResolveWarmer(w PostTransferResolveWarmer) *TransferUsecase {
	if u != nil {
		u.resolveWarmer = w
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
	ErrTransferFromWalletEmpty        = errors.New("transfer_uc: brand walletAddress is empty")
	ErrTransferToWalletEmpty          = errors.New("transfer_uc: avatar walletAddress is empty")
	ErrTransferOwnerMismatch          = errors.New("transfer_uc: token current owner mismatch")
	ErrTransferTokenDocNotReady       = errors.New("transfer_uc: token doc is not ready")
	ErrTransferResolveAfterFailed     = errors.New("transfer_uc: post-transfer resolve failed")
	ErrTransferInventoryCleanupFailed = errors.New("transfer_uc: inventory cleanup failed")
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

	ProductID        string
	MintAddress      string
	TokenBlueprintID string

	FromWallet  string // brand wallet
	ToWallet    string // avatar wallet
	TxSignature string

	FromDisplayName string
	ToDisplayName   string
}

// TransferToAvatarByVerifiedScan does:
// 0) verify scan (avatarId, productId) => matched? + scanned modelId/tokenBlueprintId
// 1) tokens/{productId} から mintAddress/brandId/tokenBlueprintId/toAddress を取得
// 2) orders を avatarId + paid=true で検索し、未移転 item を特定（modelId + tokenBlueprintId で厳密一致）
// 3) lock(item単位)
// 4) brand wallet / avatar wallet を解決
// 5) transfer(PENDING) 起票（attempt採番→pending作成）
// 6) token.toAddress が brand wallet を指しているか検証
// 7) brand signer を取得
// 8) mintAddress を brand -> avatar へ transfer
// 9) transfer(SUCCEEDED) 更新
// 10) orders item を transferred=true で確定更新
// 11) tokens/{productId}.toAddress を avatar wallet に更新
// 12) wallet テーブル（avatar wallet）の tokens に mintAddress を追加
// 13) resolve warmup を実行（期待値: resolve まで完了）
// 14) inventory cleanup
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
	vres, err := u.verifier.VerifyMatch(ctx, mallquery.VerifyInput{
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
	if scannedModelID == "" {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: scanned modelId empty (productId=%s)", productID)
	}
	if scannedTBID == "" {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: scanned tokenBlueprintId empty (productId=%s)", productID)
	}

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

	var (
		targetOrderID     string
		targetInventoryID string
		targetModelID     string
	)

	for _, o := range orders {
		if !o.Paid {
			continue
		}

		inventoryID, modelID, ok := findUntransferredItemByModelAndTB(o, scannedModelID, scannedTBID)
		if !ok {
			continue
		}

		targetOrderID = o.ID
		targetInventoryID = inventoryID
		targetModelID = modelID
		break
	}

	if targetOrderID == "" || targetInventoryID == "" || targetModelID == "" {
		return TransferByVerifiedScanResult{}, ErrTransferNoEligibleOrder
	}

	now := u.now().UTC()

	// 3) lock
	if err := u.orderRepo.LockTransferItem(ctx, targetOrderID, targetModelID, now); err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: lock failed orderId=%s modelId=%s: %w",
			targetOrderID, targetModelID, err,
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
			_ = u.orderRepo.UnlockTransferItem(context.Background(), targetOrderID, targetModelID)
		}
	}()

	// 4) resolve wallets
	fromWallet, err := u.brandWallet.ResolveBrandWalletAddress(ctx, brandID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: resolve brand wallet failed brandId=%s: %w", brandID, err)
	}
	if fromWallet == "" {
		return TransferByVerifiedScanResult{}, ErrTransferFromWalletEmpty
	}

	toWallet, err := u.avatarWallet.ResolveAvatarWalletAddress(ctx, avatarID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: resolve avatar wallet failed avatarId=%s: %w", avatarID, err)
	}
	if toWallet == "" {
		return TransferByVerifiedScanResult{}, ErrTransferToWalletEmpty
	}

	// 4.5) create transfer record (PENDING)
	attempt, err := u.transferRepo.NextAttempt(ctx, productID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: next attempt failed productId=%s: %w", productID, err)
	}

	tr, err := transferdom.NewPending(
		attempt,
		productID,
		targetOrderID,
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
	if currentOwner != "" && currentOwner != fromWallet {
		msg := fmt.Sprintf("productId=%s tokenOwner=%s expectedBrandWallet=%s",
			productID, currentOwner, fromWallet,
		)
		markFailed(transferdom.ErrorTypeMismatch, msg, nil)
		return TransferByVerifiedScanResult{}, fmt.Errorf("%w: %s", ErrTransferOwnerMismatch, msg)
	}

	// 6) signer
	fromSigner, err := u.secrets.GetBrandSigner(ctx, brandID)
	if err != nil {
		msg := fmt.Sprintf("get brand signer failed brandId=%s wallet=%s: %v", brandID, fromWallet, err)
		markFailed(transferdom.ErrorTypeSecretInvalid, msg, nil)
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: get brand signer failed brandId=%s wallet=%s: %w",
			brandID, fromWallet, err,
		)
	}

	var toSigner any = nil

	// 7) execute transfer
	execOut, err := u.executor.ExecuteTransfer(ctx, ExecuteTransferInput{
		ProductID:        productID,
		AvatarID:         avatarID,
		BrandID:          brandID,
		ModelID:          targetModelID,
		TokenBlueprintID: scannedTBID,

		MintAddress: mint,
		Amount:      1,

		FromWalletAddress: fromWallet,
		ToWalletAddress:   toWallet,

		FromSigner: fromSigner,
		ToSigner:   toSigner,
	})
	if err != nil {
		msg := fmt.Sprintf("execute transfer failed orderId=%s modelId=%s mint=%s: %v",
			targetOrderID, targetModelID, mint, err,
		)
		markFailed(transferdom.ErrorTypeTransferFailed, msg, nil)
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: execute transfer failed orderId=%s modelId=%s mint=%s: %w",
			targetOrderID, targetModelID, mint, err,
		)
	}

	tx := execOut.TxSignature

	// 7.5) transfer record -> SUCCEEDED
	markSucceeded(tx)

	// 8) mark transferred true
	if err := u.orderRepo.MarkTransferredItem(ctx, targetOrderID, targetModelID, now); err != nil {
		msg := fmt.Sprintf("mark transferred failed orderId=%s modelId=%s tx=%s: %v",
			targetOrderID, targetModelID, tx, err,
		)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: mark transferred failed orderId=%s modelId=%s tx=%s: %w",
			targetOrderID, targetModelID, tx, err,
		)
	}

	// 9) update tokens/{productId}.toAddress = avatar wallet
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
	if err := u.walletUpdate.AddMintToAvatarWalletItems(ctx, avatarID, mint, now, tx); err != nil {
		msg := fmt.Sprintf("update wallet items failed avatarId=%s mint=%s tx=%s: %v",
			avatarID, mint, tx, err,
		)
		markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: %s", msg)
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
	if u.inventoryUC != nil {
		if err := u.inventoryUC.ReleaseAfterTransfer(
			ctx,
			targetInventoryID,
			targetModelID,
			productID,
			targetOrderID,
			now,
		); err != nil {
			msg := fmt.Sprintf("inventory cleanup failed inventoryId=%s modelId=%s productId=%s orderId=%s tx=%s: %v",
				targetInventoryID, targetModelID, productID, targetOrderID, tx, err,
			)
			markFailed(transferdom.ErrorTypeUnknown, msg, &tx)
			return TransferByVerifiedScanResult{}, fmt.Errorf("%w: %s", ErrTransferInventoryCleanupFailed, msg)
		}
	}

	fromDisplayName := u.resolveBrandDisplayName(ctx, brandID)
	toDisplayName := u.resolveAvatarDisplayName(ctx, avatarID)

	locked = false

	return TransferByVerifiedScanResult{
		MatchedOrderID:     targetOrderID,
		MatchedInventoryID: targetInventoryID,
		MatchedModelID:     targetModelID,

		ProductID:        productID,
		MintAddress:      mint,
		TokenBlueprintID: scannedTBID,

		FromWallet:  fromWallet,
		ToWallet:    toWallet,
		TxSignature: tx,

		FromDisplayName: fromDisplayName,
		ToDisplayName:   toDisplayName,
	}, nil
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

// findUntransferredItemByModelAndTB returns (inventoryId, modelId, true) if order has an item where:
// - item.ModelID == scannedModelID
// - item.InventoryID is not empty
// - item.Transferred == false
// - tokenBlueprintId extracted from item.InventoryID matches scannedTBID (strict)
func findUntransferredItemByModelAndTB(o orderdom.Order, scannedModelID string, scannedTBID string) (string, string, bool) {
	m := scannedModelID
	tb := scannedTBID
	if m == "" || tb == "" {
		return "", "", false
	}

	for _, it := range o.Items {
		if it.ModelID != m {
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
		if itemTB != tb {
			continue
		}

		return it.InventoryID, it.ModelID, true
	}

	return "", "", false
}

// inventoryId は "__" 区切りで、2つめのセグメントが tokenBlueprintId
func extractTokenBlueprintIDFromInventoryID(inventoryID string) string {
	s := inventoryID
	if s == "" {
		return ""
	}

	parts := strings.Split(s, "__")
	if len(parts) < 2 {
		return ""
	}

	return parts[1]
}
