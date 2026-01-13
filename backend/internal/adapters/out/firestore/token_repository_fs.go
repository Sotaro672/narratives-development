// backend/internal/adapters/out/firestore/token_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

func nonEmptyStringAny(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
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
		// Firestore の Data() は通常 time.Time を返すが、念のため fallback
		return false
	}
}

// minted=false なのに mintedAt / 署名があるなど、
// 「既にミント済みの痕跡」があるかを判定する。
func hasMintedEvidence(raw map[string]any) bool {
	if raw == nil {
		return false
	}
	// mintedAt がある
	if v, ok := raw["mintedAt"]; ok && hasNonZeroTimestampAny(v) {
		return true
	}
	// tx signature がある（どれか1つ）
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

	raw := mintSnap.Data()

	// ★ 重要: minted=false でも mintedAt/署名が入っている「不整合状態」を検知したら
	// その場で minted=true に修復して、以降は "already minted" 扱いにする。
	if !m.Minted && hasMintedEvidence(raw) {
		// best-effort repair（失敗しても read は止めない）
		_, _ = p.mintsCol.Doc(mintID).Update(ctx, []firestore.Update{
			{Path: "minted", Value: true},
		})
		return nil, fmt.Errorf("mint %s is already minted", mintID)
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

	// 1-2) products は「配列（productId一覧）」のみ対応する
	productIDs := make([]string, 0)

	if v, ok := raw["products"]; ok {
		switch vv := v.(type) {
		case []interface{}:
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
		default:
			// 型が想定外の場合は何もしない（ProductIDs 空のまま）
		}
	}

	// ★ tokens 既存チェック（既にミント済みの product があれば止める）
	if len(productIDs) > 0 && p.tokensCol != nil {
		already := make([]string, 0, len(productIDs))

		for _, pid := range productIDs {
			pid = strings.TrimSpace(pid)
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

	mintID := strings.TrimSpace(id)
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}

	updates := []firestore.Update{
		{Path: "minted", Value: true},
		{Path: "mintedAt", Value: firestore.ServerTimestamp},
		{Path: "onChainTxSignature", Value: strings.TrimSpace(result.Signature)},
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
// - ✅ tokens には tokenBlueprintId を保存する（商品型特定に必要）
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

	mintID := strings.TrimSpace(id)
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}
	if len(minted) == 0 {
		return fmt.Errorf("no minted results provided")
	}

	// mints/{id} を再取得して brandId / tokenBlueprintId を参照
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

	brandID := strings.TrimSpace(m.BrandID)
	if brandID == "" {
		return fmt.Errorf("mint %s has empty brandId in MarkProductsAsMinted", mintID)
	}
	tbID := strings.TrimSpace(m.TokenBlueprintID)
	if tbID == "" {
		return fmt.Errorf("mint %s has empty tokenBlueprintId in MarkProductsAsMinted", mintID)
	}

	// ✅ cache: toAddress を brands/{brandId} から取得
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
	toAddress := strings.TrimSpace(b.WalletAddress)
	if toAddress == "" {
		return fmt.Errorf("brand %s has empty walletAddress (toAddress) in MarkProductsAsMinted", brandID)
	}

	// ✅ cache: metadataUri を token_blueprints/{tokenBlueprintId} から取得（A案）
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
	metadataURI := strings.TrimSpace(tb.MetadataURI)
	if metadataURI == "" {
		return fmt.Errorf("tokenBlueprint %s has empty metadataUri in MarkProductsAsMinted", tbID)
	}

	// 代表として最後の MintResult を利用
	var lastResult *tokendom.MintResult
	for _, mt := range minted {
		if mt.Result != nil {
			lastResult = mt.Result
		}
	}
	if lastResult == nil {
		return fmt.Errorf("no valid MintResult found in minted list")
	}

	// ★ tokens upsert と mint の minted 更新を batch でまとめて atomic にする
	batch := p.client.Batch()

	// tokens: docID=productId
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
			// ✅ brandId は残す（ブランドでの絞り込み等）
			"brandId": strings.TrimSpace(m.BrandID),

			// ✅ NEW: tokenBlueprintId は保存する（商品型特定に必要）
			"tokenBlueprintId": strings.TrimSpace(m.TokenBlueprintID),

			// ✅ mintAddress は保持
			"mintAddress": strings.TrimSpace(mt.Result.MintAddress),

			// ✅ onChainTxSignature / mintedAt は従来どおり
			"onChainTxSignature": strings.TrimSpace(mt.Result.Signature),
			"mintedAt":           firestore.ServerTimestamp,

			// ✅ 体感高速化: Firestore 側にキャッシュ
			"toAddress":   toAddress,
			"metadataUri": metadataURI,
		}

		batch.Set(p.tokensCol.Doc(docID), data, firestore.MergeAll)
	}

	// mints/{id}: minted=true + mintedAt + 署名
	batch.Update(p.mintsCol.Doc(mintID), []firestore.Update{
		{Path: "minted", Value: true},
		{Path: "mintedAt", Value: firestore.ServerTimestamp},
		{Path: "onChainTxSignature", Value: strings.TrimSpace(lastResult.Signature)},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("batch commit failed in MarkProductsAsMinted mintID=%s: %w", mintID, err)
	}

	return nil
}
