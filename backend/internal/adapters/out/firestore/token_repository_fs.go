// backend/internal/adapters/out/firestore/token_repository_fs.go
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
	LeafIndex             int64     `firestore:"leafIndex"`
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

	id := productID
	if id == "" {
		return nil, mallquery.ErrInvalidProductID
	}

	snap, err := r.Client.Collection("tokens").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

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
		ToAddress:          d.ToAddress,
		MetadataURI:        d.MetadataURI,
		AssetID:            d.AssetID,
		OnChainTxSignature: d.OnChainTxSignature,
	}

	if !d.MintedAt.IsZero() {
		out.MintedAt = d.MintedAt.UTC().Format(time.RFC3339Nano)
	}

	return out, nil
}

// ============================================================
// TokenQuery (assetId -> productId(docId) + brandId + metadataUri)
// ============================================================
//
// 重要:
// - 見つからない asset は tokendom.ErrNotFound を返す（handler が 404 に変換できる）
// - これにより、token テーブル削除済みデータで 500 にならず UI 側が「取得できませんでした」表示に落とせます。
func (r *TokenReaderFS) ResolveTokenByAssetID(
	ctx context.Context,
	assetID string,
) (tokendom.ResolveTokenByAssetIDResult, error) {
	if r == nil || r.Client == nil {
		return tokendom.ResolveTokenByAssetIDResult{}, errors.New("token_reader_fs: firestore client is nil")
	}

	a := strings.Trim(assetID, " \t\r\n")
	if a == "" {
		return tokendom.ResolveTokenByAssetIDResult{}, tokendom.ErrInvalidAssetID
	}

	iter := r.Client.Collection("tokens").Where("assetId", "==", a).Limit(1).Documents(ctx)
	defer iter.Stop()

	docSnap, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return tokendom.ResolveTokenByAssetIDResult{}, tokendom.ErrNotFound
		}
		return tokendom.ResolveTokenByAssetIDResult{}, err
	}

	var d tokenDoc
	if err := docSnap.DataTo(&d); err != nil {
		return tokendom.ResolveTokenByAssetIDResult{}, err
	}

	productID := docSnap.Ref.ID
	if productID == "" {
		return tokendom.ResolveTokenByAssetIDResult{}, errors.New("token_reader_fs: empty doc id")
	}

	asset := strings.Trim(d.AssetID, " \t\r\n")
	if asset == "" {
		return tokendom.ResolveTokenByAssetIDResult{}, tokendom.ErrInvalidAssetID
	}

	return tokendom.ResolveTokenByAssetIDResult{
		ProductID:   productID,
		BrandID:     d.BrandID,
		MetadataURI: d.MetadataURI,
		AssetID:     asset,
	}, nil
}

// ============================================================
// TokenQuery (tokenBlueprintId -> []assetId)
// ============================================================
//
// 同じ tokenBlueprintId を持つ tokens を検索し、assetId 一覧を返します。
// - 空文字の assetId は除外
// - 重複 assetId は除外
func (r *TokenReaderFS) ListAssetIDsByTokenBlueprintID(
	ctx context.Context,
	tokenBlueprintID string,
) (tokendom.ListAssetIDsByTokenBlueprintIDResult, error) {
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
	seen := make(map[string]struct{})

	for {
		docSnap, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return tokendom.ListAssetIDsByTokenBlueprintIDResult{}, err
		}

		var d tokenDoc
		if err := docSnap.DataTo(&d); err != nil {
			return tokendom.ListAssetIDsByTokenBlueprintIDResult{}, err
		}

		assetID := strings.Trim(d.AssetID, " \t\r\n")
		if assetID == "" {
			continue
		}

		if _, exists := seen[assetID]; exists {
			continue
		}

		seen[assetID] = struct{}{}
		assetIDs = append(assetIDs, assetID)
	}

	return tokendom.ListAssetIDsByTokenBlueprintIDResult{
		TokenBlueprintID: tbID,
		AssetIDs:         assetIDs,
	}, nil
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
	return &TokenOwnerUpdaterFS{
		Client:           client,
		TokensCollection: "tokens",
	}
}

func (r *TokenOwnerUpdaterFS) UpdateToAddressByProductID(
	ctx context.Context,
	productID string,
	newToAddress string,
	now time.Time,
	txSignature string,
) error {
	if r == nil || r.Client == nil {
		return ErrTokenOwnerUpdaterNotConfigured
	}

	pid := productID
	if pid == "" {
		return ErrTokenOwnerUpdaterInvalidID
	}

	col := r.TokensCollection
	if col == "" {
		col = "tokens"
	}

	updates := map[string]any{
		"toAddress": newToAddress,
		"updatedAt": now.UTC(),
	}

	if txSignature != "" {
		updates["onChainTxSignature"] = txSignature
		updates["transferredAt"] = now.UTC()
	}

	_, err := r.Client.Collection(col).Doc(pid).Set(ctx, updates, firestore.MergeAll)
	return err
}

// ============================================================
// TokenQuery (productId/docId -> token)
// ============================================================
//
// domain token.TokenQueryPort を満たすための productId 直引きです。
// 既存の GetByProductID は mall preview 用に *dto.TokenInfo を返すため、
// 戻り値型の衝突を避ける目的で GetTokenByProductID という別名にしています。
func (r *TokenReaderFS) GetTokenByProductID(
	ctx context.Context,
	productID string,
) (tokendom.GetTokenByProductIDResult, error) {
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

	if snap.Data() == nil {
		return tokendom.GetTokenByProductIDResult{}, tokendom.ErrNotFound
	}

	var d tokenDoc
	if err := snap.DataTo(&d); err != nil {
		return tokendom.GetTokenByProductIDResult{}, err
	}

	leafIndex := uint64(0)
	if d.LeafIndex >= 0 {
		leafIndex = uint64(d.LeafIndex)
	}

	return tokendom.GetTokenByProductIDResult{
		ProductID:             id,
		BrandID:               d.BrandID,
		TokenBlueprintID:      d.TokenBlueprintID,
		MetadataURI:           d.MetadataURI,
		AssetStandard:         tokendom.AssetStandard(d.AssetStandard),
		Cluster:               d.Cluster,
		AssetID:               strings.Trim(d.AssetID, " \t\r\n"),
		TreeAddress:           strings.Trim(d.TreeAddress, " \t\r\n"),
		LeafIndex:             leafIndex,
		CoreCollectionAddress: strings.Trim(d.CoreCollectionAddress, " \t\r\n"),
	}, nil
}
