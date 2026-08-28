// backend/internal/application/port/avatar_reader.go
package port

import (
	"context"

	avatardom "narratives/internal/domain/avatar"
)

// AvatarDisplayResolver resolves avatar information required for transfer display.
type AvatarDisplayResolver interface {
	GetByID(
		ctx context.Context,
		id string,
	) (avatardom.Avatar, error)
}
