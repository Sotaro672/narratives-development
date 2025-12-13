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
// Firestore 上の想定構造:
//
// - id                 : string
// - brandId            : string
// - tokenBlueprintId   : string
// - products           : map[string]string      // ★ productId → mintAddress（作成時は "" でよい）
// - createdAt          : time.Time
// - createdBy          : string
// - mintedAt           : *time.Time
// - minted             : bool
// - scheduledBurnDate  : *time.Time            // 任意: バーン予定日時
type Mint struct {
	ID string `json:"id"`

	BrandID          string            `json:"brandId"`
	TokenBlueprintID string            `json:"tokenBlueprintId"`
	Products         map[string]string `json:"products"` // ★ productId → mintAddress

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`

	MintedAt *time.Time `json:"mintedAt,omitempty"`
	Minted   bool       `json:"minted"`

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
//
// NewMint : brandId / tokenBlueprintId / products / createdBy / createdAt を受け取って
// Mint エンティティを生成する。
func NewMint(
	id string,
	brandID string,
	tokenBlueprintID string,
	productIDs []string, // ★ production / inspection から取得した productId の一覧
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

	// productId 群を map[productId]mintAddress に詰め替える。
	// 作成時点では mintAddress は未定なので "" を入れておく。
	productMap := normalizeIDListToMap(productIDs)

	// ★ ここでは「0件だとエラー」はチェックしない。
	//    ・Usecase 側 (UpdateRequestInfo) ですでに
	//      「passedProductIDs が 0件ならエラー」をチェックしている
	//    ・ここで ErrInvalidProducts を返すと二重チェックになり、
	//      ログ上の原因切り分けが難しくなる

	cb := strings.TrimSpace(createdBy)
	if cb == "" {
		return Mint{}, ErrInvalidCreatedBy
	}

	if createdAt.IsZero() {
		return Mint{}, ErrInvalidCreatedAt
	}

	m := Mint{
		ID:                strings.TrimSpace(id),
		BrandID:           bid,
		TokenBlueprintID:  tbID,
		Products:          productMap,
		CreatedAt:         createdAt.UTC(),
		CreatedBy:         cb,
		MintedAt:          nil,
		Minted:            false,
		ScheduledBurnDate: nil,
	}

	// 一貫性チェック
	if err := m.validate(); err != nil {
		return Mint{}, err
	}

	return m, nil
}

// ------------------------------------------------------
// validation
// ------------------------------------------------------
//
// Products については：
//   - nil でも OK（empty map と同等扱い）
//   - 非空の場合、キー(productId) が空文字でないことだけを見る
//   - 件数 0 でエラーにはしない（Usecase 側でチェック済み）
func (m Mint) validate() error {
	// （ID は採番を repo 側に任せるケースがあるため、ここでは必須にしない）
	// if strings.TrimSpace(m.ID) == "" { return ErrInvalidMintID }

	if strings.TrimSpace(m.BrandID) == "" {
		return ErrInvalidBrandID
	}
	if strings.TrimSpace(m.TokenBlueprintID) == "" {
		return ErrInvalidTokenBlueprintID
	}
	if m.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if strings.TrimSpace(m.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}

	// minted / mintedAt の整合性チェック
	if m.Minted && m.MintedAt == nil {
		return ErrInconsistentMintedStatus
	}
	if !m.Minted && m.MintedAt != nil {
		return ErrInconsistentMintedStatus
	}

	// products チェック（「ゼロ件 NG」はしない）
	if m.Products != nil {
		for pid := range m.Products {
			if strings.TrimSpace(pid) == "" {
				return ErrInvalidProducts
			}
		}
	}

	return nil
}

// ------------------------------------------------------
// Helpers
// ------------------------------------------------------

// normalizeIDListToMap は raw な productId 配列から
// map[productId]mintAddress(string) を作るヘルパ。
// ・空文字は除外
// ・重複 productId は 1 つにまとめる
// ・mintAddress は作成時点では "" で初期化
func normalizeIDListToMap(raw []string) map[string]string {
	out := make(map[string]string, len(raw))

	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		// すでに登録済みならスキップ（productId はユニーク）
		if _, ok := out[id]; ok {
			continue
		}
		out[id] = "" // mintAddress はミント完了後に埋める想定
	}

	return out
}
