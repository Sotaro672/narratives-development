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

func NewTokenResolverFS(fs *firestore.Client, col string) usecase.TokenResolver {
	return &tokenResolverFS{fs: fs, col: col}
}

func (r *tokenResolverFS) ResolveTokenByProductID(ctx context.Context, productID string) (usecase.TokenForTransfer, error) {
	if r == nil || r.fs == nil {
		return usecase.TokenForTransfer{}, errTokenResolverNotConfigured
	}
	pid := productID
	if pid == "" {
		return usecase.TokenForTransfer{}, errors.New("tokenResolverFS: productId is empty")
	}
	col := r.col
	if col == "" {
		col = "tokens"
	}

	snap, err := r.fs.Collection(col).Doc(pid).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return usecase.TokenForTransfer{}, errTokenDocNotFound
		}
		return usecase.TokenForTransfer{}, err
	}
	raw := snap.Data()
	if raw == nil {
		return usecase.TokenForTransfer{}, errTokenDocNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok {
					if s != "" {
						return s
					}
				}
			}
		}
		return ""
	}

	return usecase.TokenForTransfer{
		ProductID: pid,
		BrandID:   getStr("brandId", "brandID"),
		MintAddress: getStr(
			"mintAddress",
			"mint_address",
		),
		TokenBlueprintID: getStr(
			"tokenBlueprintId",
			"tokenBlueprintID",
			"token_blueprint_id",
		),
		ToAddress: getStr("toAddress", "to_address"),
	}, nil
}
