// backend/internal/application/usecase/transfer_usecase.go
package usecase

/*
責任と機能:
- 「スキャン結果（modelId, tokenBlueprintId）」と「購入済み未移転（= item単位で transferred=false）の注文（orders）」を突合し、
  一致した場合にブランド（brandId）の walletAddress へ token transfer を申請/実行する。
- 競合（二重処理）を防ぐために、order の item 単位で transfer lock を取得してから処理する。
- transfer 成功後に orders/{orderId} の該当 item の transferred/transferredAt を更新する。
  （order.Paid は true 前提。transfer対象 item は transferred=false のものだけ）
- 外部依存（Firestore/SecretManager/Solana/TokenOperation 等）は Port(interface) に閉じ込め、
  Usecase は「手順（オーケストレーション）」のみを担う。
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

// OrderRepoForTransfer is the minimal port needed for transfer orchestration.
//
// ✅ 注意: Order の transferred/transferredAt は廃止し、item 単位で保持する。
// よって repo は「paid=true の注文一覧」を返し、Usecase 側で item.transferred を見て対象を決める。
//
// Lock/Mark は item 単位の排他・確定更新をトランザクションで担保する想定。
// itemKey は現状 modelId を採用（同一order内で modelId が重複しない前提）。
// ※ もし同一order内で同一modelIdが複数行になり得るなら、itemKey を itemIndex / productId 等に変更してください。
type OrderRepoForTransfer interface {
	ListPaidByAvatarID(ctx context.Context, avatarID string) ([]orderdom.Order, error)

	LockTransferItem(ctx context.Context, orderID string, itemModelID string, now time.Time) error
	UnlockTransferItem(ctx context.Context, orderID string, itemModelID string) error

	MarkTransferredItem(ctx context.Context, orderID string, itemModelID string, at time.Time) error
}

// ModelTokenBlueprintResolver resolves tokenBlueprintId for a given modelId.
// （orders.items には tokenBlueprintId が無いので、別Repo/Queryで引く想定）
type ModelTokenBlueprintResolver interface {
	ResolveTokenBlueprintIDByModelID(ctx context.Context, modelID string) (string, error)
}

// BrandWalletResolver resolves brand walletAddress by brandId.
type BrandWalletResolver interface {
	ResolveBrandWalletAddress(ctx context.Context, brandID string) (string, error)
}

// WalletSecretProvider provides a signing capability for a walletAddress.
//
// 実装側では Secret Manager / GCS 等から秘密鍵を取得し、署名に使える形にする。
// Usecase は秘密鍵のフォーマットに依存しない。
type WalletSecretProvider interface {
	// GetSigner returns an opaque signer object (implementation-defined).
	// 例: solana.PrivateKey / ed25519.PrivateKey / etc
	GetSigner(ctx context.Context, walletAddress string) (any, error)
}

// TokenTransferExecutor executes transfer using signers.
//
// “申請された側が自動署名”の期待値に合わせ、
// 実装は「申請→受領側署名→実行」まで内包してよい（Usecase は 1 回呼ぶだけ）。
type TokenTransferExecutor interface {
	ExecuteTransfer(ctx context.Context, in ExecuteTransferInput) (ExecuteTransferResult, error)
}

type ExecuteTransferInput struct {
	// business identifiers
	OrderID          string
	AvatarID         string
	BrandID          string
	ModelID          string
	TokenBlueprintID string

	// destination
	ToWalletAddress string

	// signers
	FromSigner any
	ToSigner   any
}

type ExecuteTransferResult struct {
	TxSignature string
}

// ============================================================
// Usecase
// ============================================================

type TransferUsecase struct {
	orderRepo   OrderRepoForTransfer
	modelQ      ModelTokenBlueprintResolver
	brandWallet BrandWalletResolver
	secrets     WalletSecretProvider
	executor    TokenTransferExecutor

	now func() time.Time
}

func NewTransferUsecase(
	orderRepo OrderRepoForTransfer,
	modelQ ModelTokenBlueprintResolver,
	brandWallet BrandWalletResolver,
	secrets WalletSecretProvider,
	executor TokenTransferExecutor,
) *TransferUsecase {
	return &TransferUsecase{
		orderRepo:   orderRepo,
		modelQ:      modelQ,
		brandWallet: brandWallet,
		secrets:     secrets,
		executor:    executor,
		now:         time.Now,
	}
}

var (
	ErrTransferNotConfigured         = errors.New("transfer_uc: not configured")
	ErrTransferAvatarIDEmpty         = errors.New("transfer_uc: avatarId is empty")
	ErrTransferBrandIDEmpty          = errors.New("transfer_uc: brandId is empty")
	ErrTransferModelIDEmpty          = errors.New("transfer_uc: modelId is empty")
	ErrTransferTokenBlueprintIDEmpty = errors.New("transfer_uc: tokenBlueprintId is empty")
	ErrTransferNoEligibleOrder       = errors.New("transfer_uc: no eligible order/item found")
)

// TransferByScanInput is the entry point input.
// スキャン結果（modelId/tokenBlueprintId）をもとに、avatarId の購入済み未移転(item)を探し、brand へ移転する。
type TransferByScanInput struct {
	AvatarID         string
	BrandID          string
	ModelID          string
	TokenBlueprintID string
}

type TransferByScanResult struct {
	MatchedOrderID string
	MatchedModelID string
	TxSignature    string
	ToWallet       string
}

// TransferToBrandByScan does:
// 1) orders を avatarId + paid=true で検索（itemの transferred 判定は usecase 側）
// 2) 各 order の items にスキャンした modelId かつ item.transferred=false があるか確認
// 3) modelId から tokenBlueprintId を引いて、スキャン結果 tokenBlueprintId と一致するか確認
// 4) 一致した item を lock（排他）
// 5) brand walletAddress を解決し、必要な signer を取得
// 6) transfer 実行
// 7) orders/{orderId} の該当 item transferred=true を保存（成功確定）
// 8) 失敗時は unlock（best-effort）
func (u *TransferUsecase) TransferToBrandByScan(ctx context.Context, in TransferByScanInput) (TransferByScanResult, error) {
	if u == nil || u.orderRepo == nil || u.modelQ == nil || u.brandWallet == nil || u.secrets == nil || u.executor == nil {
		return TransferByScanResult{}, ErrTransferNotConfigured
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	brandID := strings.TrimSpace(in.BrandID)
	modelID := strings.TrimSpace(in.ModelID)
	tbID := strings.TrimSpace(in.TokenBlueprintID)

	if avatarID == "" {
		return TransferByScanResult{}, ErrTransferAvatarIDEmpty
	}
	if brandID == "" {
		return TransferByScanResult{}, ErrTransferBrandIDEmpty
	}
	if modelID == "" {
		return TransferByScanResult{}, ErrTransferModelIDEmpty
	}
	if tbID == "" {
		return TransferByScanResult{}, ErrTransferTokenBlueprintIDEmpty
	}

	orders, err := u.orderRepo.ListPaidByAvatarID(ctx, avatarID)
	if err != nil {
		return TransferByScanResult{}, err
	}
	if len(orders) == 0 {
		return TransferByScanResult{}, ErrTransferNoEligibleOrder
	}

	// 念のため model->tokenBlueprint を検証（スキャン値と一致しないなら誤転送防止）
	resolvedTB, err := u.modelQ.ResolveTokenBlueprintIDByModelID(ctx, modelID)
	if err != nil {
		return TransferByScanResult{}, fmt.Errorf("transfer_uc: resolve tokenBlueprintId by modelId failed: %w", err)
	}
	resolvedTB = strings.TrimSpace(resolvedTB)
	if resolvedTB == "" {
		return TransferByScanResult{}, fmt.Errorf("transfer_uc: tokenBlueprintId empty for modelId=%s", _mask(modelID))
	}
	if resolvedTB != tbID {
		return TransferByScanResult{}, fmt.Errorf(
			"transfer_uc: tokenBlueprint mismatch model=%s resolved=%s scanned=%s",
			_mask(modelID), _mask(resolvedTB), _mask(tbID),
		)
	}

	// order 探索：paid=true かつ items に (modelId一致 & transferred=false) があるもの
	var (
		targetOrderID string
		targetModelID string
	)
	for _, o := range orders {
		if !o.Paid {
			continue
		}
		if m, ok := findUntransferredItemModelID(o, modelID); ok {
			targetOrderID = strings.TrimSpace(o.ID)
			targetModelID = m
			break
		}
	}
	if targetOrderID == "" || targetModelID == "" {
		return TransferByScanResult{}, ErrTransferNoEligibleOrder
	}

	now := u.now().UTC()

	// 4) lock (item単位)
	if err := u.orderRepo.LockTransferItem(ctx, targetOrderID, targetModelID, now); err != nil {
		return TransferByScanResult{}, fmt.Errorf("transfer_uc: lock failed orderId=%s modelId=%s: %w", _mask(targetOrderID), _mask(targetModelID), err)
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

	// 5) brand wallet resolve
	toWallet, err := u.brandWallet.ResolveBrandWalletAddress(ctx, brandID)
	if err != nil {
		return TransferByScanResult{}, fmt.Errorf("transfer_uc: resolve brand wallet failed brandId=%s: %w", _mask(brandID), err)
	}
	toWallet = strings.TrimSpace(toWallet)
	if toWallet == "" {
		return TransferByScanResult{}, fmt.Errorf("transfer_uc: empty brand walletAddress brandId=%s", _mask(brandID))
	}

	// signer 取得（受領側 signer（brand）を必須）
	toSigner, err := u.secrets.GetSigner(ctx, toWallet)
	if err != nil {
		return TransferByScanResult{}, fmt.Errorf("transfer_uc: get brand signer failed wallet=%s: %w", _mask(toWallet), err)
	}

	// FromSigner（送付側）が必要ならここで取得する（現状は executor 側で解決してもよい）
	var fromSigner any = nil

	// 6) execute transfer
	out, err := u.executor.ExecuteTransfer(ctx, ExecuteTransferInput{
		OrderID:          targetOrderID,
		AvatarID:         avatarID,
		BrandID:          brandID,
		ModelID:          targetModelID,
		TokenBlueprintID: tbID,
		ToWalletAddress:  toWallet,
		FromSigner:       fromSigner,
		ToSigner:         toSigner,
	})
	if err != nil {
		return TransferByScanResult{}, fmt.Errorf("transfer_uc: execute transfer failed orderId=%s modelId=%s: %w", _mask(targetOrderID), _mask(targetModelID), err)
	}

	// 7) mark transferred true (item単位)
	if err := u.orderRepo.MarkTransferredItem(ctx, targetOrderID, targetModelID, now); err != nil {
		// transfer 自体は成功しているので、ここは重要（要リカバリ）
		return TransferByScanResult{}, fmt.Errorf(
			"transfer_uc: mark transferred failed orderId=%s modelId=%s tx=%s: %w",
			_mask(targetOrderID), _mask(targetModelID), _mask(out.TxSignature), err,
		)
	}

	locked = false

	log.Printf("[transfer_uc] OK orderId=%s avatarId=%s brandId=%s modelId=%s tokenBlueprintId=%s toWallet=%s tx=%s",
		_mask(targetOrderID), _mask(avatarID), _mask(brandID), _mask(targetModelID), _mask(tbID), _mask(toWallet), _mask(out.TxSignature),
	)

	return TransferByScanResult{
		MatchedOrderID: targetOrderID,
		MatchedModelID: targetModelID,
		TxSignature:    out.TxSignature,
		ToWallet:       toWallet,
	}, nil
}

// ============================================================
// Local helpers
// ============================================================

// findUntransferredItemModelID returns (modelId, true) if order has an item where:
// - item.ModelID == scanned modelID
// - item.Transferred == false
func findUntransferredItemModelID(o orderdom.Order, scannedModelID string) (string, bool) {
	m := strings.TrimSpace(scannedModelID)
	if m == "" {
		return "", false
	}
	for _, it := range o.Items {
		if strings.TrimSpace(it.ModelID) != m {
			continue
		}
		// ✅ item単位のTransferredを見る（order.Transferred は廃止）
		if it.Transferred {
			continue
		}
		return m, true
	}
	return "", false
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
