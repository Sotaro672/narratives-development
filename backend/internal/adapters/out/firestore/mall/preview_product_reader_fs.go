// backend/internal/adapters/out/firestore/mall/preview_product_reader_fs.go
package mall

import (
	"context"

	"cloud.google.com/go/firestore"

	mallquery "narratives/internal/application/query/mall"
	productdom "narratives/internal/domain/product"
)

// previewProductReaderFS: Firestore -> domain.Product (for PreviewQuery)
type previewProductReaderFS struct {
	fs *firestore.Client
}

func NewPreviewProductReaderFS(fs *firestore.Client) mallquery.ProductReader {
	return previewProductReaderFS{fs: fs}
}

func (r previewProductReaderFS) GetByID(ctx context.Context, productID string) (productdom.Product, error) {
	if r.fs == nil {
		return productdom.Product{}, mallquery.ErrPreviewQueryNotConfigured
	}
	id := productID
	if id == "" {
		return productdom.Product{}, mallquery.ErrInvalidProductID
	}

	doc, err := r.fs.Collection("products").Doc(id).Get(ctx)
	if err != nil {
		return productdom.Product{}, err
	}

	var p productdom.Product
	if err := doc.DataTo(&p); err != nil {
		return productdom.Product{}, err
	}

	p.ID = doc.Ref.ID
	return p, nil
}
