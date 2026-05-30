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
	usecase "narratives/internal/application/usecase"
	tokendom "narratives/internal/domain/token"
)

// ========================================
// errors
// ========================================

var (
	ErrTokenOwnerUpdaterNotConfigured = errors.New("token_owner_updater_fs: not configured")
	ErrTokenOwnerUpdaterInvalidID     = errors.New("token_owner_updater_fs: productId is empty")
)

// ========================================
// mints ベースの MintRequestPort 実装
// ========================================

// mints/{id} のビュー
// ★ products は decode 時は扱わない（Data() から柔軟に取得するため）
type mintDoc struct {
	BrandID          string `firestore:"brandId"`
	TokenBlueprintID string `firestore:"tokenBlueprintId"`
	Minted           bool   `firestore:"minted"`
}

// token_blueprints/{id} のビュー
type tokenBlueprintDoc struct {
	Name        string `firestore:"name"`        // 実際のフィールド名に合わせる
	Symbol      string `firestore:"symbol"`      // 実際のフィールド名に合わせる
	MetadataURI string `firestore:"metadataUri"` // 実際のフィールド名に合わせる
}

// brands/{id} のビュー（ブランドウォレット）
type brandDoc struct {
	WalletAddress string `firestore:"walletAddress"` // 実際のフィールド名に合わせる
}

// Firestore tokens collection DTO（実データのフィールド名を正として固定）
//
// NOTE:
// - mintAddress 逆引き（TokenQuery）と productId 直引き（TokenReader）の両方で共用します。
// - tokens には productId フィールドは保存せず、docID = productId として扱います。
type tokenDoc struct {
	BrandID            string    `firestore:"brandId"`
	MetadataURI        string    `firestore:"metadataUri"`
	MintAddress        string    `firestore:"mintAddress"`
	MintedAt           time.Time `firestore:"mintedAt"`
	OnChainTxSignature string    `firestore:"onChainTxSignature"`
	ToAddress          string    `firestore:"toAddress"`
	TokenBlueprintID   string    `firestore:"tokenBlueprintId"`
}

// MintRequestPortFS は mints を起点に MintRequestForUsecase を組み立てる実装です。
type MintRequestPortFS struct {
	client             *firestore.Client
	mintsCol           *firestore.CollectionRef
	tokenBlueprintsCol *firestore.CollectionRef
	brandsCol          *firestore.CollectionRef

	// 1商品=1トークン用のトークン保存コレクション
	tokensCol *firestore.CollectionRef
}

// コンパイル時に MintRequestPort を満たしていることを確認
var _ usecase.MintRequestPort = (*MintRequestPortFS)(nil)

// NewMintRequestPortFS は mints ベースの MintRequestPort 実装を生成します。
func NewMintRequestPortFS(
	client *firestore.Client,
	mintsColName string,
	tokenBlueprintsColName string,
	brandsColName string,
) *MintRequestPortFS {
	return &MintRequestPortFS{
		client:             client,
		mintsCol:           client.Collection(mintsColName),
		tokenBlueprintsCol: client.Collection(tokenBlueprintsColName),
		brandsCol:          client.Collection(brandsColName),
		tokensCol:          client.Collection("tokens"),
	}
}

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

