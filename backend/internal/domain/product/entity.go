// backend/internal/domain/product/entity.go
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
	ID               string           `json:"id"`
	ModelID          string           `json:"modelId"`
	ProductionID     string           `json:"productionId"`
	InspectionResult InspectionResult `json:"inspectionResult"`
	ConnectedToken   *string          `json:"connectedToken"`

	PrintedAt   *time.Time `json:"printedAt"`
	InspectedAt *time.Time `json:"inspectedAt"`
	InspectedBy *string    `json:"inspectedBy"`
}

// PrintLog は「印刷した Product の履歴」を保持するエンティティ。
// 1 レコードで 1 回の印刷バッチを表し、productIds にそのとき印刷された Product ID 一覧を持ちます。
type PrintLog struct {
	ID           string    `json:"id"`
	ProductionID string    `json:"productionId"`
	ProductIDs   []string  `json:"productIds"`
	PrintedBy    string    `json:"printedBy"`
	PrintedAt    time.Time `json:"printedAt"`
	// QR ペイロード一覧（例: 各 productId に対応する URL）
	// Firestore には保存せず、レスポンス専用に使う想定。
	QrPayloads []string `json:"qrPayloads,omitempty"`
}

// TokenConnectionStatus はトークン接続状態の列挙
type TokenConnectionStatus string

const (
	TokenConnected    TokenConnectionStatus = "connected"
	TokenDisconnected TokenConnectionStatus = "notConnected"
)

// ===============================
// Inspection batch (inspections_by_production)
// ===============================

// InspectionStatus は inspections_by_production のステータス
type InspectionStatus string

const (
	InspectionStatusInspecting InspectionStatus = "inspecting"
	InspectionStatusCompleted  InspectionStatus = "completed"
)

// InspectionItem は 1 つの productId に対する検査状態を表す
type InspectionItem struct {
	ProductID        string            `json:"productId"`
	InspectionResult *InspectionResult `json:"inspectionResult"` // create 時は nil
	InspectedBy      *string           `json:"inspectedBy"`      // create 時は nil
	InspectedAt      *time.Time        `json:"inspectedAt"`      // create 時は nil
}

// InspectionBatch は 1 productionId に紐づく inspections_by_production ドキュメント
//
//	inspections_by_production/{productionId} {
//	  "productionId": "...",
//	  "status": "inspecting" | "completed",
//	  "inspections": [
//	    { "productId": "...", "inspectionResult": null, "inspectedBy": null, "inspectedAt": null },
//	    ...
//	  ]
//	}
type InspectionBatch struct {
	ProductionID string           `json:"productionId"`
	Status       InspectionStatus `json:"status"`
	Inspections  []InspectionItem `json:"inspections"`
}

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

	ErrInvalidInspectedAt = errors.New("product: invalid inspectedAt")
	ErrInvalidInspectedBy = errors.New("product: invalid inspectedBy")

	ErrInvalidCoherence = errors.New("product: invalid field coherence")

	// PrintLog 用エラー
	ErrInvalidPrintLogID           = errors.New("printLog: invalid id")
	ErrInvalidPrintLogProductionID = errors.New("printLog: invalid productionId")
	ErrInvalidPrintLogProductIDs   = errors.New("printLog: invalid productIds")
	ErrInvalidPrintLogPrintedBy    = errors.New("printLog: invalid printedBy")
	ErrInvalidPrintLogPrintedAt    = errors.New("printLog: invalid printedAt")

	// InspectionBatch 用エラー
	ErrInvalidInspectionProductionID = errors.New("inspection: invalid productionId")
	ErrInvalidInspectionStatus       = errors.New("inspection: invalid status")
	ErrInvalidInspectionProductIDs   = errors.New("inspection: invalid productIds")
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

// NewPrintLog は PrintLog エンティティのコンストラクタです。
// 空白除去などを行ったうえでバリデーションします。
// QrPayloads はここでは扱わず、後続の処理（usecase など）で必要に応じて詰める想定です。
func NewPrintLog(
	id string,
	productionID string,
	productIDs []string,
	printedBy string,
	printedAt time.Time,
) (PrintLog, error) {
	pl := PrintLog{
		ID:           strings.TrimSpace(id),
		ProductionID: strings.TrimSpace(productionID),
		ProductIDs:   normalizeIDList(productIDs),
		PrintedBy:    strings.TrimSpace(printedBy),
		PrintedAt:    printedAt.UTC(),
		// QrPayloads は任意フィールドなのでデフォルト nil のまま
	}
	if err := pl.validate(); err != nil {
		return PrintLog{}, err
	}
	return pl, nil
}

