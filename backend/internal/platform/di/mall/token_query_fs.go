// backend/internal/platform/di/mall/token_query_fs.go
package mall

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	tokendom "narratives/internal/domain/token"
)

// TokenQueryFS (mintAddress -> productId(docId), brandId, metadataUri)
// - outfs.NewTokenQueryFS(...) が無い前提で、DI側に最小実装を置く
type tokenQueryFS struct {
	client *firestore.Client
	col    string
}

func newTokenQueryFS(client *firestore.Client) *tokenQueryFS {
	return &tokenQueryFS{
		client: client,
		col:    "tokens",
	}
}

func (q *tokenQueryFS) ResolveTokenByMintAddress(
	ctx context.Context,
	mintAddress string,
) (tokendom.ResolveTokenByMintAddressResult, error) {

	if q == nil || q.client == nil {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("tokenQueryFS is not initialized")
	}

	m := strings.TrimSpace(mintAddress)
	if m == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("mintAddress is empty")
	}

	it := q.client.Collection(q.col).
		Where("mintAddress", "==", m).
		Limit(2).
		Documents(ctx)
	defer it.Stop()

	// 1件目
	doc, err := it.Next()
	if errors.Is(err, iterator.Done) {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("token not found for mintAddress")
	}
	if err != nil {
		return tokendom.ResolveTokenByMintAddressResult{}, err
	}

	// 2件目があればユニーク違反
	doc2, err := it.Next()
	if err == nil && doc2 != nil {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("multiple tokens found for mintAddress")
	}
	if err != nil && !errors.Is(err, iterator.Done) {
		return tokendom.ResolveTokenByMintAddressResult{}, err
	}

	raw := doc.Data()

	brandID, _ := raw["brandId"].(string)
	metadataURI, _ := raw["metadataUri"].(string)

	brandID = strings.TrimSpace(brandID)
	metadataURI = strings.TrimSpace(metadataURI)

	// Firestore では docID が productId（あなたの設計前提）
	productID := strings.TrimSpace(doc.Ref.ID)
	if productID == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("resolved productId is empty")
	}

	return tokendom.ResolveTokenByMintAddressResult{
		ProductID:   productID,
		MintAddress: m,
		BrandID:     brandID,
		MetadataURI: metadataURI,
	}, nil
}