func nonEmptyStringAny(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func hasNonZeroTimestampAny(v any) bool {
	if v == nil {
		return false
	}

	switch t := v.(type) {
	case time.Time:
		return !t.IsZero()
	case *time.Time:
		return t != nil && !t.IsZero()
	default:
		return false
	}
}

// minted=false なのに mintedAt / 署名があるなど、
// 「既にミント済みの痕跡」があるかを判定する。
func hasMintedEvidence(raw map[string]any) bool {
	if raw == nil {
		return false
	}

	if v, ok := raw["mintedAt"]; ok && hasNonZeroTimestampAny(v) {
		return true
	}

	for _, k := range []string{"onChainTxSignature", "onchainTxSignature", "txSignature", "signature"} {
		if s := nonEmptyStringAny(raw[k]); s != "" {
			return true
		}
	}

	return false
}

// LoadForMinting は mintID を受け取り、
// mints + token_blueprints + brands から MintRequestForUsecase を構築して返します。
func (p *MintRequestPortFS) LoadForMinting(
	ctx context.Context,
	id string,
) (*usecase.MintRequestForUsecase, error) {
	if p == nil || p.client == nil || p.mintsCol == nil {
		return nil, fmt.Errorf("MintRequestPortFS is not initialized")
	}

	mintID := id
	if mintID == "" {
		return nil, fmt.Errorf("mint id is empty")
	}

	mintSnap, err := p.mintsCol.Doc(mintID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("mint %s not found", mintID)
		}
		return nil, fmt.Errorf("get mint %s: %w", mintID, err)
	}

	var m mintDoc
	if err := mintSnap.DataTo(&m); err != nil {
		return nil, fmt.Errorf("decode mint %s: %w", mintID, err)
	}

	raw := mintSnap.Data()

	if !m.Minted && hasMintedEvidence(raw) {
		_, _ = p.mintsCol.Doc(mintID).Update(ctx, []firestore.Update{
			{Path: "minted", Value: true},
		})
		return nil, fmt.Errorf("mint %s is already minted", mintID)
	}

	if m.Minted {
		return nil, fmt.Errorf("mint %s is already minted", mintID)
	}

	brandID := m.BrandID
	if brandID == "" {
		return nil, fmt.Errorf("mint %s has empty brandId", mintID)
	}

	tbID := m.TokenBlueprintID
	if tbID == "" {
		return nil, fmt.Errorf("mint %s has empty tokenBlueprintId", mintID)
	}

	productIDs := make([]string, 0)

	if v, ok := raw["products"]; ok {
		switch vv := v.(type) {
		case []interface{}:
			for _, x := range vv {
				if s, ok := x.(string); ok && s != "" {
					productIDs = append(productIDs, s)
				}
			}
		case []string:
			for _, s := range vv {
				if s != "" {
					productIDs = append(productIDs, s)
				}
			}
		}
	}

	if len(productIDs) > 0 && p.tokensCol != nil {
		already := make([]string, 0, len(productIDs))

		for _, pid := range productIDs {
			if pid == "" {
				continue
			}

			_, err := p.tokensCol.Doc(pid).Get(ctx)
			if err == nil {
				already = append(already, pid)
				continue
			}
			if status.Code(err) == codes.NotFound {
				continue
			}

			return nil, fmt.Errorf("check token for product %s: %w", pid, err)
		}

		if len(already) > 0 {
			return nil, fmt.Errorf("tokens already exist for products: %v", already)
		}
	}

	tbSnap, err := p.tokenBlueprintsCol.Doc(tbID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("tokenBlueprint %s not found for mint %s", tbID, mintID)
		}
		return nil, fmt.Errorf("get tokenBlueprint %s: %w", tbID, err)
	}

	var tb tokenBlueprintDoc
	if err := tbSnap.DataTo(&tb); err != nil {
		return nil, fmt.Errorf("decode tokenBlueprint %s: %w", tbID, err)
	}

	name := tb.Name
	symbol := tb.Symbol
	metadataURI := tb.MetadataURI

	if name == "" || symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint %s has empty name or symbol", tbID)
	}
	if metadataURI == "" {
		return nil, fmt.Errorf("tokenBlueprint %s has empty metadataUri", tbID)
	}

	brandSnap, err := p.brandsCol.Doc(brandID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("brand %s not found for mint %s", brandID, mintID)
		}
		return nil, fmt.Errorf("get brand %s: %w", brandID, err)
	}

	var b brandDoc
	if err := brandSnap.DataTo(&b); err != nil {
		return nil, fmt.Errorf("decode brand %s: %w", brandID, err)
	}

	toAddress := b.WalletAddress
	if toAddress == "" {
		return nil, fmt.Errorf("brand %s has empty walletAddress", brandID)
	}

	dto := &usecase.MintRequestForUsecase{
		ID:              mintID,
		ToAddress:       toAddress,
		ProductIDs:      productIDs,
		BlueprintName:   name,
		BlueprintSymbol: symbol,
		MetadataURI:     metadataURI,
	}

	return dto, nil
}

// MarkAsMinted はチェーンミント結果をもとに mints/{mintID} を更新します。
// ★ 注意: mints には mintAddress を保存しない方針のため、mintAddress 更新は行いません。
func (p *MintRequestPortFS) MarkAsMinted(
	ctx context.Context,
	id string,
	result *tokendom.MintResult,
) error {
	if p == nil || p.client == nil || p.mintsCol == nil {
		return fmt.Errorf("MintRequestPortFS is not initialized")
	}
	if result == nil {
		return fmt.Errorf("mint result is nil")
	}

	mintID := id
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}

	updates := []firestore.Update{
		{Path: "minted", Value: true},
		{Path: "mintedAt", Value: firestore.ServerTimestamp},
		{Path: "onChainTxSignature", Value: result.Signature},
	}

	_, err := p.mintsCol.Doc(mintID).Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("mint %s not found when updating as minted", mintID)
		}
		return fmt.Errorf("update mint %s as minted: %w", mintID, err)
	}

	return nil
}

