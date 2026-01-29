// backend/internal/platform/di/mall/container_local_adapters.go
package mall

import (
	"context"
	"errors"
	"strings"

	avatardom "narratives/internal/domain/avatar"
)

// Adapter: modelId -> productBlueprintId (usecase port)
type modelPBIDResolverAdapter struct {
	r interface {
		GetIDByModelID(ctx context.Context, modelID string) (string, error)
	}
}

func (a modelPBIDResolverAdapter) GetProductBlueprintIDByModelID(ctx context.Context, modelID string) (string, error) {
	if a.r == nil {
		return "", errors.New("modelPBIDResolverAdapter: resolver is nil")
	}
	return a.r.GetIDByModelID(ctx, strings.TrimSpace(modelID))
}

// Adapter: avatarId -> avatarName (sharedquery.AvatarNameReader)
type avatarNameReaderAdapter struct {
	repo interface {
		GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
	}
}

func (a avatarNameReaderAdapter) GetNameByID(ctx context.Context, avatarID string) (string, error) {
	id := strings.TrimSpace(avatarID)
	if id == "" {
		return "", errors.New("avatarNameReaderAdapter: avatarID is empty")
	}

	av, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(av.AvatarName), nil
}
