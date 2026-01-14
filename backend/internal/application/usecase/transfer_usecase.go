// backend/internal/application/usecase/transfer_usecase.go
package usecase

/*
責任と機能（今回の要件版）:
- /mall/me/orders/scan/verify で matched=true になった productId を起点に、
  tokens/{productId}.mintAddress の Solana トークン（SPL/NFT想定）を
  brand の wallet → avatar の wallet へ自動移譲する。
- 競合（二重処理）を防ぐために、order の item 単位で transfer lock を取得してから処理する。
- transfer 成功後に orders/{orderId} の該当 item の transferred/transferredAt を更新する。
- さらに tokens/{productId}.toAddress を「今の owner (= avatar wallet)」として必ず更新する。
- 外部依存（Firestore/SecretManager/Solana/TokenOperation 等）は Port(interface) に閉じ込め、
  Usecase は「手順（オーケストレーション）」のみを担う。

重要:
- tokens/{productId}.toAddress が “今の owner” を表す運用なら、
  転送前に toAddress が brand wallet と一致しているか（または空）を検証するのが安全。
  （すでに別 owner の可能性があるのに転送してしまう事故を避ける）
*/

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
)

// ============================================================
// Ports
// ============================================================

// ScanVerifier verifies whether scan(productId) matches purchased(untransferred) items for avatar.
// application/query/mall の OrderScanVerifyQuery を DI で差し込む想定。
type ScanVerifier interface {
	Verify(ctx context.Context, avatarID, productID string) (ScanVerifyResult, error)
}

type ScanVerifyResult struct {
	AvatarID  string
	ProductID string

	ScannedModelID          string
	ScannedTokenBlueprintID string

	Matched bool
}

// OrderRepoForTransfer is the minimal port needed for transfer orchestration.
//
// ✅ 注意: Order の transferred/transferredAt は廃止し、item 単位で保持する。
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

// BrandWalletResolver resolves brand walletAddress by brandId.
type BrandWalletResolver interface {
	ResolveBrandWalletAddress(ctx context.Context, brandID string) (string, error)
}

// AvatarWalletResolver resolves avatar walletAddress by avatarId.
type AvatarWalletResolver interface {
	ResolveAvatarWalletAddress(ctx context.Context, avatarID string) (string, error)
}

// WalletSecretProvider provides a signing capability for a walletAddress.
type WalletSecretProvider interface {
	GetSigner(ctx context.Context, walletAddress string) (any, error)
}

// TokenTransferExecutor executes transfer using signers.
// NFT想定なら amount=1。実装側で ATA 作成等も内包してよい。
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
	Amount      uint64 // ✅ NEW: SPL transfer amount（NFT想定=1）

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

// ============================================================
// Usecase
// ============================================================

type TransferUsecase struct {
	verifier    ScanVerifier
	orderRepo   OrderRepoForTransfer
	tokenRepo   TokenResolver
	tokenUpdate TokenOwnerUpdater

	brandWallet  BrandWalletResolver
	avatarWallet AvatarWalletResolver

	secrets  WalletSecretProvider
	executor TokenTransferExecutor

	now func() time.Time
}

func NewTransferUsecase(
	verifier ScanVerifier,
	orderRepo OrderRepoForTransfer,
	tokenRepo TokenResolver,
	tokenUpdate TokenOwnerUpdater,
	brandWallet BrandWalletResolver,
	avatarWallet AvatarWalletResolver,
	secrets WalletSecretProvider,
	executor TokenTransferExecutor,
) *TransferUsecase {
	return &TransferUsecase{
		verifier:     verifier,
		orderRepo:    orderRepo,
		tokenRepo:    tokenRepo,
		tokenUpdate:  tokenUpdate,
		brandWallet:  brandWallet,
		avatarWallet: avatarWallet,
		secrets:      secrets,
		executor:     executor,
		now:          time.Now,
	}
}

