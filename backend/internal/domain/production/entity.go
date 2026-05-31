// backend/internal/domain/production/entity.go
package production

import (
	"errors"
	"fmt"
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
	PrintedBy          *string // 印刷担当者
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

// New creates a persisted Production.
// ID is required.
func New(
	id, productBlueprintID, assigneeID string,
	models []ModelQuantity,
	printed bool,
	printedAt *time.Time,
	createdBy *string,
	createdAt time.Time,
) (Production, error) {
	p := Production{
		ID:                 id,
		ProductBlueprintID: productBlueprintID,
		AssigneeID:         assigneeID,
		Models:             models,
		Printed:            printed,
		PrintedAt:          printedAt,
		PrintedBy:          nil,
		CreatedBy:          createdBy,
		CreatedAt:          createdAt,
	}

	if err := p.Validate(); err != nil {
		return Production{}, err
	}
	return p, nil
}

// NewForCreate creates a Production before repository ID assignment.
// ID is not required here because repository.Create may generate it.
func NewForCreate(
	productBlueprintID, assigneeID string,
	models []ModelQuantity,
	printed bool,
	printedAt *time.Time,
	createdBy *string,
	createdAt time.Time,
) (Production, error) {
	p := Production{
		ID:                 "",
		ProductBlueprintID: productBlueprintID,
		AssigneeID:         assigneeID,
		Models:             models,
		Printed:            printed,
		PrintedAt:          printedAt,
		PrintedBy:          nil,
		CreatedBy:          createdBy,
		CreatedAt:          createdAt,
	}

	if err := p.ValidateForCreate(); err != nil {
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

	return p.Validate()
}

// ApplyUpdate applies update values to an existing Production and validates it.
func (p *Production) ApplyUpdate(
	assigneeID string,
	models []ModelQuantity,
	printed *bool,
	printedAt *time.Time,
	printedBy *string,
	updatedBy *string,
	updatedAt time.Time,
) error {
	if assigneeID != "" {
		p.AssigneeID = assigneeID
	}

	if len(models) > 0 {
		p.Models = models
	}

	if printed != nil {
		p.Printed = *printed
		if !p.Printed {
			p.PrintedAt = nil
			p.PrintedBy = nil
		}
	}

	if printedAt != nil && !printedAt.IsZero() {
		t := printedAt.UTC()
		p.PrintedAt = &t
		p.Printed = true
	}

	if printedBy != nil {
		if *printedBy == "" {
			p.PrintedBy = nil
		} else {
			v := *printedBy
			p.PrintedBy = &v
			p.Printed = true
		}
	}

	if !p.Printed {
		p.PrintedAt = nil
		p.PrintedBy = nil
	}

	if updatedBy != nil {
		if *updatedBy == "" {
			p.UpdatedBy = nil
		} else {
			v := *updatedBy
			p.UpdatedBy = &v
		}
	}

	if !updatedAt.IsZero() {
		p.UpdatedAt = updatedAt.UTC()
	}

	return p.Validate()
}

// ===== Validation =====

// Validate validates a persisted Production.
// ID is required.
func (p Production) Validate() error {
	return p.validate(true)
}

// ValidateForCreate validates a Production before repository ID assignment.
// ID is not required.
func (p Production) ValidateForCreate() error {
	return p.validate(false)
}

func (p Production) validate(requireID bool) error {
	if requireID && p.ID == "" {
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
		if mq.ModelID == "" {
			return ErrInvalidModelID
		}
		if mq.Quantity <= 0 {
			return ErrInvalidQuantity
		}
	}

	if p.PrintedBy != nil && *p.PrintedBy == "" {
		return ErrInvalidPrintedBy
	}
	if p.CreatedBy != nil && *p.CreatedBy == "" {
		return ErrInvalidCreatedAt
	}
	if p.UpdatedBy != nil && *p.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}

	if p.Printed {
		if p.PrintedAt == nil || p.PrintedAt.IsZero() {
			return ErrInvalidPrintedAt
		}
	} else {
		if p.PrintedAt != nil {
			return ErrInvalidPrintedAt
		}
		if p.PrintedBy != nil {
			return ErrInvalidPrintedBy
		}
	}

	if !p.CreatedAt.IsZero() && !p.UpdatedAt.IsZero() && p.UpdatedAt.Before(p.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	return nil
}
