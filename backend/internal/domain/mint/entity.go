// backend/internal/domain/mint/entity.go
package mint

import (
	"errors"
	"time"
)

// ------------------------------------------------------
// Entity: Mint (mints テーブル 1 レコード)
// ------------------------------------------------------
//
// Firestore 上の正しい構造:
//
// - id                 : string
// - brandId            : string
// - tokenBlueprintId   : string
// - products           : []string
// - createdAt          : time.Time
// - createdBy          : string
// - mintedAt           : *time.Time
// - minted             : bool
// - scheduledBurnDate  : *time.Time
// - onChainTxSignature : string
type Mint struct {
	ID string `json:"id"`

	BrandID          string   `json:"brandId"`
	TokenBlueprintID string   `json:"tokenBlueprintId"`
	Products         []string `json:"products"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`

	MintedAt *time.Time `json:"mintedAt,omitempty"`
	Minted   bool       `json:"minted"`

	ScheduledBurnDate *time.Time `json:"scheduledBurnDate,omitempty"`

	OnChainTxSignature string `json:"onChainTxSignature,omitempty"`
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
	productIDs []string,
	createdBy string,
	createdAt time.Time,
) (Mint, error) {
	if brandID == "" {
		return Mint{}, ErrInvalidBrandID
	}

	if tokenBlueprintID == "" {
		return Mint{}, ErrInvalidTokenBlueprintID
	}

	// ここでは productIDs を補正しない。
	// 空文字や不正値は validate() で ErrInvalidProducts として検出する。
	//
	// 0件をエラーにするかどうかは Usecase 側の責務。
	products := productIDs

	if createdBy == "" {
		return Mint{}, ErrInvalidCreatedBy
	}

	if createdAt.IsZero() {
		return Mint{}, ErrInvalidCreatedAt
	}

	m := Mint{
		ID:                 id,
		BrandID:            brandID,
		TokenBlueprintID:   tokenBlueprintID,
		Products:           products,
		CreatedAt:          createdAt.UTC(),
		CreatedBy:          createdBy,
		MintedAt:           nil,
		Minted:             false,
		ScheduledBurnDate:  nil,
		OnChainTxSignature: "",
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
//   - 非空の場合、productId が空文字でないことだけを見る
//   - 件数 0 でエラーにはしない（Usecase 側でチェック済み）
func (m Mint) validate() error {
	if m.BrandID == "" {
		return ErrInvalidBrandID
	}
	if m.TokenBlueprintID == "" {
		return ErrInvalidTokenBlueprintID
	}
	if m.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if m.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}

	if m.Minted && m.MintedAt == nil {
		return ErrInconsistentMintedStatus
	}
	if !m.Minted && m.MintedAt != nil {
		return ErrInconsistentMintedStatus
	}

	for _, productID := range m.Products {
		if productID == "" {
			return ErrInvalidProducts
		}
	}

	return nil
}