var (
	ErrTransferNotConfigured    = errors.New("transfer_uc: not configured")
	ErrTransferAvatarIDEmpty    = errors.New("transfer_uc: avatarId is empty")
	ErrTransferProductIDEmpty   = errors.New("transfer_uc: productId is empty")
	ErrTransferNotMatched       = errors.New("transfer_uc: scan is not matched")
	ErrTransferNoEligibleOrder  = errors.New("transfer_uc: no eligible order/item found")
	ErrTransferMintEmpty        = errors.New("transfer_uc: mintAddress is empty")
	ErrTransferBrandIDEmpty     = errors.New("transfer_uc: brandId is empty")
	ErrTransferFromWalletEmpty  = errors.New("transfer_uc: brand walletAddress is empty")
	ErrTransferToWalletEmpty    = errors.New("transfer_uc: avatar walletAddress is empty")
	ErrTransferOwnerMismatch    = errors.New("transfer_uc: token current owner mismatch")
	ErrTransferTokenDocNotReady = errors.New("transfer_uc: token doc is not ready")
)

// TransferByVerifiedScanInput is the entry point input.
// verify(matched=true) になった productId を起点に、brand→avatar へ移譲する。
type TransferByVerifiedScanInput struct {
	AvatarID  string
	ProductID string
}

type TransferByVerifiedScanResult struct {
	MatchedOrderID string
	MatchedModelID string

	ProductID        string
	MintAddress      string
	TokenBlueprintID string

	FromWallet  string // brand wallet
	ToWallet    string // avatar wallet
	TxSignature string
}

