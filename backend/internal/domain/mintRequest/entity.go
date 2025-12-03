// backend/internal/domain/mintRequest/entity.go
package mintrequest

import (
	"errors"
	"strings"
	"time"
)

// ===============================
// Status
// ===============================

// MintRequestStatus mirrors TS: 'planning' | 'requested' | 'minted'
type MintRequestStatus string

const (
	StatusPlanning  MintRequestStatus = "planning"
	StatusRequested MintRequestStatus = "requested"
	StatusMinted    MintRequestStatus = "minted"
)

// IsValidStatus は有効なステータスかどうかを判定します。
func IsValidStatus(s MintRequestStatus) bool {
	return s == StatusPlanning || s == StatusRequested || s == StatusMinted
}

// ===============================
// Entity
// ===============================
//
// InspectionBatch (inspection/entity.go) と同じ思想で、
// 「生産単位での NFT ミント要求」を表すエンティティ。
//
//	export interface MintRequest {
//	  id: string
//	  tokenBlueprintId: string | null        -> TokenBlueprintID *string
//	  productionId: string                  -> ProductionID
//	  mintQuantity: number                  -> MintQuantity
//	  burnDate: string | null               -> ScheduledBurnDate *time.Time
//	  status: MintRequestStatus             -> Status
//	  requestedBy: string | null            -> RequestedBy *string
//	  requestedAt: string | null            -> RequestedAt *time.Time
//	  mintedAt?: string | null              -> MintedAt *time.Time
//	}
//
// createdAt/updatedAt/deletedAt などの監査項目は InspectionBatch と同様、
// Firestore メタデータや別レイヤで扱う前提でドメインからは外しています。
type MintRequest struct {
	ID string `json:"id"`

	ProductionID string            `json:"productionId"`
	Status       MintRequestStatus `json:"status"`

	MintQuantity int `json:"mintQuantity"`

	RequestedBy       *string    `json:"requestedBy"`
	RequestedAt       *time.Time `json:"requestedAt"`
	MintedAt          *time.Time `json:"mintedAt"`
	ScheduledBurnDate *time.Time `json:"scheduledBurnDate"`
	TokenBlueprintID  *string    `json:"tokenBlueprintId"`
}

// ===============================
// Errors
// ===============================

var (
	ErrInvalidID                    = errors.New("mintRequest: invalid id")
	ErrInvalidProductionID          = errors.New("mintRequest: invalid productionId")
	ErrInvalidMintQuantity          = errors.New("mintRequest: invalid mintQuantity")
	ErrInvalidStatus                = errors.New("mintRequest: invalid status")
	ErrInvalidRequestedBy           = errors.New("mintRequest: invalid requestedBy")
	ErrInvalidRequestedAt           = errors.New("mintRequest: invalid requestedAt")
	ErrInvalidMintedAt              = errors.New("mintRequest: invalid mintedAt")
	ErrInvalidScheduledBurnDate     = errors.New("mintRequest: invalid scheduledBurnDate")
	ErrInvalidTokenBlueprintID      = errors.New("mintRequest: invalid tokenBlueprintId")
	ErrInvalidMintRequestTransition = errors.New("mintRequest: invalid status transition")
)

// ===============================
// Constructors
// ===============================

// NewMintRequest は planning 状態の MintRequest を新規作成します。
// requestedX / mintedAt / burnDate / tokenBlueprintId はすべて nil 初期化します。
func NewMintRequest(
	id string,
	productionID string,
	mintQuantity int,
) (MintRequest, error) {

	m := MintRequest{
		ID:           strings.TrimSpace(id),
		ProductionID: strings.TrimSpace(productionID),
		Status:       StatusPlanning,

		MintQuantity: mintQuantity,

		RequestedBy:       nil,
		RequestedAt:       nil,
		MintedAt:          nil,
		ScheduledBurnDate: nil,
		TokenBlueprintID:  nil,
	}

	if err := m.validate(); err != nil {
		return MintRequest{}, err
	}
	return m, nil
}

// ===============================
// Behavior
// ===============================

