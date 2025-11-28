// backend/internal/adapters/out/firestore/model_history_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// ModelHistoryRepositoryFS
// ------------------------------------------------------------
//
// Firestore 上の models_history コレクションに、
// ModelVariation の履歴スナップショットを保存するためのリポジトリです。
// スキーマ構成は product_blueprints_history と同じく、
//
//   models_history/{productBlueprintID}/versions/{version}/variations/{variationID}
//
// という 3 階層構造を想定しています。
// ------------------------------------------------------------

type ModelHistoryRepositoryFS struct {
	Client *firestore.Client
}

// NewModelHistoryRepositoryFS は Firestore クライアントから
// ModelHistoryRepositoryFS を生成します。
func NewModelHistoryRepositoryFS(client *firestore.Client) *ModelHistoryRepositoryFS {
	return &ModelHistoryRepositoryFS{Client: client}
}

// ルートコレクション: models_history
func (r *ModelHistoryRepositoryFS) modelsHistoryCol() *firestore.CollectionRef {
	return r.Client.Collection("models_history")
}

// 特定の productBlueprintID 配下の versions サブコレクション
func (r *ModelHistoryRepositoryFS) versionsCol(productBlueprintID string) *firestore.CollectionRef {
	id := strings.TrimSpace(productBlueprintID)
	return r.modelsHistoryCol().Doc(id).Collection("versions")
}

// ------------------------------------------------------------
// SaveVariationHistory
// ------------------------------------------------------------
//
// 1 件の ModelVariation のスナップショットを Firestore に保存します。
// - productBlueprintID ごとにドキュメントをまとめる
// - version ごとに subcollection「versions/{version}」を掘る
// - その配下に subcollection「variations/{variationID}」として保存
//
//	models_history/{productBlueprintID}/versions/{version}/variations/{variationID}
//
// ------------------------------------------------------------
func (r *ModelHistoryRepositoryFS) SaveVariationHistory(
	ctx context.Context,
	v modeldom.ModelVariation,
) error {
	if r.Client == nil {
		return errors.New("ModelHistoryRepositoryFS: Firestore client is nil")
	}

	pbID := strings.TrimSpace(v.ProductBlueprintID)
	if pbID == "" {
		return modeldom.ErrInvalidBlueprintID
	}

	// Version は 1 以上を想定（0 や負値の場合はエラー扱い）
	if v.Version <= 0 {
		return errors.New("ModelHistoryRepositoryFS: invalid version (must be > 0)")
	}

	variationID := strings.TrimSpace(v.ID)
	if variationID == "" {
		return modeldom.ErrInvalidID
	}

	// version ドキュメントIDは単純に数値を文字列化（product_blueprints_history と揃える）
	versionDocID := fmt.Sprintf("%d", v.Version)

	docRef := r.
		versionsCol(pbID).
		Doc(versionDocID).
		Collection("variations").
		Doc(variationID)

	// ModelVariation 構造体そのものを保存（firestore タグに従ってマッピングされる想定）
	_, err := docRef.Set(ctx, v)
	if err != nil {
		return fmt.Errorf("ModelHistoryRepositoryFS: failed to save variation history: %w", err)
	}

	return nil
}
