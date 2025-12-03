package product

import (
	"errors"
	"fmt"
	"strings"
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
	ConnectedToken   *string          `json:"connectedToken"`

	PrintedAt   *time.Time `json:"printedAt"`
	InspectedAt *time.Time `json:"inspectedAt"`
	InspectedBy *string    `json:"inspectedBy"`
}

// TokenConnectionStatus はトークン接続状態の列挙
type TokenConnectionStatus string

const (
	TokenConnected    TokenConnectionStatus = "connected"
	TokenDisconnected TokenConnectionStatus = "notConnected"
)

// ===============================
// Errors
// ===============================

var (
	ErrInvalidID               = errors.New("product: invalid id")
	ErrInvalidModelID          = errors.New("product: invalid modelId")
	ErrInvalidProductionID     = errors.New("product: invalid productionId")
	ErrInvalidInspectionResult = errors.New("product: invalid inspectionResult")
	ErrInvalidConnectedToken   = errors.New("product: invalid connectedToken")

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
	connectedToken *string,
	printedAt *time.Time,
	inspectedAt *time.Time,
	inspectedBy *string,
) (Product, error) {

	if inspection == "" {
		inspection = InspectionNotYet
	}

	p := Product{
		ID:               strings.TrimSpace(id),
		ModelID:          strings.TrimSpace(modelID),
		ProductionID:     strings.TrimSpace(productionID),
		InspectionResult: inspection,
		ConnectedToken:   normalizeStrPtr(connectedToken),

		PrintedAt:   normalizeTimePtr(printedAt),
		InspectedAt: normalizeTimePtr(inspectedAt),
		InspectedBy: normalizeStrPtr(inspectedBy),
	}

	if err := p.validate(); err != nil {
		return Product{}, err
	}
	return p, nil
}

func NewFromStringTimes(
	id, modelID, productionID string,
	inspection InspectionResult,
	connectedToken *string,
	printedAtStr string,
	inspectedAtStr string,
	inspectedBy *string,
) (Product, error) {

	var printedAtPtr *time.Time
	if strings.TrimSpace(printedAtStr) != "" {
		t, err := parseTime(printedAtStr, ErrInvalidPrintedAt)
		if err != nil {
			return Product{}, err
		}
		printedAtPtr = &t
	}

	var inspectedAtPtr *time.Time
	if strings.TrimSpace(inspectedAtStr) != "" {
		t, err := parseTime(inspectedAtStr, ErrInvalidInspectedAt)
		if err != nil {
			return Product{}, err
		}
		inspectedAtPtr = &t
	}

	return New(
		id, modelID, productionID,
		inspection, connectedToken,
		printedAtPtr,
		inspectedAtPtr, inspectedBy,
	)
}

// ===============================
// Behavior
// ===============================

func (p *Product) ConnectToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInvalidConnectedToken
	}
	p.ConnectedToken = &token
	return nil
}

func (p *Product) DisconnectToken() {
	p.ConnectedToken = nil
}

func (p Product) ConnectionStatus() TokenConnectionStatus {
	if p.ConnectedToken != nil {
		return TokenConnected
	}
	return TokenDisconnected
}

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
	by = strings.TrimSpace(by)
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
	by = strings.TrimSpace(by)
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

	if p.ConnectedToken != nil && strings.TrimSpace(*p.ConnectedToken) == "" {
		return ErrInvalidConnectedToken
	}

	// printedAt: あればゼロでないことだけチェック
	if p.PrintedAt != nil && p.PrintedAt.IsZero() {
		return ErrInvalidPrintedAt
	}

	// ★ 検査結果との整合性チェック
	switch p.InspectionResult {

	// 検査が確定している状態は by/at 必須
	case InspectionPassed, InspectionFailed, InspectionNotManufactured:
		if p.InspectedBy == nil || strings.TrimSpace(*p.InspectedBy) == "" {
			return ErrInvalidInspectedBy
		}
		if p.InspectedAt == nil || p.InspectedAt.IsZero() {
			return ErrInvalidInspectedAt
		}

	// まだ検査していない状態。過去データの互換性のため、
	// inspectedBy/inspectedAt が入っていてもエラーにはしない。
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

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil {
		return nil
	}
	if p.IsZero() {
		return nil
	}
	utc := p.UTC()
	return &utc
}

// ID のスライスをトリムし、空文字を除去する。
// 結果が空なら nil を返します（バリデーションで検知）。
func normalizeIDList(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
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
