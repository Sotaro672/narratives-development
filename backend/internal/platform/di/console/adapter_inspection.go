// backend/internal/platform/di/console/adapter_inspection.go
package console

import (
	"context"
	"errors"

	inspectiondom "narratives/internal/domain/inspection"
	productdom "narratives/internal/domain/product"
)

// ========================================
// inspection 用: products.UpdateInspectionResult アダプタ
// ========================================

type inspectionProductRepoAdapter struct {
	repo interface {
		UpdateInspectionResult(ctx context.Context, productID string, result productdom.InspectionResult) error
	}
}

func (a *inspectionProductRepoAdapter) UpdateInspectionResult(
	ctx context.Context,
	productID string,
	result inspectiondom.InspectionResult,
) error {
	if a == nil || a.repo == nil {
		return errors.New("inspectionProductRepoAdapter: repo is nil")
	}
	return a.repo.UpdateInspectionResult(ctx, productID, productdom.InspectionResult(result))
}
