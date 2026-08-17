// backend/internal/adapters/out/firestore/token_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
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

var (
	ErrTokenOwnerUpdaterNotConfigured = errors.New("token_owner_updater_fs: not configured")
	ErrTokenOwnerUpdaterInvalidID     = errors.New("token_owner_updater_fs: productId is empty")
)

// Firestore tokens collection DTO（実データのフィールド名を正として固定）
//
// NOTE:
// - assetId 逆引き（TokenQuery）と productId 直引き（TokenReader）の両方で共用します。
// - tokens には productId フィールドは保存せず、docID = productId として扱います。
type tokenDoc struct {
	AssetStandard         string    `firestore:"assetStandard"`
	Cluster               string    `firestore:"cluster"`
	AssetID               string    `firestore:"assetId"`
	TreeAddress           string    `firestore:"treeAddress"`
	LeafIndex             *int64    `firestore:"leafIndex"`
	CoreCollectionAddress string    `firestore:"coreCollectionAddress"`
	BrandID               string    `firestore:"brandId"`
	MetadataURI           string    `firestore:"metadataUri"`
	MintedAt              time.Time `firestore:"mintedAt"`
	OnChainTxSignature    string    `firestore:"onChainTxSignature"`
	ToAddress             string    `firestore:"toAddress"`
	TokenBlueprintID      string    `firestore:"tokenBlueprintId"`
}

// ========================================
// TokenReaderFS
// ========================================

type TokenReaderFS struct {
	Client *firestore.Client
}

func NewTokenReaderFS(client *firestore.Client) *TokenReaderFS {
	return &TokenReaderFS{Client: client}
}

// ============================================================
// preview_query.go の mall.TokenReader を満たす
// ============================================================

func (r *TokenReaderFS) GetByProductID(ctx context.Context, productID string) (*dto.TokenInfo, error) {
	if r == nil || r.Client == nil {
		return nil, mallquery.ErrPreviewQueryNotConfigured
	}
	if productID == "" {
		return nil, mallquery.ErrInvalidProductID
	}

	snap, err := r.Client.Collection("tokens").Doc(productID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	d, err := decodeTokenDoc(snap)
	if err != nil {
		return nil, err
	}

	return &dto.TokenInfo{
		ProductID:          snap.Ref.ID,
		BrandID:            d.BrandID,
		TokenBlueprintID:   d.TokenBlueprintID,
		ToAddress:          d.ToAddress,
		MetadataURI:        d.MetadataURI,
		AssetID:            d.AssetID,
		OnChainTxSignature: d.OnChainTxSignature,
		MintedAt:           d.MintedAt.Format(time.RFC3339Nano),
	}, nil
}

// ============================================================
// TokenQuery (assetId -> productId(docId) + brandId + metadataUri)
// ============================================================
//
// 重要:
// - 見つからない asset は tokendom.ErrNotFound を返す（handler が 404 に変換できる）
// - これにより、token テーブル削除済みデータで 500 にならず UI 側が「取得できませんでした」表示に落とせます。
func (r *TokenReaderFS) ResolveTokenByAssetID(ctx context.Context, assetID string) (tokendom.ResolveTokenByAssetIDResult, error) {
	if r == nil || r.Client == nil {
		return tokendom.ResolveTokenByAssetIDResult{}, errors.New("token_reader_fs: firestore client is nil")
	}

	a := strings.Trim(assetID, " \t\r\n")
	if a == "" {
		return tokendom.ResolveTokenByAssetIDResult{}, tokendom.ErrInvalidAssetID
	}

	iter := r.Client.Collection("tokens").Where("assetId", "==", a).Limit(2).Documents(ctx)
	defer iter.Stop()

	first, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return tokendom.ResolveTokenByAssetIDResult{}, tokendom.ErrNotFound
		}
		return tokendom.ResolveTokenByAssetIDResult{}, err
	}

	d, err := decodeTokenDoc(first)
	if err != nil {
		return tokendom.ResolveTokenByAssetIDResult{}, err
	}
	if d.AssetID != a {
		return tokendom.ResolveTokenByAssetIDResult{}, fmt.Errorf("token_reader_fs: assetId mismatch for productId %q", first.Ref.ID)
	}

	second, err := iter.Next()
	if err == nil {
		return tokendom.ResolveTokenByAssetIDResult{}, fmt.Errorf("token_reader_fs: duplicate assetId %q in productIds %q and %q", a, first.Ref.ID, second.Ref.ID)
	}
	if !errors.Is(err, iterator.Done) {
		return tokendom.ResolveTokenByAssetIDResult{}, err
	}

	return tokendom.ResolveTokenByAssetIDResult{
		ProductID:   first.Ref.ID,
		BrandID:     d.BrandID,
		MetadataURI: d.MetadataURI,
		AssetID:     d.AssetID,
	}, nil
}

