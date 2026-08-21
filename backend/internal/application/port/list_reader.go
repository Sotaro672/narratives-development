// backend\internal\application\port\list_reader.go
package port

import (
	"context"

	listdom "narratives/internal/domain/list"
)

type ListGetter interface {
	GetByID(
		ctx context.Context,
		id string,
	) (
		listdom.List,
		error,
	)
}

type ListImageLister interface {
	ListByListID(
		ctx context.Context,
		listID string,
	) (
		[]listdom.ListImage,
		error,
	)
}
