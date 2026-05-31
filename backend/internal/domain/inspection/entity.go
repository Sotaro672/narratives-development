// backend/internal/domain/inspection/entity.go
package inspection

import (
	"errors"
	"time"
)

// ===============================
// InspectionResult（検査結果の種類）
// ===============================

type InspectionResult string

const (
	// InspectionNotYet は、まだ Inspector で明示的な結果が入力されていない状態です。
	// ネガティブ制では、Complete 時に passed として確定します。
	InspectionNotYet InspectionResult = "notYet"

	// InspectionPassed は、実際に製造され、ミント対象となる productId です。
	InspectionPassed InspectionResult = "passed"

	// InspectionFailed は、不良などによりミント対象外となる productId です。
	InspectionFailed InspectionResult = "failed"

	// InspectionNotManufactured は、余ったラベルなど、実際には製造されなかった productId です。
	InspectionNotManufactured InspectionResult = "notManufactured"
)

// ===============================
// InspectionStatus（バッチ全体の状態）
// ===============================

type InspectionStatus string

const (
	InspectionStatusInspecting InspectionStatus = "inspecting"
	InspectionStatusCompleted  InspectionStatus = "completed"
)

// ------------------------------------------------------
// InspectionItem: productId ごとの検査結果
// ------------------------------------------------------

type InspectionItem struct {
	ProductID        string            `json:"productId"`
	ModelID          string            `json:"modelId"`
	InspectionResult *InspectionResult `json:"inspectionResult"`
	InspectedBy      *string           `json:"inspectedBy"`
	InspectedAt      *time.Time        `json:"inspectedAt"`
}

// ------------------------------------------------------
// InspectionBatch: inspections テーブル 1 レコード
// ------------------------------------------------------

type InspectionBatch struct {
	ProductionID string           `json:"productionId"`
	Status       InspectionStatus `json:"status"`
	MintID       *string          `json:"mintId,omitempty"` // mints テーブルのID（未申請なら nil）
	Quantity     int              `json:"quantity"`         // item の合計数
	TotalPassed  int              `json:"totalPassed"`      // ミント対象となる passed の数
	Inspections  []InspectionItem `json:"inspections"`
}

// ===============================
// Errors（inspection 専用）
// ===============================

var (
	ErrInvalidInspectionProductionID = errors.New("inspection: invalid productionId")
	ErrInvalidInspectionStatus       = errors.New("inspection: invalid status")
	ErrInvalidInspectionProductIDs   = errors.New("inspection: invalid productIds")
	ErrInvalidInspectionMintID       = errors.New("inspection: invalid mintId")

	ErrInvalidInspectionResult      = errors.New("inspection: invalid inspectionResult")
	ErrInvalidInspectedBy           = errors.New("inspection: invalid inspectedBy")
	ErrInvalidInspectedAt           = errors.New("inspection: invalid inspectedAt")
	ErrInvalidInspectionQuantity    = errors.New("inspection: invalid quantity")
	ErrInvalidInspectionTotalPassed = errors.New("inspection: invalid totalPassed")
	ErrInconsistentInspectionTotals = errors.New("inspection: inconsistent totals")

	ErrNotFound = errors.New("inspection: not found")
)

// ===============================
// Constructors
// ===============================

// NewInspectionBatch は、1 production に対応する InspectionBatch を作成します。
//
// ネガティブ制では、初期状態の各 productId は notYet です。
// Inspector では failed / notManufactured のみを明示的に入力し、
// Complete 時に notYet の productId を passed として確定します。
func NewInspectionBatch(
	productionID string,
	status InspectionStatus,
	productIDs []string,
) (InspectionBatch, error) {
	if productionID == "" {
		return InspectionBatch{}, ErrInvalidInspectionProductionID
	}

	if !IsValidInspectionStatus(status) {
		return InspectionBatch{}, ErrInvalidInspectionStatus
	}

	if len(productIDs) == 0 {
		return InspectionBatch{}, ErrInvalidInspectionProductIDs
	}

	inspections := make([]InspectionItem, 0, len(productIDs))
	for _, id := range productIDs {
		r := InspectionNotYet
		inspections = append(inspections, InspectionItem{
			ProductID:        id,
			ModelID:          "", // modelId はアプリケーション層で埋める
			InspectionResult: &r,
			InspectedBy:      nil,
			InspectedAt:      nil,
		})
	}

	batch := InspectionBatch{
		ProductionID: productionID,
		Status:       status,
		MintID:       nil,
		Quantity:     len(inspections),
		TotalPassed:  0,
		Inspections:  inspections,
	}

	batch.RecalculateTotals()

	if err := batch.validate(); err != nil {
		return InspectionBatch{}, err
	}

	return batch, nil
}

