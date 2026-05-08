// backend\internal\domain\product\product.go
package product

import (
	"errors"
	"time"
)

// ===============================
// Types (mirror TS)
// ===============================

// InspectionResult は検査結果の列挙
type InspectionResult string

const (
	InspectionNotYet          InspectionResult = "notYet"
	InspectionPassed          InspectionResult = "passed"
	InspectionFailed          InspectionResult = "failed"
	InspectionNotManufactured InspectionResult = "notManufactured"
)

// Inspection は検査更新APIのリクエストボディ用
type Inspection struct {
	InspectionResult InspectionResult `json:"inspectionResult"`
	InspectedBy      string           `json:"inspectedBy"`
	InspectedAt      *time.Time       `json:"inspectedAt,omitempty"`
}

// Product エンティティ（TS の仕様に合わせる）
type Product struct {
	ID               string           `json:"id"`
	ModelID          string           `json:"modelId"`
	ProductionID     string           `json:"productionId"`
	InspectionResult InspectionResult `json:"inspectionResult"`

	PrintedAt   *time.Time `json:"printedAt"`
	InspectedAt *time.Time `json:"inspectedAt"`
	InspectedBy *string    `json:"inspectedBy"`
}

// ===============================
// Errors
// ===============================

var (
	ErrInvalidID               = errors.New("product: invalid id")
	ErrInvalidModelID          = errors.New("product: invalid modelId")
	ErrInvalidProductionID     = errors.New("product: invalid productionId")
	ErrInvalidInspectionResult = errors.New("product: invalid inspectionResult")

	ErrInvalidPrintedAt   = errors.New("product: invalid printedAt")
	ErrInvalidInspectedAt = errors.New("product: invalid inspectedAt")
	ErrInvalidInspectedBy = errors.New("product: invalid inspectedBy")
)

// ===============================
// Constructors
// ===============================

func New(
	id, modelID, productionID string,
	inspection InspectionResult,
	printedAt *time.Time,
	inspectedAt *time.Time,
	inspectedBy *string,
) (Product, error) {

	if inspection == "" {
		inspection = InspectionNotYet
	}

	p := Product{
		ID:               id,
		ModelID:          modelID,
		ProductionID:     productionID,
		InspectionResult: inspection,

		PrintedAt:   printedAt,
		InspectedAt: inspectedAt,
		InspectedBy: inspectedBy,
	}

	if err := p.validate(); err != nil {
		return Product{}, err
	}
	return p, nil
}

// ===============================
// Behavior
// ===============================

// printedBy は保持しない方針なので、by は受け取らず printedAt のみを更新
func (p *Product) MarkPrinted(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidPrintedAt
	}
	utc := at.UTC()
	p.PrintedAt = &utc
	return nil
}

func (p *Product) MarkInspected(result InspectionResult, by string, at time.Time) error {
	if result != InspectionPassed && result != InspectionFailed {
		return ErrInvalidInspectionResult
	}
	if by == "" {
		return ErrInvalidInspectedBy
	}
	if at.IsZero() {
		return ErrInvalidInspectedAt
	}

	p.InspectionResult = result
	p.InspectedBy = &by
	utc := at.UTC()
	p.InspectedAt = &utc
	return nil
}

// ★ 追加: notManufactured へ確定する
func (p *Product) MarkNotManufactured(by string, at time.Time) error {
	if by == "" {
		return ErrInvalidInspectedBy
	}
	if at.IsZero() {
		return ErrInvalidInspectedAt
	}

	p.InspectionResult = InspectionNotManufactured
	p.InspectedBy = &by
	utc := at.UTC()
	p.InspectedAt = &utc
	return nil
}

func (p *Product) ClearInspection() {
	p.InspectionResult = InspectionNotYet
	p.InspectedAt = nil
	p.InspectedBy = nil
}

// ===============================
// Validation
// ===============================

func (p Product) validate() error {
	if p.ID == "" {
		return ErrInvalidID
	}
	if p.ModelID == "" {
		return ErrInvalidModelID
	}
	if p.ProductionID == "" {
		return ErrInvalidProductionID
	}

	if !IsValidInspectionResult(p.InspectionResult) {
		return ErrInvalidInspectionResult
	}

	// printedAt: あればゼロでないことだけチェック
	if p.PrintedAt != nil && p.PrintedAt.IsZero() {
		return ErrInvalidPrintedAt
	}

	// ★ 検査結果との整合性チェック
	switch p.InspectionResult {

	// 検査が確定している状態は by/at 必須
	case InspectionPassed, InspectionFailed, InspectionNotManufactured:
		if p.InspectedBy == nil || *p.InspectedBy == "" {
			return ErrInvalidInspectedBy
		}
		if p.InspectedAt == nil || p.InspectedAt.IsZero() {
			return ErrInvalidInspectedAt
		}

	// まだ検査していない状態。
	case InspectionNotYet:
		// 何もしない（coherence はチェックしない）

	default:
		// IsValidInspectionResult で弾いているのでここには来ない想定
	}

	return nil
}

// ===============================
// Helpers
// ===============================

// ID のスライスを空文字を除去する。
// 結果が空なら nil を返します（バリデーションで検知）。
func normalizeIDList(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Valid inspection result
func IsValidInspectionResult(v InspectionResult) bool {
	switch v {
	case InspectionNotYet, InspectionPassed, InspectionFailed, InspectionNotManufactured:
		return true
	default:
		return false
	}
}
