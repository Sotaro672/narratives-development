// backend/internal/adapters/out/firestore/model_history_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"log"
	"time"

	model "narratives/internal/domain/model"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// ModelHistoryRepositoryFS は models のバージョンスナップショットを
// Firestore に保存・取得するための実装です。
//
// パス構造:
//
//	product_blueprints_history/{blueprintID}/models/{version}/variations/{variationID}
type ModelHistoryRepositoryFS struct {
	Client *firestore.Client
}

func NewModelHistoryRepositoryFS(client *firestore.Client) *ModelHistoryRepositoryFS {
	return &ModelHistoryRepositoryFS{Client: client}
}

// ベースコレクションのヘルパー（バージョンごとの variations サブコレクション）
func (r *ModelHistoryRepositoryFS) historyVariationsCol(
	blueprintID string,
	version int64,
) *firestore.CollectionRef {
	versionDocID := fmt.Sprintf("%d", version)

	return r.Client.
		Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("models").
		Doc(versionDocID).
		Collection("variations")
}

// SaveSnapshot:
//
// 指定された blueprintID + blueprintVersion に対して、
// variations（ライブの ModelVariation 一式）のスナップショットを
// 1) variations サブコレクション
// 2) models/{version} ドキュメント本体（サマリ）
// の両方に保存する。
func (r *ModelHistoryRepositoryFS) SaveSnapshot(
	ctx context.Context,
	blueprintID string,
	blueprintVersion int64,
	variations []model.ModelVariation,
) error {
	if r.Client == nil {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: firestore client is nil")
	}
	if blueprintID == "" {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: blueprintID is empty")
	}
	if blueprintVersion <= 0 {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: blueprintVersion must be > 0")
	}

	log.Printf(
		"[ModelHistoryRepositoryFS] SaveSnapshot blueprintID=%s version=%d variations=%d",
		blueprintID, blueprintVersion, len(variations),
	)

	col := r.historyVariationsCol(blueprintID, blueprintVersion)

	batch := r.Client.Batch()
	now := time.Now().UTC()

	// models/{version} ドキュメントに格納するサマリ用配列
	var snapshotList []map[string]any

	for _, v := range variations {
		if v.ID == "" {
			// ID 無しはスキップ
			continue
		}

		docRef := col.Doc(v.ID)

		// variations サブコレクションに保存するデータ
		data := map[string]any{
			"id":                 v.ID,
			"productBlueprintId": v.ProductBlueprintID,
			"modelNumber":        v.ModelNumber,
			"size":               v.Size,
			"color": map[string]any{
				"name": v.Color.Name,
				"rgb":  v.Color.RGB,
			},
			"measurements": v.Measurements,
			"version":      v.Version,
			"createdAt":    v.CreatedAt,
			"createdBy":    v.CreatedBy,
			"updatedAt":    v.UpdatedAt,
			"updatedBy":    v.UpdatedBy,
		}

		// UpdatedAt がゼロなら now で補完
		if t, ok := data["updatedAt"].(time.Time); !ok || t.IsZero() {
			data["updatedAt"] = now
		}

		batch.Set(docRef, data)

		// models/{version} ドキュメントに格納する「サマリ」も作成
		snap := map[string]any{
			"id":                 v.ID,
			"productBlueprintId": v.ProductBlueprintID,
			"modelNumber":        v.ModelNumber,
			"size":               v.Size,
			"color": map[string]any{
				"name": v.Color.Name,
				"rgb":  v.Color.RGB,
			},
			"measurements": v.Measurements,
			"version":      v.Version,
		}
		snapshotList = append(snapshotList, snap)
	}

	// variations サブコレクションを一括書き込み
	if _, err := batch.Commit(ctx); err != nil {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: batch.Commit: %w", err)
	}

	// models/{version} ドキュメント本体にもサマリを保存
	versionDocID := fmt.Sprintf("%d", blueprintVersion)
	metaRef := r.Client.
		Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("models").
		Doc(versionDocID)

	meta := map[string]any{
		"productBlueprintId": blueprintID,
		"version":            blueprintVersion,
		"variationCount":     len(variations),
		"createdAt":          now,
		"variations":         snapshotList, // ★ 各バージョンの中身をここに格納
	}

	// 既存フィールドがあっても上書きマージで更新
	if _, err := metaRef.Set(ctx, meta, firestore.MergeAll); err != nil {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: meta Set: %w", err)
	}

	log.Printf(
		"[ModelHistoryRepositoryFS] SaveSnapshot completed blueprintID=%s version=%d",
		blueprintID, blueprintVersion,
	)

	return nil
}

// ListByProductBlueprintIDAndVersion:
//
// 指定された blueprintID + version に紐づく ModelVariation の履歴をすべて返す。
func (r *ModelHistoryRepositoryFS) ListByProductBlueprintIDAndVersion(
	ctx context.Context,
	blueprintID string,
	version int64,
) ([]model.ModelVariation, error) {
	if r.Client == nil {
		return nil, fmt.Errorf("ModelHistoryRepositoryFS.ListByProductBlueprintIDAndVersion: firestore client is nil")
	}
	if blueprintID == "" {
		return nil, fmt.Errorf("ModelHistoryRepositoryFS.ListByProductBlueprintIDAndVersion: blueprintID is empty")
	}
	if version <= 0 {
		return nil, fmt.Errorf("ModelHistoryRepositoryFS.ListByProductBlueprintIDAndVersion: version must be > 0")
	}

	col := r.historyVariationsCol(blueprintID, version)
	iter := col.Documents(ctx)
	defer iter.Stop()

	var out []model.ModelVariation

	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("ModelHistoryRepositoryFS.ListByProductBlueprintIDAndVersion: iter.Next: %w", err)
		}

		var mv model.ModelVariation

		// Firestore → struct へデコード
		if err := doc.DataTo(&mv); err != nil {
			// うまくデコードできない場合は最低限 ID だけでも入れてスキップ
			mv = model.ModelVariation{
				ID: doc.Ref.ID,
			}
		}

		// 念のため ID が空のときは DocID を補完
		if mv.ID == "" {
			mv.ID = doc.Ref.ID
		}

		out = append(out, mv)
	}

	return out, nil
}
