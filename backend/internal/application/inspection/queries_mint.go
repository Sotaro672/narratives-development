// backend/internal/application/inspection/queries_mint.go
package inspection

import (
	"context"
	"fmt"
	"strings"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

// inspectionId に紐づく mint を 1 件取得する
func (u *InspectionUsecase) GetMintByInspectionID(
	ctx context.Context,
	inspectionID string,
) (mintdom.Mint, error) {

	if u.mintRepo == nil {
		return mintdom.Mint{}, fmt.Errorf("mintRepo is nil")
	}

	iid := strings.TrimSpace(inspectionID)
	if iid == "" {
		return mintdom.Mint{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	return u.mintRepo.GetByInspectionID(ctx, iid)
}
