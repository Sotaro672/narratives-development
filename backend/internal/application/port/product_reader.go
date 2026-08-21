// backend\internal\application\port\product_reader.go
package port

import (
	"context"

	productdom "narratives/internal/domain/product"
)

type ProductGetter interface {
	GetByID(
		ctx context.Context,
		productID string,
	) (
		productdom.Product,
		error,
	)
}
