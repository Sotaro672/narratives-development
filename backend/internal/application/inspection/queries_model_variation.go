// backend/internal/application/inspection/queries_model_variation.go
package inspection

import (
	"context"
	"fmt"
	"strings"

	modeldom "narratives/internal/domain/model"
)

// ModelVariation を 1 件取得する（UI で modelId → size/color/rgb 等の解決に使う想定）
func (u *InspectionUsecase) GetModelVariationByID(
	ctx context.Context,
	variationID string,
) (*modeldom.ModelVariation, error) {

	if u.modelRepo == nil {
		return nil, fmt.Errorf("modelRepo is nil")
	}

	vid := strings.TrimSpace(variationID)
	if vid == "" {
		return nil, modeldom.ErrInvalid
	}

	return u.modelRepo.GetModelVariationByID(ctx, vid)
}
