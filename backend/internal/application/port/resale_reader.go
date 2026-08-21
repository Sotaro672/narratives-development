// backend\internal\application\port\resale_reader.go
package port

import (
	"context"

	resaledom "narratives/internal/domain/resale"
)

type ResaleGetter interface {
	GetByID(
		ctx context.Context,
		id string,
	) (
		resaledom.Resale,
		error,
	)
}

type ResaleImageLister interface {
	ListByResaleID(
		ctx context.Context,
		resaleID string,
	) (
		[]resaledom.ResaleImage,
		error,
	)
}
