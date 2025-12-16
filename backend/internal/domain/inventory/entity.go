// backend/internal/domain/inventory/entity.go
package inventory

import (
	"errors"
	"sort"
	"strings"
	"time"
)

// ------------------------------------------------------
// Entity: Mint (inventories / mints テーブル 1 レコード)
// ------------------------------------------------------
//
// Firestore 上の想定構造（★この定義を正）:
//
// - id                 : string
// - tokenBlueprintId   : string
// - productBlueprintId : string
// - products           : []string            // ★ productId のみ
// - accumulation       : integer
// - createdAt          : time.Time
// - updatedAt          : time.Time
type Mint struct {
	ID string `json:"id"`

	TokenBlueprintID   string   `json:"tokenBlueprintId"`
	ProductBlueprintID string   `json:"productBlueprintId"`
	Products           []string `json:"products"` // ★ productId のみ

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

	products := normalizeIDs(productIDs)

	ca := createdAt.UTC()

	m := Mint{
		ID:                 strings.TrimSpace(id),
		TokenBlueprintID:   tbID,
		ProductBlueprintID: pbID,
		Products:           products,
		Accumulation:       accumulation,
		CreatedAt:          ca,
		UpdatedAt:          ca,
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
//   - nil でも OK（empty slice と同等扱い）
//   - 非空の場合、要素が空文字でないこと
//   - 重複は許容しない（normalize で除去される想定だが念のため弾く）
func (m Mint) validate() error {
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

	if m.Products != nil {
		seen := map[string]struct{}{}
		for _, pid := range m.Products {
			pid = strings.TrimSpace(pid)
			if pid == "" {
				return ErrInvalidProducts
			}
			if _, ok := seen[pid]; ok {
				return ErrInvalidProducts
			}
			seen[pid] = struct{}{}
		}
	}

	return nil
}

// ------------------------------------------------------
// Helpers
// ------------------------------------------------------

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