// Request: planning -> requested (requestedBy/At を設定)
func (m *MintRequest) Request(by string, at time.Time) error {
	if m.Status != StatusPlanning {
		return ErrInvalidMintRequestTransition
	}

	by = strings.TrimSpace(by)
	if by == "" {
		return ErrInvalidRequestedBy
	}
	if at.IsZero() {
		return ErrInvalidRequestedAt
	}
	at = at.UTC()

	m.Status = StatusRequested
	m.RequestedBy = &by
	m.RequestedAt = &at
	m.MintedAt = nil

	return nil
}

// MarkMinted: requested -> minted (mintedAt を設定)
func (m *MintRequest) MarkMinted(at time.Time) error {
	if m.Status != StatusRequested {
		return ErrInvalidMintRequestTransition
	}
	if at.IsZero() {
		return ErrInvalidMintedAt
	}
	at = at.UTC()

	m.Status = StatusMinted
	m.MintedAt = &at
	return nil
}

// UpdateQuantity は planning 中のみミント数量を変更できます。
func (m *MintRequest) UpdateQuantity(q int) error {
	if m.Status != StatusPlanning {
		return ErrInvalidMintRequestTransition
	}
	if q <= 0 {
		return ErrInvalidMintQuantity
	}
	m.MintQuantity = q
	return nil
}

// SetTokenBlueprintID は planning / requested 状態でのみトークン設計IDを設定可能とします。
// minted になった後は変更不可、というドメインルール想定です。
func (m *MintRequest) SetTokenBlueprintID(tokenBlueprintID *string) error {
	if m.Status == StatusMinted {
		return ErrInvalidMintRequestTransition
	}
	if tokenBlueprintID != nil {
		id := strings.TrimSpace(*tokenBlueprintID)
		if id == "" {
			return ErrInvalidTokenBlueprintID
		}
		m.TokenBlueprintID = &id
		return nil
	}
	// nil を許容（＝未設定）
	m.TokenBlueprintID = nil
	return nil
}

// SetScheduledBurnDate は任意の状態で設定可能とします（必要なら制約を追加）。
func (m *MintRequest) SetScheduledBurnDate(t *time.Time) error {
	if t == nil {
		m.ScheduledBurnDate = nil
		return nil
	}
	if t.IsZero() {
		return ErrInvalidScheduledBurnDate
	}
	utc := t.UTC()
	m.ScheduledBurnDate = &utc
	return nil
}

// ===============================
// Validation
// ===============================

func (m MintRequest) validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(m.ProductionID) == "" {
		return ErrInvalidProductionID
	}
	if m.MintQuantity <= 0 {
		return ErrInvalidMintQuantity
	}
	if !IsValidStatus(m.Status) {
		return ErrInvalidStatus
	}

	// TokenBlueprintID の形式チェック（ある場合のみ）
	if m.TokenBlueprintID != nil {
		if strings.TrimSpace(*m.TokenBlueprintID) == "" {
			return ErrInvalidTokenBlueprintID
		}
	}

	// ScheduledBurnDate の妥当性チェック
	if m.ScheduledBurnDate != nil && m.ScheduledBurnDate.IsZero() {
		return ErrInvalidScheduledBurnDate
	}

	// ステータスとフィールドの整合性
	switch m.Status {
	case StatusPlanning:
		// planning 中はまだリクエスト/ミント情報が入っていない前提
		if m.RequestedBy != nil || m.RequestedAt != nil || m.MintedAt != nil {
			return ErrInvalidStatus
		}

	case StatusRequested:
		if m.RequestedBy == nil || strings.TrimSpace(*m.RequestedBy) == "" {
			return ErrInvalidRequestedBy
		}
		if m.RequestedAt == nil || m.RequestedAt.IsZero() {
			return ErrInvalidRequestedAt
		}
		if m.MintedAt != nil {
			return ErrInvalidMintedAt
		}

	case StatusMinted:
		if m.RequestedBy == nil || strings.TrimSpace(*m.RequestedBy) == "" {
			return ErrInvalidRequestedBy
		}
		if m.RequestedAt == nil || m.RequestedAt.IsZero() {
			return ErrInvalidRequestedAt
		}
		if m.MintedAt == nil || m.MintedAt.IsZero() {
			return ErrInvalidMintedAt
		}
		if m.MintedAt.Before(*m.RequestedAt) {
			return ErrInvalidMintedAt
		}
	}

	return nil
}

// Validate は外部公開用のラッパーです。
func (m MintRequest) Validate() error {
	return m.validate()
}
