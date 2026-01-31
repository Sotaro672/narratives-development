// backend/internal/application/inspection/queries_screen_dto.go
package inspection

import (
	"context"
	"strings"

	inspectiondto "narratives/internal/application/inspection/dto"
	mintdom "narratives/internal/domain/mint"
)

// 画面用 DTO（InspectionBatch + Mint を結合して返す）
func (u *InspectionUsecase) GetBatchForScreenByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondto.InspectionBatchForScreenDTO, error) {

	batch, err := u.GetBatchByProductionID(ctx, productionID)
	if err != nil {
		return inspectiondto.InspectionBatchForScreenDTO{}, err
	}

	// batch.MintID は *string なので nil 安全に扱う
	var mintDTO *inspectiondto.MintDTO
	if u.mintRepo != nil && batch.MintID != nil {
		mintID := strings.TrimSpace(*batch.MintID)
		if mintID != "" {
			m, err := u.mintRepo.GetByInspectionID(ctx, mintID)
			if err == nil {
				dto := inspectiondto.NewMintDTO(m, batch.ProductionID)
				mintDTO = &dto
			} else if err != mintdom.ErrNotFound {
				return inspectiondto.InspectionBatchForScreenDTO{}, err
			}
		}
	}

	return inspectiondto.NewInspectionBatchForScreenDTO(batch, mintDTO), nil
}