// ============================================================
// TokenQuery (tokenBlueprintId -> []assetId)
// ============================================================
//
// 同じ tokenBlueprintId を持つ tokens を検索し、assetId 一覧を返します。
// - Firestore の assetId をそのまま返します。
// - 空文字や重複 assetId は DB 不整合として error にします。
func (r *TokenReaderFS) ListAssetIDsByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) (tokendom.ListAssetIDsByTokenBlueprintIDResult, error) {
	if r == nil || r.Client == nil {
		return tokendom.ListAssetIDsByTokenBlueprintIDResult{}, errors.New("token_reader_fs: firestore client is nil")
	}

	tbID := strings.Trim(tokenBlueprintID, " \t\r\n")
	if tbID == "" {
		return tokendom.ListAssetIDsByTokenBlueprintIDResult{}, tokendom.ErrInvalidTokenBlueprintID
	}

	iter := r.Client.Collection("tokens").Where("tokenBlueprintId", "==", tbID).Documents(ctx)
	defer iter.Stop()

	assetIDs := make([]string, 0)
	seen := make(map[string]string)

	for {
		docSnap, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return tokendom.ListAssetIDsByTokenBlueprintIDResult{}, err
		}

		d, err := decodeTokenDoc(docSnap)
		if err != nil {
			return tokendom.ListAssetIDsByTokenBlueprintIDResult{}, err
		}
		if d.TokenBlueprintID != tbID {
			return tokendom.ListAssetIDsByTokenBlueprintIDResult{}, fmt.Errorf("token_reader_fs: tokenBlueprintId mismatch for productId %q", docSnap.Ref.ID)
		}
		if existingProductID, exists := seen[d.AssetID]; exists {
			return tokendom.ListAssetIDsByTokenBlueprintIDResult{}, fmt.Errorf("token_reader_fs: duplicate assetId %q in productIds %q and %q", d.AssetID, existingProductID, docSnap.Ref.ID)
		}

		seen[d.AssetID] = docSnap.Ref.ID
		assetIDs = append(assetIDs, d.AssetID)
	}

	return tokendom.ListAssetIDsByTokenBlueprintIDResult{TokenBlueprintID: tbID, AssetIDs: assetIDs}, nil
}

// ========================================
// TokenOwnerUpdaterFS
// ========================================

type TokenOwnerUpdaterFS struct {
	Client *firestore.Client

	// collection name (default "tokens")
	TokensCollection string
}

func NewTokenOwnerUpdaterFS(client *firestore.Client) *TokenOwnerUpdaterFS {
	return &TokenOwnerUpdaterFS{Client: client, TokensCollection: "tokens"}
}

func (r *TokenOwnerUpdaterFS) UpdateToAddressByProductID(ctx context.Context, productID string, newToAddress string, now time.Time, txSignature string) error {
	if r == nil || r.Client == nil {
		return ErrTokenOwnerUpdaterNotConfigured
	}
	if productID == "" {
		return ErrTokenOwnerUpdaterInvalidID
	}

	col := r.TokensCollection
	if col == "" {
		col = "tokens"
	}

	updates := []firestore.Update{
		{Path: "toAddress", Value: newToAddress},
		{Path: "updatedAt", Value: now.UTC()},
	}

	if txSignature != "" {
		updates = append(updates,
			firestore.Update{Path: "onChainTxSignature", Value: txSignature},
			firestore.Update{Path: "transferredAt", Value: now.UTC()},
		)
	}

	_, err := r.Client.Collection(col).Doc(productID).Update(ctx, updates)
	return err
}

