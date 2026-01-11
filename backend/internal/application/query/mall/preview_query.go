// backend/internal/application/query/mall/preview_query.go
package mall

import (
	"context"
	"errors"
	"strings"

	productdom "narratives/internal/domain/product"
)

// ------------------------------------------------------------
// Errors
// ------------------------------------------------------------

var (
	ErrPreviewQueryNotConfigured = errors.New("preview_query: not configured")
	ErrInvalidProductID          = errors.New("preview_query: invalid productId")
	ErrModelIDEmpty              = errors.New("preview_query: resolved modelId is empty")
)

// ------------------------------------------------------------
// Ports (dependency interfaces)
// ------------------------------------------------------------

// ProductReader is a minimal read port for preview usecases.
// We only need "productId -> product -> modelId".
//
// NOTE:
// If your existing repository method name differs (e.g. FindByID / Get / Read),
// create a tiny adapter that implements this interface.
type ProductReader interface {
	GetByID(ctx context.Context, productID string) (productdom.Product, error)
}

// ------------------------------------------------------------
// Query
// ------------------------------------------------------------

// PreviewQuery resolves preview entry info from productId.
// This struct is intended to be injected as cont.PreviewQ.
type PreviewQuery struct {
	ProductRepo ProductReader
}

// NewPreviewQuery constructs PreviewQuery.
func NewPreviewQuery(productRepo ProductReader) *PreviewQuery {
	return &PreviewQuery{
		ProductRepo: productRepo,
	}
}

// ResolveModelIDByProductID resolves modelId from productId.
//
// This method name/signature MUST match what preview_handler.go expects.
// (Your compile error indicates this is the missing method.)
func (q *PreviewQuery) ResolveModelIDByProductID(
	ctx context.Context,
	productID string,
) (string, error) {
	if q == nil || q.ProductRepo == nil {
		return "", ErrPreviewQueryNotConfigured
	}

	id := strings.TrimSpace(productID)
	if id == "" {
		return "", ErrInvalidProductID
	}

	p, err := q.ProductRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	modelID := strings.TrimSpace(p.ModelID)
	if modelID == "" {
		return "", ErrModelIDEmpty
	}

	return modelID, nil
}
