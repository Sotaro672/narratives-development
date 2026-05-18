// backend/internal/application/mint/inspection_batch_loader.go
package mint

import (
	"context"
	"errors"

	inspectiondom "narratives/internal/domain/inspection"
)

// inspection batch を productionId から 1件取得する。
// production / inspection / mint の docId は同一値として扱う。
func (u *MintUsecase) loadInspectionBatchByProductionID(
	ctx context.Context,
	productionID string,
) (*inspectiondom.InspectionBatch, error) {
	if u == nil || u.inspRepo == nil {
		return nil, errors.New("inspection repo is nil")
	}

	if productionID == "" {
		return nil, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := u.inspRepo.GetByProductionID(ctx, productionID)
	if err != nil {
		return nil, err
	}

	return &batch, nil
}