// TransferToAvatarByVerifiedScan does:
// 0) verify scan (avatarId, productId) => matched? + scanned modelId/tokenBlueprintId
// 1) tokens/{productId} から mintAddress/brandId/tokenBlueprintId/toAddress を取得
// 2) orders を avatarId + paid=true で検索し、未移転 item を特定（modelId + tokenBlueprintId で厳密一致）
// 3) lock(item単位)
// 4) brand wallet / avatar wallet を解決
// 5) (optional) token.toAddress が brand wallet を指しているか検証（誤転送防止）
// 6) brand signer を取得
// 7) mintAddress を brand → avatar へ transfer (amount=1)
// 8) orders item を transferred=true で確定更新
// 9) tokens/{productId}.toAddress を avatar wallet に更新（今の owner を正にする）
// 10) 失敗時は unlock（best-effort）
func (u *TransferUsecase) TransferToAvatarByVerifiedScan(ctx context.Context, in TransferByVerifiedScanInput) (TransferByVerifiedScanResult, error) {
	if u == nil ||
		u.verifier == nil ||
		u.orderRepo == nil ||
		u.tokenRepo == nil ||
		u.tokenUpdate == nil ||
		u.brandWallet == nil ||
		u.avatarWallet == nil ||
		u.secrets == nil ||
		u.executor == nil {
		return TransferByVerifiedScanResult{}, ErrTransferNotConfigured
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	productID := strings.TrimSpace(in.ProductID)

	if avatarID == "" {
		return TransferByVerifiedScanResult{}, ErrTransferAvatarIDEmpty
	}
	if productID == "" {
		return TransferByVerifiedScanResult{}, ErrTransferProductIDEmpty
	}

	// 0) verify
	vres, err := u.verifier.Verify(ctx, avatarID, productID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: verify failed: %w", err)
	}
	if !vres.Matched {
		return TransferByVerifiedScanResult{}, ErrTransferNotMatched
	}

	scannedModelID := strings.TrimSpace(vres.ScannedModelID)
	scannedTBID := strings.TrimSpace(vres.ScannedTokenBlueprintID)
	if scannedModelID == "" {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: scanned modelId empty (productId=%s)", _mask(productID))
	}
	if scannedTBID == "" {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: scanned tokenBlueprintId empty (productId=%s)", _mask(productID))
	}

	// 1) token doc
	tok, err := u.tokenRepo.ResolveTokenByProductID(ctx, productID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: resolve token failed productId=%s: %w", _mask(productID), err)
	}

	brandID := strings.TrimSpace(tok.BrandID)
	mint := strings.TrimSpace(tok.MintAddress)
	tokenTBID := strings.TrimSpace(tok.TokenBlueprintID)
	currentOwner := strings.TrimSpace(tok.ToAddress) // tokens/{productId}.toAddress

	if brandID == "" {
		return TransferByVerifiedScanResult{}, ErrTransferBrandIDEmpty
	}
	if mint == "" {
		return TransferByVerifiedScanResult{}, ErrTransferMintEmpty
	}

	// ✅ safety: tokenBlueprintId も一致を要求（ズレると誤転送になる）
	if tokenTBID != "" && tokenTBID != scannedTBID {
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: tokenBlueprint mismatch productId=%s scanned=%s tokenDoc=%s",
			_mask(productID), _mask(scannedTBID), _mask(tokenTBID),
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
		targetOrderID string
		targetModelID string
	)
	for _, o := range orders {
		if !o.Paid {
			continue
		}
		if mid, ok := findUntransferredItemByModelAndTB(o, scannedModelID, scannedTBID); ok {
			targetOrderID = strings.TrimSpace(o.ID)
			targetModelID = strings.TrimSpace(mid)
			break
		}
	}
	if targetOrderID == "" || targetModelID == "" {
		return TransferByVerifiedScanResult{}, ErrTransferNoEligibleOrder
	}

	now := u.now().UTC()

	// 3) lock (item単位)
	if err := u.orderRepo.LockTransferItem(ctx, targetOrderID, targetModelID, now); err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: lock failed orderId=%s modelId=%s: %w",
			_mask(targetOrderID), _mask(targetModelID), err,
		)
	}

	locked := true
	defer func() {
		// 失敗時に best-effort unlock
		if locked {
			if uerr := u.orderRepo.UnlockTransferItem(context.Background(), targetOrderID, targetModelID); uerr != nil {
				log.Printf("[transfer_uc] WARN: unlock failed orderId=%s modelId=%s err=%v", _mask(targetOrderID), _mask(targetModelID), uerr)
			}
		}
	}()

	// 4) resolve wallets
	fromWallet, err := u.brandWallet.ResolveBrandWalletAddress(ctx, brandID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: resolve brand wallet failed brandId=%s: %w", _mask(brandID), err)
	}
	fromWallet = strings.TrimSpace(fromWallet)
	if fromWallet == "" {
		return TransferByVerifiedScanResult{}, ErrTransferFromWalletEmpty
	}

	toWallet, err := u.avatarWallet.ResolveAvatarWalletAddress(ctx, avatarID)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: resolve avatar wallet failed avatarId=%s: %w", _mask(avatarID), err)
	}
	toWallet = strings.TrimSpace(toWallet)
	if toWallet == "" {
		return TransferByVerifiedScanResult{}, ErrTransferToWalletEmpty
	}

	// 5) ✅ safety: token current owner check (best-effort but recommended)
	// 運用上 tokens.toAddress が必ず「今の owner」なら、
	// ここが brand wallet と一致していない場合は誤転送リスクが高いので止める。
	//
	// ただし legacy で空のケースもあり得るので、空は許容（= キャッシュ未整備）。
	if currentOwner != "" && currentOwner != fromWallet {
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"%w: productId=%s tokenOwner=%s expectedBrandWallet=%s",
			ErrTransferOwnerMismatch, _mask(productID), _mask(currentOwner), _mask(fromWallet),
		)
	}

	// 6) signer（送付元=brand は必須）
	fromSigner, err := u.secrets.GetSigner(ctx, fromWallet)
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf("transfer_uc: get brand signer failed wallet=%s: %w", _mask(fromWallet), err)
	}

	// 受領側 signer は通常不要（必要なら取得して executor に渡す）
	var toSigner any = nil

	// 7) execute transfer (brand -> avatar)
	execOut, err := u.executor.ExecuteTransfer(ctx, ExecuteTransferInput{
		ProductID:        productID,
		AvatarID:         avatarID,
		BrandID:          brandID,
		ModelID:          targetModelID,
		TokenBlueprintID: scannedTBID,

		MintAddress: mint,
		Amount:      1, // ✅ NFT想定

		FromWalletAddress: fromWallet,
		ToWalletAddress:   toWallet,

		FromSigner: fromSigner,
		ToSigner:   toSigner,
	})
	if err != nil {
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: execute transfer failed orderId=%s modelId=%s mint=%s: %w",
			_mask(targetOrderID), _mask(targetModelID), _mask(mint), err,
		)
	}

	tx := strings.TrimSpace(execOut.TxSignature)

	// 8) mark transferred true (item単位)
	if err := u.orderRepo.MarkTransferredItem(ctx, targetOrderID, targetModelID, now); err != nil {
		// transfer 自体は成功しているので、ここは重要（要リカバリ）
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: mark transferred failed orderId=%s modelId=%s tx=%s: %w",
			_mask(targetOrderID), _mask(targetModelID), _mask(tx), err,
		)
	}

	// 9) ✅ update tokens/{productId}.toAddress = avatar wallet (今の owner)
	if err := u.tokenUpdate.UpdateToAddressByProductID(ctx, productID, toWallet, now, tx); err != nil {
		// transfer は成功 & order も transferred 済みなので、ここも重要（要リカバリ）
		return TransferByVerifiedScanResult{}, fmt.Errorf(
			"transfer_uc: update token owner failed productId=%s to=%s tx=%s: %w",
			_mask(productID), _mask(toWallet), _mask(tx), err,
		)
	}

	locked = false

	log.Printf(
		"[transfer_uc] OK productId=%s orderId=%s avatarId=%s brandId=%s modelId=%s tokenBlueprintId=%s mint=%s fromWallet=%s toWallet=%s tx=%s",
		_mask(productID),
		_mask(targetOrderID),
		_mask(avatarID),
		_mask(brandID),
		_mask(targetModelID),
		_mask(scannedTBID),
		_mask(mint),
		_mask(fromWallet),
		_mask(toWallet),
		_mask(tx),
	)

	return TransferByVerifiedScanResult{
		MatchedOrderID: targetOrderID,
		MatchedModelID: targetModelID,

		ProductID:        productID,
		MintAddress:      mint,
		TokenBlueprintID: scannedTBID,

		FromWallet:  fromWallet,
		ToWallet:    toWallet,
		TxSignature: tx,
	}, nil
}

// ============================================================
// Local helpers
// ============================================================

// findUntransferredItemByModelAndTB returns (modelId, true) if order has an item where:
// - item.ModelID == scannedModelID
// - item.Transferred == false
// - tokenBlueprintId extracted from item.InventoryID matches scannedTBID (strict)
func findUntransferredItemByModelAndTB(o orderdom.Order, scannedModelID string, scannedTBID string) (string, bool) {
	m := strings.TrimSpace(scannedModelID)
	tb := strings.TrimSpace(scannedTBID)
	if m == "" || tb == "" {
		return "", false
	}
	for _, it := range o.Items {
		if strings.TrimSpace(it.ModelID) != m {
			continue
		}
		if it.Transferred {
			continue
		}
		itemTB := strings.TrimSpace(extractTokenBlueprintIDFromInventoryID(it.InventoryID))
		if itemTB == "" {
			continue
		}
		if itemTB != tb {
			continue
		}
		return m, true
	}
	return "", false
}

// inventoryId は "__" 区切りで、2つめのセグメントが tokenBlueprintId（あなたの実データ仕様）
func extractTokenBlueprintIDFromInventoryID(inventoryID string) string {
	s := strings.TrimSpace(inventoryID)
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "__")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func _mask(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
