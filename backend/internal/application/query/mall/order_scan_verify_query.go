// backend/internal/application/query/mall/order_scan_verify_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

/*
責任と機能:
- preview_query.go のスキャン結果（= productId から解決した modelId と tokenBlueprintId）と、
  order_purchased_query.go の検索結果（= avatarId の購入済み(paid=true)かつ items.transfer=false の (modelId, tokenBlueprintId) 集合）を突合する。
- 一致判定:
  - scannedModelId と scannedTokenBlueprintId が、purchasedPairs のどれか1つと完全一致すれば OK。
- Firestore への直接依存は持たず、既存 Query を合成して検証する（Query Orchestration）。

今回の修正ポイント（あなたの要件）:
- scannedModelId は products/{productId}.modelId（= PreviewQ が productId から解決する modelId）を使う
- scannedTokenBlueprintId は tokens/{productId}.tokenBlueprintId（docId=productId）を使う
- productBlueprintId への fallback は廃止（ここがズレの原因だったため）
*/

var (
	ErrOrderScanVerifyQueryNotConfigured  = errors.New("order_scan_verify_query: not configured")
	ErrOrderScanVerifyAvatarIDEmpty       = errors.New("order_scan_verify_query: avatarId is empty")
	ErrOrderScanVerifyProductIDEmpty      = errors.New("order_scan_verify_query: productId is empty")
	ErrOrderScanVerifyTokenNotFound       = errors.New("order_scan_verify_query: token not found for productId")
	ErrOrderScanVerifyTokenBlueprintEmpty = errors.New("order_scan_verify_query: tokenBlueprintId is empty")
)

// ModelTokenPair is a minimal pair used for matching.
type ModelTokenPair struct {
	ModelID          string `json:"modelId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

// PurchasedPairsProvider is the minimal interface we need from order_purchased_query.go.
type PurchasedPairsProvider interface {
	ListEligiblePairsByAvatarID(ctx context.Context, avatarID string) (OrderPurchasedResult, error)
}

// ScanResultProvider is the minimal interface we need from preview_query.go.
// IMPORTANT:
// - PreviewModelInfo.Token は tokens/{productId} 由来の情報を含むこと（docId=productId）
// - TokenInfo に TokenBlueprintID が載っていること（= tokens/{productId}.tokenBlueprintId）
type ScanResultProvider interface {
	ResolveModelInfoByProductID(ctx context.Context, productID string) (*PreviewModelInfo, error)
}

type OrderScanVerifyQuery struct {
	PurchasedQ PurchasedPairsProvider
	PreviewQ   ScanResultProvider
}

func NewOrderScanVerifyQuery(purchasedQ PurchasedPairsProvider, previewQ ScanResultProvider) *OrderScanVerifyQuery {
	return &OrderScanVerifyQuery{
		PurchasedQ: purchasedQ,
		PreviewQ:   previewQ,
	}
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

// ✅ A案: handler 側の ScanVerifyQuery interface に合わせるためのメソッドを追加
// - 実体は VerifyMatch を呼ぶ薄いアダプタ
func (q *OrderScanVerifyQuery) VerifyScanPurchasedByAvatarID(ctx context.Context, avatarID string, productID string) (VerifyResult, error) {
	return q.VerifyMatch(ctx, VerifyInput{
		AvatarID:  avatarID,
		ProductID: productID,
	})
}

// VerifyMatch verifies whether the scanned pair exists in purchased(untransferred) pairs.
func (q *OrderScanVerifyQuery) VerifyMatch(ctx context.Context, in VerifyInput) (VerifyResult, error) {
	if q == nil || q.PurchasedQ == nil || q.PreviewQ == nil {
		return VerifyResult{}, ErrOrderScanVerifyQueryNotConfigured
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	productID := strings.TrimSpace(in.ProductID)

	if avatarID == "" {
		return VerifyResult{}, ErrOrderScanVerifyAvatarIDEmpty
	}
	if productID == "" {
		return VerifyResult{}, ErrOrderScanVerifyProductIDEmpty
	}

	// 1) scan side: productId -> modelId + tokenBlueprintId(tokens/{productId}.tokenBlueprintId)
	info, err := q.PreviewQ.ResolveModelInfoByProductID(ctx, productID)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("order_scan_verify_query: preview resolve failed: %w", err)
	}
	if info == nil {
		return VerifyResult{}, fmt.Errorf("order_scan_verify_query: preview resolve returned nil")
	}

	scannedModelID := strings.TrimSpace(info.ModelID)
	if scannedModelID == "" {
		return VerifyResult{}, fmt.Errorf("order_scan_verify_query: scanned modelId is empty")
	}

	// token must exist (tokens/{productId} が存在する or TokenRepo 注入されている)
	if info.Token == nil {
		return VerifyResult{}, ErrOrderScanVerifyTokenNotFound
	}

	// scanned tokenBlueprintId is tokens/{productId}.tokenBlueprintId (docId=productId)
	scannedTBID := strings.TrimSpace(info.Token.TokenBlueprintID)
	if scannedTBID == "" {
		return VerifyResult{}, ErrOrderScanVerifyTokenBlueprintEmpty
	}

	// 2) purchased side: avatarId -> paid orders -> items.transfer=false -> (modelId,tbId)
	purchased, err := q.PurchasedQ.ListEligiblePairsByAvatarID(ctx, avatarID)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("order_scan_verify_query: purchased pairs resolve failed: %w", err)
	}

	// 3) dedup to []ModelTokenPair
	seen := map[string]struct{}{}
	outPairs := make([]ModelTokenPair, 0, len(purchased.Pairs))
	for _, p := range purchased.Pairs {
		mid := strings.TrimSpace(p.ModelID)
		tbid := strings.TrimSpace(p.TokenBlueprintID)
		if mid == "" || tbid == "" {
			continue
		}
		k := mid + "::" + tbid
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		outPairs = append(outPairs, ModelTokenPair{ModelID: mid, TokenBlueprintID: tbid})
	}

	// 4) match
	var match *ModelTokenPair
	for i := range outPairs {
		p := outPairs[i]
		if p.ModelID == scannedModelID && p.TokenBlueprintID == scannedTBID {
			cp := p // copy
			match = &cp
			break
		}
	}

	return VerifyResult{
		AvatarID:                avatarID,
		ProductID:               productID,
		ScannedModelID:          scannedModelID,
		ScannedTokenBlueprintID: scannedTBID,
		PurchasedPairs:          outPairs,
		Matched:                 match != nil,
		Match:                   match,
	}, nil
}
