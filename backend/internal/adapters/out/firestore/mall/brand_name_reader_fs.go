// backend/internal/adapters/out/firestore/mall/brand_name_reader_fs.go
package mall

import (
	"context"

	"cloud.google.com/go/firestore"

	outfs "narratives/internal/adapters/out/firestore"
	branddom "narratives/internal/domain/brand"
)

type BrandNameReaderFS struct {
	repo *outfs.BrandRepositoryFS
}

func NewBrandNameReaderFS(client *firestore.Client) *BrandNameReaderFS {
	return &BrandNameReaderFS{
		repo: outfs.NewBrandRepositoryFS(client),
	}
}

func (r *BrandNameReaderFS) TryGetBrandName(ctx context.Context, brandID string) (string, bool, error) {
	if r == nil || r.repo == nil || brandID == "" {
		return "", false, nil
	}

	b, err := r.repo.GetByID(ctx, brandID)
	if err != nil {
		if err == branddom.ErrNotFound {
			return "", false, nil
		}
		return "", false, err
	}

	if b.Name == "" {
		return "", false, nil
	}

	return b.Name, true, nil
}
