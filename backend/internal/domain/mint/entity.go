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
// - status             : string
// - createdAt          : time.Time
// - createdBy          : string
// - mintedAt           : *time.Time
// - minted             : bool
// - scheduledBurnDate  : *time.Time
// - onChainTxSignature : string
//
// NOTE:
// - 1 product = 1 mint task に分解するため、親 Mint は全体進捗を status で管理します。
// - minted は既存互換のため残します。
// - 新規実装では status を正とし、全 product task が完了した時点で minted=true にします。
type Mint struct {
	ID string `json:"id"`

	BrandID          string   `json:"brandId"`
	TokenBlueprintID string   `json:"tokenBlueprintId"`
	Products         []string `json:"products"`

	Status MintStatus `json:"status"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`

	MintedAt *time.Time `json:"mintedAt,omitempty"`
	Minted   bool       `json:"minted"`

	ScheduledBurnDate *time.Time `json:"scheduledBurnDate,omitempty"`

	// 互換用:
	// 旧フローでは代表tx signatureをここに保持していました。
	// 1件ごとmint化後は product task 側に signature を持たせ、
	// 親には最後に成功した signature または代表 signature を保存します。
	OnChainTxSignature string `json:"onChainTxSignature,omitempty"`
}

// ------------------------------------------------------
// Status
// ------------------------------------------------------

type MintStatus string

const (
	// MintStatusCreated は Mint 親レコード作成直後の状態です。
	MintStatusCreated MintStatus = "CREATED"

	// MintStatusQueued は product 単位の mint task 作成後、
	// worker 実行待ちになった状態です。
	MintStatusQueued MintStatus = "QUEUED"

	// MintStatusMinting は少なくとも1件の product task が処理中の状態です。
	MintStatusMinting MintStatus = "MINTING"

	// MintStatusPartiallyMinted は一部 product が MINTED 済みで、
	// まだ未完了 product が残っている状態です。
	MintStatusPartiallyMinted MintStatus = "PARTIALLY_MINTED"

	// MintStatusMinted は全 product task が MINTED になった状態です。
	MintStatusMinted MintStatus = "MINTED"

	// MintStatusFailedRetryable は一時的な失敗で、再実行可能な状態です。
	MintStatusFailedRetryable MintStatus = "FAILED_RETRYABLE"

	// MintStatusFailedFatal は再実行しても成功しない可能性が高い失敗状態です。
	MintStatusFailedFatal MintStatus = "FAILED_FATAL"
)

func (s MintStatus) IsValid() bool {
	switch s {
	case MintStatusCreated,
		MintStatusQueued,
		MintStatusMinting,
		MintStatusPartiallyMinted,
		MintStatusMinted,
		MintStatusFailedRetryable,
		MintStatusFailedFatal:
		return true
	default:
		return false
	}
}

func (s MintStatus) IsFinished() bool {
	return s == MintStatusMinted ||
		s == MintStatusFailedFatal
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
	ErrInvalidMintStatus        = errors.New("mint: invalid status")
	ErrInconsistentMintedStatus = errors.New("mint: inconsistent minted / mintedAt / status")
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
		Status:             MintStatusCreated,
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
// Behavior
// ------------------------------------------------------

func (m *Mint) MarkQueued() error {
	if m == nil {
		return ErrInvalidMintID
	}

	if m.Minted {
		return ErrInconsistentMintedStatus
	}

	m.Status = MintStatusQueued
	return m.validate()
}

func (m *Mint) MarkMinting() error {
	if m == nil {
		return ErrInvalidMintID
	}

	if m.Minted {
		return ErrInconsistentMintedStatus
	}

	m.Status = MintStatusMinting
	return m.validate()
}

func (m *Mint) MarkPartiallyMinted() error {
	if m == nil {
		return ErrInvalidMintID
	}

	if m.Minted {
		return ErrInconsistentMintedStatus
	}

	m.Status = MintStatusPartiallyMinted
	return m.validate()
}

func (m *Mint) MarkMinted(now time.Time, representativeSignature string) error {
	if m == nil {
		return ErrInvalidMintID
	}
	if now.IsZero() {
		return ErrInvalidMintedAt
	}

	t := now.UTC()
	m.Minted = true
	m.MintedAt = &t
	m.Status = MintStatusMinted

	if representativeSignature != "" {
		m.OnChainTxSignature = representativeSignature
	}

	return m.validate()
}

func (m *Mint) MarkFailedRetryable() error {
	if m == nil {
		return ErrInvalidMintID
	}

	if m.Minted {
		return ErrInconsistentMintedStatus
	}

	m.Status = MintStatusFailedRetryable
	return m.validate()
}

func (m *Mint) MarkFailedFatal() error {
	if m == nil {
		return ErrInvalidMintID
	}

	if m.Minted {
		return ErrInconsistentMintedStatus
	}

	m.Status = MintStatusFailedFatal
	return m.validate()
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

	status := m.Status
	if status == "" {
		status = MintStatusCreated
	}

	if !status.IsValid() {
		return ErrInvalidMintStatus
	}

	if m.Minted {
		if m.MintedAt == nil {
			return ErrInconsistentMintedStatus
		}
		if status != MintStatusMinted {
			return ErrInconsistentMintedStatus
		}
	}

	if !m.Minted {
		if m.MintedAt != nil {
			return ErrInconsistentMintedStatus
		}
		if status == MintStatusMinted {
			return ErrInconsistentMintedStatus
		}
	}

	for _, productID := range m.Products {
		if productID == "" {
			return ErrInvalidProducts
		}
	}

	return nil
}
