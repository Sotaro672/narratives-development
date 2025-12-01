// backend/internal/domain/production/entity.go
package production

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// 汎用エラー（リポジトリ/サービス共通）
var (
	ErrNotFound = errors.New("production: not found")
	ErrConflict = errors.New("production: conflict")
	ErrInvalid  = errors.New("production: invalid")
)

// 判定ヘルパー
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }
func IsInvalid(err error) bool  { return errors.Is(err, ErrInvalid) }

// ラップヘルパー（原因を保持）
func WrapInvalid(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrInvalid, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrInvalid, msg, err)
}

func WrapConflict(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrConflict, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrConflict, msg, err)
}

func WrapNotFound(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrNotFound, msg, err)
}

// ===== Types (mirror TS) =====

type ModelQuantity struct {
	ModelID  string
	Quantity int
}

// ProductionStatus mirrors TS
type ProductionStatus string

const (
	StatusManufacturing ProductionStatus = "manufacturing"
	StatusInspected     ProductionStatus = "inspected"
	StatusPrinted       ProductionStatus = "printed"
	StatusPlanning      ProductionStatus = "planning"
	StatusDeleted       ProductionStatus = "deleted"
	StatusSuspended     ProductionStatus = "suspended"
)

func IsValidStatus(s ProductionStatus) bool {
	switch s {
	case StatusManufacturing, StatusPrinted, StatusInspected, StatusPlanning, StatusDeleted, StatusSuspended:
		return true
	default:
		return false
	}
}

// Production mirrors shared/types（複数モデル構成)
type Production struct {
	ID                 string
	ProductBlueprintID string
	AssigneeID         string
	Models             []ModelQuantity // [{modelId, quantity}]
	Status             ProductionStatus
	PrintedAt          *time.Time
	PrintedBy          *string // ★ 追加: 印刷担当者
	InspectedAt        *time.Time
	CreatedBy          *string
	CreatedAt          time.Time // optional（ゼロ許容）
	UpdatedAt          time.Time // optional（ゼロ許容）
	UpdatedBy          *string
	DeletedAt          *time.Time
	DeletedBy          *string
}

// ===== Errors =====
var (
	ErrInvalidID                 = errors.New("production: invalid id")
	ErrInvalidProductBlueprintID = errors.New("production: invalid productBlueprintId")
	ErrInvalidAssigneeID         = errors.New("production: invalid assigneeId")
	ErrInvalidModels             = errors.New("production: invalid models")
	ErrInvalidModelID            = errors.New("production: invalid modelId")
	ErrInvalidQuantity           = errors.New("production: invalid quantity")
	ErrInvalidStatus             = errors.New("production: invalid status")
	ErrInvalidPrintedAt          = errors.New("production: invalid printedAt")
	ErrInvalidPrintedBy          = errors.New("production: invalid printedBy") // ★ 追加
	ErrInvalidInspectedAt        = errors.New("production: invalid inspectedAt")
	ErrInvalidCreatedAt          = errors.New("production: invalid createdAt")
	ErrInvalidUpdatedAt          = errors.New("production: invalid updatedAt")
	ErrInvalidUpdatedBy          = errors.New("production: invalid updatedBy")
	ErrInvalidDeletedAt          = errors.New("production: invalid deletedAt")
	ErrInvalidDeletedBy          = errors.New("production: invalid deletedBy")
	ErrTransition                = errors.New("production: invalid status transition")
)

// ===== Constructors =====

// New creates a Production. If status is empty, defaults to manufacturing.
func New(
	id, productBlueprintID, assigneeID string,
	models []ModelQuantity,
	status ProductionStatus,
	printedAt, inspectedAt *time.Time,
	createdBy *string,
	createdAt time.Time,
) (Production, error) {
	if status == "" {
		status = StatusManufacturing
	}
	p := Production{
		ID:                 strings.TrimSpace(id),
		ProductBlueprintID: strings.TrimSpace(productBlueprintID),
		AssigneeID:         strings.TrimSpace(assigneeID),
		Models:             normalizeModels(models),
		Status:             status,
		PrintedAt:          printedAt,
		// PrintedBy はコンストラクタでは nil 初期化（後から更新）
		PrintedBy:   nil,
		InspectedAt: inspectedAt,
		CreatedBy:   normalizePtr(createdBy),
		CreatedAt:   createdAt, // ゼロ許容
	}
	if err := p.validate(); err != nil {
		return Production{}, err
	}
	return p, nil
}

