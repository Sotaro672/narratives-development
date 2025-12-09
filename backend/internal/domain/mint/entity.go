// backend/internal/domain/mint/entity.go
package mint

import (
	"errors"
	"strings"
	"time"
)

// ------------------------------------------------------
// Entity: Mint (mints テーブル 1 レコード)
// ------------------------------------------------------
//
// 追加: brandId を保持（tokenBlueprintId と同様、MintRequest 時に必須）
// 追加: inspectionId を保持（inspectionResults:passed の productId を取得した inspections テーブルの ID）
//
// 想定テーブル構造:
//
// - id                 : string
// - inspectionId       : string                // ★ 追加: 元になった inspections ドキュメントID（= productionId）
// - brandId            : string
// - tokenBlueprintId   : string
// - products           : []string
// - createdAt          : time.Time
// - createdBy          : string
// - mintedAt           : *time.Time
// - minted             : bool
// - scheduledBurnDate  : *time.Time           // バーン予定日時・任意
type Mint struct {
	ID string `json:"id"`
	// 検査結果（inspectionResults: passed の productId）を取得した inspections テーブル側の ID
	// 実体としては inspections コレクションのドキュメント ID（= productionId）を想定。
	InspectionID     string     `json:"inspectionId"`
	BrandID          string     `json:"brandId"`
	TokenBlueprintID string     `json:"tokenBlueprintId"`
	Products         []string   `json:"products"`
	CreatedAt        time.Time  `json:"createdAt"`
	CreatedBy        string     `json:"createdBy"`
	MintedAt         *time.Time `json:"mintedAt,omitempty"`
	Minted           bool       `json:"minted"`
	// 任意フィールド: バーン予定日時（未設定なら nil）
	ScheduledBurnDate *time.Time `json:"scheduledBurnDate,omitempty"`
}

// ------------------------------------------------------
// Errors
// ------------------------------------------------------

var (
	ErrInvalidMintID            = errors.New("mint: invalid id")
	ErrInvalidBrandID           = errors.New("mint: invalid brandId")
	ErrInvalidTokenBlueprintID  = errors.New("mint: invalid tokenBlueprintId")
	ErrInvalidProducts          = errors.New("mint: invalid products")
	ErrInvalidCreatedBy         = errors.New("mint: invalid createdBy")
	ErrInvalidCreatedAt         = errors.New("mint: invalid createdAt")
	ErrInvalidMintedAt          = errors.New("mint: invalid mintedAt")
	ErrInconsistentMintedStatus = errors.New("mint: inconsistent minted / mintedAt")
	ErrNotFound                 = errors.New("mint: not found")
)

// ------------------------------------------------------
// Constructors
// ------------------------------------------------------

// NewMint : brandId を追加対応
// ※ 現段階では inspectionId はコンストラクタ引数には含めず、
//
//	必要に応じて usecase / repository 側で別途セットする想定。
func NewMint(
	id string,
	brandID string,
	tokenBlueprintID string,
	products []string,
	createdBy string,
	createdAt time.Time,
) (Mint, error) {

	bid := strings.TrimSpace(brandID)
	if bid == "" {
		return Mint{}, ErrInvalidBrandID
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return Mint{}, ErrInvalidTokenBlueprintID
	}

	prods := normalizeIDList(products)
	if len(prods) == 0 {
		return Mint{}, ErrInvalidProducts
	}

	cb := strings.TrimSpace(createdBy)
	if cb == "" {
		return Mint{}, ErrInvalidCreatedBy
	}

	if createdAt.IsZero() {
		return Mint{}, ErrInvalidCreatedAt
	}

	m := Mint{
		ID:               strings.TrimSpace(id),
		InspectionID:     "", // ★ 後から usecase 側で埋める想定
		BrandID:          bid,
		TokenBlueprintID: tbID,
		Products:         prods,
		CreatedAt:        createdAt.UTC(),
		CreatedBy:        cb,
		MintedAt:         nil,
		Minted:           false,
		// 新規作成時点では ScheduledBurnDate は未定なので nil
		ScheduledBurnDate: nil,
	}

	// 一貫性チェック
	if err := m.validate(); err != nil {
		return Mint{}, err
	}

	return m, nil
}

// ------------------------------------------------------
// internal validation（既存の validate が別ファイルにある前提）
// ------------------------------------------------------
// ここでは validate の定義は変更しない想定。
// InspectionID が空でも許容し、後から埋められるようにしておく。

// ------------------------------------------------------
// Helpers
// ------------------------------------------------------

func normalizeIDList(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))

	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}
