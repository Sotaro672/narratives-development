// backend/internal/adapters/out/firestore/mall/transfer_token_resolver_fs.go
package mall

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
)

var (
	errTokenResolverNotConfigured = errors.New("firestore.mall: tokenResolverFS not configured")
	errTokenDocNotFound           = errors.New("firestore.mall: token doc not found")
)

type tokenResolverFS struct {
	fs  *firestore.Client
	col string
}

func NewTokenResolverFS(
	fs *firestore.Client,
	col string,
) usecase.TokenResolver {
	return &tokenResolverFS{
		fs:  fs,
		col: col,
	}
}

func (r *tokenResolverFS) ResolveTokenByProductID(
	ctx context.Context,
	productID string,
) (usecase.TokenForTransfer, error) {
	if r == nil || r.fs == nil {
		return usecase.TokenForTransfer{},
			errTokenResolverNotConfigured
	}
	if productID == "" {
		return usecase.TokenForTransfer{},
			errors.New("tokenResolverFS: productId is empty")
	}

	col := r.col
	if col == "" {
		col = "tokens"
	}

	snap, err := r.fs.Collection(col).
		Doc(productID).
		Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return usecase.TokenForTransfer{},
				errTokenDocNotFound
		}

		return usecase.TokenForTransfer{}, err
	}

	if snap == nil || !snap.Exists() {
		return usecase.TokenForTransfer{},
			errTokenDocNotFound
	}

	raw := snap.Data()
	if raw == nil {
		return usecase.TokenForTransfer{},
			errTokenDocNotFound
	}

	getStr := func(key string) string {
		value, ok := raw[key]
		if !ok {
			return ""
		}

		s, ok := value.(string)
		if !ok {
			return ""
		}

		return s
	}

	return usecase.TokenForTransfer{
		ProductID:        productID,
		BrandID:          getStr("brandId"),
		AssetID:          getStr("assetId"),
		TokenBlueprintID: getStr("tokenBlueprintId"),
	}, nil
}
