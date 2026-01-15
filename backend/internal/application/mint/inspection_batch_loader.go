// backend/internal/application/mint/inspection_batch_loader.go
package mint

import (
	"context"
	"errors"
	"strings"

	inspectiondom "narratives/internal/domain/inspection"
)

// inspection batch を productionId から 1件取得（互換）
// - inspRepo が GetByProductionID を持つ場合はそれを優先
// - 無い場合は ListByProductionID で代替
func (u *MintUsecase) loadInspectionBatchByProductionID(ctx context.Context, productionID string) (*inspectiondom.InspectionBatch, error) {
	if u == nil || u.inspRepo == nil {
		return nil, errors.New("inspection repo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return nil, errors.New("productionID is empty")
	}

	// 1) GetByProductionID があれば最優先
	if getter, ok := any(u.inspRepo).(interface {
		GetByProductionID(ctx context.Context, productionID string) (inspectiondom.InspectionBatch, error)
	}); ok {
		b, err := getter.GetByProductionID(ctx, pid)
		if err != nil {
			return nil, err
		}
		return &b, nil
	}

	// 2) ListByProductionID のみでもOK
	if lister, ok := any(u.inspRepo).(interface {
		ListByProductionID(ctx context.Context, productionIDs []string) ([]inspectiondom.InspectionBatch, error)
	}); ok {
		list, err := lister.ListByProductionID(ctx, []string{pid})
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, inspectiondom.ErrNotFound
		}
		// productionId 一致を優先
		for i := range list {
			if strings.TrimSpace(list[i].ProductionID) == pid {
				b := list[i]
				return &b, nil
			}
		}
		b := list[0]
		return &b, nil
	}

	return nil, errors.New("inspection repo does not support GetByProductionID/ListByProductionID")
}
