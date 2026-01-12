// backend/internal/adapters/out/firestore/token_reader_fs.go
package firestore

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mallquery "narratives/internal/application/query/mall"
)

type TokenReaderFS struct {
	Client *firestore.Client
}

func NewTokenReaderFS(client *firestore.Client) *TokenReaderFS {
	return &TokenReaderFS{Client: client}
}

// preview_query.go の mall.TokenReader を満たす
func (r *TokenReaderFS) GetByProductID(ctx context.Context, productID string) (*mallquery.TokenInfo, error) {
	if r == nil || r.Client == nil {
		return nil, mallquery.ErrPreviewQueryNotConfigured
	}
	id := strings.TrimSpace(productID)
	if id == "" {
		return nil, mallquery.ErrInvalidProductID
	}

	snap, err := r.Client.Collection("tokens").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil // ✅ token が無いのは正常（未mintなど）
		}
		return nil, err
	}

	raw := snap.Data()
	out := &mallquery.TokenInfo{
		ProductID: id,
	}

	// map から柔軟に読む（Firestoreがスキーマレスなので）
	if v, ok := raw["brandId"].(string); ok {
		out.BrandID = strings.TrimSpace(v)
	}
	if v, ok := raw["tokenBlueprintId"].(string); ok {
		out.TokenBlueprintID = strings.TrimSpace(v)
	}
	if v, ok := raw["mintAddress"].(string); ok {
		out.MintAddress = strings.TrimSpace(v)
	}

	// ✅ 命名統一: TokenInfo 側は OnChainTxSignature にするのがおすすめ
	if v, ok := raw["onChainTxSignature"].(string); ok {
		out.OnChainTxSignature = strings.TrimSpace(v)
	} else if v, ok := raw["onchainTxSignature"].(string); ok {
		out.OnChainTxSignature = strings.TrimSpace(v)
	} else if v, ok := raw["txSignature"].(string); ok {
		out.OnChainTxSignature = strings.TrimSpace(v)
	} else if v, ok := raw["signature"].(string); ok {
		out.OnChainTxSignature = strings.TrimSpace(v)
	}

	// Firestore 側に productId が入っている場合はそれを優先
	if v, ok := raw["productId"].(string); ok && strings.TrimSpace(v) != "" {
		out.ProductID = strings.TrimSpace(v)
	}

	return out, nil
}