// ============================================================
// TokenQuery (productId/docId -> token)
// ============================================================
//
// domain token.TokenQueryPort を満たすための productId 直引きです。
// 既存の GetByProductID は mall preview 用に *dto.TokenInfo を返すため、
// 戻り値型の衝突を避ける目的で GetTokenByProductID という別名にしています。
func (r *TokenReaderFS) GetTokenByProductID(ctx context.Context, productID string) (tokendom.GetTokenByProductIDResult, error) {
	if r == nil || r.Client == nil {
		return tokendom.GetTokenByProductIDResult{}, errors.New("token_reader_fs: firestore client is nil")
	}

	id := strings.Trim(productID, " \t\r\n")
	if id == "" {
		return tokendom.GetTokenByProductIDResult{}, tokendom.ErrInvalidProductID
	}

	snap, err := r.Client.Collection("tokens").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tokendom.GetTokenByProductIDResult{}, tokendom.ErrNotFound
		}
		return tokendom.GetTokenByProductIDResult{}, err
	}

	d, err := decodeTokenDoc(snap)
	if err != nil {
		return tokendom.GetTokenByProductIDResult{}, err
	}

	return tokendom.GetTokenByProductIDResult{
		ProductID:             snap.Ref.ID,
		BrandID:               d.BrandID,
		TokenBlueprintID:      d.TokenBlueprintID,
		MetadataURI:           d.MetadataURI,
		AssetStandard:         tokendom.AssetStandard(d.AssetStandard),
		Cluster:               d.Cluster,
		AssetID:               d.AssetID,
		TreeAddress:           d.TreeAddress,
		LeafIndex:             uint64(*d.LeafIndex),
		CoreCollectionAddress: d.CoreCollectionAddress,
	}, nil
}

func decodeTokenDoc(snap *firestore.DocumentSnapshot) (tokenDoc, error) {
	if snap == nil || snap.Ref == nil || snap.Ref.ID == "" {
		return tokenDoc{}, errors.New("token_reader_fs: invalid token document snapshot")
	}

	var d tokenDoc
	if err := snap.DataTo(&d); err != nil {
		return tokenDoc{}, fmt.Errorf("token_reader_fs: decode token document %q: %w", snap.Ref.ID, err)
	}
	if err := validateTokenDoc(d); err != nil {
		return tokenDoc{}, fmt.Errorf("token_reader_fs: invalid token document %q: %w", snap.Ref.ID, err)
	}

	return d, nil
}

func validateTokenDoc(d tokenDoc) error {
	if d.AssetStandard == "" {
		return errors.New("assetStandard is empty")
	}
	if d.Cluster == "" {
		return errors.New("cluster is empty")
	}
	if d.AssetID == "" {
		return errors.New("assetId is empty")
	}
	if d.TreeAddress == "" {
		return errors.New("treeAddress is empty")
	}
	if d.LeafIndex == nil {
		return errors.New("leafIndex is missing")
	}
	if *d.LeafIndex < 0 {
		return errors.New("leafIndex is negative")
	}
	if d.CoreCollectionAddress == "" {
		return errors.New("coreCollectionAddress is empty")
	}
	if d.BrandID == "" {
		return errors.New("brandId is empty")
	}
	if d.MetadataURI == "" {
		return errors.New("metadataUri is empty")
	}
	if d.MintedAt.IsZero() {
		return errors.New("mintedAt is zero")
	}
	if d.OnChainTxSignature == "" {
		return errors.New("onChainTxSignature is empty")
	}
	if d.ToAddress == "" {
		return errors.New("toAddress is empty")
	}
	if d.TokenBlueprintID == "" {
		return errors.New("tokenBlueprintId is empty")
	}

	return nil
}