// NewNow is a convenience constructor with CreatedAt=now (UTC).
func NewNow(
	id, productBlueprintID, assigneeID string,
	models []ModelQuantity,
	status ProductionStatus,
) (Production, error) {
	now := time.Now().UTC()
	return New(id, productBlueprintID, assigneeID, models, status, nil, nil, nil, now)
}

// NewFromStringTimes parses printedAt/inspectedAt/createdAt/updatedAt/deletedAt from ISO8601 strings.
// Pass "" for optional times. createdBy/updatedBy/deletedBy は nil または空白で未設定。
func NewFromStringTimes(
	id, productBlueprintID, assigneeID string,
	models []ModelQuantity,
	status ProductionStatus,
	printedAt, inspectedAt, createdAt string,
	createdBy *string,
	updatedAt string,
	updatedBy *string,
	deletedAt string,
	deletedBy *string,
) (Production, error) {
	var (
		printedPtr   *time.Time
		inspectedPtr *time.Time
		created      time.Time
		updated      time.Time
		deletedPtr   *time.Time
	)

	if strings.TrimSpace(printedAt) != "" {
		t, err := parseTime(printedAt)
		if err != nil {
			return Production{}, fmt.Errorf("%w: %v", ErrInvalidPrintedAt, err)
		}
		printedPtr = &t
	}
	if strings.TrimSpace(inspectedAt) != "" {
		t, err := parseTime(inspectedAt)
		if err != nil {
			return Production{}, fmt.Errorf("%w: %v", ErrInvalidInspectedAt, err)
		}
		inspectedPtr = &t
	}
	if strings.TrimSpace(createdAt) != "" {
		t, err := parseTime(createdAt)
		if err != nil {
			return Production{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
		}
		created = t
	}
	if strings.TrimSpace(updatedAt) != "" {
		t, err := parseTime(updatedAt)
		if err != nil {
			return Production{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
		}
		updated = t
	}
	if strings.TrimSpace(deletedAt) != "" {
		t, err := parseTime(deletedAt)
		if err != nil {
			return Production{}, fmt.Errorf("%w: %v", ErrInvalidDeletedAt, err)
		}
		deletedPtr = &t
	}

	p, err := New(id, productBlueprintID, assigneeID, models, status, printedPtr, inspectedPtr, createdBy, created)
	if err != nil {
		return Production{}, err
	}
	// 追加フィールド
	p.UpdatedAt = updated
	p.UpdatedBy = normalizePtr(updatedBy)
	p.DeletedAt = deletedPtr
	p.DeletedBy = normalizePtr(deletedBy)

	if err := p.validate(); err != nil {
		return Production{}, err
	}
	return p, nil
}

// ===== Behavior (state transitions) =====

// MarkPrinted: manufacturing/planning/suspended -> printed
func (p *Production) MarkPrinted(at time.Time) error {
	if !(p.Status == StatusManufacturing || p.Status == StatusPlanning || p.Status == StatusSuspended) {
		return ErrTransition
	}
	if at.IsZero() {
		return ErrInvalidPrintedAt
	}
	at = at.UTC()
	p.Status = StatusPrinted
	p.PrintedAt = &at
	return nil
}

// MarkInspected: printed -> inspected
func (p *Production) MarkInspected(at time.Time) error {
	if p.Status != StatusPrinted {
		return ErrTransition
	}
	if at.IsZero() {
		return ErrInvalidInspectedAt
	}
	at = at.UTC()
	p.Status = StatusInspected
	p.InspectedAt = &at
	return nil
}

// ResetToManufacturing: any -> manufacturing (clears timestamps)
func (p *Production) ResetToManufacturing() {
	p.Status = StatusManufacturing
	p.PrintedAt = nil
	p.PrintedBy = nil
	p.InspectedAt = nil
}

// ===== Validation =====

func (p Production) validate() error {
	if p.ID == "" {
		return ErrInvalidID
	}
	if p.ProductBlueprintID == "" {
		return ErrInvalidProductBlueprintID
	}
	if p.AssigneeID == "" {
		return ErrInvalidAssigneeID
	}
	if len(p.Models) == 0 {
		return ErrInvalidModels
	}
	for _, mq := range p.Models {
		if strings.TrimSpace(mq.ModelID) == "" {
			return ErrInvalidModelID
		}
		if mq.Quantity <= 0 {
			return ErrInvalidQuantity
		}
	}
	if !IsValidStatus(p.Status) {
		return ErrInvalidStatus
	}

	// PrintedBy は nil または非空文字列のみ許容
	if p.PrintedBy != nil && strings.TrimSpace(*p.PrintedBy) == "" {
		return ErrInvalidPrintedBy
	}

	// Status/time coherence
	switch p.Status {
	case StatusManufacturing, StatusPlanning, StatusSuspended:
		// times may be nil
	case StatusPrinted:
		if p.PrintedAt == nil {
			return ErrInvalidPrintedAt
		}
		if p.InspectedAt != nil && p.InspectedAt.Before(*p.PrintedAt) {
			return ErrInvalidInspectedAt
		}
	case StatusInspected:
		if p.PrintedAt == nil {
			return ErrInvalidPrintedAt
		}
		if p.InspectedAt == nil {
			return ErrInvalidInspectedAt
		}
		if p.InspectedAt.Before(*p.PrintedAt) {
			return ErrInvalidInspectedAt
		}
	case StatusDeleted:
		if p.DeletedAt == nil || p.DeletedAt.IsZero() {
			return ErrInvalidDeletedAt
		}
	}

	// CreatedAt/UpdatedAt は optional（ゼロ許容）
	if !p.CreatedAt.IsZero() && !p.UpdatedAt.IsZero() && p.UpdatedAt.Before(p.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// ===== Helpers =====

func normalizeModels(in []ModelQuantity) []ModelQuantity {
	out := make([]ModelQuantity, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, mq := range in {
		id := strings.TrimSpace(mq.ModelID)
		if id == "" || mq.Quantity <= 0 {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			// 既出はスキップ（必要なら数量を合算するロジックに変更可）
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ModelQuantity{ModelID: id, Quantity: mq.Quantity})
	}
	return out
}

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}
	// RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	// Fallback layouts
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

func normalizePtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func dedupTrim(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	return out
}

// ===== ProductBlueprint (mirror TS) =====

// ItemType: "tops" | "bottoms" | "other"
type ItemType string

const (
	ItemTypeTops    ItemType = "tops"
	ItemTypeBottoms ItemType = "bottoms"
	ItemTypeOther   ItemType = "other"
)

// ProductIdTagType: "qr" | "nfc"
type ProductIdTagType string

const (
	ProductIdTagQR  ProductIdTagType = "qr"
	ProductIdTagNFC ProductIdTagType = "nfc"
)

// FileRef mirrors { name: string; url: string }
type FileRef struct {
	Name string
	URL  string
}

// ProductIdTag mirrors TS
type ProductIdTag struct {
	Type           ProductIdTagType
	LogoDesignFile *FileRef // optional
}

// ProductBlueprint mirrors TS
type ProductBlueprint struct {
	ID               string
	ProductName      string
	BrandID          string
	ItemType         ItemType
	ModelVariation   []string
	Fit              string
	Material         string
	Weight           float64
	QualityAssurance []string
	ProductIdTag     ProductIdTag
	AssigneeID       string
	CreatedBy        *string
	CreatedAt        time.Time
	UpdatedBy        *string
	UpdatedAt        time.Time
	DeletedBy        *string
	DeletedAt        *time.Time
}

// Optional: minimal constructor/validator (can be extended as needed)
func NewProductBlueprint(
	id, productName, brandID string,
	itemType ItemType,
	modelVariation []string,
	fit, material string,
	weight float64,
	qa []string,
	tag ProductIdTag,
	assigneeID string,
	createdBy *string,
	createdAt time.Time,
) ProductBlueprint {
	return ProductBlueprint{
		ID:               strings.TrimSpace(id),
		ProductName:      strings.TrimSpace(productName),
		BrandID:          strings.TrimSpace(brandID),
		ItemType:         itemType,
		ModelVariation:   dedupTrim(modelVariation),
		Fit:              strings.TrimSpace(fit),
		Material:         strings.TrimSpace(material),
		Weight:           weight,
		QualityAssurance: dedupTrim(qa),
		ProductIdTag:     tag,
		AssigneeID:       strings.TrimSpace(assigneeID),
		CreatedBy:        normalizePtr(createdBy),
		CreatedAt:        createdAt.UTC(),
	}
}