// NewInspectionBatch は inspections_by_production 用のバッチを生成します。
// - productionID は必須
// - status は inspecting / completed のいずれか
// - productIDs は 1 件以上必要
// - 各 InspectionItem の inspectionResult / inspectedBy / inspectedAt は nil で初期化
func NewInspectionBatch(
	productionID string,
	status InspectionStatus,
	productIDs []string,
) (InspectionBatch, error) {

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return InspectionBatch{}, ErrInvalidInspectionProductionID
	}
	if !IsValidInspectionStatus(status) {
		return InspectionBatch{}, ErrInvalidInspectionStatus
	}

	ids := normalizeIDList(productIDs)
	if len(ids) == 0 {
		return InspectionBatch{}, ErrInvalidInspectionProductIDs
	}

	inspections := make([]InspectionItem, 0, len(ids))
	for _, id := range ids {
		inspections = append(inspections, InspectionItem{
			ProductID:        id,
			InspectionResult: nil,
			InspectedBy:      nil,
			InspectedAt:      nil,
		})
	}

	batch := InspectionBatch{
		ProductionID: pid,
		Status:       status,
		Inspections:  inspections,
	}

	if err := batch.validate(); err != nil {
		return InspectionBatch{}, err
	}
	return batch, nil
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
	at = at.UTC()
	p.PrintedAt = &at
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
	at = at.UTC()

	p.InspectionResult = result
	p.InspectedBy = &by
	p.InspectedAt = &at
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

	// printedAt: あればゼロでないことだけチェック（printedBy は保持しない）
	if p.PrintedAt != nil && p.PrintedAt.IsZero() {
		return ErrInvalidPrintedAt
	}

	// inspected pair coherence
	switch p.InspectionResult {
	case InspectionPassed, InspectionFailed:
		if p.InspectedBy == nil || strings.TrimSpace(*p.InspectedBy) == "" {
			return ErrInvalidInspectedBy
		}
		if p.InspectedAt == nil || p.InspectedAt.IsZero() {
			return ErrInvalidInspectedAt
		}
	case InspectionNotYet, InspectionNotManufactured:
		if p.InspectedBy != nil || p.InspectedAt != nil {
			return ErrInvalidCoherence
		}
	}

	return nil
}

func (pl PrintLog) validate() error {
	if pl.ID == "" {
		return ErrInvalidPrintLogID
	}
	if pl.ProductionID == "" {
		return ErrInvalidPrintLogProductionID
	}
	if len(pl.ProductIDs) == 0 {
		return ErrInvalidPrintLogProductIDs
	}
	for _, pid := range pl.ProductIDs {
		if strings.TrimSpace(pid) == "" {
			return ErrInvalidPrintLogProductIDs
		}
	}
	if strings.TrimSpace(pl.PrintedBy) == "" {
		return ErrInvalidPrintLogPrintedBy
	}
	if pl.PrintedAt.IsZero() {
		return ErrInvalidPrintLogPrintedAt
	}
	// QrPayloads は任意なのでここではバリデーションしない
	return nil
}

// InspectionBatch のバリデーション
func (b InspectionBatch) validate() error {
	if strings.TrimSpace(b.ProductionID) == "" {
		return ErrInvalidInspectionProductionID
	}
	if !IsValidInspectionStatus(b.Status) {
		return ErrInvalidInspectionStatus
	}
	if len(b.Inspections) == 0 {
		return ErrInvalidInspectionProductIDs
	}

	for _, ins := range b.Inspections {
		if strings.TrimSpace(ins.ProductID) == "" {
			return ErrInvalidInspectionProductIDs
		}

		if ins.InspectionResult != nil {
			// 結果が入っている場合は、結果の妥当性 + inspectedBy / inspectedAt の整合性をチェック
			if !IsValidInspectionResult(*ins.InspectionResult) {
				return ErrInvalidInspectionResult
			}
			if ins.InspectedBy == nil || strings.TrimSpace(*ins.InspectedBy) == "" {
				return ErrInvalidInspectedBy
			}
			if ins.InspectedAt == nil || ins.InspectedAt.IsZero() {
				return ErrInvalidInspectedAt
			}
		} else {
			// inspectionResult が nil の場合、他も nil であるべき
			if ins.InspectedBy != nil || ins.InspectedAt != nil {
				return ErrInvalidCoherence
			}
		}
	}

	return nil
}

// Validate is an exported wrapper for the internal validate method.
// Firestore adaptersなど、product パッケージ外からドメインバリデーションを呼ぶために使用する。
func (b InspectionBatch) Validate() error {
	return b.validate()
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

// Valid inspection
func IsValidInspectionResult(v InspectionResult) bool {
	switch v {
	case InspectionNotYet, InspectionPassed, InspectionFailed, InspectionNotManufactured:
		return true
	default:
		return false
	}
}

// Valid inspection status
func IsValidInspectionStatus(s InspectionStatus) bool {
	switch s {
	case InspectionStatusInspecting, InspectionStatusCompleted:
		return true
	default:
		return false
	}
}
