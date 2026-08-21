package port

import (
	"context"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

type TokenBlueprintGetter interface {
	GetByID(
		ctx context.Context,
		id string,
	) (
		*tbdom.TokenBlueprint,
		error,
	)
}
