// backend/internal/adapters/out/firestore/mint_request_port_fs.go
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
// Firestore 実装: MintRequestPort
// ========================================
//
// TokenUsecase のために、MintRequest を Firestore から読み書きする
// アダプタです。
// ドメイン型 mintRequest に強く依存しないよう、
// このファイル内で Firestore のドキュメントを直接マッピングしています。
//
// コレクション名やフィールド名はプロジェクトの実態に合わせて
// 適宜修正してください。
// ========================================

// mintRequestDoc は Firestore 上の mintRequests ドキュメントのビューです。
// フィールド名は Firestore の実際のフィールドに合わせて変更してください。
type mintRequestDoc struct {
	Status          string `firestore:"status"`
	ToAddress       string `firestore:"toAddress"`
	Amount          int64  `firestore:"amount"`
	MetadataURI     string `firestore:"metadataUri"`
	BlueprintName   string `firestore:"blueprintName"`
	BlueprintSymbol string `firestore:"blueprintSymbol"`

	// ミント後に更新する想定のフィールド（存在しなくても良い）
	OnChainTxSignature string `firestore:"onChainTxSignature"`
	MintAddress        string `firestore:"mintAddress"`
}

// MintRequestPortFS は Firestore バックエンドの MintRequestPort 実装です。
type MintRequestPortFS struct {
	client        *firestore.Client
	collectionRef *firestore.CollectionRef
}

// コンパイル時に usecase.MintRequestPort を満たしていることを確認
var _ usecase.MintRequestPort = (*MintRequestPortFS)(nil)

// NewMintRequestPortFS は Firestore クライアントから MintRequestPort 実装を生成します.
//
// colName はコレクション名です。デフォルト "mintRequests" を使いたい場合は
// 空文字を渡してください。
func NewMintRequestPortFS(client *firestore.Client, colName string) *MintRequestPortFS {
	if colName == "" {
		colName = "mintRequests" // ※必要に応じて実際のコレクション名に合わせてください
	}
	return &MintRequestPortFS{
		client:        client,
		collectionRef: client.Collection(colName),
	}
}

// LoadForMinting は指定された MintRequest をミント用 DTO に変換して返します。
func (p *MintRequestPortFS) LoadForMinting(
	ctx context.Context,
	id string,
) (*usecase.MintRequestForUsecase, error) {
	if p == nil || p.client == nil || p.collectionRef == nil {
		return nil, fmt.Errorf("MintRequestPortFS is not initialized")
	}

	docID := strings.TrimSpace(id)
	if docID == "" {
		return nil, fmt.Errorf("mint request id is empty")
	}

	snap, err := p.collectionRef.Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("mint request %s not found", docID)
		}
		return nil, fmt.Errorf("get mint request %s: %w", docID, err)
	}

	var mr mintRequestDoc
	if err := snap.DataTo(&mr); err != nil {
		return nil, fmt.Errorf("decode mint request %s: %w", docID, err)
	}

	// TODO: ステータスチェック
	// 例:
	//   if mr.Status != "readyToMint" {
	//       return nil, fmt.Errorf("mint request %s is not ready to mint (status=%s)", docID, mr.Status)
	//   }

	to := strings.TrimSpace(mr.ToAddress)
	if to == "" {
		return nil, fmt.Errorf("mint request %s has empty toAddress", docID)
	}

	amount := mr.Amount
	if amount <= 0 {
		// 0 以下の場合は NFT 前提で 1 をデフォルトにする
		amount = 1
	}

	metadataURI := strings.TrimSpace(mr.MetadataURI)
	if metadataURI == "" {
		return nil, fmt.Errorf("mint request %s has empty metadataUri", docID)
	}

	name := strings.TrimSpace(mr.BlueprintName)
	symbol := strings.TrimSpace(mr.BlueprintSymbol)
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("mint request %s has empty name or symbol", docID)
	}

	dto := &usecase.MintRequestForUsecase{
		ID:              docID,
		ToAddress:       to,
		Amount:          uint64(amount),
		BlueprintName:   name,
		BlueprintSymbol: symbol,
		MetadataURI:     metadataURI,
	}

	return dto, nil
}

// MarkAsMinted は、チェーンミント結果をもとに MintRequest ドキュメントを更新します。
func (p *MintRequestPortFS) MarkAsMinted(
	ctx context.Context,
	id string,
	result *tokendom.MintResult,
) error {
	if p == nil || p.client == nil || p.collectionRef == nil {
		return fmt.Errorf("MintRequestPortFS is not initialized")
	}
	if result == nil {
		return fmt.Errorf("mint result is nil")
	}

	docID := strings.TrimSpace(id)
	if docID == "" {
		return fmt.Errorf("mint request id is empty")
	}

	// Firestore のフィールド名は実態に合わせて変更してください。
	updates := []firestore.Update{
		{
			Path:  "status",
			Value: "minted", // TODO: 実際のステータス値に合わせて調整
		},
		{
			Path:  "onChainTxSignature",
			Value: result.Signature,
		},
		{
			Path:  "mintAddress",
			Value: result.MintAddress,
		},
		{
			Path:  "mintedAt",
			Value: firestore.ServerTimestamp, // サーバー時刻でミント日時を記録
		},
	}

	_, err := p.collectionRef.Doc(docID).Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("mint request %s not found when updating as minted", docID)
		}
		return fmt.Errorf("update mint request %s as minted: %w", docID, err)
	}

	return nil
}
