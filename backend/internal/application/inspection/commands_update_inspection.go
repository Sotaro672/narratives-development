// backend/internal/application/inspection/commands_update_inspection.go
package inspection

import (
	"context"
	"fmt"
	"strings"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
)

// inspections 内の 1 productId 分を更新する
func (u *InspectionUsecase) UpdateInspectionForProduct(
	ctx context.Context,
	productionID string,
	productID string,
	result *inspectiondom.InspectionResult,
	inspectedBy *string,
	inspectedAt *time.Time,
	status *inspectiondom.InspectionStatus,
) (inspectiondom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}
	pdID := strings.TrimSpace(productID)
	if pdID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	found := false
	for i := range batch.Inspections {
		if strings.TrimSpace(batch.Inspections[i].ProductID) != pdID {
			continue
		}
		found = true

		item := &batch.Inspections[i]

		if result != nil {
			if !inspectiondom.IsValidInspectionResult(*result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}
			r := *result
			item.InspectionResult = &r
		}

		if inspectedBy != nil {
			v := strings.TrimSpace(*inspectedBy)
			if v == "" {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedBy
			}
			item.InspectedBy = &v
		}

		if inspectedAt != nil {
			atUTC := inspectedAt.UTC()
			if atUTC.IsZero() {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
			}
			item.InspectedAt = &atUTC
		}

		break
	}

	if !found {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	if status != nil {
		if !inspectiondom.IsValidInspectionStatus(*status) {
			return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionStatus
		}
		batch.Status = *status
	}

	passedCount := 0
	for _, ins := range batch.Inspections {
		if ins.InspectionResult != nil && *ins.InspectionResult == inspectiondom.InspectionPassed {
			passedCount++
		}
	}
	batch.TotalPassed = passedCount

	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if result != nil && u.productRepo != nil {
		if err := u.productRepo.UpdateInspectionResult(ctx, pdID, *result); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}
	}

	return updated, nil
}
