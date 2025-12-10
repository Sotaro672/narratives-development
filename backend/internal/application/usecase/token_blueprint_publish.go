// backend/internal/application/usecase/token_blueprint_publish.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Arweave 用ポート
// ============================================================

// ArweaveUploader は、メタデータ JSON を Arweave 互換サービスにアップロードし、
// 公開 URI（例: https://arweave.net/xxxx）を返すためのポートです。
type ArweaveUploader interface {
	UploadJSON(ctx context.Context, metadataJSON []byte) (string, error)
}

// ============================================================
// PublishTokenBlueprint メソッド本体
// ============================================================

// PublishTokenBlueprint は、指定された TokenBlueprint を元に
//
//  1. メタデータ JSON を生成
//  2. Arweave（互換サービス）にアップロードして URI を取得
//  3. token_blueprints.{id}.metadataUri を更新
//
// という処理をまとめて行います。
//
// ※ ArweaveUploader / TokenMetadataBuilder は引数で受け取るので、
// 既存の TokenBlueprintUsecase 構造体定義やコンストラクタを変更せずに済みます。
func (u *TokenBlueprintUsecase) PublishTokenBlueprint(
	ctx context.Context,
	uploader ArweaveUploader,
	builder *TokenMetadataBuilder,
	id string,
) (*tbdom.TokenBlueprint, error) {

	if u == nil {
		return nil, fmt.Errorf("token blueprint usecase is nil")
	}
	if uploader == nil || builder == nil {
		return nil, fmt.Errorf("arweave uploader or metadata builder is nil")
	}

	tbID := strings.TrimSpace(id)
	if tbID == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	// 1) TokenBlueprint を取得
	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, fmt.Errorf("tokenBlueprint %s not found", tbID)
	}

	// 2) メタデータ JSON を生成（tb は *TokenBlueprint なので値にして渡す）
	metaJSON, err := builder.BuildFromBlueprint(*tb)
	if err != nil {
		return nil, fmt.Errorf("build metadata: %w", err)
	}

	// 3) Arweave にアップロード
	uri, err := uploader.UploadJSON(ctx, metaJSON)
	if err != nil {
		return nil, fmt.Errorf("upload metadata to arweave: %w", err)
	}

	// 4) TokenBlueprint の metadataUri を更新
	//    RepositoryPort.Update のシグネチャ:
	//      Update(ctx, id, tokenBlueprint.UpdateTokenBlueprintInput) (*TokenBlueprint, error)
	input := tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri, // ★ domain 側にフィールド追加が必要
	}

	updated, err := u.tbRepo.Update(ctx, tbID, input)
	if err != nil {
		return nil, fmt.Errorf("update token blueprint metadataUri: %w", err)
	}

	return updated, nil
}
