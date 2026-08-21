// backend\internal\application\port\cart_reader.go
package port

import (
	"context"

	cartdom "narratives/internal/domain/cart"
)

type CartGetter interface {
	GetByAvatarID(
		ctx context.Context,
		avatarID string,
	) (
		*cartdom.Cart,
		error,
	)
}
