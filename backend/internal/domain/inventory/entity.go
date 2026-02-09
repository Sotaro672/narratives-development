// backend/internal/domain/inventory/entity.go
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
	// 概念的契約：Stock に存在する modelId と整合する（Normalize で再生成される）
	ModelIDs []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ------------------------------
// Conceptual Contracts (Domain)
// ------------------------------
//
// このファイルに追加する「概念的契約」は、永続化（Firestore）や DTO ではなく、
// ドメインの整合性（不変条件）を保証するためのものです。
//
// - Normalize(): 余計な空白や重複、0以下の予約などを正規化し、整合性を揃える
// - Validate(): 必須項目と整合性をチェックする（Normalize 後の利用を推奨）
// - BuildMintID(): Mint の識別子規約（productBlueprintId__tokenBlueprintId）を表す
//   ※ "/" 置換などストレージ都合の sanitize は adapter 層の責務
//

// BuildMintID はドメイン上の識別子規約（productBlueprintId__tokenBlueprintId）を返します。
// 注意：Firestore 等の制約に合わせた sanitize（"/" -> "_" 等）は adapter 層で行ってください。
func BuildMintID(productBlueprintID, tokenBlueprintID string) string {
	pb := strings.TrimSpace(productBlueprintID)
	tb := strings.TrimSpace(tokenBlueprintID)
	if pb == "" || tb == "" {
		return ""
	}
	return pb + "__" + tb
}

// Normalize は Mint の整合性（不変条件）を揃えたコピーを返します。
// - ID / BlueprintID / ModelIDs の Trim
// - Stock の各 ModelStock を正規化
// - 空の model を drop
// - ModelIDs を Stock から再生成（不整合を防ぐ）
// - UpdatedAt がゼロなら CreatedAt と合わせる（CreatedAt がゼロなら触らない）
func (m Mint) Normalize() Mint {
	m.ID = strings.TrimSpace(m.ID)
	m.TokenBlueprintID = strings.TrimSpace(m.TokenBlueprintID)
	m.ProductBlueprintID = strings.TrimSpace(m.ProductBlueprintID)

	// Stock 正規化
	if m.Stock != nil {
		out := map[string]ModelStock{}
		for modelID, ms := range m.Stock {
			modelID = strings.TrimSpace(modelID)
			if modelID == "" {
				continue
			}
			nms := ms.Normalize()

			hasProducts := len(nms.Products) > 0
			hasReserved := len(nms.ReservedByOrder) > 0
			if !hasProducts && !hasReserved {
				continue
			}
			out[modelID] = nms
		}
		if len(out) == 0 {
			m.Stock = nil
		} else {
			m.Stock = out
		}
	}

	// ModelIDs は Stock から再生成（概念的契約）
	m.ModelIDs = modelIDsFromStock(m.Stock)

	// CreatedAt/UpdatedAt の軽い整合（ドメインとしての最低限）
	if !m.CreatedAt.IsZero() && m.UpdatedAt.IsZero() {
		m.UpdatedAt = m.CreatedAt
	}
	if m.CreatedAt.IsZero() && !m.UpdatedAt.IsZero() {
		// CreatedAt が無いのに UpdatedAt だけあるのは変なので寄せる
		m.CreatedAt = m.UpdatedAt
	}

	return m
}

// Validate は Mint の必須項目と整合性を検証します。
// 推奨：保存・処理の境界で m = m.Normalize() してから Validate() を呼ぶ。
func (m Mint) Validate() error {
	if strings.TrimSpace(m.TokenBlueprintID) == "" {
		return ErrInvalidTokenBlueprintID
	}
	if strings.TrimSpace(m.ProductBlueprintID) == "" {
		return ErrInvalidProductBlueprintID
	}

	// 在庫が空はユースケース次第で許容されることもあるが、
	// ここでは「Mint として意味がある」最低条件として Stock を必須にしない。
	// 必須にしたい場合はユースケース側で判定する。
	if m.Stock == nil {
		return nil
	}

	for modelID, ms := range m.Stock {
		if strings.TrimSpace(modelID) == "" {
			return ErrInvalidModelID
		}
		if err := ms.Validate(); err != nil {
			return err
		}
	}

	// ModelIDs は補助フィールド。整合性は Normalize で担保する想定だが、Validate でも軽く確認
	want := modelIDsFromStock(m.Stock)
	got := normalizeModelIDs(m.ModelIDs)
	if !stringSliceEqual(want, got) {
		// ここでは専用エラーを増やさず、ドメイン整合性違反として既存の ErrInvalidProducts を流用しない。
		// 必要なら ErrInvalidModelIDs を追加してください。
		return errors.New("invalid modelIds (not consistent with stock)")
	}

	return nil
}

