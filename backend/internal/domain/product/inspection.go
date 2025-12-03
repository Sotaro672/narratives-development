// backend/internal/domain/product/inspection.go
package product

import (
	"errors"
	"strings"
	"time"
)

// ===============================
// Inspection batch (inspections)
// ===============================

// InspectionStatus は inspections のステータス
type InspectionStatus string

const (
	InspectionStatusInspecting InspectionStatus = "inspecting"
	InspectionStatusCompleted  InspectionStatus = "completed"
)

// InspectionItem は 1 つの productId に対する検査状態を表す
type InspectionItem struct {
	ProductID        string            `json:"productId"`
	InspectionResult *InspectionResult `json:"inspectionResult"`
	InspectedBy      *string           `json:"inspectedBy"`
	InspectedAt      *time.Time        `json:"inspectedAt"`
}

// InspectionBatch は 1 productionId に紐づく inspections ドキュメント
type InspectionBatch struct {
	ProductionID string           `json:"productionId"`
	Status       InspectionStatus `json:"status"`
	Inspections  []InspectionItem `json:"inspections"`
}

// ===============================
// Errors（inspection 専用）
// ===============================

var (
	ErrInvalidInspectionProductionID = errors.New("inspection: invalid productionId")
	ErrInvalidInspectionStatus       = errors.New("inspection: invalid status")
	ErrInvalidInspectionProductIDs   = errors.New("inspection: invalid productIds")
)

// ===============================
// Constructors
// ===============================

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
		r := InspectionNotYet
		inspections = append(inspections, InspectionItem{
			ProductID:        id,
			InspectionResult: &r,
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
// Behavior / Validation
// ===============================

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
			if !IsValidInspectionResult(*ins.InspectionResult) {
				return ErrInvalidInspectionResult
			}

			switch *ins.InspectionResult {
			case InspectionPassed, InspectionFailed:
				if ins.InspectedBy == nil || strings.TrimSpace(*ins.InspectedBy) == "" {
					return ErrInvalidInspectedBy
				}
				if ins.InspectedAt == nil || ins.InspectedAt.IsZero() {
					return ErrInvalidInspectedAt
				}
			case InspectionNotYet, InspectionNotManufactured:
				if ins.InspectedBy != nil || ins.InspectedAt != nil {
					return ErrInvalidCoherence
				}
			}
		} else {
			if ins.InspectedBy != nil || ins.InspectedAt != nil {
				return ErrInvalidCoherence
			}
		}
	}
	return nil
}

// Exported wrapper
func (b InspectionBatch) Validate() error {
	return b.validate()
}

// ===============================
// Status validator
// ===============================

func IsValidInspectionStatus(s InspectionStatus) bool {
	return s == InspectionStatusInspecting || s == InspectionStatusCompleted
}
