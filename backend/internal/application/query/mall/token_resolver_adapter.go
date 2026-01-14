// backend\internal\application\query\mall\token_resolver_adapter.go
package mall

import (
	"context"
	"errors"
	"strings"

	usecase "narratives/internal/application/usecase"
)

var ErrTokenResolverNil = errors.New("token_resolver_adapter: token reader is nil")

type tokenResolverAdapter struct {
	r TokenReader // = PreviewQuery.TokenRepo と同じ interface
}

func NewTokenResolverAdapter(r TokenReader) usecase.TokenResolver {
	return &tokenResolverAdapter{r: r}
}

func (a *tokenResolverAdapter) ResolveTokenByProductID(ctx context.Context, productID string) (usecase.TokenForTransfer, error) {
	if a == nil || a.r == nil {
		return usecase.TokenForTransfer{}, ErrTokenResolverNil
	}

	pid := strings.TrimSpace(productID)
	if pid == "" {
		return usecase.TokenForTransfer{}, errors.New("token_resolver_adapter: productId is empty")
	}

	tok, err := a.r.GetByProductID(ctx, pid)
	if err != nil {
		return usecase.TokenForTransfer{}, err
	}
	if tok == nil {
		// TransferUsecase 側で ErrTransferTokenDocNotReady を出したいなら、
		// ここで nil 相当を返すのではなく “空” を返して上位で弾く、でもOK。
		// ただ、現状 usecase.TokenResolver は (TokenForTransfer, error) なので、
		// 「未mint/未作成」を error に寄せるのが安全です。
		return usecase.TokenForTransfer{}, usecase.ErrTransferTokenDocNotReady
	}

	return usecase.TokenForTransfer{
		ProductID:        pid,
		BrandID:          strings.TrimSpace(tok.BrandID),
		MintAddress:      strings.TrimSpace(tok.MintAddress),
		TokenBlueprintID: strings.TrimSpace(tok.TokenBlueprintID),
		ToAddress:        strings.TrimSpace(tok.ToAddress),
	}, nil
}
