// backend/internal/domain/inventory/entity.go
package inventory

import (
	"errors"
	"sort"
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
// - Products: productId の配列（重複なし・ソート済み）
// - Accumulation: その model の在庫数（= len(Products)）
// - ReservedByOrder: orderId -> qty（予約数）
// - ReservedCount: 予約数合計（= sum(ReservedByOrder)）
type ModelStock struct {
	Products []string
	// Accumulation は「物理在庫数」。products の件数と整合する想定。
	Accumulation int

	// ReservedByOrder は「注文による引当」。
	// 発送時に productId を触れないため、注文時点でここに qty を積む。
	ReservedByOrder map[string]int
	ReservedCount   int
}

// Mint は inventories の 1 ドキュメント（= inventory）を表します。
// 期待値：
// - docId: productBlueprintId__tokenBlueprintId（※ docId 自体の sanitize は永続化層の責務）
// - stock: modelId ごとに products + accumulation + reserved を並列保持
type Mint struct {
	ID                 string
	TokenBlueprintID   string
	ProductBlueprintID string

	// modelId -> { products, accumulation, reservedByOrder, reservedCount }
	Stock map[string]ModelStock

	// クエリ用（Firestore の array-contains などで検索するための補助）
	// 契約：Stock に存在する modelId と整合する（※このドメインでは「直さない」、不正ならエラー）
	ModelIDs []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ------------------------------
// Conceptual Contracts (Domain)
// ------------------------------
//
// 方針（A）:
// - ドメインは “直さない”
// - 不正ならエラー（Validate で弾く）
//
// したがって Normalize は存在しません。
// 入力整形（Trim/重複排除/ソート等）をしたい場合は上位層の責務です。
// ------------------------------

// BuildMintID はドメイン上の識別子規約（productBlueprintId__tokenBlueprintId）を返します。
// 注意：Firestore 等の制約に合わせた sanitize（"/" -> "_" 等）は adapter 層で行ってください。
func BuildMintID(productBlueprintID, tokenBlueprintID string) string {
	if productBlueprintID == "" || tokenBlueprintID == "" {
		return ""
	}
	return productBlueprintID + "__" + tokenBlueprintID
}

// Validate は Mint の必須項目と整合性を検証します。
func (m Mint) Validate() error {
	if m.TokenBlueprintID == "" {
		return ErrInvalidTokenBlueprintID
	}
	if m.ProductBlueprintID == "" {
		return ErrInvalidProductBlueprintID
	}

	// Stock が空はユースケース次第で許容されるので、ここでは必須にしない
	if m.Stock == nil {
		// Stock が無いなら ModelIDs も空/nil であるべき
		if len(m.ModelIDs) != 0 {
			return errors.New("invalid modelIds (stock is nil but modelIds is not empty)")
		}
		return nil
	}

	// Stock の中身検証
	for modelID, ms := range m.Stock {
		if modelID == "" {
			return ErrInvalidModelID
		}
		// products/reserved が両方空なら意味がない（以前は Normalize で drop していたが、今は不正扱い）
		if len(ms.Products) == 0 && len(ms.ReservedByOrder) == 0 {
			return ErrInvalidProducts
		}
		if err := ms.Validate(); err != nil {
			return err
		}
	}

	// ModelIDs 整合性チェック（集合一致）
	if err := validateModelIDsConsistency(m.Stock, m.ModelIDs); err != nil {
		return err
	}

	return nil
}

// Validate は ModelStock の整合性を検証します。
// ※ Normalize はしない（不正ならエラー）
func (ms ModelStock) Validate() error {
	// Products: 空文字なし、ソート済み、重複なし
	if err := validateSortedUniqueNonEmptyStrings(ms.Products); err != nil {
		return ErrInvalidProducts
	}

	// Accumulation は Products の件数と一致する必要がある
	if ms.Accumulation != len(ms.Products) {
		return errors.New("invalid accumulation (must equal len(products))")
	}

	// ReservedByOrder: key は空でない、qty は正、ReservedCount は sum と一致
	var sum int
	for oid, n := range ms.ReservedByOrder {
		if oid == "" || n <= 0 {
			return errors.New("invalid reservedByOrder (empty orderId or non-positive qty)")
		}
		sum += n
	}
	if ms.ReservedCount != sum {
		return errors.New("invalid reservedCount (must equal sum(reservedByOrder))")
	}

	return nil
}

// ------------------------------
// internal helpers (domain-level)
// ------------------------------

func validateSortedUniqueNonEmptyStrings(xs []string) error {
	if len(xs) == 0 {
		return nil
	}
	for _, s := range xs {
		if s == "" {
			return errors.New("contains empty string")
		}
	}
	if !sort.StringsAreSorted(xs) {
		return errors.New("not sorted")
	}
	for i := 1; i < len(xs); i++ {
		if xs[i] == xs[i-1] {
			return errors.New("contains duplicates")
		}
	}
	return nil
}

func validateModelIDsConsistency(stock map[string]ModelStock, modelIDs []string) error {
	// modelIDs 側の基本検証（空・重複・未ソートを弾く）
	if err := validateSortedUniqueNonEmptyStrings(modelIDs); err != nil {
		return errors.New("invalid modelIds (must be sorted, unique, non-empty)")
	}

	stockKeys := make([]string, 0, len(stock))
	stockSet := make(map[string]struct{}, len(stock))
	for k, ms := range stock {
		if k == "" {
			return errors.New("invalid stock key (empty modelId)")
		}
		// products/reserved が両方空なら不正（呼び出し元でも弾いているが二重で守る）
		if len(ms.Products) == 0 && len(ms.ReservedByOrder) == 0 {
			return errors.New("invalid stock (empty model entry)")
		}
		if _, ok := stockSet[k]; ok {
			// map なので通常起きないが一応
			return errors.New("invalid stock (duplicate modelId)")
		}
		stockSet[k] = struct{}{}
		stockKeys = append(stockKeys, k)
	}
	sort.Strings(stockKeys)

	if len(stockKeys) != len(modelIDs) {
		return errors.New("invalid modelIds (not consistent with stock)")
	}
	for i := range stockKeys {
		if stockKeys[i] != modelIDs[i] {
			return errors.New("invalid modelIds (not consistent with stock)")
		}
	}
	return nil
}
