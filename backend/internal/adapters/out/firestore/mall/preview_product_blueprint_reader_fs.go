// backend/internal/adapters/out/firestore/mall/preview_product_blueprint_reader_fs.go
package mall

import (
	"context"

	"cloud.google.com/go/firestore"

	mallquery "narratives/internal/application/query/mall"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// previewProductBlueprintReaderFS: ProductBlueprintReader adapter (for PreviewQuery)
type previewProductBlueprintReaderFS struct {
	fs *firestore.Client
	pb interface {
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
		GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error)
	}
}

func NewPreviewProductBlueprintReaderFS(
	fs *firestore.Client,
	pb interface {
		GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
		GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error)
	},
) mallquery.ProductBlueprintReader {
	return previewProductBlueprintReaderFS{fs: fs, pb: pb}
}

func (r previewProductBlueprintReaderFS) GetIDByModelID(ctx context.Context, modelID string) (string, error) {
	if r.fs == nil {
		return "", mallquery.ErrPreviewQueryNotConfigured
	}
	id := modelID
	if id == "" {
		return "", mallquery.ErrInvalidModelID
	}

	snap, err := r.fs.Collection("models").Doc(id).Get(ctx)
	if err != nil {
		return "", err
	}

	data := snap.Data()
	if data == nil {
		return "", nil
	}

	for _, k := range []string{"productBlueprintId", "productBlueprintID", "product_blueprint_id"} {
		if v, ok := data[k]; ok {
			if s, ok := v.(string); ok {
				if s != "" {
					return s, nil
				}
			}
		}
	}

	return "", nil
}

func (r previewProductBlueprintReaderFS) GetPatchByID(ctx context.Context, id string) (productbpdom.Patch, error) {
	if r.pb == nil {
		return productbpdom.Patch{}, mallquery.ErrPreviewQueryNotConfigured
	}
	return r.pb.GetPatchByID(ctx, id)
}

func (r previewProductBlueprintReaderFS) GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error) {
	if r.pb == nil {
		return productbpdom.ProductBlueprint{}, mallquery.ErrPreviewQueryNotConfigured
	}
	return r.pb.GetByID(ctx, id)
}
