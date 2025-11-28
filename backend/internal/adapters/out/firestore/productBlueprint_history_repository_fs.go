// backend/internal/adapters/out/firestore/productBlueprint_history_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintHistoryRepositoryFS implements ProductBlueprintHistoryRepo
// using Firestore の
// product_blueprints_history/{blueprintId}/versions/{1,2,3,...}
// サブコレクションを利用する実装。
type ProductBlueprintHistoryRepositoryFS struct {
	Client *firestore.Client
}

func NewProductBlueprintHistoryRepositoryFS(client *firestore.Client) *ProductBlueprintHistoryRepositoryFS {
	return &ProductBlueprintHistoryRepositoryFS{Client: client}
}

// コンパイル時チェック: interface 満たしているか
var _ pbdom.ProductBlueprintHistoryRepo = (*ProductBlueprintHistoryRepositoryFS)(nil)

// historyCol: product_blueprints_history/{blueprintId}/versions
func (r *ProductBlueprintHistoryRepositoryFS) historyCol(blueprintID string) *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("versions")
}

// SaveSnapshot は、ライブの ProductBlueprint をそのままスナップショットとして保存する。
// - ドキュメントパス: product_blueprints_history/{pb.ID}/versions/{1,2,3,...}
// - UpdatedAt/UpdatedBy は ProductBlueprint 側の値をそのまま利用。
// - 連番 version 自体はドメインには持たず、このリポジトリ内でのみ index として管理する。
func (r *ProductBlueprintHistoryRepositoryFS) SaveSnapshot(
	ctx context.Context,
	pb pbdom.ProductBlueprint,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	blueprintID := strings.TrimSpace(pb.ID)
	if blueprintID == "" {
		return pbdom.ErrInvalidID
	}

	// UpdatedAt / CreatedAt 補完
	if pb.UpdatedAt.IsZero() {
		pb.UpdatedAt = time.Now().UTC()
	}
	if pb.CreatedAt.IsZero() {
		pb.CreatedAt = pb.UpdatedAt
	}

	hCol := r.historyCol(blueprintID)

	// -------------------------
	// ★ 直近の index を取得して nextIndex を決定
	//     - index フィールドで降順ソート → 先頭の index + 1
	//     - 1 件も無ければ 1 から開始
	// -------------------------
	var nextIndex int
	q := hCol.OrderBy("index", firestore.Desc).Limit(1)
	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("SaveSnapshot: get latest index failed: %w", err)
	}

	if len(snaps) == 0 {
		nextIndex = 1
	} else {
		data := snaps[0].Data()
		cur := 0
		if v, ok := data["index"]; ok {
			switch x := v.(type) {
			case int64:
				cur = int(x)
			case int:
				cur = x
			case float64:
				cur = int(x)
			}
		}
		if cur <= 0 {
			nextIndex = 1
		} else {
			nextIndex = cur + 1
		}
	}

	// docID を "1", "2", ... の文字列にする
	docID := fmt.Sprintf("%d", nextIndex)
	docRef := hCol.Doc(docID)

	// 既存の productBlueprintToDoc を流用してフィールド構成を揃える
	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return err
	}

	// history 用のメタ情報
	data["id"] = blueprintID
	data["index"] = nextIndex                     // ★ 連番
	data["historyUpdatedAt"] = pb.UpdatedAt.UTC() // 履歴としての時刻
	if pb.UpdatedBy != nil {
		if s := strings.TrimSpace(*pb.UpdatedBy); s != "" {
			data["historyUpdatedBy"] = s
		}
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return err
	}
	return nil
}

// ListByProductBlueprintID は、指定された productBlueprintID に紐づく
// 履歴 ProductBlueprint 一覧を、新しい順（index 降順）で返す。
// LogCard 側では ProductBlueprint.UpdatedAt / UpdatedBy を利用する想定。
func (r *ProductBlueprintHistoryRepositoryFS) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, pbdom.ErrInvalidID
	}

	// index（1,2,3,...）の降順で取得
	q := r.historyCol(productBlueprintID).OrderBy("index", firestore.Desc)

	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.ProductBlueprint, 0, len(snaps))
	for _, snap := range snaps {
		data := snap.Data()
		if data == nil {
			continue
		}

		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}

		// UpdatedAt / UpdatedBy は、historyUpdatedAt / historyUpdatedBy があればそちらを優先する。
		if t, ok := data["historyUpdatedAt"].(time.Time); ok && !t.IsZero() {
			pb.UpdatedAt = t.UTC()
		}
		if v, ok := data["historyUpdatedBy"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			pb.UpdatedBy = &s
		}

		out = append(out, pb)
	}

	return out, nil
}
