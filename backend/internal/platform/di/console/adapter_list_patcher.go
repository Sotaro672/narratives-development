// backend/internal/platform/di/console/adapter_list_patcher.go
package console

import (
	"context"
	"strings"
	"time"

	listdom "narratives/internal/domain/list"
)

// ============================================================
// Adapter: ListRepositoryFS -> usecase.ListPatcher
// ============================================================

type listPatcherAdapter struct {
	repo interface {
		Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
	}
}

func (a *listPatcherAdapter) UpdateImageID(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (listdom.List, error) {
	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)
	if listID == "" {
		return listdom.List{}, listdom.ErrNotFound
	}

	patch := listdom.ListPatch{
		ImageID:   &imageID,
		UpdatedAt: &now,
		UpdatedBy: updatedBy,
	}
	return a.repo.Update(ctx, listID, patch)
}
