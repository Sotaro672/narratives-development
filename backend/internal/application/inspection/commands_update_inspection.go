// backend/internal/application/inspection/commands_update_inspection.go
package inspection

import (
	"context"
	"fmt"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
)

// inspections 内の 1 productId 分を更新する。
//
// ネガティブ制では、通常は failed / notManufactured を明示的に入力します。
// ただし、誤って failed / notManufactured にした productId を戻すため、
// 修正操作として passed への更新も許可します。
func (u *InspectionUsecase) UpdateInspectionForProduct(
	ctx context.Context,
	productionID string,
	productID string,
	result *inspectiondom.InspectionResult,
	inspectedBy *string,
	inspectedAt *time.Time,
) (inspectiondom.InspectionBatch, error) {

	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	pdID := productID
	if pdID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	if result == nil {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	if *result != inspectiondom.InspectionPassed &&
		*result != inspectiondom.InspectionFailed &&
		*result != inspectiondom.InspectionNotManufactured {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	if inspectedBy == nil || *inspectedBy == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedBy
	}

	if inspectedAt == nil {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
	}

	atUTC := inspectedAt.UTC()
	if atUTC.IsZero() {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	switch *result {
	case inspectiondom.InspectionPassed:
		if err := batch.MarkPassed(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	case inspectiondom.InspectionFailed:
		if err := batch.MarkFailed(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	case inspectiondom.InspectionNotManufactured:
		if err := batch.MarkNotManufactured(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	default:
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if u.productRepo != nil {
		if err := u.productRepo.UpdateInspectionResult(ctx, pdID, *result); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}
	}
	return updated, nil
}
