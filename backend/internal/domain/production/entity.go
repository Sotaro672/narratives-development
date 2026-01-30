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

// Production mirrors shared/types（複数モデル構成)
type Production struct {
	ID                 string
	ProductBlueprintID string
	AssigneeID         string
	Models             []ModelQuantity // [{modelId, quantity}]
	Printed            bool            // printed:boolean
	PrintedAt          *time.Time
	PrintedBy          *string // ★ 印刷担当者
	CreatedBy          *string
	CreatedAt          time.Time // optional（ゼロ許容）
	UpdatedAt          time.Time // optional（ゼロ許容）
	UpdatedBy          *string
}

// ===== Errors =====
var (
	ErrInvalidID                 = errors.New("production: invalid id")
	ErrInvalidProductBlueprintID = errors.New("production: invalid productBlueprintId")
	ErrInvalidAssigneeID         = errors.New("production: invalid assigneeId")
	ErrInvalidModels             = errors.New("production: invalid models")
	ErrInvalidModelID            = errors.New("production: invalid modelId")
	ErrInvalidQuantity           = errors.New("production: invalid quantity")
	ErrInvalidPrintedAt          = errors.New("production: invalid printedAt")
	ErrInvalidPrintedBy          = errors.New("production: invalid printedBy")
	ErrInvalidCreatedAt          = errors.New("production: invalid createdAt")
	ErrInvalidUpdatedAt          = errors.New("production: invalid updatedAt")
	ErrInvalidUpdatedBy          = errors.New("production: invalid updatedBy")
	ErrTransition                = errors.New("production: invalid printed transition")
)

// ===== Constructors =====

// New creates a Production.
func New(
	id, productBlueprintID, assigneeID string,
	models []ModelQuantity,
	printed bool,
	printedAt *time.Time,
	createdBy *string,
	createdAt time.Time,
) (Production, error) {
	p := Production{
		ID:                 strings.TrimSpace(id),
		ProductBlueprintID: strings.TrimSpace(productBlueprintID),
		AssigneeID:         strings.TrimSpace(assigneeID),
		Models:             normalizeModels(models),
		Printed:            printed,
		PrintedAt:          printedAt,
		// PrintedBy はコンストラクタでは nil 初期化（後から更新）
		PrintedBy: nil,
		CreatedBy: normalizePtr(createdBy),
		CreatedAt: createdAt, // ゼロ許容
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
	printed bool,
) (Production, error) {
	now := time.Now().UTC()
	return New(id, productBlueprintID, assigneeID, models, printed, nil, nil, now)
}

// NewFromStringTimes parses printedAt/createdAt/updatedAt from ISO8601 strings.
// Pass "" for optional times. createdBy/updatedBy は nil または空白で未設定。
func NewFromStringTimes(
	id, productBlueprintID, assigneeID string,
	models []ModelQuantity,
	printed bool,
	printedAt, createdAt string,
	createdBy *string,
	updatedAt string,
	updatedBy *string,
) (Production, error) {
	var (
		printedPtr *time.Time
		created    time.Time
		updated    time.Time
	)

	if strings.TrimSpace(printedAt) != "" {
		t, err := parseTime(printedAt)
		if err != nil {
			return Production{}, fmt.Errorf("%w: %v", ErrInvalidPrintedAt, err)
		}
		printedPtr = &t
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

	p, err := New(id, productBlueprintID, assigneeID, models, printed, printedPtr, createdBy, created)
	if err != nil {
		return Production{}, err
	}
	// 追加フィールド
	p.UpdatedAt = updated
	p.UpdatedBy = normalizePtr(updatedBy)

	if err := p.validate(); err != nil {
		return Production{}, err
	}
	return p, nil
}

// ===== Behavior (state transitions) =====

// MarkPrinted: unprinted(false) -> printed(true)
func (p *Production) MarkPrinted(at time.Time) error {
	if p.Printed {
		return ErrTransition
	}
	if at.IsZero() {
		return ErrInvalidPrintedAt
	}
	at = at.UTC()
	p.Printed = true
	p.PrintedAt = &at
	return nil
}

// ResetToUnprinted: printed(true) -> unprinted(false) (clears timestamps)
func (p *Production) ResetToUnprinted() {
	p.Printed = false
	p.PrintedAt = nil
	p.PrintedBy = nil
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

	// PrintedBy は nil または非空文字列のみ許容
	if p.PrintedBy != nil && strings.TrimSpace(*p.PrintedBy) == "" {
		return ErrInvalidPrintedBy
	}

	// Printed/time coherence
	if p.Printed {
		if p.PrintedAt == nil || p.PrintedAt.IsZero() {
			return ErrInvalidPrintedAt
		}
	} else {
		// 未印刷なら PrintedAt/PrintedBy は未設定が原則
		if p.PrintedAt != nil {
			return ErrInvalidPrintedAt
		}
		if p.PrintedBy != nil {
			return ErrInvalidPrintedBy
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
