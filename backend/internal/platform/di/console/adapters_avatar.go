// backend/internal/platform/di/console/adapters_avatar.go
package console

import (
	"context"
	"errors"
	"strings"

	avatardom "narratives/internal/domain/avatar"
)

// ✅ Adapter: avatarId -> avatarName (sharedquery.AvatarNameReader)
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
