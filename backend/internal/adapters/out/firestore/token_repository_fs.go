// backend/internal/adapters/out/firestore/token_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	tokendom "narratives/internal/domain/token"
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

// MintRequestPortFS は mints を起点に MintRequestForUsecase を組み立てる実装です。
type MintRequestPortFS struct {
	client             *firestore.Client
	mintsCol           *firestore.CollectionRef
	tokenBlueprintsCol *firestore.CollectionRef
	brandsCol          *firestore.CollectionRef
	// ★ 1商品=1トークン用のトークン保存コレクション
	tokensCol *firestore.CollectionRef
}

// コンパイル時に MintRequestPort を満たしていることを確認
var _ usecase.MintRequestPort = (*MintRequestPortFS)(nil)

// NewMintRequestPortFS は mints ベースの MintRequestPort 実装を生成します。
//
// mintsColName / tokenBlueprintsColName / brandsColName は、
// それぞれ Firestore の実際のコレクション名に合わせて渡してください。
// 例:
//
//	NewMintRequestPortFS(client, "mints", "token_blueprints", "brands")
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
		// ★ トークン保存先はひとまず固定で "tokens" コレクションを想定
		tokensCol: client.Collection("tokens"),
	}
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

	mintID := strings.TrimSpace(id)
	if mintID == "" {
		return nil, fmt.Errorf("mint id is empty")
	}

	// 1) mints/{mintID} を取得
	mintSnap, err := p.mintsCol.Doc(mintID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("mint %s not found", mintID)
		}
		return nil, fmt.Errorf("get mint %s: %w", mintID, err)
	}

	// 基本フィールドは struct に decode
	var m mintDoc
	if err := mintSnap.DataTo(&m); err != nil {
		return nil, fmt.Errorf("decode mint %s: %w", mintID, err)
	}

	// すでに minted 済みならエラーにする（ or スキップ）
	if m.Minted {
		return nil, fmt.Errorf("mint %s is already minted", mintID)
	}

	brandID := strings.TrimSpace(m.BrandID)
	if brandID == "" {
		return nil, fmt.Errorf("mint %s has empty brandId", mintID)
	}
	tbID := strings.TrimSpace(m.TokenBlueprintID)
	if tbID == "" {
		return nil, fmt.Errorf("mint %s has empty tokenBlueprintId", mintID)
	}

	// 1-2) products は旧・新スキーマ両対応で取得する
	raw := mintSnap.Data()
	productIDs := make([]string, 0)

	if v, ok := raw["products"]; ok {
		switch vv := v.(type) {
		case []interface{}:
			// 旧: products: [ "productId1", "productId2", ... ]
			for _, x := range vv {
				if s, ok := x.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						productIDs = append(productIDs, s)
					}
				}
			}
		case []string:
			for _, s := range vv {
				s = strings.TrimSpace(s)
				if s != "" {
					productIDs = append(productIDs, s)
				}
			}
		case map[string]interface{}:
			// 新: products: { "productId1": "mintAddress1", ... }
			for k := range vv {
				k = strings.TrimSpace(k)
				if k != "" {
					productIDs = append(productIDs, k)
				}
			}
		case map[string]string:
			for k := range vv {
				k = strings.TrimSpace(k)
				if k != "" {
					productIDs = append(productIDs, k)
				}
			}
		default:
			// 型が想定外の場合は何もしない（ProductIDs 空のまま）
		}
	}

	// ★ 追加: すでに tokens に存在する productId がないか検査
	if len(productIDs) > 0 && p.tokensCol != nil {
		already := make([]string, 0, len(productIDs))

		for _, pid := range productIDs {
			pid = strings.TrimSpace(pid)
			if pid == "" {
				continue
			}

			// tokens/{productId} をチェック
			_, err := p.tokensCol.Doc(pid).Get(ctx)
			if err == nil {
				// ドキュメントが存在 = すでにミント済み
				already = append(already, pid)
				continue
			}
			if status.Code(err) == codes.NotFound {
				// まだミントされていないので OK
				continue
			}

			// それ以外のエラーはそのまま返す
			return nil, fmt.Errorf("check token for product %s: %w", pid, err)
		}

		if len(already) > 0 {
			return nil, fmt.Errorf("tokens already exist for products: %v", already)
		}
	}

	// 2) token_blueprints/{tokenBlueprintId} を取得
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

	name := strings.TrimSpace(tb.Name)
	symbol := strings.TrimSpace(tb.Symbol)
	metadataURI := strings.TrimSpace(tb.MetadataURI)

	if name == "" || symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint %s has empty name or symbol", tbID)
	}
	if metadataURI == "" {
		return nil, fmt.Errorf("tokenBlueprint %s has empty metadataUri", tbID)
	}

	// 3) brands/{brandId} から ToAddress（ブランドウォレット）を取得
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

	toAddress := strings.TrimSpace(b.WalletAddress)
	if toAddress == "" {
		return nil, fmt.Errorf("brand %s has empty walletAddress", brandID)
	}

	// ★ ProductIDs を DTO に渡す（1商品=1Mint 用）
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

	mintID := strings.TrimSpace(id)
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}

	updates := []firestore.Update{
		{
			Path:  "minted",
			Value: true,
		},
		{
			Path:  "mintedAt",
			Value: firestore.ServerTimestamp,
		},
		{
			Path:  "onChainTxSignature",
			Value: result.Signature,
		},
		{
			Path:  "mintAddress",
			Value: result.MintAddress,
		},
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
// - tokens コレクションに [productId, mintAddress] を 1:1 で保存
// - mints/{id} 自体も minted=true に更新（代表の MintResult を利用）
func (p *MintRequestPortFS) MarkProductsAsMinted(
	ctx context.Context,
	id string,
	minted []usecase.MintedTokenForUsecase,
) error {
	if p == nil || p.client == nil || p.mintsCol == nil || p.tokensCol == nil {
		return fmt.Errorf("MintRequestPortFS is not initialized (tokensCol may be nil)")
	}
	mintID := strings.TrimSpace(id)
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}
	if len(minted) == 0 {
		return fmt.Errorf("no minted results provided")
	}

	// 対応する mints/{id} を再取得して、brandId / tokenBlueprintId 等を参照
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

	// tokens コレクションに 1商品=1トークンのレコードを作成
	// ここでは「productId を docID」にして 1:1 を保証する方針。
	for _, mt := range minted {
		productID := strings.TrimSpace(mt.ProductID)
		if productID == "" {
			continue
		}
		if mt.Result == nil {
			continue
		}

		docID := productID

		data := map[string]interface{}{
			"brandId":            m.BrandID,
			"tokenBlueprintId":   m.TokenBlueprintID,
			"productId":          productID,
			"mintAddress":        mt.Result.MintAddress,
			"onChainTxSignature": mt.Result.Signature,
			"mintedAt":           firestore.ServerTimestamp,
			// 必要に応じて scheduledBurnDate 等もここへコピーしてよい
		}

		// 既に存在していた場合は上書き (1 productId = 1 token の前提)
		if _, err := p.tokensCol.Doc(docID).Set(ctx, data); err != nil {
			return fmt.Errorf("failed to upsert token doc for product %s: %w", productID, err)
		}
	}

	// mints/{id} 自体も minted=true に更新（代表として最後の MintResult を利用）
	var lastResult *tokendom.MintResult
	for _, mt := range minted {
		if mt.Result != nil {
			lastResult = mt.Result
		}
	}
	if lastResult == nil {
		return fmt.Errorf("no valid MintResult found in minted list")
	}

	if err := p.MarkAsMinted(ctx, mintID, lastResult); err != nil {
		return fmt.Errorf("failed to mark mint %s as minted after per-product tokens: %w", mintID, err)
	}

	return nil
}
