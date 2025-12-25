// backend\internal\domain\inventory\entity.go
package inventory

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound                  = errors.New("inventory not found")
	ErrInvalidMintID             = errors.New("invalid inventory id")
	ErrInvalidTokenBlueprintID   = errors.New("invalid tokenBlueprintID")
	ErrInvalidProductBlueprintID = errors.New("invalid productBlueprintID")
	ErrInvalidModelID            = errors.New("invalid modelID")
	ErrInvalidProducts           = errors.New("invalid products")
)

// ModelStock は modelId ごとの在庫を表します。
// - Products: productId -> true
// - Accumulation: その model の在庫数（= len(Products)）
type ModelStock struct {
	Products     map[string]bool
	Accumulation int
}

// Mint は inventories の 1 ドキュメント（= inventory）を表します。
// 期待値：
// - id: productBlueprintId__tokenBlueprintId
// - stock: modelId ごとに products + accumulation を並列保持
type Mint struct {
	ID                 string
	TokenBlueprintID   string
	ProductBlueprintID string

	// modelId -> { products, accumulation }
	Stock map[string]ModelStock

	// クエリ用（Firestore の array-contains などで検索するための補助）
	ModelIDs []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewMint は「1つの modelId の在庫」を起点に inventories ドキュメントを作るコンストラクタです。
// docId（ID）は repo 側で productBlueprintId__tokenBlueprintId を採用して設定する想定のため、
// ここでは id をそのまま受け取ります（空でも可）。
func NewMint(
	id string,
	tokenBlueprintID string,
	productBlueprintID string,
	modelID string,
	products []string,
	now time.Time,
) (Mint, error) {
	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return Mint{}, ErrInvalidTokenBlueprintID
	}

	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return Mint{}, ErrInvalidProductBlueprintID
	}

	mID := strings.TrimSpace(modelID)
	if mID == "" {
		return Mint{}, ErrInvalidModelID
	}

	ps := normalizeIDs(products)
	if len(ps) == 0 {
		return Mint{}, ErrInvalidProducts
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	// productId -> true
	prodMap := make(map[string]bool, len(ps))
	for _, pid := range ps {
		prodMap[pid] = true
	}

	stock := map[string]ModelStock{
		mID: {
			Products:     prodMap,
			Accumulation: len(prodMap),
		},
	}

	return Mint{
		ID:                 strings.TrimSpace(id),
		TokenBlueprintID:   tbID,
		ProductBlueprintID: pbID,
		Stock:              stock,
		ModelIDs:           []string{mID},
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
