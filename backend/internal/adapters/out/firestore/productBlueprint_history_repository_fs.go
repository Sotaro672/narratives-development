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
// product_blueprints_history/{blueprintId}/versions/{version}
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
// - ドキュメントパス: product_blueprints_history/{pb.ID}/versions/{pb.Version}
// - version は ProductBlueprint.Version を利用する。
// - UpdatedAt/UpdatedBy は ProductBlueprint 側の値をそのまま利用。
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
	if pb.Version <= 0 {
		return pbdom.ErrInvalidVersion
	}

	// UpdatedAt が空なら現在時刻で補完（ログ用）
	if pb.UpdatedAt.IsZero() {
		pb.UpdatedAt = time.Now().UTC()
	}
	if pb.CreatedAt.IsZero() {
		pb.CreatedAt = pb.UpdatedAt
	}

	docID := fmt.Sprintf("%d", pb.Version)
	docRef := r.historyCol(blueprintID).Doc(docID)

	// 既存の productBlueprintToDoc を流用してフィールド構成を揃える
	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return err
	}

	// history 用のメタ情報
	data["id"] = blueprintID
	data["version"] = pb.Version
	data["historyUpdatedAt"] = pb.UpdatedAt.UTC()
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
// 履歴 ProductBlueprint 一覧を version の降順（新しい順）で返す。
// LogCard 側では ProductBlueprint.Version / UpdatedAt / UpdatedBy を利用する想定。
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

	q := r.historyCol(productBlueprintID).OrderBy("version", firestore.Desc)

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

		// version は docToProductBlueprint 内で "version" を読んだ値が入っている想定だが、
		// 念のため docID から復元するフォールバックはここでは行わず、
		// 0 の場合はそのまま（古いデータとして扱う）。
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
