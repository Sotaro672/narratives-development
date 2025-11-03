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

// Product エンティティ（TSの仕様に合わせる）
type Product struct {
	ID               string           `json:"id"`               // 例: 'product_001'
	ModelID          string           `json:"modelId"`          // モデルID
	ProductionID     string           `json:"productionId"`     // 生産計画ID
	InspectionResult InspectionResult `json:"inspectionResult"` // 検査結果
	ConnectedToken   *string          `json:"connectedToken"`   // 接続されたトークンID（null可）

	PrintedAt  *time.Time `json:"printedAt"`  // 製造日時（null可）
	PrintedBy  *string    `json:"printedBy"`  // 製造者（null可）
	InspectedAt *time.Time `json:"inspectedAt"` // 検査日時（null可）
	InspectedBy *string    `json:"inspectedBy"` // 検査者（null可）

	UpdatedAt time.Time `json:"updatedAt"` // 更新日時（必須）
	UpdatedBy string    `json:"updatedBy"` // 更新者（必須）
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

	ErrInvalidPrintedAt = errors.New("product: invalid printedAt")
	ErrInvalidPrintedBy = errors.New("product: invalid printedBy")

	ErrInvalidInspectedAt = errors.New("product: invalid inspectedAt")
	ErrInvalidInspectedBy = errors.New("product: invalid inspectedBy")

	ErrInvalidUpdatedAt = errors.New("product: invalid updatedAt")
	ErrInvalidUpdatedBy = errors.New("product: invalid updatedBy")

	ErrInvalidCoherence = errors.New("product: invalid field coherence")
)

// ===============================
// Constructors
// ===============================

func New(
	id, modelID, productionID string,
	inspection InspectionResult,
	connectedToken *string,
	printedAt *time.Time,
	printedBy *string,
	inspectedAt *time.Time,
	inspectedBy *string,
	updatedAt time.Time,
	updatedBy string,
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

		PrintedAt:  normalizeTimePtr(printedAt),
		PrintedBy:  normalizeStrPtr(printedBy),
		InspectedAt: normalizeTimePtr(inspectedAt),
		InspectedBy: normalizeStrPtr(inspectedBy),

		UpdatedAt: updatedAt.UTC(),
		UpdatedBy: strings.TrimSpace(updatedBy),
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
	printedAtStr string,   // "" => nil
	printedBy *string,     // nil or non-empty
	inspectedAtStr string, // "" => nil
	inspectedBy *string,   // nil or non-empty
	updatedAtStr string,
	updatedBy string,
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
	ua, err := parseTime(updatedAtStr, ErrInvalidUpdatedAt)
	if err != nil {
		return Product{}, err
	}
	return New(
		id, modelID, productionID,
		inspection, connectedToken,
		printedAtPtr, printedBy,
		inspectedAtPtr, inspectedBy,
		ua, updatedBy,
	)
}

// ===============================
// Behavior
// ===============================

// ConnectToken sets a token id (non-empty).
func (p *Product) ConnectToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInvalidConnectedToken
	}
	p.ConnectedToken = &token
	return nil
}

// DisconnectToken clears the token connection.
func (p *Product) DisconnectToken() {
	p.ConnectedToken = nil
}

// ConnectionStatus returns 'connected' when ConnectedToken is set.
func (p Product) ConnectionStatus() TokenConnectionStatus {
	if p.ConnectedToken != nil {
		return TokenConnected
	}
	return TokenDisconnected
}

// MarkPrinted sets printed fields coherently.
func (p *Product) MarkPrinted(by string, at time.Time) error {
	by = strings.TrimSpace(by)
	if by == "" {
		return ErrInvalidPrintedBy
	}
	if at.IsZero() {
		return ErrInvalidPrintedAt
	}
	at = at.UTC()
	p.PrintedBy = &by
	p.PrintedAt = &at
	return nil
}

// ClearPrinted clears printed fields.
func (p *Product) ClearPrinted() {
	p.PrintedBy = nil
	p.PrintedAt = nil
}

// MarkInspected sets inspection to passed/failed with inspector and time.
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
	at = at.UTC()
	p.InspectionResult = result
	p.InspectedBy = &by
	p.InspectedAt = &at
	return nil
}