// ===============================
// Behavior / Validation
// ===============================

func (b InspectionBatch) validate() error {
	if b.ProductionID == "" {
		return ErrInvalidInspectionProductionID
	}

	if !IsValidInspectionStatus(b.Status) {
		return ErrInvalidInspectionStatus
	}

	if len(b.Inspections) == 0 {
		return ErrInvalidInspectionProductIDs
	}

	if b.MintID != nil && *b.MintID == "" {
		return ErrInvalidInspectionMintID
	}

	if b.Quantity != len(b.Inspections) || b.Quantity <= 0 {
		return ErrInvalidInspectionQuantity
	}

	if b.TotalPassed < 0 || b.TotalPassed > b.Quantity {
		return ErrInvalidInspectionTotalPassed
	}

	productIDs := make(map[string]struct{}, len(b.Inspections))

	actualPassed := 0

	for _, ins := range b.Inspections {
		if ins.ProductID == "" {
			return ErrInvalidInspectionProductIDs
		}

		if _, exists := productIDs[ins.ProductID]; exists {
			return ErrInvalidInspectionProductIDs
		}
		productIDs[ins.ProductID] = struct{}{}

		if ins.InspectionResult == nil {
			return ErrInvalidInspectionResult
		}

		if !IsValidInspectionResult(*ins.InspectionResult) {
			return ErrInvalidInspectionResult
		}

		switch *ins.InspectionResult {
		case InspectionPassed:
			actualPassed++

			if ins.InspectedBy == nil || *ins.InspectedBy == "" {
				return ErrInvalidInspectedBy
			}
			if ins.InspectedAt == nil || ins.InspectedAt.IsZero() {
				return ErrInvalidInspectedAt
			}

		case InspectionFailed, InspectionNotManufactured:
			if ins.InspectedBy == nil || *ins.InspectedBy == "" {
				return ErrInvalidInspectedBy
			}
			if ins.InspectedAt == nil || ins.InspectedAt.IsZero() {
				return ErrInvalidInspectedAt
			}

		case InspectionNotYet:
			if b.Status == InspectionStatusCompleted {
				return ErrInvalidInspectionResult
			}

			if ins.InspectedBy != nil || ins.InspectedAt != nil {
				return ErrInvalidInspectionResult
			}
		}
	}

	if b.TotalPassed != actualPassed {
		return ErrInconsistentInspectionTotals
	}

	return nil
}

// Validate は InspectionBatch の公開バリデーションメソッドです。
func (b InspectionBatch) Validate() error {
	return b.validate()
}

// Complete は検品バッチを完了状態にします。
//
// ネガティブ制では、Inspector で明示的に入力するのは failed / notManufactured です。
// Complete 時点で notYet のまま残っている productId は、実際に製造されたものとして
// passed に確定し、ミント対象になります。
func (b *InspectionBatch) Complete(by string, at time.Time) error {
	if by == "" {
		return ErrInvalidInspectedBy
	}

	atUTC := at.UTC()
	if atUTC.IsZero() {
		return ErrInvalidInspectedAt
	}

	for i := range b.Inspections {
		item := &b.Inspections[i]

		if item.InspectionResult == nil || *item.InspectionResult == InspectionNotYet {
			r := InspectionPassed
			item.InspectionResult = &r
			item.InspectedBy = &by
			item.InspectedAt = &atUTC
			continue
		}

		switch *item.InspectionResult {
		case InspectionPassed, InspectionFailed, InspectionNotManufactured:
			if item.InspectedBy == nil || *item.InspectedBy == "" {
				return ErrInvalidInspectedBy
			}
			if item.InspectedAt == nil || item.InspectedAt.IsZero() {
				return ErrInvalidInspectedAt
			}

		default:
			return ErrInvalidInspectionResult
		}
	}

	b.Status = InspectionStatusCompleted
	b.RecalculateTotals()

	return b.validate()
}