// Normalize は ModelStock の整合性（不変条件）を揃えたコピーを返します。
// - Products の Trim / 重複排除 / ソート
// - Accumulation を len(Products) に一致させる
// - ReservedByOrder の key Trim / qty>0 のみ残す
// - ReservedCount を sum(ReservedByOrder) に一致させる
func (ms ModelStock) Normalize() ModelStock {
	ms.Products = normalizeIDs(ms.Products)
	if len(ms.Products) == 0 {
		ms.Products = nil
	}
	ms.Accumulation = len(ms.Products)

	if ms.ReservedByOrder != nil {
		rbo := map[string]int{}
		var sum int
		for oid, n := range ms.ReservedByOrder {
			oid = strings.TrimSpace(oid)
			if oid == "" || n <= 0 {
				continue
			}
			rbo[oid] = n
			sum += n
		}
		if len(rbo) == 0 {
			ms.ReservedByOrder = nil
			ms.ReservedCount = 0
		} else {
			ms.ReservedByOrder = rbo
			ms.ReservedCount = sum
		}
	} else {
		ms.ReservedCount = 0
	}

	return ms
}

// Validate は ModelStock の整合性を検証します。
// 推奨：境界で Normalize() 後に呼ぶ（Normalize が契約の中心）。
func (ms ModelStock) Validate() error {
	// Products が空でも予約があれば意味があるケースはあり得るので、ここでは products 必須にしない。
	// ただし Products があれば必ず Accumulation と一致する必要がある。
	ps := normalizeIDs(ms.Products)
	if len(ps) != len(ms.Products) {
		// ここで厳密に弾くかは好みだが、契約違反として検知
		return ErrInvalidProducts
	}
	if ms.Accumulation != len(ms.Products) {
		return errors.New("invalid accumulation (must equal len(products))")
	}

	// ReservedCount 整合
	var sum int
	for oid, n := range ms.ReservedByOrder {
		oid = strings.TrimSpace(oid)
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

	stock := map[string]ModelStock{
		mID: {
			Products:     ps,
			Accumulation: len(ps),

			ReservedByOrder: map[string]int{},
			ReservedCount:   0,
		},
	}

	out := Mint{
		ID:                 strings.TrimSpace(id),
		TokenBlueprintID:   tbID,
		ProductBlueprintID: pbID,
		Stock:              stock,
		ModelIDs:           []string{mID},
		CreatedAt:          now,
		UpdatedAt:          now,
	}.Normalize()

	// コンストラクタは契約を満たすものを返したいので Validate
	if err := out.Validate(); err != nil {
		return Mint{}, err
	}
	return out, nil
}

// ------------------------------
// internal helpers (domain-level)
// ------------------------------

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

func normalizeModelIDs(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
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
	if len(out) == 0 {
		return nil
	}
	return out
}

func modelIDsFromStock(stock map[string]ModelStock) []string {
	if stock == nil {
		return nil
	}
	out := make([]string, 0, len(stock))
	for modelID, ms := range stock {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		// products/reserved が両方空なら Normalize で消える想定だが、念のため
		if len(ms.Products) == 0 && len(ms.ReservedByOrder) == 0 {
			continue
		}
		out = append(out, modelID)
	}
	return normalizeModelIDs(out)
}

func stringSliceEqual(a, b []string) bool {
	aa := normalizeModelIDs(a)
	bb := normalizeModelIDs(b)
	if len(aa) != len(bb) {
		return false
	}
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
