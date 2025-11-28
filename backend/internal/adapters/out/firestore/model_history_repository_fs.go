// backend/internal/adapters/out/firestore/model_history_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	modeldom "narratives/internal/domain/model"
)

// ============================================================
// Firestore implementation for ModelHistoryRepo
// ============================================================
//
// 保存先：product_blueprints_history/{blueprintId}/models/{version}/variations/{variationId}
//
// version は productBlueprint の version に合わせる
//
// ============================================================

type ModelHistoryRepositoryFS struct {
	Client *firestore.Client
}

func NewModelHistoryRepositoryFS(client *firestore.Client) *ModelHistoryRepositoryFS {
	return &ModelHistoryRepositoryFS{Client: client}
}

var _ modeldom.ModelHistoryRepo = (*ModelHistoryRepositoryFS)(nil)

// variations history collection path
func (r *ModelHistoryRepositoryFS) variationsCol(blueprintID string, version int64) *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("models").
		Doc(fmt.Sprintf("%d", version)).
		Collection("variations")
}

// ============================================================
// SaveSnapshot
// ============================================================
//
// blueprintVersion に紐づく ModelVariation の完全スナップショットを保存する。
// 保存先：
//
//	product_blueprints_history/{blueprintId}/models/{version}/variations/{variationId}
func (r *ModelHistoryRepositoryFS) SaveSnapshot(
	ctx context.Context,
	blueprintID string,
	blueprintVersion int64,
	variations []modeldom.ModelVariation,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return modeldom.ErrInvalidBlueprintID
	}
	if blueprintVersion <= 0 {
		return modeldom.ErrInvalidVersion
	}

	col := r.variationsCol(blueprintID, blueprintVersion)

	// variations が空の場合は何も保存しない（要件に合わせる）
	for _, v := range variations {
		docRef := col.Doc(v.ID)

		// UpdatedAt/CreatedAt が空なら補完
		if v.CreatedAt.IsZero() {
			v.CreatedAt = time.Now().UTC()
		}
		if v.UpdatedAt.IsZero() {
			v.UpdatedAt = v.CreatedAt
		}

		// Firestore 用 map へ変換
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
			"createdAt":    v.CreatedAt.UTC(),
			"updatedAt":    v.UpdatedAt.UTC(),
		}

		if v.CreatedBy != nil {
			data["createdBy"] = strings.TrimSpace(*v.CreatedBy)
		}
		if v.UpdatedBy != nil {
			data["updatedBy"] = strings.TrimSpace(*v.UpdatedBy)
		}

		if _, err := docRef.Set(ctx, data); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// ListByProductBlueprintIDAndVersion
// ============================================================
//
// 指定 blueprintId + version に紐づく ModelVariation の履歴をすべて返す。
// つまり LogCard に表示するための Model バージョン一覧。
// ============================================================

func (r *ModelHistoryRepositoryFS) ListByProductBlueprintIDAndVersion(
	ctx context.Context,
	blueprintID string,
	version int64,
) ([]modeldom.ModelVariation, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}
	if version <= 0 {
		return nil, modeldom.ErrInvalidVersion
	}

	col := r.variationsCol(blueprintID, version)

	snaps, err := col.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]modeldom.ModelVariation, 0, len(snaps))

	for _, snap := range snaps {
		data := snap.Data()
		if data == nil {
			continue
		}

		getStr := func(k string) string {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
			return ""
		}

		// Color
		var color modeldom.Color
		if c, ok := data["color"].(map[string]any); ok {
			if n, ok := c["name"].(string); ok {
				color.Name = strings.TrimSpace(n)
			}
			switch rv := c["rgb"].(type) {
			case int64:
				color.RGB = int(rv)
			case int:
				color.RGB = rv
			case float64:
				color.RGB = int(rv)
			}
		}

		// measurements
		measurements := modeldom.Measurements{}
		if raw, ok := data["measurements"].(map[string]any); ok {
			for k, v := range raw {
				switch x := v.(type) {
				case int64:
					measurements[k] = int(x)
				case int:
					measurements[k] = x
				case float64:
					measurements[k] = int(x)
				}
			}
		}

		mv := modeldom.ModelVariation{
			ID:                 getStr("id"),
			ProductBlueprintID: getStr("productBlueprintId"),
			ModelNumber:        getStr("modelNumber"),
			Size:               getStr("size"),
			Color:              color,
			Measurements:       measurements,
		}

		if t, ok := data["createdAt"].(time.Time); ok {
			mv.CreatedAt = t.UTC()
		}
		if t, ok := data["updatedAt"].(time.Time); ok {
			mv.UpdatedAt = t.UTC()
		}
		if s := getStr("createdBy"); s != "" {
			mv.CreatedBy = &s
		}
		if s := getStr("updatedBy"); s != "" {
			mv.UpdatedBy = &s
		}

		out = append(out, mv)
	}

	return out, nil
}