// MarkFailed は指定した productId を failed として記録します。
func (b *InspectionBatch) MarkFailed(productID string, by string, at time.Time) error {
	return b.mark(productID, InspectionFailed, by, at)
}

// MarkNotManufactured は指定した productId を notManufactured として記録します。
func (b *InspectionBatch) MarkNotManufactured(productID string, by string, at time.Time) error {
	return b.mark(productID, InspectionNotManufactured, by, at)
}

// MarkPassed は指定した productId を passed として記録します。
//
// ネガティブ制では通常 Inspector から直接 passed を入力しません。
// ただし、ドメイン上は Complete 後の確定状態として passed を保持します。
func (b *InspectionBatch) MarkPassed(productID string, by string, at time.Time) error {
	return b.mark(productID, InspectionPassed, by, at)
}

func (b *InspectionBatch) mark(
	productID string,
	result InspectionResult,
	by string,
	at time.Time,
) error {
	if productID == "" {
		return ErrInvalidInspectionProductIDs
	}

	if !IsValidInspectionResult(result) || result == InspectionNotYet {
		return ErrInvalidInspectionResult
	}

	if by == "" {
		return ErrInvalidInspectedBy
	}

	atUTC := at.UTC()
	if atUTC.IsZero() {
		return ErrInvalidInspectedAt
	}

	for i := range b.Inspections {
		item := &b.Inspections[i]
		if item.ProductID != productID {
			continue
		}

		item.InspectionResult = &result
		item.InspectedBy = &by
		item.InspectedAt = &atUTC

		b.RecalculateTotals()

		return b.validate()
	}

	return ErrInvalidInspectionProductIDs
}

// RecalculateTotals は Quantity と TotalPassed を inspections から再計算します。
func (b *InspectionBatch) RecalculateTotals() {
	b.Quantity = len(b.Inspections)

	totalPassed := 0
	for _, item := range b.Inspections {
		if item.InspectionResult != nil && *item.InspectionResult == InspectionPassed {
			totalPassed++
		}
	}

	b.TotalPassed = totalPassed
}

// MintTargetProductIDs はミント対象となる productId の一覧を返します。
//
// ミント対象は passed の productId のみです。
func (b InspectionBatch) MintTargetProductIDs() []string {
	out := make([]string, 0, b.TotalPassed)

	for _, item := range b.Inspections {
		if item.InspectionResult != nil && *item.InspectionResult == InspectionPassed {
			out = append(out, item.ProductID)
		}
	}

	return out
}

// ExcludedProductIDs はミント対象外となる productId の一覧を返します。
//
// failed / notManufactured が対象外です。
func (b InspectionBatch) ExcludedProductIDs() []string {
	out := make([]string, 0)

	for _, item := range b.Inspections {
		if item.InspectionResult == nil {
			continue
		}

		switch *item.InspectionResult {
		case InspectionFailed, InspectionNotManufactured:
			out = append(out, item.ProductID)
		}
	}

	return out
}

// ===============================
// Status / Result validator
// ===============================

func IsValidInspectionStatus(s InspectionStatus) bool {
	return s == InspectionStatusInspecting || s == InspectionStatusCompleted
}

func IsValidInspectionResult(r InspectionResult) bool {
	switch r {
	case InspectionNotYet, InspectionPassed, InspectionFailed, InspectionNotManufactured:
		return true
	default:
		return false
	}
}