// ClearInspection resets inspection to notYet and clears fields.
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
	// connectedToken optional; when present must be non-empty
	if p.ConnectedToken != nil && strings.TrimSpace(*p.ConnectedToken) == "" {
		return ErrInvalidConnectedToken
	}

	// printed pair coherence: both nil or both valid
	if (p.PrintedAt == nil) != (p.PrintedBy == nil) {
		return ErrInvalidCoherence
	}
	if p.PrintedBy != nil && strings.TrimSpace(*p.PrintedBy) == "" {
		return ErrInvalidPrintedBy
	}
	if p.PrintedAt != nil && p.PrintedAt.IsZero() {
		return ErrInvalidPrintedAt
	}

	// inspected pair coherence driven by inspection result
	switch p.InspectionResult {
	case InspectionPassed, InspectionFailed:
		if p.InspectedBy == nil || strings.TrimSpace(*p.InspectedBy) == "" {
			return ErrInvalidInspectedBy
		}
		if p.InspectedAt == nil || p.InspectedAt.IsZero() {
			return ErrInvalidInspectedAt
		}
	case InspectionNotYet, InspectionNotManufactured:
		// should not have inspected fields
		if p.InspectedBy != nil || p.InspectedAt != nil {
			return ErrInvalidCoherence
		}
	}

	// updated fields required
	if p.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if strings.TrimSpace(p.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
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

// IsValidInspectionResult は検査結果が有効か判定
func IsValidInspectionResult(v InspectionResult) bool {
	switch v {
	case InspectionNotYet, InspectionPassed, InspectionFailed, InspectionNotManufactured:
		return true
	default:
		return false
	}
}

// ProductsTableDDL defines the SQL for the products table migration.
const ProductsTableDDL = `
-- Migration: Initialize/Update Product domain
-- Mirrors backend/internal/domain/product/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS products (
  id                TEXT        PRIMARY KEY,
  model_id          TEXT        NOT NULL,
  production_id     TEXT        NOT NULL,
  inspection_result TEXT        NOT NULL CHECK (inspection_result IN ('notYet','passed','failed','notManufactured')),
  connected_token   TEXT        NULL,

  printed_at        TIMESTAMPTZ NULL,
  printed_by        TEXT        NULL,

  inspected_at      TIMESTAMPTZ NULL,
  inspected_by      TEXT        NULL,

  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by        TEXT        NOT NULL,

  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Non-empty checks
  CONSTRAINT chk_products_ids_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(model_id)) > 0
    AND char_length(trim(production_id)) > 0
  ),
  CONSTRAINT chk_products_connected_token_non_empty CHECK (
    connected_token IS NULL OR char_length(trim(connected_token)) > 0
  ),
  CONSTRAINT chk_products_printed_by_non_empty CHECK (
    printed_by IS NULL OR char_length(trim(printed_by)) > 0
  ),
  CONSTRAINT chk_products_inspected_by_non_empty CHECK (
    inspected_by IS NULL OR char_length(trim(inspected_by)) > 0
  ),
  CONSTRAINT chk_products_updated_by_non_empty CHECK (
    char_length(trim(updated_by)) > 0
  ),

  -- Printed coherence: both NULL or both present
  CONSTRAINT chk_products_printed_coherence CHECK (
    (printed_by IS NULL AND printed_at IS NULL)
    OR
    (printed_by IS NOT NULL AND char_length(trim(printed_by)) > 0 AND printed_at IS NOT NULL)
  ),

  -- Coherence with inspection_result:
  -- passed/failed: inspected_by, inspected_at required
  -- notYet/notManufactured: inspected_by, inspected_at must be NULL
  CONSTRAINT chk_products_inspection_coherence CHECK (
    (inspection_result IN ('notYet','notManufactured') AND inspected_by IS NULL AND inspected_at IS NULL)
    OR
    (inspection_result IN ('passed','failed') AND inspected_by IS NOT NULL AND char_length(trim(inspected_by)) > 0 AND inspected_at IS NOT NULL)
  ),

  -- Optional FK: disconnect token automatically when deleted
  CONSTRAINT fk_products_connected_token
    FOREIGN KEY (connected_token) REFERENCES tokens(mint_address) ON DELETE SET NULL
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_products_model_id           ON products(model_id);
CREATE INDEX IF NOT EXISTS idx_products_production_id      ON products(production_id);
CREATE INDEX IF NOT EXISTS idx_products_inspection_result  ON products(inspection_result);
CREATE INDEX IF NOT EXISTS idx_products_printed_at         ON products(printed_at);
CREATE INDEX IF NOT EXISTS idx_products_inspected_at       ON products(inspected_at);
CREATE INDEX IF NOT EXISTS idx_products_updated_at         ON products(updated_at);

COMMIT;
`
