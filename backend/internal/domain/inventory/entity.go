// backend\internal\domain\inventory\entity.go
package inventory

import (
	"errors"
	"strings"
	"time"
)

// ------------------------------------------------------
// Entity: Mint (mints テーブル 1 レコード)
// ------------------------------------------------------
//
// Firestore 上の想定構造（★この定義を正）:
//
// - id                 : string
// - tokenBlueprintId   : string
// - productBlueprintId : string
// - products           : map[string]string      // ★ productId → mintAddress（作成時は "" でよい）
// - accumulation       : integer
// - createdAt          : time.Time
// - updatedAt          : time.Time
type Mint struct {
	ID string `json:"id"`

	TokenBlueprintID   string            `json:"tokenBlueprintId"`
	ProductBlueprintID string            `json:"productBlueprintId"`
	Products           map[string]string `json:"products"` // ★ productId → mintAddress

	Accumulation int `json:"accumulation"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ------------------------------------------------------
// Errors
// ------------------------------------------------------

var (
	ErrInvalidMintID             = errors.New("mint: invalid id")
	ErrInvalidTokenBlueprintID   = errors.New("mint: invalid tokenBlueprintId")
	ErrInvalidProductBlueprintID = errors.New("mint: invalid productBlueprintId")
	ErrInvalidProducts           = errors.New("mint: invalid products")
	ErrInvalidAccumulation       = errors.New("mint: invalid accumulation")
	ErrInvalidCreatedAt          = errors.New("mint: invalid createdAt")
	ErrInvalidUpdatedAt          = errors.New("mint: invalid updatedAt")
	ErrNotFound                  = errors.New("mint: not found")
)

// ------------------------------------------------------
// Constructors
// ------------------------------------------------------
//
// NewMint : tokenBlueprintId / productBlueprintId / products / accumulation / createdAt を受け取って
// Mint エンティティを生成する。
func NewMint(
	id string,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string, // ★ productBlueprint から取得した productId の一覧
	accumulation int,
	createdAt time.Time,
) (Mint, error) {

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return Mint{}, ErrInvalidTokenBlueprintID
	}

	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return Mint{}, ErrInvalidProductBlueprintID
	}

	if accumulation < 0 {
		return Mint{}, ErrInvalidAccumulation
	}

	if createdAt.IsZero() {
		return Mint{}, ErrInvalidCreatedAt
	}

	// productId 群を map[productId]mintAddress に詰め替える。
	// 作成時点では mintAddress は未定なので "" を入れておく。
	productMap := normalizeIDListToMap(productIDs)

	ca := createdAt.UTC()

	m := Mint{
		ID:                 strings.TrimSpace(id),
		TokenBlueprintID:   tbID,
		ProductBlueprintID: pbID,
		Products:           productMap,
		Accumulation:       accumulation,
		CreatedAt:          ca,
		UpdatedAt:          ca, // 作成時は createdAt と同一で初期化
	}

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
//   - 件数 0 でエラーにはしない（Usecase 側でチェックする前提）
func (m Mint) validate() error {
	// （ID は採番を repo 側に任せるケースがあるため、ここでは必須にしない）
	// if strings.TrimSpace(m.ID) == "" { return ErrInvalidMintID }

	if strings.TrimSpace(m.TokenBlueprintID) == "" {
		return ErrInvalidTokenBlueprintID
	}
	if strings.TrimSpace(m.ProductBlueprintID) == "" {
		return ErrInvalidProductBlueprintID
	}

	if m.Accumulation < 0 {
		return ErrInvalidAccumulation
	}

	if m.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if m.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if m.UpdatedAt.Before(m.CreatedAt) {
		return ErrInvalidUpdatedAt
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
