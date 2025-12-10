// backend\internal\adapters\out\firestore\token_repository_fs.go
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
type mintDoc struct {
	BrandID          string   `firestore:"brandId"`
	TokenBlueprintID string   `firestore:"tokenBlueprintId"`
	Products         []string `firestore:"products"`
	Minted           bool     `firestore:"minted"`
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
	}
}

// LoadForMinting は mintID を受け取り、
// mints + tokenBlueprints + brands から MintRequestForUsecase を構築して返します。
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

	// 4) Amount を決める
	// ここは要件次第だが、シンプルに「products の件数」を数量とする例。
	amount := len(m.Products)
	if amount <= 0 {
		// products が空でも最低 1 はミントしたい場合
		amount = 1
	}

	dto := &usecase.MintRequestForUsecase{
		ID:              mintID,
		ToAddress:       toAddress,
		Amount:          uint64(amount),
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
