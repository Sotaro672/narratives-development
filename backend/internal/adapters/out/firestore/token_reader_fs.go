// backend/internal/adapters/out/firestore/token_reader_fs.go
package firestore

import (
	"context"
	"strings"
	"time"

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

// Firestore tokens collection DTO（実データのフィールド名を正として固定）
type tokenDoc struct {
	BrandID            string    `firestore:"brandId"`
	MetadataURI        string    `firestore:"metadataUri"`
	MintAddress        string    `firestore:"mintAddress"`
	MintedAt           time.Time `firestore:"mintedAt"` // timestamp
	OnChainTxSignature string    `firestore:"onChainTxSignature"`
	ToAddress          string    `firestore:"toAddress"`
	TokenBlueprintID   string    `firestore:"tokenBlueprintId"`
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

	// doc はあるが中身が無いケースは token無し相当として扱う
	if snap.Data() == nil {
		return nil, nil
	}

	var d tokenDoc
	if err := snap.DataTo(&d); err != nil {
		return nil, err
	}

	out := &mallquery.TokenInfo{
		ProductID:          id,
		BrandID:            strings.TrimSpace(d.BrandID),
		TokenBlueprintID:   strings.TrimSpace(d.TokenBlueprintID),
		MintAddress:        strings.TrimSpace(d.MintAddress),
		ToAddress:          strings.TrimSpace(d.ToAddress),
		MetadataURI:        strings.TrimSpace(d.MetadataURI),
		OnChainTxSignature: strings.TrimSpace(d.OnChainTxSignature),
	}

	// mintedAt は Firestore timestamp（time.Time）として受け、文字列に落とす（DTOがstring前提のため）
	if !d.MintedAt.IsZero() {
		out.MintedAt = d.MintedAt.UTC().Format(time.RFC3339Nano)
	}

	return out, nil
}