// MarkProductsAsMinted は「1商品=1Mint」でミントした結果を Firestore に反映します。
// - tokens コレクションに [productId, mintAddress] を 1:1 で保存（docID=productId）
// - tokens には productId フィールドは保存しない（docID が productId なので不要）
// - tokens には tokenBlueprintId を保存する（商品型特定に必要）
// - tokens に toAddress / metadataUri をキャッシュとして保存する（体感高速化）
// - mints/{id} 自体も minted=true に更新（代表の MintResult を利用。ただし mintAddress は保存しない）
func (p *MintRequestPortFS) MarkProductsAsMinted(
	ctx context.Context,
	id string,
	minted []usecase.MintedTokenForUsecase,
) error {
	if p == nil || p.client == nil || p.mintsCol == nil || p.tokensCol == nil {
		return fmt.Errorf("MintRequestPortFS is not initialized (tokensCol may be nil)")
	}

	mintID := id
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}
	if len(minted) == 0 {
		return fmt.Errorf("no minted results provided")
	}

	mintSnap, err := p.mintsCol.Doc(mintID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("mint %s not found when MarkProductsAsMinted", mintID)
		}
		return fmt.Errorf("get mint %s in MarkProductsAsMinted: %w", mintID, err)
	}

	var m mintDoc
	if err := mintSnap.DataTo(&m); err != nil {
		return fmt.Errorf("decode mint %s in MarkProductsAsMinted: %w", mintID, err)
	}

	brandID := m.BrandID
	if brandID == "" {
		return fmt.Errorf("mint %s has empty brandId in MarkProductsAsMinted", mintID)
	}

	tbID := m.TokenBlueprintID
	if tbID == "" {
		return fmt.Errorf("mint %s has empty tokenBlueprintId in MarkProductsAsMinted", mintID)
	}

	if p.brandsCol == nil {
		return fmt.Errorf("brandsCol is nil in MintRequestPortFS")
	}

	brandSnap, err := p.brandsCol.Doc(brandID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("brand %s not found for mint %s", brandID, mintID)
		}
		return fmt.Errorf("get brand %s in MarkProductsAsMinted: %w", brandID, err)
	}

	var b brandDoc
	if err := brandSnap.DataTo(&b); err != nil {
		return fmt.Errorf("decode brand %s in MarkProductsAsMinted: %w", brandID, err)
	}

	toAddress := b.WalletAddress
	if toAddress == "" {
		return fmt.Errorf("brand %s has empty walletAddress (toAddress) in MarkProductsAsMinted", brandID)
	}

	if p.tokenBlueprintsCol == nil {
		return fmt.Errorf("tokenBlueprintsCol is nil in MintRequestPortFS")
	}

	tbSnap, err := p.tokenBlueprintsCol.Doc(tbID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("tokenBlueprint %s not found for mint %s", tbID, mintID)
		}
		return fmt.Errorf("get tokenBlueprint %s in MarkProductsAsMinted: %w", tbID, err)
	}

	var tb tokenBlueprintDoc
	if err := tbSnap.DataTo(&tb); err != nil {
		return fmt.Errorf("decode tokenBlueprint %s in MarkProductsAsMinted: %w", tbID, err)
	}

	metadataURI := tb.MetadataURI
	if metadataURI == "" {
		return fmt.Errorf("tokenBlueprint %s has empty metadataUri in MarkProductsAsMinted", tbID)
	}

	var lastResult *tokendom.MintResult
	for _, mt := range minted {
		if mt.Result != nil {
			lastResult = mt.Result
		}
	}

	if lastResult == nil {
		return fmt.Errorf("no valid MintResult found in minted list")
	}

	batch := p.client.Batch()

	for _, mt := range minted {
		productID := mt.ProductID
		if productID == "" || mt.Result == nil {
			continue
		}

		data := map[string]interface{}{
			"brandId":            m.BrandID,
			"tokenBlueprintId":   m.TokenBlueprintID,
			"mintAddress":        mt.Result.MintAddress,
			"onChainTxSignature": mt.Result.Signature,
			"mintedAt":           firestore.ServerTimestamp,
			"toAddress":          toAddress,
			"metadataUri":        metadataURI,
		}

		batch.Set(p.tokensCol.Doc(productID), data, firestore.MergeAll)
	}

	batch.Update(p.mintsCol.Doc(mintID), []firestore.Update{
		{Path: "minted", Value: true},
		{Path: "mintedAt", Value: firestore.ServerTimestamp},
		{Path: "onChainTxSignature", Value: lastResult.Signature},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("batch commit failed in MarkProductsAsMinted mintID=%s: %w", mintID, err)
	}

	return nil
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
		MintAddress:        d.MintAddress,
		ToAddress:          d.ToAddress,
		MetadataURI:        d.MetadataURI,
		OnChainTxSignature: d.OnChainTxSignature,
	}

	if !d.MintedAt.IsZero() {
		out.MintedAt = d.MintedAt.UTC().Format(time.RFC3339Nano)
	}

	return out, nil
}

// ============================================================
// TokenQuery (mintAddress -> productId(docId) + brandId + metadataUri)
// ============================================================

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
			return tokendom.ResolveTokenByMintAddressResult{}, tokendom.ErrNotFound
		}
		return tokendom.ResolveTokenByMintAddressResult{}, err
	}

	var d tokenDoc
	if err := docSnap.DataTo(&d); err != nil {
		return tokendom.ResolveTokenByMintAddressResult{}, err
	}

	productID := docSnap.Ref.ID
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
