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

// ModelHistoryRepositoryFS は models のスナップショットを
// Firestore に保存・取得するための実装です。
//
// パス構造:
//
//	product_blueprints_history/{blueprintID}/models/{version}/variations/{variationID}
//
// version は 1,2,3,... の連番として自動採番する。
type ModelHistoryRepositoryFS struct {
	Client *firestore.Client
}

func NewModelHistoryRepositoryFS(client *firestore.Client) *ModelHistoryRepositoryFS {
	return &ModelHistoryRepositoryFS{Client: client}
}

// models コレクション（バージョンごとの metadata）へのヘルパー
func (r *ModelHistoryRepositoryFS) modelsCol(blueprintID string) *firestore.CollectionRef {
	return r.Client.
		Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("models")
}

// ベースコレクションのヘルパー（version ごとの variations サブコレクション）
func (r *ModelHistoryRepositoryFS) historyVariationsCol(
	blueprintID string,
	version int64,
) *firestore.CollectionRef {
	versionDocID := fmt.Sprintf("%d", version)

	return r.modelsCol(blueprintID).
		Doc(versionDocID).
		Collection("variations")
}

// 最新バージョン番号を取得する（なければ 0 を返す）
func (r *ModelHistoryRepositoryFS) getLatestVersion(
	ctx context.Context,
	blueprintID string,
) (int64, error) {
	col := r.modelsCol(blueprintID)

	// version フィールドの降順で 1 件だけ取得
	iter := col.OrderBy("version", firestore.Desc).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			// まだ 1 件も無い場合は 0 とみなす
			return 0, nil
		}
		return 0, err
	}

	data := doc.Data()
	if data == nil {
		return 0, nil
	}

	var v int64
	switch x := data["version"].(type) {
	case int64:
		v = x
	case int:
		v = int64(x)
	case float64:
		v = int64(x)
	default:
		v = 0
	}

	return v, nil
}

// SaveSnapshot:
//
// 指定された blueprintID に対して、
// variations（ライブの ModelVariation 一式）のスナップショットを
// 1) versions サブコレクション（models/{version}/variations/*）
// 2) models/{version} ドキュメント本体（サマリ）
// の両方に保存する。
func (r *ModelHistoryRepositoryFS) SaveSnapshot(
	ctx context.Context,
	blueprintID string,
	variations []model.ModelVariation,
) error {
	if r.Client == nil {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: firestore client is nil")
	}
	if blueprintID == "" {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: blueprintID is empty")
	}

	// 直近の version を取得し、次の version を決定（1 からスタート）
	latestVersion, err := r.getLatestVersion(ctx, blueprintID)
	if err != nil {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: getLatestVersion: %w", err)
	}
	newVersion := latestVersion + 1

	log.Printf(
		"[ModelHistoryRepositoryFS] SaveSnapshot blueprintID=%s version=%d variations=%d",
		blueprintID, newVersion, len(variations),
	)

	col := r.historyVariationsCol(blueprintID, newVersion)
	now := time.Now().UTC()

	// models/{version} ドキュメントに格納するサマリ用配列
	var snapshotList []map[string]any

	// variations サブコレクションをトランザクションで一括書き込み
	if err := r.Client.RunTransaction(ctx, func(txCtx context.Context, tx *firestore.Transaction) error {
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
				"createdAt":    v.CreatedAt,
				"createdBy":    v.CreatedBy,
				"updatedAt":    v.UpdatedAt,
				"updatedBy":    v.UpdatedBy,
			}

			// UpdatedAt がゼロなら now で補完
			if t, ok := data["updatedAt"].(time.Time); !ok || t.IsZero() {
				data["updatedAt"] = now
			}

			if err := tx.Set(docRef, data); err != nil {
				return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: tx.Set variation: %w", err)
			}

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
			}
			snapshotList = append(snapshotList, snap)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("ModelHistoryRepositoryFS.SaveSnapshot: transaction (variations) failed: %w", err)
	}

	// models/{version} ドキュメント本体にもサマリを保存
	metaRef := r.modelsCol(blueprintID).Doc(fmt.Sprintf("%d", newVersion))

	meta := map[string]any{
		"productBlueprintId": blueprintID,
		"version":            newVersion,
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
		blueprintID, newVersion,
	)

	return nil
}

// ListByProductBlueprintID:
//
// 指定された blueprintID に紐づく **最新バージョン** の
// ModelVariation 履歴をすべて返す。
func (r *ModelHistoryRepositoryFS) ListByProductBlueprintID(
	ctx context.Context,
	blueprintID string,
) ([]model.ModelVariation, error) {
	if r.Client == nil {
		return nil, fmt.Errorf("ModelHistoryRepositoryFS.ListByProductBlueprintID: firestore client is nil")
	}
	if blueprintID == "" {
		return nil, fmt.Errorf("ModelHistoryRepositoryFS.ListByProductBlueprintID: blueprintID is empty")
	}

	// 最新 version を取得
	latestVersion, err := r.getLatestVersion(ctx, blueprintID)
	if err != nil {
		return nil, fmt.Errorf("ModelHistoryRepositoryFS.ListByProductBlueprintID: getLatestVersion: %w", err)
	}
	if latestVersion == 0 {
		// まだ履歴がない場合は空配列
		return []model.ModelVariation{}, nil
	}

	col := r.historyVariationsCol(blueprintID, latestVersion)
	iter := col.Documents(ctx)
	defer iter.Stop()

	var out []model.ModelVariation

	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("ModelHistoryRepositoryFS.ListByProductBlueprintID: iter.Next: %w", err)
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
