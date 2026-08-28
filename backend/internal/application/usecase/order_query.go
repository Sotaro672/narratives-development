// backend/internal/application/usecase/order_query.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// =======================
// Queries
// =======================

func (u *OrderUsecase) GetByID(
	ctx context.Context,
	id string,
) (orderdom.Order, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *OrderUsecase) ListByAvatarID(
	ctx context.Context,
	avatarID string,
	sort common.Sort,
	page common.Page,
) (common.PageResult[orderdom.Order], error) {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return common.PageResult[orderdom.Order]{},
			fmt.Errorf("order usecase: avatarId is required")
	}

	return u.repo.ListByAvatarID(
		ctx,
		avatarID,
		sort,
		page,
	)
}
