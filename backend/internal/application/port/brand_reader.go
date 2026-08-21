// backend\internal\application\port\brand_reader.go
package port

import (
	"context"

	branddom "narratives/internal/domain/brand"
)

type BrandGetter interface {
	GetByID(
		ctx context.Context,
		id string,
	) (
		branddom.Brand,
		error,
	)
}
