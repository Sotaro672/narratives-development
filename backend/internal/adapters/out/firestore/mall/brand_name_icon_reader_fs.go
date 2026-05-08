// backend/internal/adapters/out/firestore/mall/brand_name_icon_reader_fs.go
package mall

import (
	"context"

	"cloud.google.com/go/firestore"

	outfs "narratives/internal/adapters/out/firestore"
	branddom "narratives/internal/domain/brand"
)

type BrandNameIconReaderFS struct {
	repo *outfs.BrandRepositoryFS
}

func NewBrandNameIconReaderFS(client *firestore.Client) *BrandNameIconReaderFS {
	return &BrandNameIconReaderFS{
		repo: outfs.NewBrandRepositoryFS(client),
	}
}

func (r *BrandNameIconReaderFS) TryGetBrandNameIcon(ctx context.Context, brandID string) (string, string, bool, error) {
	if r == nil || r.repo == nil || brandID == "" {
		return "", "", false, nil
	}

	b, err := r.repo.GetByID(ctx, brandID)
	if err != nil {
		if err == branddom.ErrNotFound {
			return "", "", false, nil
		}
		return "", "", false, err
	}

	if b.Name == "" {
		return "", "", false, nil
	}

	return b.Name, b.BrandIcon, true, nil
}
