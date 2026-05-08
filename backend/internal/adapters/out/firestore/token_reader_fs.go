// backend/internal/adapters/out/firestore/token_reader_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mallquery "narratives/internal/application/query/mall"
	dto "narratives/internal/application/query/mall/dto"
	tokendom "narratives/internal/domain/token"
)

type TokenReaderFS struct {
	Client *firestore.Client
}

func NewTokenReaderFS(client *firestore.Client) *TokenReaderFS {
	return &TokenReaderFS{Client: client}
}

// Firestore tokens collection DTO（実データのフィールド名を正として固定）
//
// NOTE:
// - 同一 package firestore 内で tokenDoc を重複定義するとコンパイルエラーになります。
// - mintAddress 逆引き（TokenQuery）と productId 直引き（TokenReader）の両方で共用します。
type tokenDoc struct {
	BrandID            string    `firestore:"brandId"`
	MetadataURI        string    `firestore:"metadataUri"`
	MintAddress        string    `firestore:"mintAddress"`
	MintedAt           time.Time `firestore:"mintedAt"` // timestamp
	OnChainTxSignature string    `firestore:"onChainTxSignature"`
	ToAddress          string    `firestore:"toAddress"`
	TokenBlueprintID   string    `firestore:"tokenBlueprintId"`
}

// 必要に応じて compile-time interface check を入れる場合は下記を有効化してください。
// var _ tokendom.TokenQueryPort = (*TokenReaderFS)(nil)

// ============================================================
// preview_query.go の mall.TokenReader を満たす
// ============================================================

func (r *TokenReaderFS) GetByProductID(ctx context.Context, productID string) (*dto.TokenInfo, error) {
	if r == nil || r.Client == nil {
		return nil, mallquery.ErrPreviewQueryNotConfigured
	}
	id := productID
	if id == "" {
		return nil, mallquery.ErrInvalidProductID
	}

	snap, err := r.Client.Collection("tokens").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil // token が無いのは正常（未mintなど）
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

	out := &dto.TokenInfo{
		ProductID:          id,
		BrandID:            d.BrandID,
		TokenBlueprintID:   d.TokenBlueprintID,
		MintAddress:        d.MintAddress,
		ToAddress:          d.ToAddress,
		MetadataURI:        d.MetadataURI,
		OnChainTxSignature: d.OnChainTxSignature,
	}

	// mintedAt は Firestore timestamp（time.Time）として受け、文字列に落とす（DTOがstring前提のため）
	if !d.MintedAt.IsZero() {
		out.MintedAt = d.MintedAt.UTC().Format(time.RFC3339Nano)
	}

	return out, nil
}

// ============================================================
// TokenQuery (mintAddress -> productId(docId) + brandId + metadataUri)
// ============================================================
//
// 重要:
// - 見つからない mint は tokendom.ErrNotFound を返す（handler が 404 に変換できる）
// - これにより、token テーブル削除済みデータで 500 にならず UI 側が「取得できませんでした」表示に落とせます。
func (r *TokenReaderFS) ResolveTokenByMintAddress(
	ctx context.Context,
	mintAddress string,
) (tokendom.ResolveTokenByMintAddressResult, error) {
	if r == nil || r.Client == nil {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("token_reader_fs: firestore client is nil")
	}

	m := strings.TrimSpace(mintAddress)
	if m == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, tokendom.ErrInvalidMintAddress
	}

	iter := r.Client.Collection("tokens").Where("mintAddress", "==", m).Limit(1).Documents(ctx)
	defer iter.Stop()

	docSnap, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			// handler 側で 404 に落とせるよう ErrNotFound を返す
			return tokendom.ResolveTokenByMintAddressResult{}, tokendom.ErrNotFound
		}
		return tokendom.ResolveTokenByMintAddressResult{}, err
	}

	var d tokenDoc
	if err := docSnap.DataTo(&d); err != nil {
		return tokendom.ResolveTokenByMintAddressResult{}, err
	}

	productID := docSnap.Ref.ID // docId = productId
	if productID == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, errors.New("token_reader_fs: empty doc id")
	}

	mint := strings.TrimSpace(d.MintAddress)
	if mint == "" {
		return tokendom.ResolveTokenByMintAddressResult{}, tokendom.ErrInvalidMintAddress
	}

	return tokendom.ResolveTokenByMintAddressResult{
		ProductID:   productID,
		BrandID:     d.BrandID,
		MetadataURI: d.MetadataURI,
		MintAddress: mint,
	}, nil
}

// ============================================================
// TokenQuery (tokenBlueprintId -> []mintAddress)
// ============================================================
//
// 同じ tokenBlueprintId を持つ tokens を検索し、mintAddress 一覧を返します。
// - 空文字の mintAddress は除外
// - 重複 mintAddress は除外
func (r *TokenReaderFS) ListMintAddressesByTokenBlueprintID(
	ctx context.Context,
	tokenBlueprintID string,
) (tokendom.ListMintAddressesByTokenBlueprintIDResult, error) {
	if r == nil || r.Client == nil {
		return tokendom.ListMintAddressesByTokenBlueprintIDResult{}, errors.New("token_reader_fs: firestore client is nil")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return tokendom.ListMintAddressesByTokenBlueprintIDResult{}, tokendom.ErrInvalidTokenBlueprintID
	}

	iter := r.Client.Collection("tokens").Where("tokenBlueprintId", "==", tbID).Documents(ctx)
	defer iter.Stop()

	mintAddresses := make([]string, 0)
	seen := make(map[string]struct{})

	for {
		docSnap, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return tokendom.ListMintAddressesByTokenBlueprintIDResult{}, err
		}

		var d tokenDoc
		if err := docSnap.DataTo(&d); err != nil {
			return tokendom.ListMintAddressesByTokenBlueprintIDResult{}, err
		}

		addr := strings.TrimSpace(d.MintAddress)
		if addr == "" {
			continue
		}
		if _, exists := seen[addr]; exists {
			continue
		}

		seen[addr] = struct{}{}
		mintAddresses = append(mintAddresses, addr)
	}

	return tokendom.ListMintAddressesByTokenBlueprintIDResult{
		TokenBlueprintID: tbID,
		MintAddresses:    mintAddresses,
	}, nil
}
